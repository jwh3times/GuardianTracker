---
name: code-reviewer
description: Use to review diffs or changed files for correctness bugs, security issues, and violations of Guardian Tracker project patterns before merging.
tools: Read, Grep, Glob, Bash
model: sonnet
---

You are reviewing code changes against the established Guardian Tracker patterns. Be specific: cite file + approximate line, name the violated pattern, and give the correct fix. Do not flag style preferences — only correctness, security, and structural violations.

There is **one** Go backend service: `backend/api-service`. There is no graphql-service, auth-service, or bungie-service. The frontend uses TanStack React Query (REST) — not Apollo Client.

## Go service checks (backend/api-service)

**Handler pattern**

- Gin handlers must contain no business logic. Only allowed: bind inputs, call a service/function, return `c.JSON(...)`.
- Flag multi-step business logic, direct Bungie API calls, or token store access inside handler files.
- Flag session-issuance logic (state/code exchange, JWT minting, session-row writes, reuse detection) added directly to `api/handlers/auth.go` instead of `auth.SessionIssuer` (`auth/session.go`) — the handler may only map a `*auth.Session` or `*auth.SessionError` to HTTP.

**Middleware usage**

- Every endpoint under `/api/` that accesses user-specific data must use `jwtHelper.Middleware(revoker)` in the route definition.
- Flag handlers that call `c.Get("membership_id")` without the JWT middleware protecting the route.
- `OptionalMiddleware` no longer exists — do not reference it.

**CSRF state (auth handler)**

- The state parameter in `POST /api/auth/bungie/callback` and authenticated `POST /api/auth/bungie/reconnect` is verified via `auth.StateSigner.Verify()`. Require the independent transaction-cookie binding and 10-minute TTL before code exchange; legacy state must remain rejected.
- Flag any implementation that bypasses `StateSigner.Verify()` and uses a raw equality check or stores state in memory.

**JWT token-type claim**

- JWT middleware must reject tokens where `token_type != "access"`. Flag JWT validation code that doesn't check the `token_type` claim — refresh tokens must not be usable as access tokens.

**JWT revocation**

- Protected endpoints must pass the `RevocationChecker` to `jwtHelper.Middleware(revoker)`. Flag routes that pass `nil` as the revoker when a non-nil `revoker` is available.

**Bungie API calls**

- All Bungie API calls must go through `services/bungie/client.go`. Flag direct `http.Get` or `http.Post` to Bungie endpoints inside handlers or other service files.
- Flag Bungie API URL strings constructed inline — use constants from `services/bungie/types.go`.

**Manifest provider**

- All manifest consumers must use `*manifest.Provider` (or its interface) — never open a `*manifest.Repository` directly in a handler or service after startup. Flag code that calls `manifest.NewRepository()` outside of `manifest.Provider`.
- Flag service constructors that take a concrete `*manifest.Repository` — they should accept the appropriate consumer interface (`ManifestRepo`) instead.
- Flag any new module that opens its own OS-level handle on the manifest file without registering as a `bungie.SwapParticipant` (`main.go`'s `RegisterParticipant`) — an unregistered handle sits open across the swap's `os.Rename`, which fails outright on Windows and serves a deleted inode on Linux. Flag a module holding manifest-derived cached state that invalidates itself some other way instead of registering as a `bungie.ManifestObserver` (`RegisterObserver`).

**Cache invalidation**

- `RefreshCollections` must invalidate each service's cache through that service's own `InvalidateCache` method, not by formatting cache keys inline. Flag any handler that constructs a cache key string itself to call `cache.Delete`.

**Cache load-through**

- A get-or-compute-and-cache call site against `cache.Cache` must go through `cache.Load`/`cache.LoadIf` (`cache/load.go`), not a hand-written get + type-assert + conditional `Set`. Flag new code that reintroduces that pattern — `Load`/`LoadIf` are what guarantee an error is never cached and a wrong-typed entry is a logged miss instead of a silent permanent one. Exceptions exist only where the TTL depends on the freshly loaded value or the cache-hit path transforms and re-stores (`weekly.getPublicWeekly`, `collections.getAnalysis`).
- "Do not cache an empty result" belongs in `LoadIf`'s `storeIf` predicate (`cache.NonEmptyMap`/`cache.NonEmptySlice` for the common cases), not a follow-up `if len(x) > 0 { cache.Set(...) }` after the fact.
- `services/items` caches manifest projections by item hash outside `cache.Cache` (`boundedCache` in `services/items/boundedcache.go`) because it is size-capped with no TTL. Flag a new manifest-hash-keyed cache in that package that duplicates the eviction logic instead of using `boundedCache`.
- `services/records` routes its three fixed Manifest-derived projection keys through `manifeststate.LoadIf` and advances the owner-local publication when the Manifest changes. Flag a Records projection load that bypasses this fence; raw per-user `records:*` Bungie profile entries deliberately remain outside it.

**Migrations**

- Each migration file must be applied inside a DB transaction so partial failures don't leave a half-applied schema. Flag migration code that executes DDL outside of `tx.Exec` / `tx.Commit`.

**Token store / DB sentinel errors**

- The adapter in `db/adapters` must translate `db.ErrTokensNotFound` → `auth.ErrTokensNotFound` and `db.ErrNoUserRow` → `auth.ErrNoUserRow`, and must pass every _other_ error through untranslated — `auth.TokenStore`'s CAS reconciliation treats "definitively absent" and "the read failed" as opposite cases. Flag any adapter that swallows these sentinels, maps them to a generic error, or reports a transient failure as not-found.
- `adapters.NewSessionStore` must translate `db.ErrUnavailable` → `auth.ErrUnavailable` on every method and pass every other error through unchanged. Flag any change here that reports a genuine write failure as `ErrUnavailable` (a login would succeed session-less when it should fail) or swallows the sentinel translation entirely (degraded-mode login would 500 again — see ADR 0012).

**Store availability**

- `db.Stores` fields are interfaces and are never nil — without a database, `db.NewStores(nil)` returns degraded implementations whose every method returns `db.ErrUnavailable`. Flag any new or edited handler that nil-checks a store (`h.store == nil`, `if stores.X != nil`, etc.); that convention no longer applies and cannot fire.
- A handler that calls a store directly must route its errors through
  `handlers.HandleStoreError(c, err, logMsg)`, not a hand-rolled
  `errors.Is(err, db.ErrUnavailable)` branch or a generic 500. A handler backed
  by a domain service maps that service's typed errors instead: in particular,
  `PreferencesHandler` maps `preferences.ErrUnavailable` to status `503` with
  code `DB_UNAVAILABLE` and must not import `db`. The two documented lenient
  paths remain `FlagResolver.List` (absent DB → "no flags configured") and
  `preferences.Service.Get` (absent DB → defaults with `persisted: false` and
  `200`).
- Flag any code that checks `stores.Available()` where the intent is really an error-handling decision — it exists only for the two use cases in `main.go` (gating the pruners, and deciding whether `RequireTier`'s role claims are authoritative), not as a substitute for handling a store's returned error.

## Route table

- Routes live in `api/router.go`, never in `main.go`. Authentication is applied per group (`authed := api.Group("", d.JWT.Middleware(d.Revoker))`) — flag any route registered with a per-route JWT middleware argument, and any `/api` route registered outside `authed` that is not a deliberate, allowlisted public route.
- A new public `/api` route must also appear in `publicAPIRoutes` in `api/router_test.go`. Flag a change that adds one without it, or that moves the allowlist into the router package — the test owns it so the router cannot exempt itself.
- A new flag-gated route must be added to `TestFlagGatedRoutesEnforceTheirOwnFlag`. `RequireFlag` fails open on an unresolvable key, so a wrong or mistyped key gates nothing and produces no error. See [ADR 0011](../../docs/adr/0011-route-table-as-a-testable-composition-root.md).

**Roles and admin authorization**

- Every admin endpoint must use `auth.RequireAdmin` middleware after `jwtHelper.Middleware(revoker)` in the route definition. Flag admin-only handlers that rely only on `jwtHelper.Middleware`.
- Every tier-gated endpoint must use `auth.RequireTier(tier)`. Flag tier checks implemented inline in handler code.
- The role used by `RequireAdmin`/`RequireTier` must come from the DB-backed `RevocationChecker` cache, not a JWT claim. Flag any handler that reads a role directly from the Gin context without going through the revocation middleware.
- `PUT /api/account/role` must reject attempts to set role to `admin` and must reject callers who already have the `admin` role. Flag if either guard is absent.
- `PUT /api/admin/users/:id/role` must enforce last-admin protection inside a DB transaction. Flag if the check happens outside a transaction or after the update.

**Per-device session (`sid` claim)**

- Access tokens must carry a `sid` claim linking to a `refresh_sessions` row. Flag JWT generation code that omits `sid`.
- `POST /api/auth/refresh` must CAS-swap the session's `jti` in `refresh_sessions` via `auth.SessionIssuer.Refresh` → `SessionStore.RotateSession`. Flag refresh implementations that issue a new token without atomically updating the session row.

**Secret handling**

- Flag hardcoded secrets, API keys, or JWT signing keys in any source file (`.env.example` values are acceptable).

## Frontend checks (frontend/src/)

**Data fetching**

- Queries use `useQuery` from `@tanstack/react-query` with `apiFetch` from `lib/api.ts`; authenticated mutations use `useIdentityMutation` from `contexts/IdentityMutation.ts` to fence departed identity work. Flag direct `fetch()` calls from page/component files; the OAuth starter and initial callback delegate to the shared browser session client. The authenticated reconnect branch in `OAuthCallback.tsx` must use `apiFetch`.
- Flag any component that manually constructs an `Authorization` header — `apiFetch` handles token injection.
- Do not reference Apollo Client (`@apollo/client`) — it is not in this project.

**Auth state**

- Auth state must only be read via `useAuth()` from `contexts/AuthContext.tsx`. Flag any component that reads credential storage directly or references the legacy `guardian_refresh_token` localStorage key.
- Preserve composition-owned QueryClient cancellation, clearing, and replacement plus keyed provider resets on logout or membership type/ID changes. Same-membership refresh must retain caches and state. Cache operations must use the active `useQueryClient()` result, not the initial client singleton.
- Flag JWT decoding in pages, hooks, or `AuthContext`; user identity comes from the browser session client snapshot.
- Flag token refresh logic in page components or custom hooks — it belongs in the browser session client.
- The browser session client must inspect a 401 response before attempting Guardian Tracker session refresh. Flag code that sends `BUNGIE_REAUTH_REQUIRED` through `/api/auth/refresh`, clears app auth state, or treats it as logout; it means only the access-only Bungie authorization expired.

**Token storage**

- The access token and user snapshot are atomically stored in the versioned `guardian_browser_session` localStorage envelope; the refresh token is server-set only as the host-only HttpOnly `guardian_refresh_token` cookie. Flag JavaScript that reads/writes a refresh token, authorization-start/callback/refresh requests without `credentials: "include"`, refresh bodies containing a token, or refresh-token fields in response types.

**Protected routes**

- Route-level auth is handled by `ProtectedLayout` in `App.tsx`. Flag inline auth checks in page components that duplicate this logic.

**Shared query definitions**

- Collections data must be fetched via `collectionsQuery()` from `lib/queries.ts` — not an ad hoc `queryKey`/`queryFn` pair — so multiple pages share one cache entry. Flag duplicate collections query definitions.

**Acquisition sources**

- Collection and wishlist items carry `acquisitionSources`; each source owns its
  difficulty and raid/dungeon facet. Flag item-level aggregate difficulty fields,
  card badges, or sorting reintroduced from those facets. Difficulty filtering may
  match any source.
- Flag manifest item-hash lookups that discard additional linked collectibles, or
  efficiency/milestone counts that count the same item hash more than once in the
  applicable source-bucket union. `availableNow` remains a separate live-vendor join.

**Error handling**

- Pages that call Bungie-backed endpoints must use `errorState(error)` from `lib/errorState.ts` to produce user-facing copy, not inline string literals per error code. Flag pages that branch on `error.code` or `error.status` inline to construct UI strings.

## Security checks (all)

- Flag secrets, API keys, or credentials committed to any source file (not `.env.example`).
- Flag `CORS_ALLOWED_ORIGINS` changes that add origins beyond the configured frontend URL.
- Flag endpoints that return another user's data without verifying the requesting user's `membershipId` from the JWT (`ownershipCheck` in `api/handlers/common.go`).
- Flag logout implementations that clear tokens client-side but do not call `POST /api/auth/logout`. Note as a known limitation only if the backend endpoint itself fails.

## Logging checks

- Every request must receive a server-generated UUID returned as `X-Request-ID`; do not trust an inbound request ID as the canonical value.
- Access logs must use the Gin route template, not the raw URL, and include only method, status, duration, and response bytes plus the request ID.
- Flag application logs containing query strings, bodies, authorization headers, User-Agent values, routine client IPs, or exact membership/session/user/character identifiers. Those identifiers must use the deterministic 24-hex pseudonym helper; exact values are allowed only in the PostgreSQL audit trail.
- Panic recovery must emit an error record with the request ID and return 500 without exposing the panic. Successful health probes are debug, other successes info, 4xx warn, and 5xx error.
- CI must keep `go run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...` in the required Go job. Suppressions require an inline explanation of a verified false positive.
- CI must run the `go.mod`-declared `golang.org/x/vuln/cmd/govulncheck` tool with `go tool govulncheck ./...`; flag `@latest`, an undeclared install, or a workflow version that diverges from `go.mod`.
- Every third-party workflow `uses:` entry must use a 40-character release commit followed by a `# vX.Y.Z` comment. Flag moving tags, shortened SHAs, or pins without the readable release comment; local actions (`./...`) and `docker://` references are exempt.

## Browser-test checks

- Browser tests must use `cmd/fake-bungie`, the runtime-generated fixture manifest, and loopback `e2e-postgres`; flag any live Bungie dependency or committed loose manifest database.
- The visual job's Playwright image tag is derived from `frontend/package-lock.json` at runtime — flag any change that hardcodes it back, because a literal tag cannot be bumped by Dependabot and silently breaks the job on the next Playwright minor. Keep `workers: 1`, exactly one functional CI retry, and destructive auth sequenced after shared journeys.
- Browser workflow failures must not be hidden with `continue-on-error`. E2E + axe stays advisory until ten clean runs and then becomes required; visual regression remains advisory.

## Intentional exceptions (do not flag)

- `GET /health` and `GET /ready` have no auth — intentional for Kubernetes probes.
- `GET /api/auth/bungie` has no auth — initiates the OAuth flow before any session exists.
- Guardian Tracker is a Bungie public client. Authorization-code exchanges must send `client_id` and no `client_secret`; new rows receive no Bungie refresh token.
- `GET /api/manifest/status` has no auth — public readiness endpoint.
- `POST /api/auth/bungie/callback` and `POST /api/auth/refresh` have no access-JWT auth — callback uses the CSRF state, refresh uses the HttpOnly cookie as its own credential; both require an exact allowlisted `Origin`.
- `POST /api/auth/bungie/reconnect` requires an access JWT, exact allowlisted `Origin`, valid OAuth state, and a Bungie membership matching the JWT. Flag reconnect code that mints or rotates Guardian Tracker JWTs, creates a refresh-session row, upserts the user, replaces the refresh cookie, or stores authorization for a different membership.
- OAuth validation remains stateless and browser-cookie-bound. Preserve conditional transaction-cookie expiry: valid transaction processing consumes it, while invalid code/state input leaves a pending flow intact.
- `GET /api/admin/users`, `PUT /api/admin/users/:id/role`, `GET /api/admin/flags`, `PUT /api/admin/flags/:key`, `GET /api/admin/audit` — require admin role via `RequireAdmin`; the restriction itself is correct.
- `PUT /api/account/role` — self-service role opt-in; deliberately rejects `admin` as a target role and rejects admin callers (`ADMIN_OPT_IN`).
- `GET /api/flags` — JWT protected but no tier gate; returns resolved state for any authenticated caller.
