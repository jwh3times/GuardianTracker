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

| Variable               | Description                   | Where to Get                                                 |
| ---------------------- | ----------------------------- | ------------------------------------------------------------ |
| `BUNGIE_API_KEY`       | Bungie API access key         | [Bungie Applications](https://www.bungie.net/en/Application) |
| `BUNGIE_CLIENT_ID`     | OAuth client ID               | Bungie application settings                                  |
| `BUNGIE_CLIENT_SECRET` | OAuth client secret           | Bungie application settings                                  |
| `JWT_SECRET`           | Token signing key (32+ chars) | `openssl rand -base64 32`                                    |
| `POSTGRES_PASSWORD`    | Database password             | Generate a strong random password                            |

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

- **Bungie OAuth 2.0** with CSRF protection — cryptographically random state parameter, stored server-side with 10-minute TTL, consumed on first use (one-time)
- **JWT tokens** — HS256 signed, access token (24h) + refresh token (30d) with rotation
- **Token-type claims** — refresh tokens cannot be used as access tokens (enforced in JWT validation)
- **Bungie token auto-refresh** — stored Bungie OAuth tokens are refreshed automatically before expiry (5-min buffer), with cleanup of fully expired tokens

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
- [ ] Wishlist endpoints backed by a real database (currently return stub data)

---

## Known Security Limitations

These are tracked as TODOs — **do not deploy to production until resolved**:

1. **JWT logout is not blacklisted** — `logout` clears client-side tokens but the server-side JWT remains valid until expiry. Implement a token blacklist (Redis) before production.
2. **Wishlist has no database persistence** — data is hardcoded mock responses; no authorization checks on actual data retrieval.
3. **In-memory CSRF state and Bungie token store** — these are lost on service restart. Acceptable for development; use Redis for production persistence.
4. **No audit logging** — authentication events (login, token refresh, failed attempts) are not logged to a persistent audit trail.
