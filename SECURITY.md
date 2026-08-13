# Security Guide — Guardian Tracker

## Reporting Security Issues

If you discover a security vulnerability, **do not open a public issue**. Email `jerryholland00@gmail.com` with details. You can expect a response within 72 hours.

---

## Credential Management

This is a public repository. Keep production runbooks, incident notes,
environment-specific commands, private security reviews, and raw research under
gitignored `private/`; do not commit them to public docs.

### CRITICAL: Rotate Credentials If Ever Committed

If you have previously committed credentials to this repository, you **must**:

1. **Rotate ALL credentials immediately:**

   - Bungie API Key: Generate a new key at <https://www.bungie.net/en/Application>
   - Bungie Client Secret: Regenerate in your Bungie application settings
   - JWT Secret: Generate a new 32+ character random string
   - Database passwords: Change in your database and update configs

2. **Remove credentials from git history:**

   ```bash
   # Option 1: BFG Repo Cleaner (recommended, much faster)
   bfg --delete-files .env
   bfg --replace-text passwords.txt  # File listing secrets to scrub

   # Option 2: git filter-branch (slower)
   git filter-branch --force --index-filter \
     "git rm --cached --ignore-unmatch .env" \
     --prune-empty --tag-name-filter cat -- --all
   ```

3. **Force push and notify all collaborators:**

   ```bash
   git push origin --force --all
   git push origin --force --tags
   ```

---

## Environment Variables

### Secrets (Never Commit!)

| Variable                        | Description                                                           | Where to Get                                                 |
| ------------------------------- | --------------------------------------------------------------------- | ------------------------------------------------------------ |
| `BUNGIE_API_KEY`                | Bungie API access key                                                 | [Bungie Applications](https://www.bungie.net/en/Application) |
| `BUNGIE_CLIENT_ID`              | OAuth client ID                                                       | Bungie application settings                                  |
| `BUNGIE_CLIENT_SECRET`          | OAuth client secret                                                   | Bungie application settings                                  |
| `JWT_SECRET`                    | Token signing key (32+ chars); also derives the OAuth state HMAC key  | `openssl rand -base64 32`                                    |
| `POSTGRES_PASSWORD`             | Database password                                                     | Generate a strong random password                            |
| `DATABASE_URL`                  | Postgres connection string (contains credentials)                     | Compose your provider's connection string                    |
| `TOKEN_ENCRYPTION_KEY`          | 32-byte base64 key — AES-256-GCM encryption of stored Bungie tokens   | `openssl rand -base64 32`                                    |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS` | (optional) previous encryption key, kept readable during key rotation | the key being rotated out                                    |

`GO_ENV` is required and accepts exactly `development` or `production`.
Production refuses to start without Postgres and token encryption; explicit
development may run with those protections disabled and emits one conspicuous
warning describing every disabled protection.

Encryption keys also have explicit positive `SMALLINT` versions:

| Variable                                | Purpose                                                           |
| --------------------------------------- | ----------------------------------------------------------------- |
| `TOKEN_ENCRYPTION_KEY_VERSION`          | Current-key ciphertext version (`1` by default for existing rows) |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION` | Exact version accepted for the optional previous key              |

### Local Development

```bash
cp .env.example .env
# Edit with development values — NEVER use production credentials locally
```

### Minikube Validation

The manifests under `k8s/` run only as a local development-validation stack.
They use `GO_ENV=development`, omit Postgres, and are not a production deployment
or secret-management guide. Never place production credentials in those
manifests. A production runtime and its secret-management procedure remain
deferred until a deployment target is selected.

---

## Implemented Security Features

### Authentication & Authorization

- **Bungie OAuth 2.0** with CSRF protection — stateless HMAC-SHA256-signed state parameter (`v1.<ts>.<nonce>.<sig>`, key derived from `JWT_SECRET` with domain separation), verified with a 10-minute TTL. Stateless by design: login survives restarts and works across replicas. Tradeoff: a state token is replayable within its TTL (the previous in-memory store was not browser-bound either, and the Bungie authorization code it accompanies is single-use)
- **JWT tokens** — HS256 signed access token (30-minute default) plus a rotating 30-day refresh credential; `tver` (token_version), `jti`, and `sid` (session id) claims. The access lifetime is configurable with the duration-based `JWT_ACCESS_TTL` setting; `JWT_EXPIRY_HOURS` remains a legacy fallback when the new setting is absent. Tokens already issued retain their original expiry until they are refreshed or revoked.
- **Browser credential split** — the access JWT and non-secret user snapshot remain in `localStorage`. The refresh JWT is delivered only as the host-only `guardian_refresh_token` cookie with `HttpOnly`, `SameSite=Lax`, and `Path=/api/auth`; production also sets `Secure`. Callback and refresh responses contain `{token,user}` and never expose the refresh token to JavaScript. There is no localStorage compatibility bridge, so sessions created before this migration sign in once again.
- **Per-device refresh sessions + reuse detection** — each login opens a row in `refresh_sessions` (the `sid` claim) holding the current refresh `jti`. `POST /api/auth/refresh` compare-and-swaps it (`RotateSession`); a replayed (already-rotated) refresh token is detected as reuse and the **whole session is revoked** with 401, rather than staying valid to expiry. Sessions are independent, so this is fully multi-device. The CAS **fails open** on DB errors, consistent with revocation below — except a definitive reuse always 401s. Session `expires_at` slides forward on each rotation to match the freshly issued refresh token, so an active session is not force-expired at its original creation time. Sessions are capped per user (oldest-by-use evicted) and a background pruner deletes expired rows, so the table can't grow unbounded. If a session row can't be persisted at login because the write itself failed, the login fails (the session is load-bearing — the access token is checked against it on every request); a deployment with no database configured is the one exception, where login succeeds without a session row since nothing checks for one there
- **Two logout scopes** — `POST /api/auth/logout` ends only the current device's session (others stay signed in; the access token is rejected within the cache window via the session check, and the Bungie OAuth token is preserved). `POST /api/auth/logout/all` bumps `token_version`, deletes every session, and evicts the Bungie token (sign out everywhere). Both responses expire the refresh cookie; definitive refresh failures expire it too.
- **JWT revocation** — sign-out-everywhere bumps `token_version` in Postgres; the access-token middleware verifies both `token_version` (Guardian Tracker user-wide) and session existence (per-device) via `RevocationChecker` with a 60-second cache window. The checks **fail open** on DB errors (availability over strict revocation — logout is not guaranteed during a DB outage)
- **Logout exposure window** — with the default 30-minute access lifetime, an access token whose session revocation cannot be observed immediately is bounded by the token lifetime plus the 60-second revocation cache window; the frontend silently refreshes expired access tokens while the refresh session remains valid
- **Token-type claims** — refresh tokens cannot be used as access tokens (enforced in middleware)
- **Bungie tokens encrypted at rest** — AES-256-GCM in the `bungie_tokens` table, with the membership row bound as AAD and exact key-version metadata stored with each encrypted row. Decryption accepts only an exact current- or previous-version match and rejects unknown versions.
- **Bungie token auto-refresh** — stored Bungie OAuth tokens are refreshed automatically before expiry (5-min buffer); refresh writes are compare-and-swap on `updated_at`, and a replica that loses the race adopts the winner's tokens instead of clobbering them

### Token-encryption key rotation

Version numbers identify keys; never reuse a version for different key material.
Existing version-1 rows remain readable during the first rotation:

1. Start with key A as `TOKEN_ENCRYPTION_KEY`, version `1`, and no previous key.
2. Generate key B, then deploy B as the current key/version `2` while retaining
   A as the previous key/version `1`:

   ```env
   TOKEN_ENCRYPTION_KEY=<key-B>
   TOKEN_ENCRYPTION_KEY_VERSION=2
   TOKEN_ENCRYPTION_KEY_PREVIOUS=<key-A>
   TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION=1
   ```

3. New or refreshed rows are encrypted with B/v2; existing A/v1 rows continue
   to decrypt only through the exact previous-version match.
4. Keep A/v1 configured until no `bungie_tokens.key_version = 1` rows remain.
   Rows are rewritten through normal login/token persistence; users whose rows
   are removed must authenticate again.
5. Remove both previous-key variables only after version 1 is no longer stored.

Unknown, zero, negative, duplicate, or keyless versions are configuration/data
errors; do not change a version independently of its key.

### Roles, Feature Flags & Admin Console

- **Role tiers** — `standard(0) < beta(1) < alpha(2) < admin(3)` stored in `users.role`. Authorization **always reads the role from the DB** (the `RevocationChecker` cache now holds `{token_version, role}`), never from the JWT — the JWT carries role only as a display hint. Role changes therefore take effect within the 60-second cache window with no token churn.
- **Admin bootstrap** — admin can only be minted via `ADMIN_MEMBERSHIP_IDS` (comma-separated Bungie membership IDs pinned to admin on every login upsert) or granted by an existing admin. There is **no self-service path to admin**: `PUT /api/account/role` accepts only standard/beta/alpha and rejects admin callers. `ADMIN_MEMBERSHIP_IDS` is non-secret config (membership IDs), but treat it as sensitive — it controls who holds admin.
- **Server-side enforcement** — `RequireAdmin`/`RequireTier` gate admin and tier-locked endpoints, and `RequireFlag` enforces feature-flag access (see _Feature-flag enforcement_ below); UI hiding mirrors these gates rather than being the boundary. In degraded mode (no DB) roles resolve to standard and admin/tier endpoints return 503 (flag gates fail open).
- **Last-admin protection** — demoting the final admin is refused inside the same transaction (all admin rows are `SELECT … FOR UPDATE`-locked so concurrent demotions can't race the system to zero admins).
- **Audit trail** — authentication events (login success/failure, logout, logout-all, token refresh), session security events (refresh-token reuse, session termination), self opt-in role changes, admin role changes, and feature-flag changes are persisted to the unified `audit_log` table and exposed to admins via `GET /api/admin/audit` and the `/admin` Audit Log UI panel. Role and flag changes are written in their mutation transaction (atomic); auth/session events are best-effort (a DB outage can drop an event). **Client IP address and User-Agent are captured and stored** for security forensics, retained for `AUDIT_RETENTION_DAYS` (default 180), and pruned hourly once past expiry. IP addresses are trusted only from `TRUSTED_PROXIES` (gin `SetTrustedProxies`) so they cannot be spoofed by a client; configure this in production to your platform's ingress range.

### Feature-flag enforcement

- **Enforced routes** — `RequireFlag` (`auth/roles.go`) gates the routes whose features the UI also gates: `weekly-planner` (`/api/weekly/recommendations`), `global-search` (`/api/items/search`), `catalysts-crafting` (`/api/catalysts`, `/api/crafting`), and `triumphs-seals` (`/api/seals`). A disabled flag returns `404 FEATURE_DISABLED`; an enabled flag above the caller's tier returns `403 TIER_LOCKED`.
- **Fail-open** — if the flag store is unavailable (degraded/DB-less mode), a lookup errors, or the key is unknown, the request is allowed. Feature flags are rollout and upsell controls, not security boundaries — a flag-table hiccup must not take down core pages. Real access boundaries (the admin console) use `RequireAdmin`, which fails closed (503) when roles cannot be resolved.

### Input Validation

- **Membership ID validation**: numeric-only, 10–25 chars
- **Membership type allowlist** — only valid Bungie platform types accepted
- **Authorization code length limit** — max 500 chars to prevent DoS on OAuth callback

### Rate Limiting

- **Bungie API client**: configurable RPS (default 10 req/s, burst 20) with rate.Limiter
- **Bungie API retries**: exponential backoff with Retry-After header respect on 429 responses

### CORS

- **Strict origin validation** — only explicitly configured origins allowed (set via `CORS_ALLOWED_ORIGINS`)
- **Credentials**: allowed only with explicit origin match
- **Credential-issuing endpoints** — `POST /api/auth/bungie/callback` and `POST /api/auth/refresh` require an exact allowlisted `Origin`; missing or unlisted origins are rejected. CORS responses vary on `Origin`.

The refresh-cookie design assumes the frontend and API are same-site. A future
cross-site production topology must revisit the cookie policy and would require
`SameSite=None; Secure` plus an explicit CSRF design before deployment.

### HTTP Security (Go service)

- **Server timeouts**: ReadTimeout 30s, WriteTimeout 60s, IdleTimeout 120s — prevents slowloris-style attacks
- **Graceful shutdown** with 30s timeout
- **API response headers**: every API response sets `X-Content-Type-Options: nosniff` and `Referrer-Policy: no-referrer`; auth responses also set `Cache-Control: no-store`
- **Frontend CSP**: inline scripts are disallowed, object embedding is disabled, the base URI is restricted to self, and framing is limited to self. Google Fonts origins used by the app are allowlisted. `style-src 'unsafe-inline'` remains a documented residual XSS-hardening risk while the current component system still uses inline styles.
- **Request correlation**: the server assigns every request a UUID and returns it as `X-Request-ID`; CORS exposes the header to allowed browser origins.
- **Application-log privacy**: access records use route templates and omit query strings, bodies, authorization headers, User-Agent values, and routine client IPs. Membership, session, user, and character identifiers are deterministic 24-hex pseudonyms outside the exact PostgreSQL security audit trail.

### CI supply-chain controls

- **Immutable workflow dependencies**: every third-party GitHub Action is pinned to a reviewed 40-character release commit with a readable release-version comment. A repository policy test rejects moving tags or missing comments.
- **Automated pin maintenance**: the `github-actions` Dependabot ecosystem advances action SHAs and release comments together.
- **Reproducible Go vulnerability scan**: `govulncheck` is declared in the backend Go module at v1.6.0 and CI invokes it with `go tool`, allowing Go-module Dependabot updates without an unbounded `@latest` install.
- **Reproducible frontend tooling**: `.nvmrc` selects one exact Node 26 patch for
  local and CI tooling. The production builder and development image pin that
  patch on Alpine 3.24 to a multi-platform OCI index digest, and clean container
  installs use `npm ci`. npm enforces the Node 26 engine range, and a repository
  policy test rejects drift between these pins, the workflows, package engine
  metadata, and Node ambient types.

---

## Production Security Checklist

- [ ] All secrets rotated and removed from git history
- [ ] `JWT_SECRET` is 32+ characters, randomly generated
- [ ] `CORS_ALLOWED_ORIGINS` set to your production domain only
- [ ] `GO_ENV=production` on the API service
- [ ] Current and previous encryption keys have the exact positive versions intended for their stored rows
- [ ] Rate limiting enabled and tuned for expected traffic
- [ ] TLS/HTTPS configured (terminate at load balancer or ingress)
- [ ] Database connections use SSL (`sslmode=require`)
- [ ] Health endpoints (`/health`, `/ready`) expose no sensitive data; safe to expose by design on the API ingress
- [ ] Logging does not include tokens, secrets, or full OAuth codes
- [ ] Docker images built from pinned base image versions
- [ ] Kubernetes secrets not stored in version control
- [ ] `DATABASE_URL` and `TOKEN_ENCRYPTION_KEY` set (the service refuses to start in production without them); current key version verified (`1` when omitted)
