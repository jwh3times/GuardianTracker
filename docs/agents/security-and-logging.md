# Security and logging

The auth, session and authorization behavior every change must preserve, and what the API's
application logs may and may not record. The overview is `AGENTS.md` ("Auth & token flow").

## Auth and security behavior to preserve

- Access and refresh JWT verification accepts only HS256 and requires the `guardian-tracker` issuer and an unexpired `exp` claim. Session-less development tokens remain supported.
- Credentialed `GET /api/auth/bungie` creates a 10-minute host-only HttpOnly `SameSite=Lax`, `Path=/` transaction cookie (`__Host-guardian_oauth_transaction` with `Secure` in production; `guardian_oauth_transaction` in development). HMAC-signed v2 state binds its SHA-256 nonce digest; callback/reconnect verify both before code exchange, without legacy fallback. The latest start replaces prior pending browser flows. Valid transaction processing expires the cookie; invalid input preserves it.
- The access-only Bungie authorization is encrypted at rest with AES-256-GCM and exact current/previous key versions; expiry requires an authenticated reconnect of the same Bungie membership.
- The shared browser session client atomically persists access JWT/user state in the versioned `guardian_browser_session` localStorage envelope and owns authenticated transport. Web Locks coordinate authorization start, callback, authenticated reconnect, refresh, and logout across tabs; all except local logout require them. `AuthProvider` only subscribes to the public user/authenticated snapshot. The rotating refresh JWT is only in the host-only HttpOnly `guardian_refresh_token` cookie.
- Application composition observes membership type/ID boundaries, cancels and clears the old QueryClient, replaces it, remounts identity-bound providers, and runs registered boundary resets (currently the preferences client, ADR 0021, which is not itself a provider). Same-membership refresh retains caches and mounted state and runs no boundary resets. Authenticated mutations use `useIdentityMutation` to fence work from a departed identity.
- Refresh token revocation is backed by PostgreSQL.
- Callback, authenticated Bungie reconnect, and refresh require an exact allowlisted `Origin`; the cookie design assumes the frontend and API are same-site.
- Single-device logout preserves Bungie authorization; logout-all evicts it.
- Membership-scoped routes (collections, collections refresh, characters,
  character equipment, character activity history, character current activity,
  catalysts, crafting, seals, digest) authorize the caller's whole Destiny membership
  pair — platform type and ID, not ID alone — and abort the request on
  mismatch before any Bungie call or cache access.
- Admin access is controlled by explicit membership ID configuration. Self-service role updates lock the current database row, reject admins, and commit `role.optin` audit events atomically without session/token-version changes. Successful bootstrap admin upserts evict the local authorization cache; cross-replica cache propagation retains its 60-second window.

See `SECURITY.md` for public security posture and reporting guidance.

## Application logging

The API uses `log/slog` and assigns every request a server-generated UUID exposed
as `X-Request-ID`. Access records use Gin route templates and include method,
status, duration, and response bytes. They never include raw URLs/query strings,
bodies, authorization headers, User-Agent values, or routine client IPs.
Membership, session, user, and character identifiers are deterministic 24-hex
pseudonyms in application logs; exact values remain only in the PostgreSQL audit
trail. Successful app requests log at info, 4xx at warn, 5xx at error, and
successful health probes at debug. A request abandoned by its client (the
request context canceled and the failure is that cancellation) is recorded as
499 at info with no error-level handler log.
