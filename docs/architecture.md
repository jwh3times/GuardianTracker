# Architecture

Guardian Tracker is a two-service Destiny 2 companion app. Players authenticate
with Bungie OAuth, the API stores the access authorization encrypted, and the
frontend renders collection, wishlist, weekly, and settings surfaces through
direct REST calls to the API service.

## Runtime Shape

```text
React/Vite frontend (:5273)
    -> Go/Gin API service (:8081)
        -> Bungie API
        -> Postgres user data
        -> SQLite Destiny manifest
```

- **Frontend:** React 19, TypeScript, Vite, React Router, TanStack Query, and the
  custom `gt-*` design system.
- **API:** Go + Gin HTTP service with Bungie OAuth, JWT access/refresh tokens,
  manifest management, collection analysis, weekly recommendations, search,
  wishlist, preferences, roles, flags, admin endpoints, and structured request
  logging.
- **Postgres:** users, wishlist, preferences and onboarding completion, encrypted
  Bungie access authorization, Guardian Tracker refresh sessions, roles, feature
  flags, and audit log.
- **SQLite:** local copy of the Bungie Destiny 2 manifest, downloaded and
  swapped by the API service.
- **Cache:** in-memory service caches for collection results, weekly data,
  manifest-derived indexes, and role/revocation checks; the search index also
  persists as a versioned snapshot beside the manifest.

## Authentication and Sessions

Bungie OAuth login uses a public client and stateless HMAC-signed v2 CSRF state
bound to an independent browser transaction cookie. The credentialed
`GET /api/auth/bungie` sets a 10-minute host-only HttpOnly `SameSite=Lax`,
`Path=/` cookie (`__Host-guardian_oauth_transaction`, `Secure` in production;
`guardian_oauth_transaction` in development). Callback and reconnect require
the matching cookie and unexpired signed state before code exchange, with no
legacy-state fallback. The latest start supersedes an earlier pending flow in
the same browser; matching completion can occur across tabs. Processing valid
transaction input expires the cookie, while invalid input leaves it intact.
The authorization-code grant sends the public client ID without a client secret;
Bungie public clients return an expiring access token without a refresh token.
The API stores that access-only authorization AES-256-GCM encrypted in Postgres
and issues separate Guardian Tracker JWTs. One module, `auth.SessionIssuer`, owns
the whole session lifecycle — starting the OAuth flow, turning the initial
callback or a Guardian Tracker refresh into a session, reconnecting Bungie, and
ending one session or all of them; the Gin handlers only map the result to HTTP
(status code, cookie, audit event, response body).

Once the Bungie access authorization enters its five-minute expiry buffer,
strict endpoints that require it return `401 BUNGIE_REAUTH_REQUIRED`. The
frontend routes the still-authenticated user through Bungie again, then sends the
code and state to authenticated `POST /api/auth/bungie/reconnect`. Reconnect
verifies that the newly authorized Destiny membership matches the Guardian
Tracker JWT, replaces only the encrypted Bungie authorization, and returns 204.
It does not issue or rotate Guardian Tracker JWTs, create a refresh-session row,
change the user, or replace the refresh cookie.

Access tokens are short-lived bearer tokens stored with the non-secret user
snapshot in browser localStorage. Refresh tokens are rotating, per-device
sessions backed by Postgres and delivered only through a host-only HttpOnly
cookie scoped to `/api/auth`. Reused refresh tokens revoke the affected session.
Single-device logout preserves the membership-wide Bungie authorization.
Sign-out-everywhere bumps the user's token version, removes every Guardian
Tracker session, and evicts the Bungie authorization.
The production browser session client owns the access-token/user projection in
one versioned `guardian_browser_session` localStorage envelope, including durable
anonymous logout state. Its browser adapters supply persistence, the API transport,
and an origin-wide Web Lock for authorization start, callback, authenticated
reconnect, refresh, and logout. Valid legacy
access-token/user pairs migrate once when no envelope exists; legacy keys are then
removed. Authorization start, callback completion, authenticated reconnect, and
refresh require Web Locks; local logout can still
persist when coordination is unavailable, with server cleanup best effort.

`AuthProvider` is a declarative `useSyncExternalStore` projection of the shared
client and performs no JWT decoding or hydration profile request. `apiFetch`
delegates credential attachment and refresh to that same client and adapts REST
responses/errors; initial OAuth completion also delegates to the client. Bungie
reauthorization retains its separate route and authenticated reconnect request.
Application composition observes the browser session through
`lib/applicationIdentity.ts`. Becoming anonymous or changing the Destiny
membership type/ID cancels and clears the departing QueryClient, supplies a fresh
client, and remounts the keyed provider subtree. This resets preferences,
onboarding, flags, character state, and page-local drafts; same-membership refresh
retains the client and mounted state. Identity boundaries also best-effort clear
the global `gt_done:` weekly checklist storage so completion marks do not carry
into another account. `useIdentityMutation` fences mutation starts
and callbacks from a departed identity, while late cache work remains isolated
in the retired client. Preferences also guard asynchronous completion and reset
the global local preference snapshot at identity boundaries. See
[ADR 0017](./adr/0017-own-the-browser-session-projection.md).

Without a configured database, login still succeeds without a session row; a
session write failure with a database configured still fails the login, since
the access token is checked against that row on every request.

The callback, authenticated Bungie reconnect, and Guardian Tracker refresh
endpoints require an exact allowlisted browser origin. This cookie policy assumes
the frontend and API are same-site; a cross-site deployment would require a new
cookie and CSRF decision.

Authorization reads the current role from the DB-backed revocation cache rather
than trusting the JWT role hint.

## Roles, Flags, and Admin

Roles are `standard`, `beta`, `alpha`, and `admin`. Admin users can manage user
roles and feature flags through the admin console. Last-admin demotion is blocked
transactionally. Self-service opt-in uses `UserStore.SetSelfRole`: it locks the
current user row, refuses admin callers even if request middleware had a stale
role, and commits the role update and `role.optin` event in one transaction.
The event's prior role comes from that locked row. Audit failure rolls back the
update; admin refusal maps to `403 ADMIN_OPT_IN`.

Successful self-service changes evict the local authorization cache without
changing token version or sessions. Bootstrap admin upserts also evict the local
cache after success. Other replicas retain the normal 60-second cache window;
the database guard still protects self-service updates during that interval.
Admin role and flag mutations also commit their audit events transactionally;
authentication/session audit writes remain best effort. After successful session
refresh, the handler emits `refresh.success` using the verified membership and
session identifiers, with request IP/User-Agent supplied by the audit helper.
It includes no credential values; audit persistence failure leaves the successful
refresh response and replacement cookie intact. The admin audit table labels this event "Session refreshed".

Feature flags control frontend visibility and rollout state. Server-side
authorization remains the boundary for protected API surfaces.

## Bungie Data Flow

The API talks to Bungie for:

- public-client OAuth authorization-code exchange and authenticated reconnect
- Bungie account and Destiny membership data
- collections, records, characters, and weekly public data
- manifest version and manifest database download

The app does not treat Bungie data as live. User-facing surfaces should show
freshness and reset timing rather than implying real-time state.

## Manifest Pipeline

On startup and periodic checks, the API downloads the current Destiny manifest,
extracts the SQLite database, opens it through the manifest provider, and notifies
dependent services to rebuild manifest-derived indexes.

Collections, cosmetics, catalysts, crafting, triumphs, search, and item detail
views all depend on the manifest. The search index restores a matching versioned
snapshot from beside the manifest on startup, then rebuilds asynchronously when
the snapshot is missing or the manifest changes. A build that fails is retried by
the next search request rather than waiting for the next manifest swap, throttled
to one attempt per 30 seconds. Other affected endpoints can return warming
responses during cold start or manifest swap.

`services/manifeststate` provides the generation fence that keeps a slow request
from republishing state derived from a manifest that has since been replaced.
Work captures the current generation before it reads, and may install its result
only while that generation is still current; a request that loses the race still
returns its own coherent result but leaves nothing behind for anyone else.
Advancing the generation and running the owner's invalidation are one
transition, so a loader cannot observe a moved generation over uncleared state.
Items, Records, and Efficiency hold owner-local publications. Records fences the three fixed
Manifest-derived lookup tables used to enrich its projections while leaving raw
per-membership Bungie profile records untouched. Efficiency fences asynchronous
source-bucket index builds: a stale build cannot publish after a swap, while the
previous complete index remains available until the current generation succeeds.
Weekly and Collections still invalidate without a fence and adopt it as each is reworked. See
[ADR 0014](./adr/0014-own-manifest-derived-publication.md).

`services/items` owns the canonical, user-independent facts about an item — its
name, icon, slot-specific type, rarity, collection category, linked collectible
hashes, the deterministic union of acquisition sources contributed by all of
them, and farm-only status. Item detail reads those facts through one seam
rather than joining the manifest itself, which is what stops separate consumers
describing the same item differently. Collections and the wish list still carry
their own projections and move onto this seam as they are reworked. See
[ADR 0015](./adr/0015-own-item-acquisition-facts-in-items.md).

## Collection and Acquisition Model

An inventory item can be linked from several manifest collectibles. Collection
and wishlist responses preserve that multiplicity as a deterministic, deduplicated
`acquisitionSources` union. Each entry carries its source text, the difficulty tier
classified from that text, and a raid/dungeon facet. Difficulty is source-scoped;
items do not carry a single aggregate difficulty. This replaces the former item-level
`difficulty` and `sources` REST fields on collection items, and the former `sources`
field on wishlist items.

Live `availableNow` vendor data is joined separately because acquisition sources
describe provenance rather than current availability. The Collections difficulty
filter matches an item when any source has the selected tier; item cards summarize
source text or count without presenting an item-level tier, while the detail drawer
shows every source with its own tier. Wishlist items use the same acquisition-source
shape. Difficulty sorting has been removed; legacy URL and persisted
`sort=difficulty` values migrate to the default rarity sort.

The efficiency index counts an item hash once within each source bucket, and a
milestone missing count counts the union of item hashes across all matching buckets.
It returns ordered, capped source-bucket facts with an explicit cold/ready state;
internal scores do not cross that seam. `services/recommendations` turns those facts
into complete player-facing outcomes and owns wording, explanation, emphasis,
source/action-scoped difficulty, and Xûr or weekly-reset fallback selection. Weekly gathers
the player and live facts, invokes that policy once, and performs field-for-field
wire assembly. Farm-only classification retains its existing behavior and is not
inferred across an item's full source union. See
[ADR 0016](./adr/0016-own-acquisition-recommendation-outcomes.md).

## API Surface

Primary route groups:

- auth: Bungie login, initial callback, authenticated Bungie reconnect, Guardian
  Tracker session refresh, logout, logout-all
- account: Guardian Tracker user snapshot, role opt-in, feature flags
- collections: collection tree, refresh, item views, item perk pools
- weekly: recommendations, Xur inventory and authenticated location, milestones,
  reset countdowns; authenticated vendor calls validate and follow the selected
  character because Bungie's vendor inventory can be class-specific
- wishlist: user-scoped CRUD
- preferences: user preferences plus irreversible first-run onboarding completion
- records: catalysts, crafting, seals
- characters: Destiny membership characters
- admin: users, roles, flags, audit log
- health: `/health` liveness and `/ready` readiness; readiness requires the
  manifest and, when a database pool is configured, a successful database ping

See `backend/api-service/api/router.go` for the authoritative route
registration. Authentication is applied once to the authenticated group rather
than per route, and the invariants — every `/api` route behind the JWT gate,
each flag-gated route enforcing its own flag key, admin routes refusing
non-admins and degraded builds — are asserted against the built route table in
`api/router_test.go`. See
[ADR 0011](./adr/0011-route-table-as-a-testable-composition-root.md).

Preferences are owned behind the HTTP boundary by `services/preferences`, which
defines the defaults, validates partial patches, and owns irreversible,
server-stamped onboarding completion. Its consumer-side repository is keyed by
Destiny membership; the adapter in `db/adapters` resolves PostgreSQL's internal
user ID and translates storage values without exposing either detail to the
service or Gin. The store applies every supplied field in one atomic statement,
so independent partial updates cannot restore stale values.

`GET /api/preferences` remains available in degraded mode. It returns `200` and
adds `persisted: true` for every authoritative read, including a genuinely new
account whose defaults have no stored row, or `persisted: false` when the values
are unstored defaults returned because persistence is unavailable. Writes do not
degrade: `PUT /api/preferences` returns `503 DB_UNAVAILABLE` when persistence is
unavailable. `PreferencesHandler` owns request binding, typed error mapping, and
serialization; preference policy does not live in Gin.

## Request Logging

The API generates a UUID for every request, exposes it as `X-Request-ID`, and
attaches a request-scoped `log/slog` logger to the Go context. Access records use
the matched route template rather than the raw URL and include method, status,
duration, and response bytes. Successful health probes are debug records;
successful application requests are info, 4xx are warning, and 5xx are error.

Application logs never include query strings, bodies, authorization headers,
User-Agent values, or routine client IPs. Membership, session, user, and
character identifiers use deterministic 24-hex pseudonyms derived from the
first 12 SHA-256 bytes. Exact identifiers remain only in the existing Postgres
audit trail.

## Local Infrastructure

Docker Compose is the recommended full-stack development path. It starts the
frontend, API service, Postgres, pgAdmin, and profile-gated disposable test
databases.
Database and pgAdmin host ports are loopback-only; frontend and API bindings are
unchanged.

Minikube manifests under `k8s/` validate container and Kubernetes wiring. That
environment runs in development mode and is not production parity.

## Security Posture

- Secrets are read from environment files or runtime environment variables, never
  committed.
- Bungie's access-only authorization is encrypted at rest with exact
  current/previous key versions.
- CORS allows only configured origins.
- API server timeouts, body limits, no-sniff/referrer headers, and no-store auth
  responses are configured.
- The frontend CSP disallows inline scripts; inline styles remain an explicitly
  documented residual risk.
- Admin and role changes are audited.
- Audit rows include client IP and User-Agent, retained by configured policy.

See [SECURITY.md](../SECURITY.md) for the security guide and checklist.

## Tests

- Frontend: Vitest, Testing Library, MSW, type-check, lint, build.
- Browser: Playwright against the real API/Vite plus a test-only fake Bungie
  service and runtime-generated SQLite manifest; functional, WCAG 2.2 axe, and
  deterministic visual projects share one worker and isolated Postgres state.
- Backend: Go unit and integration tests, `go vet`, Staticcheck 2026.1, the
  declared `govulncheck` v1.6.0 tool, race detector in CI, Postgres-backed
  integration tests.
- Docker: CI builds production images for validation.

See [frontend/README.md](../frontend/README.md#browser-tests) for the operational
browser-test and visual-baseline procedure.

CI pins every third-party GitHub Action to a reviewed release commit and checks
the full-SHA plus release-comment policy in the format job. Dependabot manages
both those action pins and the declared Go vulnerability-scanner tool.
Frontend development, CI, and container builds take one exact Node 26 patch from
the root `.nvmrc`; a repository policy test checks the workflows, Dockerfiles,
package engine range, and Node ambient types for alignment. The three Compose
PostgreSQL services and the `Test Go Services` service container share one
`major.minor` server version across their Alpine and Debian variants; a
repository policy test enforces the alignment and rejects a retired
`postgres:<version>` reference left in a tracked file.

The browser workflow makes functional/axe a required check after exceeding ten
consecutive clean runs. Visual comparison remains advisory and runs in the
Playwright Noble image whose tag CI derives from the frontend lockfile, so the
container's browsers always match the installed `@playwright/test`.

## Related Decisions

See [docs/adr](./adr/README.md) for accepted architecture and operating decisions.
