# Security Guide — Guardian Tracker

## Reporting Security Issues

If you discover a security vulnerability, **do not open a public issue**. Email `jerryholland00@gmail.com` with details. You can expect a response within 72 hours.

---

## Credential Management

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

### Local Development

```bash
cp .env.example .env
# Edit with development values — NEVER use production credentials locally
```

### Kubernetes Production

```bash
# Create from literal values (do not put in scripts or version control)
kubectl create secret generic api-service-secrets \
  --from-literal=BUNGIE_API_KEY=your_key \
  --from-literal=BUNGIE_CLIENT_ID=your_id \
  --from-literal=BUNGIE_CLIENT_SECRET=your_secret \
  --from-literal=JWT_SECRET=your_jwt_secret \
  --from-literal=POSTGRES_PASSWORD=your_db_password

# Or from a secure temp file (delete immediately after)
kubectl create secret generic api-service-secrets --from-env-file=.env.secrets
rm .env.secrets
```

---

## Implemented Security Features

### Authentication & Authorization

- **Bungie OAuth 2.0** with CSRF protection — stateless HMAC-SHA256-signed state parameter (`v1.<ts>.<nonce>.<sig>`, key derived from `JWT_SECRET` with domain separation), verified with a 10-minute TTL. Stateless by design: login survives restarts and works across replicas. Tradeoff: a state token is replayable within its TTL (the previous in-memory store was not browser-bound either, and the Bungie authorization code it accompanies is single-use)
- **JWT tokens** — HS256 signed, access token (24h) + refresh token (30d) with rotation; `tver` (token_version), `jti`, and `sid` (session id) claims
- **Per-device refresh sessions + reuse detection** — each login opens a row in `refresh_sessions` (the `sid` claim) holding the current refresh `jti`. `POST /api/auth/refresh` compare-and-swaps it (`RotateSession`); a replayed (already-rotated) refresh token is detected as reuse and the **whole session is revoked** with 401, rather than staying valid to expiry. Sessions are independent, so this is fully multi-device. The CAS **fails open** on DB errors, consistent with revocation below — except a definitive reuse always 401s. Session `expires_at` slides forward on each rotation to match the freshly issued refresh token, so an active session is not force-expired at its original creation time. Sessions are capped per user (oldest-by-use evicted) and a background pruner deletes expired rows, so the table can't grow unbounded. If a session row can't be persisted at login, the login itself fails (the session is load-bearing — the access token is checked against it on every request)
- **Two logout scopes** — `POST /api/auth/logout` ends only the current device's session (others stay signed in; the access token is rejected within the cache window via the session check, and the account-wide Bungie token is preserved). `POST /api/auth/logout/all` bumps `token_version`, deletes every session, and evicts the Bungie token (sign out everywhere)
- **JWT revocation** — sign-out-everywhere bumps `token_version` in Postgres; the access-token middleware verifies both `token_version` (account-wide) and session existence (per-device) via `RevocationChecker` with a 60-second cache window. The checks **fail open** on DB errors (availability over strict revocation — logout is not guaranteed during a DB outage)
- **Token-type claims** — refresh tokens cannot be used as access tokens (enforced in middleware)
- **Bungie tokens encrypted at rest** — AES-256-GCM in the `bungie_tokens` table, with the membership row bound as AAD and key-version rotation support
- **Bungie token auto-refresh** — stored Bungie OAuth tokens are refreshed automatically before expiry (5-min buffer); refresh writes are compare-and-swap on `updated_at`, and a replica that loses the race adopts the winner's tokens instead of clobbering them

### Roles, Feature Flags & Admin Console

- **Role tiers** — `standard(0) < beta(1) < alpha(2) < admin(3)` stored in `users.role`. Authorization **always reads the role from the DB** (the `RevocationChecker` cache now holds `{token_version, role}`), never from the JWT — the JWT carries role only as a display hint. Role changes therefore take effect within the 60-second cache window with no token churn.
- **Admin bootstrap** — admin can only be minted via `ADMIN_MEMBERSHIP_IDS` (comma-separated Bungie membership IDs pinned to admin on every login upsert) or granted by an existing admin. There is **no self-service path to admin**: `PUT /api/account/role` accepts only standard/beta/alpha and rejects admin callers. `ADMIN_MEMBERSHIP_IDS` is non-secret config (membership IDs), but treat it as sensitive — it controls who holds admin.
- **Server-side enforcement** — `RequireAdmin`/`RequireTier` gate admin and tier-locked endpoints; UI flag hiding is convenience, not enforcement. In degraded mode (no DB) roles resolve to standard and admin/tier endpoints return 503.
- **Last-admin protection** — demoting the final admin is refused inside the same transaction (all admin rows are `SELECT … FOR UPDATE`-locked so concurrent demotions can't race the system to zero admins).
- **Audit trail** — authentication events (login success/failure, logout, logout-all, token refresh), session security events (refresh-token reuse, session termination), self opt-in role changes, admin role changes, and feature-flag changes are persisted to the unified `audit_log` table and exposed to admins via `GET /api/admin/audit` and the `/admin` Audit Log UI panel. Role and flag changes are written in their mutation transaction (atomic); auth/session events are best-effort (a DB outage can drop an event). **Client IP address and User-Agent are captured and stored** for security forensics, retained for `AUDIT_RETENTION_DAYS` (default 180), and pruned hourly once past expiry. IP addresses are trusted only from `TRUSTED_PROXIES` (gin `SetTrustedProxies`) so they cannot be spoofed by a client; configure this in production to your platform's ingress range.

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

### HTTP Security (Go service)

- **Server timeouts**: ReadTimeout 30s, WriteTimeout 60s, IdleTimeout 120s — prevents slowloris-style attacks
- **Graceful shutdown** with 30s timeout

---

## Production Security Checklist

- [ ] All secrets rotated and removed from git history
- [ ] `JWT_SECRET` is 32+ characters, randomly generated
- [ ] `CORS_ALLOWED_ORIGINS` set to your production domain only
- [ ] `GO_ENV=production` on the API service
- [ ] Rate limiting enabled and tuned for expected traffic
- [ ] TLS/HTTPS configured (terminate at load balancer or ingress)
- [ ] Database connections use SSL (`sslmode=require`)
- [ ] Health endpoints (`/health`, `/ready`) not exposed publicly (behind ingress rules)
- [ ] Logging does not include tokens, secrets, or full OAuth codes
- [ ] Docker images built from pinned base image versions
- [ ] Kubernetes secrets not stored in version control
- [ ] `DATABASE_URL` and `TOKEN_ENCRYPTION_KEY` set (the service refuses to start in production without them)
