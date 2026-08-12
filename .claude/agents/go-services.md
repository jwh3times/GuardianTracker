---
name: go-services
description: Use for any work in backend/api-service — Gin handlers, JWT auth, Bungie OAuth flow, CSRF state, DB-backed encrypted token store, manifest download, collections analysis, records, weekly, search, roles, feature flags, admin, and Go tests.
tools: Read, Write, Edit, Bash, Glob, Grep
model: sonnet
---

You are working inside the Guardian Tracker Go backend. There is **one** service: `backend/api-service` (Go + Gin). The old multi-service architecture (`auth-service`, `bungie-service`, `graphql-service`) no longer exists.

## Project layout

```
backend/api-service/
  main.go                              ← Composition root only: config, dependency wiring, manifest startup +
                                           swap participant/observer registration, server lifecycle
  api/router.go                        ← The route table. NewRouter(Deps) *gin.Engine; auth applied per group
  db/adapters/adapters.go              ← db → auth/weekly consumer-interface adapters (sentinel translation)
  db/pruner.go                         ← Session and audit retention pruners (injectable interval)
  config/config.go                     ← Typed config with env var parsing helpers; Validate() returns error
  auth/jwt.go                          ← JWT generation and validation (access 30m default, refresh 30d)
  auth/middleware.go                   ← JWT middleware for protected routes
  auth/state.go                        ← Stateless HMAC-signed OAuth state (CSRF, multi-replica safe)
  auth/tokenstore.go                   ← DB-backed encrypted Bungie OAuth token store; CAS refresh writes
  auth/crypto.go                       ← AES-256-GCM cipher; exact current/previous key versions
  auth/revocation.go                   ← JWT revocation: checks token_version (account-wide) + session
                                           existence (per-device) via RevocationChecker; 60s in-memory
                                           cache; also resolves role from DB for RequireAdmin/RequireTier
  auth/roles.go                        ← Role tiers (standard/beta/alpha/admin) + RequireAdmin/RequireTier
                                           tier-gating middleware
  auth/session.go                      ← SessionIssuer: owns the browser session lifecycle
                                           (AuthorizeURL/Login/Refresh/EndSession/EndAllSessions);
                                           failures cross as SessionError{Reason,...} — see
                                           "Session issuer" below
  auth/bungieoauth.go                  ← bungieOAuth: one Bungie OAuth token-endpoint client for both
                                           grants (authorization_code, refresh_token), shared by
                                           SessionIssuer (login) and TokenStore (Bungie-token refresh)
  api/handlers/auth.go                 ← Maps SessionIssuer results to HTTP via loginFailures/
                                           refreshFailures tables (status, cookie, audit, body); holds
                                           no session logic of its own
  api/handlers/characters.go           ← HTTP handler for characters
  api/handlers/collections.go          ← HTTP handler for collections; RefreshCollections invalidates
                                           collections + characters + records caches via service methods
  api/handlers/items.go                ← HTTP handlers for manifest-derived item detail: GetPerks (perk pool +
                                           exotic catalyst pool) and GetItem (minimal item view for deep-linked
                                           non-collectible items); no ownership check (public manifest data)
  api/handlers/wishlist.go             ← Wishlist CRUD + bulk delete/set_priority; enriches with manifest
                                           defs, collectible source, and all-rotating-vendor availability
                                           (via liveVendorIface → weekly.Service.LiveVendorItemHashes)
  api/handlers/account.go              ← Self-service role opt-in (PUT /api/account/role) +
                                           resolved feature-flag state (GET /api/flags)
  api/handlers/admin.go                ← Admin console: user roster, role management, feature-flag
                                           config, audit log feed
  api/handlers/health.go               ← Health, ready, manifest status endpoints
  api/handlers/common.go               ← Shared handler helpers (parseMembershipParams, ownershipCheck…)
  api/handlers/storeerror.go           ← HandleStoreError(c, err, logMsg) — maps db.ErrUnavailable to a
                                           503 DB_UNAVAILABLE and anything else to a logged 500 INTERNAL_ERROR;
                                           mirrors handleBungieError
  services/bungie/client.go            ← HTTP client with rate limiting + retry
  services/bungie/manifest.go          ← Manifest download, version tracking, SQLite extraction;
                                           RegisterParticipant/RegisterObserver coordinate the file swap
                                           (see "Manifest repository, provider, and swap seam" below)
  services/bungie/types.go             ← All Bungie API types, constants, helpers
  services/collections/service.go      ← Collection analysis + difficulty classification + cosmetics;
                                           uses ManifestRepo interface (satisfied by *manifest.Provider)
  services/characters/service.go       ← Character fetching; InvalidateCache method
  services/items/service.go            ← Cached weapon-perks lookup (GetWeaponPerks), item-view lookup
                                           (GetItem), and catalyst-pool lookup (GetCatalysts), each backed by its
                                           own boundedCache; all three cleared by InvalidateCache, called from
                                           OnVersionChanged (ManifestObserver)
  services/items/boundedcache.go       ← Unexported generic boundedCache[K,V]: size-capped, no TTL, keyed by
                                           item hash; not cache.Cache — cleared wholesale on manifest swap instead
                                           of expiring
  services/manifest/repository.go      ← SQLite read-only queries against manifest DB
  services/manifest/provider.go        ← Shared lazy-opening repository; implements bungie.SwapParticipant
                                           (CloseForSwap/Reopen) for the hourly manifest swap; satisfies all
                                           consumer ManifestRepo interfaces
  services/weekly/service.go           ← Weekly recommendations; Xûr inventory + XurItemHashes();
                                           milestone data; reset time math; ManifestRepo interface;
                                           bungie.ManifestObserver
  services/weekly/availability.go      ← LiveVendorItemHashes: best-effort all-rotating-vendor item
                                           availability (Xûr, Banshee-44, Ada-1, ritual vendors) for wishlist
  services/search/service.go           ← Manifest item search index with versioned disk snapshots; opens its
                                           own SQLite handle on the manifest (not manifest.Provider), so it
                                           registers itself as its own bungie.SwapParticipant (CloseForSwap/
                                           Reopen) in addition to being a bungie.ManifestObserver
                                           (OnVersionChanged kicks BuildIndex); async rebuild on update,
                                           aborted mid-scan if a swap starts
  services/records/service.go          ← Catalysts, crafting patterns, seals/triumphs; ManifestRepo
                                           interface; InvalidateCache; OnVersionChanged (ManifestObserver)
                                           evicts its three manifest-derived cache keys
  services/sources/sources.go          ← Destiny source-string vocabulary: Difficulty, IsRaidOrDungeon,
                                           ActionKind, MilestoneCategory — one faceted keyword table,
                                           not four parallel ones; collections/efficiency/weekly delegate
  cache/cache.go                       ← In-memory cache (and no-op cache interface)
  cache/load.go                        ← cache.Load[T]/cache.LoadIf[T]: typed load-through above Cache —
                                           an error is never cached, a wrong-typed entry is a logged miss,
                                           a nil Cache loads every time; LoadIf's predicate (NonEmptyMap/
                                           NonEmptySlice) is where "don't cache an empty result" lives
  db/db.go, db/migrate.go              ← Postgres pool; migration runner (each migration runs in a tx)
  db/stores.go                         ← Stores struct — interface fields, never nil; NewStores(nil)
                                           returns the degraded set; Available() reports whether a real DB backs it
  db/degraded.go                       ← ErrUnavailable sentinel; the six store interfaces (UserRepo,
                                           TokenRepo, WishlistRepo, PrefsRepo, FlagRepo, AuditRepo) + Pinger;
                                           degraded implementations whose every method returns ErrUnavailable
  db/migrations/0001_init.sql          ← Base schema DDL
  db/migrations/0002_roles_flags.sql   ← Adds role column to users, feature_flags table, role_audit
  db/migrations/0003_refresh_sessions.sql ← Adds refresh_sessions for per-device sessions + reuse detection
  db/migrations/0004_audit_log.sql     ← Unifies audit_log, drops role_audit
  db/migrations/0005_logout_session.sql ← Renames "logout" audit event to "logout.session"
                                           (matches the logout.* prefix filter)
  db/migrations/0006_remove_unused_flags.sql ← Retires wishlist-alerts and ui-tweaks flags
                                           seeded by 0002 (10 → 8 seeded flags)
  db/users.go                          ← UserStore — upsert, get, bump token_version, per-device sessions
                                           (CreateSession, RotateSession, DeleteSession, DeleteAllSessions)
  db/tokens.go                         ← BungieTokenStore — encrypted Bungie OAuth tokens
  db/wishlist.go, db/prefs.go          ← DB stores for wishlist and preferences
  db/audit.go                          ← Unified append-only audit trail store (audit_log): best-effort
                                           Log, in-transaction insertAudit, filtered/keyset List, prune
  db/flags.go                          ← Feature flags store (get all, get by key, upsert)
```

## Endpoints

| Method | Path                                                     | Auth                          | Purpose                                                                                                                                                                                                                                                                                                           |
| ------ | -------------------------------------------------------- | ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| GET    | `/api/auth/bungie`                                       | None                          | Initiate OAuth — returns `{ authUrl, state }`                                                                                                                                                                                                                                                                     |
| POST   | `/api/auth/bungie/callback`                              | Exact Origin + OAuth state    | Exchange code; set refresh cookie; return `{token,user}`                                                                                                                                                                                                                                                          |
| POST   | `/api/auth/refresh`                                      | Exact Origin + refresh cookie | Empty JSON request; rotate cookie + access token (per-session, reuse detection)                                                                                                                                                                                                                                   |
| GET    | `/api/auth/validate`                                     | JWT                           | Validate JWT                                                                                                                                                                                                                                                                                                      |
| GET    | `/api/auth/profile`                                      | JWT                           | Current user profile                                                                                                                                                                                                                                                                                              |
| POST   | `/api/auth/logout`                                       | JWT                           | End current device's session only; other devices stay signed in                                                                                                                                                                                                                                                   |
| POST   | `/api/auth/logout/all`                                   | JWT                           | Sign out everywhere: bump token_version + delete all sessions + Bungie token                                                                                                                                                                                                                                      |
| GET    | `/api/wishlist`                                          | JWT                           | List wishlist items (name, icon, sources, availableNow/From)                                                                                                                                                                                                                                                      |
| POST   | `/api/wishlist`                                          | JWT                           | Add wishlist item                                                                                                                                                                                                                                                                                                 |
| PUT    | `/api/wishlist/:id`                                      | JWT                           | Update wishlist item priority/notes                                                                                                                                                                                                                                                                               |
| DELETE | `/api/wishlist/:id`                                      | JWT                           | Remove wishlist item                                                                                                                                                                                                                                                                                              |
| POST   | `/api/wishlist/bulk`                                     | JWT                           | Bulk `delete` / `set_priority` over selected ids; partial-success `{updated, skipped}`                                                                                                                                                                                                                            |
| GET    | `/api/preferences`                                       | JWT                           | Get user preferences and `onboardedAt`                                                                                                                                                                                                                                                                            |
| PUT    | `/api/preferences`                                       | JWT                           | Update preferences; `onboardingComplete:true` stamps completion and cannot reset it                                                                                                                                                                                                                               |
| PUT    | `/api/account/role`                                      | JWT                           | Self-service opt-in to standard/beta/alpha; admin rejected, admin callers rejected                                                                                                                                                                                                                                |
| GET    | `/api/flags`                                             | JWT                           | Resolved feature-flag state for caller (enabled/accessible/locked + role)                                                                                                                                                                                                                                         |
| GET    | `/api/admin/users?q=`                                    | JWT + admin                   | User roster (id, displayName, platform, role, lastActive)                                                                                                                                                                                                                                                         |
| PUT    | `/api/admin/users/:id/role`                              | JWT + admin                   | Set any role; last-admin protected; bumps token_version + audits                                                                                                                                                                                                                                                  |
| GET    | `/api/admin/flags`                                       | JWT + admin                   | Full feature-flag config                                                                                                                                                                                                                                                                                          |
| PUT    | `/api/admin/flags/:key`                                  | JWT + admin                   | Toggle enabled / set minTier                                                                                                                                                                                                                                                                                      |
| GET    | `/api/admin/audit`                                       | JWT + admin                   | Filtered keyset-paginated audit feed                                                                                                                                                                                                                                                                              |
| GET    | `/api/characters/:membershipType/:membershipId`          | JWT                           | User characters                                                                                                                                                                                                                                                                                                   |
| GET    | `/api/collections/:membershipType/:membershipId`         | JWT                           | Collections + fetchedAt; `?include=all` adds collectedItems                                                                                                                                                                                                                                                       |
| POST   | `/api/collections/:membershipType/:membershipId/refresh` | JWT                           | Invalidate cache (collections + characters + records)                                                                                                                                                                                                                                                             |
| GET    | `/api/manifest/status`                                   | None                          | Manifest version and readiness                                                                                                                                                                                                                                                                                    |
| GET    | `/api/weekly/recommendations?characterId=`               | JWT + flag                    | Weekly data, Xûr, milestones, recommended actions + fetchedAt/resetAt; validates the optional character against the authenticated roster and falls back to the primary character                                                                                                                                  |
| GET    | `/api/items/search?q=&limit=`                            | JWT + flag                    | Manifest item search; 503 until index ready                                                                                                                                                                                                                                                                       |
| GET    | `/api/items/:itemHash`                                   | JWT                           | Minimal manifest item view for deep-linked non-collectible items; `{ itemHash, name, icon, itemType, tierType, rarity, description }`; 404 for unknown hash; 503 (`MANIFEST_NOT_READY`) while manifest warms; 400 on non-numeric hash; NOT membership-scoped                                                      |
| GET    | `/api/items/:itemHash/perks`                             | JWT                           | Weapon possible perk pool + exotic catalyst pool from manifest; `{ itemHash, perkColumns: [{role,label,perks}], catalysts: [{name,description}] }`; 200 + empty arrays for non-weapon/unknown hash or non-exotic; 503 (`MANIFEST_NOT_READY`) while manifest warms; 400 on non-numeric hash; NOT membership-scoped |
| GET    | `/api/catalysts/:membershipType/:membershipId`           | JWT + flag                    | `{ items, fetchedAt }` exotic catalyst progress incl. weapon type/icon/effect text                                                                                                                                                                                                                                |
| GET    | `/api/crafting/:membershipType/:membershipId`            | JWT + flag                    | `{ items, fetchedAt }` crafting pattern progress                                                                                                                                                                                                                                                                  |
| GET    | `/api/seals/:membershipType/:membershipId`               | JWT + flag                    | `{ items, fetchedAt }` triumph/seal completion; each triumph carries an optional `objectives` array (`{label,done,cur,max}`)                                                                                                                                                                                      |
| GET    | `/health`                                                | None                          | Liveness probe                                                                                                                                                                                                                                                                                                    |
| GET    | `/ready`                                                 | None                          | Readiness probe; requires the manifest and, when a DB pool is configured, a successful database ping                                                                                                                                                                                                              |

Error responses: `{ "error": "...", "code": "MACHINE_CODE" }`. Codes: `PRIVACY_RESTRICTION`, `ACCOUNT_NOT_FOUND`, `RATE_LIMITED`, `MANIFEST_NOT_READY`, `BUNGIE_ERROR`, `INTERNAL_ERROR`, `FORBIDDEN`, `TIER_LOCKED`, `DB_UNAVAILABLE`, `LAST_ADMIN`, `ROLE_NOT_ALLOWED`, `ADMIN_OPT_IN`, `FEATURE_DISABLED`.

## Bungie OAuth flow

1. `GET /api/auth/bungie` — `SessionIssuer.AuthorizeURL()` generates the HMAC-signed state token via `auth.StateSigner` (derived from `JWT_SECRET`); handler returns `{ authUrl, state }`
2. User visits Bungie.net, authorizes, Bungie redirects to frontend `/auth/callback?code=...&state=...`
3. Frontend credentialed-POSTs `{ code, state }` to `POST /api/auth/bungie/callback` — exact Origin required; `SessionIssuer.Login()` verifies state with 10-min TTL via `StateSigner.Verify()` (not single-use; replay bounded by Bungie's single-use auth code), exchanges the code via `bungieOAuth`, stores Bungie tokens, mints the JWT pair, and creates a `refresh_sessions` row (skipped, not failed, when no database is configured); the handler sets the HttpOnly refresh cookie and returns `{token,user}`

## Session issuer (`auth/session.go`)

`SessionIssuer` owns the whole browser session lifecycle behind one seam:
`AuthorizeURL`, `Login`, `Refresh`, `EndSession`, `EndAllSessions`.
`api/handlers/auth.go` calls it and translates the result to HTTP; it holds no
session logic itself.

- Failures return `*auth.SessionError{Reason, MembershipID, SessionID, Err}`.
  `Reason` is also the audit reason string — a new failure mode needs a
  `Reason` constant before a handler can report it. `auth.AsSessionError(err)`
  unwraps one.
- The handler maps `Reason` through `loginFailures` or `refreshFailures`
  (`api/handlers/auth.go`) to a status, message, audit event, and whether the
  refresh cookie is cleared. An unmapped `Reason` falls through to 500 rather
  than a silent success.
- `auth.ErrUnavailable`, translated from `db.ErrUnavailable` by
  `adapters.NewSessionStore` (`db/adapters/adapters.go`), separates "there is
  no database" from "the write failed": a login whose session write hits
  `ErrUnavailable` still succeeds, session-less; any other write error fails
  the login. See
  [ADR 0012](../../docs/adr/0012-session-issuance-owns-the-session-lifecycle.md).
- The `user` response body has one builder (`userPayload` in
  `api/handlers/auth.go`), used by all four endpoints that return it
  (`bungie/callback`, `refresh`, `validate`, `profile`) — all four now
  include `role`.

## JWT format

- Algorithm: HS256, secret from `JWT_SECRET` env var
- Claims: `sub` (membershipId), `membership_type`, `display_name`, `platform`, `token_type`, `tver` (token_version), `jti`, `sid` (session ID — matches a `refresh_sessions.id` row)
- `token_type` must be `"access"` for access tokens and `"refresh"` for refresh tokens
- Access expiry: configurable via duration-based `JWT_ACCESS_TTL` (default 30m); legacy `JWT_EXPIRY_HOURS` is accepted when the new setting is absent
- Refresh expiry: configurable via `JWT_REFRESH_EXPIRY_DAYS` (default 30d)
- Browser delivery: access token/user snapshot in localStorage; refresh JWT only in host-only HttpOnly `guardian_refresh_token` (`SameSite=Lax`, `/api/auth`, and `Secure` in production)

## Token store (`auth/tokenstore.go`)

DB-backed encrypted Bungie OAuth token store. Tokens are stored AES-256-GCM encrypted in the `bungie_tokens` table with exact positive current/previous key versions; unknown versions are rejected. Auto-refreshes via Bungie's OAuth refresh endpoint when within 5 minutes of expiry. Refresh writes use compare-and-swap on `updated_at` — a replica that loses the race reads and adopts the winner's tokens. The refresh-grant HTTP call goes through `bungieOAuth` (`auth/bungieoauth.go`), the same client `SessionIssuer` uses for the login grant, so the 90-day `refresh_expires_in` fallback is written once.

Sentinel errors:

- `auth.ErrTokensNotFound` — no token row exists
- `auth.ErrNoUserRow` — no users row for the membership

## Store availability (degraded mode, `db/degraded.go`)

`db.NewStores(pool)` never returns a nil field. With a `nil` pool it returns
degraded implementations of every store interface (`UserRepo`, `TokenRepo`,
`WishlistRepo`, `PrefsRepo`, `FlagRepo`, `AuditRepo`, `Pinger`) whose every
method returns `db.ErrUnavailable`. There is no store nil-guard convention —
handlers call the store directly and handle the error like any other:

```go
items, err := h.store.List(c.Request.Context(), userID)
if err != nil {
    handlers.HandleStoreError(c, err, "wishlist listing failed")
    return
}
```

`HandleStoreError(c, err, logMsg) bool` maps `errors.Is(err, db.ErrUnavailable)`
to `503 {"error": "...", "code": "DB_UNAVAILABLE"}` and anything else to a
logged `500 INTERNAL_ERROR`. All seven wishlist/preferences handlers and the
admin/audit/account handlers route store errors through it, so a missing
database now produces one response shape everywhere instead of the bare
`{"error": "database not configured"}` the wishlist handlers used to emit.
`auth.RequireTier`'s own 503 (no store involved — it never reads one) uses
matching wording so both paths read as the same failure.

`Stores.Available() bool` is the only legitimate "is there a real database"
check — used in `main.go` to gate the session/audit pruners and to decide
whether `RequireTier`'s role claims are authoritative (`auth.NewAuthz(stores.Available())`).
Do not resurrect a `store == nil` check; every store field is always a valid,
callable interface.

Two callers stay deliberately lenient on `ErrUnavailable` rather than
surfacing it: `FlagResolver.List` treats it as "no flags configured" (flags
are rollout controls, not a security boundary, so an absent DB must not hide
pages), and `GET /api/preferences` returns default preferences with `200`
instead of a 503 (the UI needs preferences to render). `auth.RevocationChecker`
already failed open on any store error before this existed. `SessionIssuer`
follows the same pattern one level removed, through its own `auth.ErrUnavailable`
sentinel rather than `db.ErrUnavailable` directly — see "Session issuer" above.

## Middleware

`jwtHelper.Middleware(revoker)` — required auth, returns 401 if JWT is invalid, missing, wrong `token_type`, revoked (token_version mismatch), or session doesn't exist. Sets `user_id`, `membership_id`, `membership_type`, `display_name`, `platform`, `token_version` on Gin context.

`auth.RequireAdmin` / `auth.RequireTier(tier)` — role-gating middleware placed after `jwtHelper.Middleware`. **Role is always read from the DB-backed RevocationChecker cache, never from the JWT claim** — so role changes propagate within the 60s window without requiring a new token.

`authz.RequireFlag(enforceFlags, handlers.Flag*)` — composes after `jwtHelper.Middleware` on the weekly, search, catalysts, crafting, and seals routes (`FlagWeeklyPlanner`, `FlagGlobalSearch`, `FlagCatalystsCrafting`, `FlagTriumphsSeals`). **Fails open**: a disabled flag returns 404 `FEATURE_DISABLED`, an enabled flag above the caller's tier returns 403 `TIER_LOCKED`, but a nil resolver, degraded mode, store error, or unknown key allows the request through instead of blocking it.

## Roles & feature flags

Tiers: `standard(0) < beta(1) < alpha(2) < admin(3)` stored in `users.role`.

- **Admin bootstrap**: `ADMIN_MEMBERSHIP_IDS` env var pins specified accounts to admin on every login. Additional admins are granted by existing admins from the admin console; there is no self-service path to admin.
- **Self opt-in**: `PUT /api/account/role` (standard/beta/alpha only; admin role rejected; admin callers blocked). Evicts the caller's RevocationChecker cache entry with **no** token_version bump (session preserved).
- **Admin-driven changes**: `PUT /api/admin/users/:id/role` bumps target's `token_version` + evicts their cache entry (forced re-sync) + writes an audit row inside the same transaction. Last-admin protection enforced in-transaction.
- **Feature flags**: stored in `feature_flags` table (`key`, `enabled`, `min_tier`). `GET /api/flags` returns resolved state (`enabled`/`accessible`/`locked`) based on caller's role — a UI hint. Server-side enforcement on the flag-gated routes (weekly, search, catalysts, crafting, seals) is `authz.RequireFlag` (see Middleware); it fails open when the flag can't be resolved. Admin toggles via `PUT /api/admin/flags/:key`. `RequireAdmin`/`RequireTier` remain the (fail-closed) gate for admin and tier-locked endpoints.

## Per-device refresh sessions

`SessionIssuer.Login` creates a `refresh_sessions` row (skipped, not failed,
with no database configured); the `sid` JWT claim holds the session ID:

- `POST /api/auth/refresh` reads the refresh JWT only from its cookie; `SessionIssuer.Refresh` compare-and-swaps the session's refresh `jti` (`SessionStore.RotateSession`). A replayed (already-rotated) token is detected as reuse → **revokes the whole session** (401, even if the revoking commit errors). Successful refresh rotates the cookie; definitive failures expire it.
- Sessions are independent → fully multi-device. `expires_at` slides forward on each rotation.
- Sessions are capped per user (`maxSessionsPerUser`); an hourly `startSessionPruner` deletes expired rows.
- `POST /api/auth/logout` — `SessionIssuer.EndSession` ends only the current session (`DeleteSession`); other devices stay signed in; Bungie token preserved.
- `POST /api/auth/logout/all` — `SessionIssuer.EndAllSessions` bumps `token_version` + deletes all sessions (`DeleteUserSessions`) + evicts the Bungie token.
- Pre-`0003` tokens (no `sid`) are adopted into a fresh session on their first `SessionIssuer.Refresh` call.

## Audit logging (`db/audit.go`)

Events persisted to `audit_log`: login, logout, logout-all, refresh failure, refresh reuse, session termination, self opt-in role changes, admin role changes, feature-flag changes. Role/flag changes are written in the mutation's transaction (atomic); auth/session events are best-effort (a DB outage can drop an event). Client IP (validated via `TRUSTED_PROXIES`) and User-Agent are retained for `AUDIT_RETENTION_DAYS` (default 180) days; an hourly pruner removes older rows.

## Route table (`api/router.go`)

`NewRouter(Deps) *gin.Engine` owns every route. `main.go` builds `Deps` and
serves; it registers no routes itself.

- **Authentication is applied once per group**, not per route:
  `authed := api.Group("", d.JWT.Middleware(d.Revoker))`. Register every
  authenticated route on `authed`, or on the `admin` subgroup inside it. Do not
  reintroduce a per-route `Middleware(...)` argument — the 24× repetition it
  replaced is what made "forgot the middleware on the new route" a silent way to
  publish an unauthenticated endpoint.
- **A new public `/api` route also needs an entry in `publicAPIRoutes` in
  `api/router_test.go`.** That allowlist deliberately lives in the test, not the
  router, so `TestEveryAPIRouteRequiresAuthentication` fails until someone grants
  the exemption on purpose. The test walks `Engine.Routes()`, so authenticated
  routes are covered automatically.
- **Flag-gated routes** pair a route with a key via
  `d.Authz.RequireFlag(d.Flags, handlers.FlagX)`. `RequireFlag` fails open on an
  unresolvable key (ADR 0006), so a wrong key gates nothing; add the route to
  `TestFlagGatedRoutesEnforceTheirOwnFlag`, which disables one key at a time and
  asserts exactly the routes gated on it are blocked.
- See [ADR 0011](../../docs/adr/0011-route-table-as-a-testable-composition-root.md).

## Manifest repository, provider, and swap seam

- `services/manifest/repository.go` — raw SQLite queries. All public methods acquire `r.mu` (RWMutex); locked variants (`*Locked`) for composite operations that must hold the lock across multiple queries.
- `services/manifest/provider.go` — single `*Provider` shared by all consumers. Opens lazily on first use; implements `bungie.SwapParticipant` (`CloseForSwap()` / `Reopen() error`). Returns `ErrNotReady` (503 `MANIFEST_NOT_READY`) while absent or swapping.
- `services/bungie/manifest.go` defines the swap seam as two interfaces. Both are registered on `*bungie.ManifestService` in `main.go`, in registration order — `manifest.Provider` is registered first, since observers may query the manifest through it — and registration must happen before the first download can fire:
  - `SwapParticipant { CloseForSwap(); Reopen() error }` — for a module holding an OS-level handle on the manifest file. `RegisterParticipant(p)`. Registered participants: `manifest.Provider`, `search.Service` (search opens its own SQLite connection instead of sharing the Provider).
  - `ManifestObserver { OnVersionChanged(version string) error }` — for a module holding manifest-derived state (a cache, an index). `RegisterObserver(o)`. Registered observers: `records`, `weekly`, `collections`, `items`, `search`, `efficiency`.
  - Swap sequence: close every participant → `os.Rename` → reopen every participant → notify every observer, in that order (observers may query the already-reopened manifest). On a **failed** rename (rollback), participants are still reopened — against the still-present old file — but observers are **deliberately not notified**, because the version did not change; `Reopen` means "reopen against whatever manifest is now live", not "a new version was installed".
  - A failing participant or observer is logged (with its `%T` type) and the loop continues rather than aborting — aborting would strand later registrants closed.
  - **Any new module that opens its own handle on the manifest file must register as a `SwapParticipant`**, or `os.Rename` runs under its open handle (fails outright on Windows; serves a deleted inode on Linux). Any module holding manifest-derived state should register as a `ManifestObserver` instead of relying on an ad hoc eviction call from `main.go`.
- `services/collections/service.go` is an observer but does not evict its per-user cache on swap: `OnVersionChanged` drops only the shared cross-user tree; each cached `*analysis` carries a `manifestVersion` stamp, and `getAnalysis` lazily rebuilds the stale manifest-derived fields (`collectibles`/`tree`/`owned`) via `refreshManifestParts` on the next read, reusing the already-fetched (rate-limited) Bungie `collected` data. `refreshManifestParts` is copy-on-write — it returns a new `*analysis` rather than mutating the cached one, since concurrent requests share that pointer without a lock.
- `services/weekly/service.go`'s per-character `daily:vendors:*` cache entries are scoped by manifest version (via a `versioner` dependency satisfied by `*bungie.ManifestService`) instead of being evicted on swap, so a swap orphans the old entries and they expire on their existing TTL; the underlying Bungie vendor response stays cached separately under an unversioned key, so nothing refetches from Bungie.

## Notable manifest methods

| Method                                | Purpose                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GetCollectiblesByItemHashes(hashes)` | Collectible defs keyed by itemHash (for wishlist source strings)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `GetWeaponTypesByName()`              | Lowercased weapon name → weapon type display name (table scan; callers cache)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `GetWeaponPerks(itemHash)`            | Socket-category → plug-set → plug-item traversal yielding ordered perk columns across every weapon socket category (Intrinsic/Barrel/Magazine/Trait N/Origin plus Scope/Launcher Barrel/Battery/Stock/Blade/Guard/Arrow/Bowstring/Haft/Grip/Rail/Bolt, and a generic "Perks" fallback for unrecognized plug-category-identifiers); weapon-only (itemType 3 + weapon socket categories); `isJunkPCI` blacklist-filters kill-tracker/empty/catalyst-socket plugs (catalysts render separately via `GetWeaponCatalysts`) instead of allowlisting known categories, so no perk-bearing column is silently dropped; dedupes by name; also exposed via `services/items/service.go` cached wrapper |
| `GetItemView(itemHash)`               | Minimal item projection (`ItemView` — name, icon, itemType, tierType, rarity, description); returns `(nil, nil)` for unknown hash; no item-type restriction (all collectible/non-collectible items); also exposed via `services/items/service.go` cached wrapper                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `GetWeaponCatalysts(itemHash)`        | Catalyst-socket plug-set traversal yielding an exotic weapon's catalyst name/description pool; `(nil, nil)` for non-weapons, non-exotics, or exotics without a detected catalyst socket; also exposed via `services/items/service.go` cached wrapper (`GetCatalysts`)                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `GetCatalystLinks()`                  | Full-manifest scan returning per-exotic-weapon catalyst text + unlock-objective hashes (`CatalystLink`); cached by `services/records` (evicted by its `OnVersionChanged`) for hash-first catalyst-record→weapon linkage                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |

## Records service

- `GetCatalysts / GetCrafting / GetSeals` return `([]T, time.Time, error)` — second value is `fetchedAt`
- `Catalyst` struct has `Type` (weapon type), `Icon` (record icon), and `Effect` (catalyst-perk effect text) fields
- `Effect` is resolved by `resolveCatalystEffect`: links the catalyst record to its weapon via `GetCatalystLinks()` objective-hash overlap first (unambiguous on both the weapon and record side), then a stripped-name match, then a catalyst-plug-name match, falling back to the record's own description and then `""`
- `OnVersionChanged` (`bungie.ManifestObserver`) evicts its three manifest-derived cache keys (weapon types, exotic weapons, catalyst links) so stale labels don't outlive a swap; per-user `records:*` profile entries are untouched — they hold raw Bungie data, which a manifest swap does not invalidate
- `InvalidateCache(membershipType, membershipId)` drops cached profile records (called by RefreshCollections)
- `Triumph.Objectives []TriumphObjective` (`omitempty`) — per-objective drill-down built by `GetSeals`: excludes explicitly-hidden objectives (`RecordObjective.Visible *bool`; `nil` = absent = visible — a plain `bool` would decode Bungie's absent-field-means-visible default backwards), falls back to `Objective N` for a blank `progressDescription` (numbered over the objectives that survive visibility filtering), normalizes a zero `completionValue` to `Max=1`, and forces `Done=true`/`Cur==Max` on every objective when the parent record is redeemed regardless of stale objective payloads. The existing top-level `Triumph.Cur`/`Max` is unchanged for response compatibility.

## Source vocabulary (`services/sources`)

Owns the Destiny source-string keyword vocabulary and every classification
derived from it, behind one faceted table (`{keyword, tier, raidDungeon}`)
instead of the four parallel keyword lists that used to live separately in
`collections`, `efficiency`, and `weekly` — nothing forced those to stay in
step, so shipping a new dungeon meant editing three files, and forgetting one
made a milestone's missing-count badge silently not appear.

- `Difficulty(source)` — `Challenging`/`Moderate`/`Easy`/`Unrated`; stops at the
  first table hit, so "Grandmaster Nightfall" scores Challenging before the
  Moderate "nightfall" rule.
- `IsRaidOrDungeon(source)` — scans the whole table rather than stopping at the
  first hit, so the same "Grandmaster Nightfall"-in-a-dungeon string still
  counts as dungeon loot even though it tiers on "grandmaster".
- `ActionKind(sourceHash, source)` — `KindExcluded`/`KindVendor`/`KindActivity` for
  the efficiency engine's "go do this" bucket classification.
- `MilestoneCategory(name)` — buckets a weekly milestone by display name.
- `services/sources/sources_test.go` asserts invariants unrepresentable across
  separate tables: raid/dungeon facets agree with `IsRaidOrDungeon` and are
  never tiered Easy, tiers stay grouped so first-hit-wins holds, and no keyword
  is duplicated.

## Collections service

- `ErrManifestNotReady` exported (aliases `manifest.ErrNotReady`); handler maps it to 503
- `UserCollections` has `FetchedAt` field
- `CollectionSummary` has `CollectedItems` field (stripped from response unless `?include=all`)
- `WithoutCollectedItems()` returns a value copy with all `CollectedItems` cleared
- `ClassifyDifficulty(source, isExotic)` — thin delegate to `services/sources.Difficulty`, which owns the
  positive-match table; returns `"Unrated"` for unmatched sources (no catch-all "Easy"); called from
  `collections` only — `weekly` calls `sources.Difficulty` directly (see `MissingItemReader` below)
- `DestinyItem.FarmOnly` — set `true` when the collectible source string contains "cannot be reacquired"; surfaced as a "Farm only" chip in the UI
- `DestinyItem.ItemType` — `toDestinyItem` names all verified cosmetic types via `bungie.ItemTypeName`, so cosmetics carry real strings (`Emblem`/`Ship`/`Sparrow`/`Ghost`/`Emote`/`Shader`/`Ornament`/`Finisher`) instead of `"Unknown"`; the frontend gallery classifies by these strings
- Shaders and ornaments both use `itemType=19` (mod), with verified subtypes 20 and 21 respectively. Gate both by subtype in `manifest.CollectibleCategory`; do NOT add all of item type 19 to `cosmeticItemTypes`. Finishers use verified item type 29.

## Weekly service

- `OnVersionChanged` (`bungie.ManifestObserver`) evicts the cached global weekly payload, whose milestone names and reward labels resolve through the manifest; per-character `daily:vendors:*` entries need no eviction call — their key is scoped by manifest version (see "Manifest repository, provider, and swap seam" above), so a swap orphans them automatically
- `Weekly` struct has `ResetAt`, `FetchedAt`, `Degraded` fields
- `Xur.Location` is optional and best-effort: character-vendor component 400 supplies
  `vendorLocationIndex`, the manifest resolves its destination, and the known Last City
  destination is presented as `The Tower`; failures omit the field.
- `Milestone.Missing` is `*int` (omitempty) — populated by `buildMilestones` for
  raid/dungeon milestones via `efficiency.MissingForMilestone`; verified current
  non-raid reward definitions contain no collectible-linked items, so others omit it.
- `XurItemHashes(ctx)` returns the set of hashes Xûr currently sells
- `LiveVendorItemHashes(ctx, membershipType, membershipID, bungieToken)` (`services/weekly/availability.go`) returns item hash → vendor display name across all rotating vendors (Xûr, Banshee-44, Ada-1, ritual vendors); best-effort with the caller's Bungie token — used by the wishlist handler for availability instead of the Xûr-only `XurItemHashes`
- `GetWeekly(..., requestedCharacterID)` validates the requested character against the authenticated roster and scopes component-402 vendor inventory, daily actions, recommendation availability, and Xûr location to that character. Character-specific caches include membership and character IDs. Xûr armor carries an optional manifest `className`; absent class data remains unlabelled.
- `MissingItemReader` is the consumer-side interface `weekly` declares for the one
  `collections` method it needs (`GetMissingItemHashes`), satisfied structurally by
  `*collections.Service`; `weekly` does not import `services/collections` at all.
  Required, never nil-guarded — a reader degrading to an empty set would silently
  render as a complete collection.

## Go patterns

- Config is loaded once at startup via `config.Load()`; `cfg.Validate()` requires `GO_ENV` to be exactly `development` or `production` and rejects invalid/missing key versions.
- Logging config accepts `LOG_LEVEL=debug|info|warn|error` and `LOG_FORMAT=text|json`; invalid values fail startup. Defaults are text/info in development and JSON/info in production.
- Gin handlers must only: bind inputs, call a service/function, return `c.JSON(...)`. No business logic in handlers.
- Errors: `c.JSON(http.StatusXXX, gin.H{"error": "...", "code": "MACHINE_CODE"})` then `return`.
- All HTTP calls to Bungie API go through `services/bungie/client.go` (rate limiting + retry). Never construct Bungie API calls inline.
- Use constants and helpers from `services/bungie/types.go` for Bungie API types.
- CGO is **enabled** (`CGO_ENABLED=1`) for SQLite.
- Migrations run in a transaction — a failed multi-statement migration cannot leave a half-applied schema.
- A get-or-compute-and-cache call site against `cache.Cache` goes through `cache.Load`/`cache.LoadIf` (`cache/load.go`), not a hand-written get + type-assert + conditional `Set`. Use `LoadIf` with `cache.NonEmptyMap`/`cache.NonEmptySlice` (or a custom predicate) when an empty result must not be cached. Skip it only when the TTL depends on the loaded value or the cache-hit path itself transforms and re-stores (`weekly.getPublicWeekly`, `collections.getAnalysis`).

## Structured logging

- Use the request-scoped `*slog.Logger` attached to `context.Context`; every request has a server-owned UUID returned as `X-Request-ID`.
- Access records use the matched route template, method, status, duration, and response bytes. Never log raw URLs/query strings, bodies, authorization headers, User-Agent values, or routine client IPs.
- Use deterministic 24-hex pseudonyms (first 12 bytes of SHA-256) for membership, session, user, and character identifiers. Exact values belong only in `audit_log`.
- Log successful health probes at debug, successful application requests at info, 4xx at warn, and 5xx/panic recovery at error.

## Environment variables

```
PORT, GO_ENV, BUNGIE_API_KEY, BUNGIE_CLIENT_ID, BUNGIE_CLIENT_SECRET
AUTH_REDIRECT_URI, JWT_SECRET, JWT_ACCESS_TTL, JWT_EXPIRY_HOURS (legacy), JWT_REFRESH_EXPIRY_DAYS
DATABASE_URL, TOKEN_ENCRYPTION_KEY, TOKEN_ENCRYPTION_KEY_VERSION
TOKEN_ENCRYPTION_KEY_PREVIOUS, TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION
ADMIN_MEMBERSHIP_IDS, AUDIT_RETENTION_DAYS, TRUSTED_PROXIES
BUNGIE_API_BASE_URL, BUNGIE_API_RPS, BUNGIE_API_BURST
MANIFEST_DB_PATH, MANIFEST_CHECK_INTERVAL
CACHE_ENABLED, CACHE_TTL_COLLECTIONS, CACHE_TTL_RECORDS
CORS_ALLOWED_ORIGINS, LOG_LEVEL, LOG_FORMAT, HTTP_TIMEOUT_SECONDS
```

## Testing

```powershell
# From backend/api-service/
go test ./...
go run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...
go tool govulncheck ./... # declared at v1.6.0 in go.mod; updated by Dependabot

# With race detector (matches CI); requires CGO + Postgres for full coverage
go test -race ./...

# Full CI-equivalent coverage (~63%)
./test-local.ps1            # start throwaway Postgres, run all tests, print total
./test-local.ps1 -Html      # also open per-line HTML report
./test-local.ps1 -Fresh     # recreate Postgres container from scratch
./test-local.ps1 -NoRace    # skip race detector (faster)
./test-local.ps1 -Down      # stop & remove test Postgres container
```

DB integration tests gated on `TEST_DATABASE_URL`. SQLite tests gated on a runtime `requireSQLite(t)` probe (skipped when `CGO_ENABLED=0`).

### Browser fixture service

`backend/api-service/cmd/fake-bungie` is test-only. From
`backend/api-service`, `go run ./cmd/fake-bungie` binds loopback port 8090,
serves `/health`, OAuth/profile/vendor/milestone/settings fixtures, and a tiny
runtime-generated zipped SQLite manifest. PUT/DELETE `/__e2e/scenario` controls
mutable scenarios. It must never contact or proxy the real Bungie API and must
never listen beyond loopback.

## Hot reload (development)

```powershell
air   # from backend/api-service/ (requires .air.toml)
```

## Running locally (without Kubernetes)

```powershell
cd backend/api-service
cp .env.example .env   # fill in secrets
go run .               # :8081
```
