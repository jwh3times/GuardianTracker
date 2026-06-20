# Guardian Tracker — Claude Code Guide

## Project Overview

Guardian Tracker is a Destiny 2 collection tracker web app. Players log in via Bungie OAuth, and the app analyzes their in-game collections to surface missing items with acquisition difficulty ratings, wish-list management, and weekly recommendations.

## Architecture

Two services: a Go API backend and a React frontend that calls it directly over REST.

```text
Frontend (React/TS :5273)
    └─► API Service (Go/Gin :8081)  — OAuth, JWT, manifest, collections
```

### Service Ports

Frontend on `5273`, API Service on `8081`. For the full port map — core services,
Docker Compose mappings, Kubernetes, and dev/cross-service wiring — see
**[Ports in the README](./README.md#ports)**, which is the single source of truth.

## Running Services

Pick the option that matches what you're doing:

- **Docker Compose** (Option A) — one command to run the whole stack. Best default for local dev, onboarding, and integration testing.
- **Minikube** (Option B) — for validating the Kubernetes manifests / deployment parity. Overkill for everyday work.
- **Individual services** (Option C) — for actively developing a single service with fast hot reload (Air / Vite).

### Option A: Docker Compose (full stack)

Runs the API service, frontend, Postgres, and Redis from their production Dockerfiles.

```powershell
cp .env.example .env      # fill in BUNGIE_* secrets for real OAuth
docker compose up --build
```

- Frontend `http://localhost:5273`, API `http://localhost:8081`, pgAdmin `http://localhost:5150`
- Postgres `:5532`, Redis `:6379`
- The Bungie manifest persists in the `manifest-data` named volume, so it isn't re-downloaded on restart.
- `database/init/01-init.sql` auto-loads into Postgres on first run.

```powershell
docker compose down        # stop (keeps volumes/data)
docker compose down -v     # stop and wipe Postgres/Redis/manifest volumes
```

### Option B: Kubernetes (Minikube)

```powershell
cd k8s
./startup.ps1
```

Dev-validation only: runs `GO_ENV: development` with no Postgres, so the api-service is
in degraded mode (in-memory tokens, no persistence). Production parity lives in the Azure
Container Apps deployment, not these manifests.

### Option C: Individual services

```powershell
# API Service
cd backend/api-service
go run .

# Frontend
cd frontend
npm start
```

### Hot Reload (Air)

The API service has `.air.toml` configured. Use `air` instead of `go run .` for hot reload during development.

### Exposing via ngrok (public HTTPS)

To test Bungie OAuth against a public HTTPS URL, tunnel the frontend with ngrok. Two things must know about the ngrok origin:

1. **CORS** — add the ngrok URL to `CORS_ALLOWED_ORIGINS` in the root `.env`, then rebuild api-service:

   ```powershell
   # root .env
   CORS_ALLOWED_ORIGINS=http://localhost:5273,https://<sub>.ngrok-free.dev

   docker compose up -d --build api-service
   ```

2. **OAuth redirect** — set `AUTH_REDIRECT_URI` to the ngrok callback and add the same URL to your Bungie app settings at <https://www.bungie.net/en/Application>.

**Caveats:**

- The frontend calls the backend at the `localhost` URLs baked in at build time, so this only works when the browser runs on the same machine as Docker.
- Free ngrok subdomains change on each restart — update `CORS_ALLOWED_ORIGINS`, `AUTH_REDIRECT_URI`, and the Bungie app redirect each time.

## Environment Setup

Copy and fill each `.env.example` before running:

```powershell
# Root (used by docker-compose)
cp .env.example .env

# API service (used when running individually)
cd backend/api-service
cp .env.example .env

# Frontend
cd frontend
cp .env.example .env.local
```

Or run `./setup.ps1` to copy all at once.

### Required secrets

- `BUNGIE_API_KEY` — from <https://www.bungie.net/en/Application>
- `BUNGIE_CLIENT_ID` — from Bungie app settings
- `BUNGIE_CLIENT_SECRET` — from Bungie app settings
- `JWT_SECRET` — 32+ char random string (`openssl rand -base64 32`)
- `DATABASE_URL` — Postgres connection string (`postgres://guardian_app:...@host:5432/guardian_tracker?sslmode=disable`)
- `TOKEN_ENCRYPTION_KEY` — 32-byte base64 key for Bungie token encryption (`openssl rand -base64 32`)
- `TOKEN_ENCRYPTION_KEY_PREVIOUS` — (optional) previous key for rotation during key migration
- `ADMIN_MEMBERSHIP_IDS` — (optional) comma-separated Bungie membership IDs pinned to the admin role at every login; the only way to bootstrap the first admin (additional admins are then granted from the Admin Console)

## Key Files

### API Service (`backend/api-service/`)

| File | Purpose |
| --- | --- |
| `main.go` | Gin router, dependency wiring, manifest startup |
| `config/config.go` | Typed config with env var parsing helpers |
| `auth/jwt.go` | JWT generation and validation (access 24h, refresh 30d) |
| `auth/middleware.go` | JWT middleware for protected routes |
| `auth/state.go` | Stateless HMAC-signed OAuth state parameter (CSRF, multi-replica safe) |
| `auth/tokenstore.go` | DB-backed encrypted Bungie OAuth token store with auto-refresh + CAS write |
| `auth/crypto.go` | AES-256-GCM cipher for Bungie token encryption; key rotation via prev key |
| `auth/revocation.go` | JWT revocation (account-wide token_version + per-device session existence) + role resolution; 60s in-memory cache |
| `auth/roles.go` | Role tiers (standard/beta/alpha/admin) + `RequireAdmin`/`RequireTier` tier-gating middleware |
| `api/handlers/auth.go` | OAuth flow, token refresh, logout, profile endpoints |
| `api/handlers/characters.go` | HTTP handler for characters |
| `api/handlers/collections.go` | HTTP handler for collections (incl. cosmetics) |
| `api/handlers/wishlist.go` | Wishlist CRUD with Postgres persistence and manifest enrichment |
| `api/handlers/account.go` | Self-service role opt-in + resolved feature-flag state (`GET /api/flags`) |
| `api/handlers/admin.go` | Admin console: user role management + feature-flag config |
| `api/handlers/health.go` | Health, ready, manifest status endpoints |
| `api/handlers/common.go` | Shared handler helpers (parseMembershipParams, ownershipCheck, etc.) |
| `services/bungie/client.go` | HTTP client with rate limiting + retry |
| `services/bungie/manifest.go` | Manifest download, version tracking, SQLite extraction |
| `services/bungie/types.go` | All Bungie API types, constants, helpers |
| `services/collections/service.go` | Collection analysis + difficulty classification + cosmetics |
| `services/characters/service.go` | Character fetching |
| `services/manifest/repository.go` | SQLite read-only queries against manifest DB |
| `services/manifest/provider.go` | Shared lazy-opening repository provider; reconnects across manifest swaps |
| `services/weekly/service.go` | Weekly recommendations; Xûr inventory; milestone data; reset time math |
| `services/search/service.go` | In-memory manifest item search index; async rebuild on manifest update |
| `services/records/service.go` | Catalysts, crafting patterns, and seals/triumphs from Bungie records API |
| `cache/cache.go` | In-memory cache (and no-op cache interface) |
| `db/db.go`, `db/migrate.go`, `db/migrations/{0001_init,0002_roles_flags,0003_refresh_sessions,0004_audit_log}.sql` | Postgres pool, migration runner, schema DDL (0002 adds roles, feature_flags, role_audit; 0003 adds refresh_sessions for per-device sessions + refresh-token reuse detection; 0004 unifies audit_log and drops role_audit) |
| `db/audit.go` | Unified append-only audit trail store (`audit_log`): best-effort `Log` + in-transaction `insertAudit`, filtered/keyset `List`, retention prune |
| `db/users.go`, `db/tokens.go`, `db/wishlist.go`, `db/prefs.go`, `db/flags.go` | DB stores for users (+roles/audit + per-device refresh_sessions), encrypted Bungie tokens, wishlist, preferences, feature flags |

**Endpoints:**

- `GET /api/auth/bungie` — initiate OAuth, returns auth URL + CSRF state
- `POST /api/auth/bungie/callback` — exchange code for JWT tokens
- `POST /api/auth/refresh` — rotate access + refresh tokens (per-session, with reuse detection)
- `GET /api/auth/validate` — validate JWT (protected)
- `GET /api/auth/profile` — current user profile (protected)
- `POST /api/auth/logout` — end the current device's session; other devices stay signed in (protected)
- `POST /api/auth/logout/all` — sign out everywhere: bump token_version + delete all sessions + Bungie token (protected)
- `GET /api/wishlist` — list wishlist items with `availableNow` (Xûr cross-check), `sources`, `icon` (JWT protected)
- `POST /api/wishlist` — add wishlist item (JWT protected)
- `PUT /api/wishlist/:id` — update wishlist item priority/notes (JWT protected)
- `DELETE /api/wishlist/:id` — remove wishlist item (JWT protected)
- `GET/PUT /api/preferences` — user preferences: card style, personalize (protected)
- `PUT /api/account/role` — self-service opt-in to standard/beta/alpha; `admin` rejected, admin callers rejected (protected)
- `GET /api/flags` — resolved feature-flag state for the caller (`enabled`/`accessible`/`locked` + role) (protected)
- `GET /api/admin/users?q=` — admin only: user roster (id, displayName, platform, role, lastActive)
- `PUT /api/admin/users/:id/role` — admin only: set any role; last-admin protected; bumps target token_version + audits
- `GET /api/admin/flags` — admin only: full feature-flag config
- `PUT /api/admin/flags/:key` — admin only: toggle `enabled` / set `minTier`
- `GET /api/admin/audit` — admin only: filtered, keyset-paginated audit feed (`type`, `actor`, `target`, `outcome`, `before`/`after`, `cursor`, `limit`)
- `GET /api/characters/:membershipType/:membershipId` — user characters (JWT protected)
- `GET /api/collections/:membershipType/:membershipId` — user collections + `fetchedAt`; `?include=all` adds `collectedItems` per category (JWT protected)
- `POST /api/collections/:membershipType/:membershipId/refresh` — invalidate cache (JWT protected)
- `GET /api/manifest/status` — manifest version and readiness
- `GET /api/weekly/recommendations` — weekly data, Xûr, milestones, recommended actions + `fetchedAt`/`resetAt` (protected)
- `GET /api/items/search?q=&limit=` — manifest item search; 503 until index ready (protected)
- `GET /api/catalysts/:membershipType/:membershipId` — `{ items, fetchedAt }` exotic catalyst progress incl. weapon type/icon (protected)
- `GET /api/crafting/:membershipType/:membershipId` — `{ items, fetchedAt }` crafting pattern progress (protected)
- `GET /api/seals/:membershipType/:membershipId` — `{ items, fetchedAt }` triumph/seal completion (protected)
- `GET /health` and `GET /ready` — health/readiness probes

Error responses carry a machine-readable `code` (`PRIVACY_RESTRICTION`, `ACCOUNT_NOT_FOUND`, `RATE_LIMITED`, `MANIFEST_NOT_READY`, `BUNGIE_ERROR`, `INTERNAL_ERROR`) that the frontend branches its error states on. Role/flag endpoints add `FORBIDDEN`, `TIER_LOCKED`, `DB_UNAVAILABLE` (degraded mode), `LAST_ADMIN`, `ROLE_NOT_ALLOWED`, and `ADMIN_OPT_IN`.

**Bungie manifest flow:**

1. On startup, service fetches manifest metadata from `https://www.bungie.net/Platform/Destiny2/Manifest/`
2. Downloads the English `.content` (SQLite) ZIP file from Bungie CDN
3. Extracts and stores at `./data/manifest.sqlite`
4. Background goroutine checks for updates every hour (configurable)

### Frontend (`frontend/src/`)

The UI uses the "Guardian Tracker" design system: a custom dark theme built on oklch design tokens and `gt-*` CSS classes (not Tailwind utilities for new work). Layout is a persistent sidebar + top bar shell. See `frontend/design/` for the source design and `frontend/README.md` for the full component map.

| Path | Purpose |
| --- | --- |
| `App.tsx` | Router, lazy-loaded pages, `ProtectedLayout` (AppShell + auth gate) |
| `index.tsx` | App root; imports `styles/{tokens,kit,app,admin}.css` |
| `contexts/AuthContext.tsx` | Auth state, localStorage persistence, token refresh; `logout` (this device) + `logoutAll` (everywhere) |
| `contexts/PreferencesContext.tsx` | User prefs (card style, "for you" badges); localStorage `guardian_prefs` |
| `contexts/CharacterContext.tsx` | Characters query + persisted active-character pick (display-only; data is account-wide) |
| `contexts/FlagsContext.tsx` | `GET /api/flags` query; `useFlag(key)` / `useFlags()` resolved gating state + role (port of design `useGT`) |
| `lib/roles.ts` | Role/tier constants, labels, colors (port of design `admin-data.js`) |
| `lib/api.ts` | `apiFetch` helper + `ApiError` (status/code) + `QueryClient` — all REST calls go through here |
| `lib/adapters.ts` | API response types → design `GTItem`/`WishlistEntry`; `relTime` |
| `lib/constants.ts` | Label constants (`RARITIES`, glyphs) + `BUNGIE_CDN` base URL |
| `lib/errorState.ts` | `errorState(error)` → UI copy; branches on `ApiError.code` (`PRIVACY_RESTRICTION`, `MANIFEST_NOT_READY`, `BUNGIE_ERROR`) |
| `lib/queries.ts` | `collectionsQuery()` — shared React Query definition used by Dashboard, Collections, and Settings |
| `styles/{tokens,kit,app,admin}.css` | Design tokens + component/shell/admin styles (plain CSS) |
| `components/AppShell.tsx` | Sidebar + top bar + mobile nav; global search; character switcher; flag-gated nav + admin nav |
| `components/Brand.tsx` | Logo mark |
| `components/kit/` | Design component kit: `Icon`, primitives, `ItemCard`, composites, `admin` (RoleBadge/Switch/RoleSelect/TierSegment/FlagCard/UserRow/LockedFeature) |
| `components/ui/` | Legacy primitives: `LoadingSpinner`, `Toast` (`ErrorBoundary` lives in `components/`) |
| `pages/Login.tsx` | Bungie OAuth initiation |
| `pages/OAuthCallback.tsx` | Handles `/auth/callback` — exchanges code, stores tokens |
| `pages/Dashboard.tsx` | Completion hero + "do this today"; real collection totals + cosmetics, real weekly |
| `pages/Collections.tsx` | Category tree + filterable item grid/list + detail drawer; DataFreshnessChip wired |
| `pages/WishList.tsx` | Wishlist management; real API with optimistic mutations |
| `pages/ThisWeek.tsx` | Weekly recommendations / Xûr / milestones (real API) |
| `pages/Catalysts.tsx` | Catalysts & crafting patterns (real API) |
| `pages/Triumphs.tsx` | Triumphs & seals (real API) |
| `pages/Settings.tsx` | Account info + early-access tier opt-in + appearance preferences + sign out |
| `pages/Admin.tsx` | Admin console: user roster + role management, feature-flag config (admin-gated route) |
| `types/api.ts` | API response types (`APIUser`, `AuthTokenResponse`, etc.) |
| `types/design.ts` | Design-system domain types (`GTItem`, `Seal`, `Weekly`, …) |

### Database (`database/init/`)

- `01-init.sql` — `guardian_app` least-privilege role bootstrap (run once after server provisioning; the application schema is in `db/migrations/0001_init.sql`, applied automatically at startup)

### Kubernetes (`k8s/`)

- `api-service.yaml`, `api-service-configmap.yaml`, `api-service-secret.yaml`
- `frontend.yaml`
- `startup.ps1` / `shutdown.ps1` — Minikube lifecycle scripts

## Development Notes

### Manifest Database

The Bungie manifest is a ~100MB SQLite file. On first run the API service downloads it (~10–30s). The version is tracked in `./data/manifest_version.txt`. The service gracefully starts without the manifest (collections endpoint returns 503 until ready).

### Token Flow

```text
1. Frontend → GET /api/auth/bungie → returns authUrl + CSRF state
2. User → Bungie.net → redirects to /auth/callback?code=...&state=...
3. Frontend → POST /api/auth/bungie/callback → gets JWT access + refresh tokens (bound to a new refresh_sessions row); user upserted in DB
4. Frontend stores tokens in localStorage (guardian_token, guardian_refresh_token)
5. lib/api.ts apiFetch injects Authorization: Bearer <token> on every request
6. React Query hooks call apiFetch for all data fetching
7. API service validates JWT on protected routes; RevocationChecker verifies token_version + session existence (60s cache)
8. API service uses stored Bungie OAuth token (AES-256-GCM encrypted in DB, auto-refreshed if expired) for Bungie API calls
9. Logout: POST /api/auth/logout ends the current session only; POST /api/auth/logout/all bumps token_version + deletes all sessions + evicts the Bungie token. Client clears localStorage either way
```

### Authentication Security

- CSRF: stateless HMAC-signed OAuth state (`auth/state.go`, key derived from `JWT_SECRET`), 10-min TTL — survives restarts and works across replicas; not single-use (replay bounded by the TTL and Bungie's single-use auth code)
- JWT: HS256, access=24h, refresh=30d, token-type claim prevents refresh tokens being used as access tokens; `tver` (token_version), `jti`, and `sid` (session id) claims
- Per-device refresh sessions with reuse detection: each login opens a `refresh_sessions` row (the `sid` claim) holding the current refresh `jti`; `POST /api/auth/refresh` compare-and-swaps it (`UserStore.RotateSession`). A replayed (already-rotated) token is detected as reuse and **revokes the whole session** (401, even if the revoking commit errors). Sessions are independent → fully multi-device; CAS fails open on genuine DB error. `expires_at` slides forward on each rotation (active sessions don't hard-expire at creation+TTL); sessions are capped per user (`maxSessionsPerUser`) and an hourly `startSessionPruner` deletes expired rows. A failed `CreateSession` fails the login/refresh (the session is load-bearing for the access token). Pre-`0003` tokens (no `sid`) are adopted into a fresh session on first refresh
- Two logout scopes: `POST /api/auth/logout` ends only the current session (others stay; Bungie token preserved); `POST /api/auth/logout/all` bumps `token_version` + deletes all sessions + evicts the Bungie token. The middleware checks both `token_version` (account-wide) and session existence (per-device) via `RevocationChecker` with a 60s in-memory cache window
- Bungie OAuth tokens stored AES-256-GCM encrypted in `bungie_tokens` table; survive scale-to-zero. Refresh writes use compare-and-swap on `updated_at` — a replica that loses the race adopts the winner's tokens
- Rate limiting: Bungie API client — 10 req/s, burst 20
- Roles & feature flags (item 13): tiers `standard(0) < beta(1) < alpha(2) < admin(3)` in `users.role`. Authorization always reads the role from the DB-backed `RevocationChecker` cache (now `{token_version, role}`), never from the JWT — so role changes propagate within the 60s window. Admin is bootstrapped only via `ADMIN_MEMBERSHIP_IDS` (pinned at login) or granted by an existing admin; there is no self-service path to admin. Self opt-in (`PUT /api/account/role`, standard/beta/alpha only) evicts the cache entry with **no** token_version bump (session preserved); admin-driven changes (`PUT /api/admin/users/:id/role`) **do** bump token_version + evict the target's cache (forced re-sync) and write an audit row, with last-admin protection enforced inside the transaction. `RequireAdmin`/`RequireTier` gate server-side; UI flag hiding is not enforcement.
- Audit logging: authentication events (login, logout, logout-all, refresh failure), session security events (refresh reuse, session termination), self opt-in role changes, admin role changes, and feature-flag changes are persisted to the unified `audit_log` table. Role and flag changes are written in the mutation's transaction (atomic); auth/session events are best-effort (a DB outage can drop an event). Client IP and User-Agent are captured and retained for `AUDIT_RETENTION_DAYS` (default 180, hourly-pruned). Set `AUDIT_RETENTION_DAYS` in `.env` to change retention. IP addresses are trusted only from `TRUSTED_PROXIES` (gin `SetTrustedProxies`) so they cannot be spoofed; configure as comma-separated CIDR/IP ranges.

## CI/CD

GitHub Actions (`.github/workflows/ci-cd.yml`):

1. **format-check** — repo-wide formatting gate: Prettier for the frontend (`npm run format:check`) and `gofmt` for the Go services. Fails if anything wasn't formatted. Fix with `npm run format` (from `frontend/`) or `gofmt -w .` (from `backend/api-service/`).
2. **test-frontend** — type-check, lint, test with vitest coverage thresholds (lines ≥70%, branches ≥65%), build
3. **test-go-services** — go vet, govulncheck, go test with race detector; db integration tests run against the Postgres service container via `TEST_DATABASE_URL`; statement-coverage gate ≥60% (ratcheting up — CI lands ~63% with cgo+Postgres; ~52% locally where the sqlite/db packages are skipped — see [Full Go coverage locally](#full-go-coverage-locally-matches-ci))
4. **build-docker-images** — builds both Docker images; build validation only (no push configured yet)

Separately, **CodeQL** (GitHub default setup) scans go, javascript-typescript, and actions on every PR — except **Dependabot-authored PRs**, which default setup does not analyze (Dependabot runs get a read-only token with no `security-events: write`). On those PRs the aggregate `CodeQL` check reports *neutral* ("configurations not found"); that is informational and does not block (see the code-scanning note below).

### Branch protection (`main`)

`main` (and `release/**`) is governed by a single repository ruleset, **"Main/Release branch rules"** (id `17717600`, Settings → Rules), which bundles:

- **Pull request required** — no direct pushes; 0 required approvals (solo repo — self-approval isn't possible; self-merge once green is allowed), stale-review dismissal + review-thread resolution on, Copilot review on push.
- **Required status checks** (strict policy — branch must be up to date): `Format Check`, `Test Frontend`, `Test Go Services`, `Build Docker Images`. **The per-language CodeQL `Analyze (...)` contexts are deliberately _not_ required status checks** — default setup never creates them on Dependabot PRs, so requiring them blocked every Dependabot PR indefinitely. CodeQL is instead enforced by the dedicated **code-scanning merge rule** below, which is default-setup-aware and degrades gracefully when an analysis didn't run.
- **Code scanning merge protection** — CodeQL, `alerts_threshold: errors_and_warnings`, `security_alerts_threshold: medium_or_higher`. This is the real CodeQL gate for human PRs.
- **Code quality** (warnings) + **deletion** and **non-fast-forward** (force-push) blocks.
- **No bypass actors** — the rules apply to everyone, admins included.

To change the gate (e.g. add/remove a required check), edit the ruleset via `gh api repos/jwh3times/GuardianTracker/rulesets/17717600` or the GitHub UI; required status-check names must match the CI job `name:` exactly. Do **not** re-add the CodeQL `Analyze (...)` contexts as required status checks — that re-breaks Dependabot PRs; gate CodeQL through the code-scanning rule instead.

## Common Tasks

### Add a new API endpoint

1. Add handler in `backend/api-service/api/handlers/`
2. Register route in `backend/api-service/main.go`
3. Add call in `frontend/src/lib/api.ts` (or inline in the React Query hook)
4. Use with `useQuery` / `useMutation` from `@tanstack/react-query` in frontend component

### Update Bungie manifest item types

Edit `backend/api-service/services/bungie/types.go` — constants and helpers live there.

### Add a new protected endpoint to api-service

Use `jwtHelper.Middleware()` on the route:

```go
api.GET("/my-endpoint", jwtHelper.Middleware(), func(c *gin.Context) {
    membershipID, _ := c.Get("membership_id")
    // ...
})
```

### Run tests

```powershell
# Go service (from backend/api-service)
go test ./...

# Frontend
cd frontend && npm test
```

#### Full Go coverage locally (matches CI)

A plain `go test ./...` on a fresh Windows checkout reports **~52%** because two
groups of tests self-skip. CI hits **~63%** because it satisfies both gates:

| Gate | Skipped tests | Why it skips locally |
| --- | --- | --- |
| **cgo / SQLite** | `services/manifest`, `services/search` `BuildIndex` | `mattn/go-sqlite3` needs cgo; with no C compiler Go sets `CGO_ENABLED=0` and the driver becomes a stub. A runtime `requireSQLite(t)` probe then calls `t.Skip`. (`-race` also needs cgo.) |
| **Postgres** | `db` package integration tests | Gated on `TEST_DATABASE_URL`; `pgx` is pure Go (no compiler needed), so this gate is independent of the cgo one. |

To close both gaps locally:

1. **Install a C toolchain** (one-time) — e.g. mingw-w64 via `scoop install mingw`,
   `choco install mingw`, or MSYS2 — and confirm `gcc --version` resolves on PATH.
2. **Run the helper script** (from `backend/api-service/`):

   ```powershell
   ./test-local.ps1          # start a throwaway Postgres, run all tests, print total coverage
   ./test-local.ps1 -Html    # also open the per-line HTML report
   ./test-local.ps1 -Fresh   # recreate the Postgres container from scratch
   ./test-local.ps1 -NoRace  # skip the race detector (faster; CI uses -race)
   ./test-local.ps1 -Down    # stop & remove the test Postgres container
   ```

`test-local.ps1` (`backend/api-service/`):

- Starts the `test-postgres` service from the root `docker-compose.yml` (a
  `test`-profiled service, container `gt-test-pg`) on host port **5533** (override
  with `-Port`) so it never collides with the docker-compose Postgres on 5532.
  Because it's a Compose service, Docker Desktop groups it under the same
  `guardiantracker` project as the rest of the stack (a plain `docker compose up`
  never starts it — the `test` profile gates it). The container is idempotent and
  left running between invocations for fast re-runs (`-Down` removes it; a one-time
  check also clears any legacy standalone `gt-test-pg` left by older script versions).
- Exports `CGO_ENABLED=1` and `TEST_DATABASE_URL=postgres://test_user:test_password@localhost:5533/test_db?sslmode=disable`,
  then runs `go test -race -coverprofile=coverage.out ./...` and prints the total.
- The test DB needs **no manual setup**: the container creates `test_db`, and the
  schema is applied automatically by the tests (`db.Migrate` runs the embedded
  migrations in `testPool`); each test creates and cleans its own rows.

PowerShell note: pass `-flag=value` args to `go.exe` quoted — `go tool cover "-func=coverage.out"`
(or the space form `-func coverage.out`). Bare `-func=coverage.out` trips a
"too many arguments" error because PowerShell mangles the token.

To do it by hand instead of the script: start any Postgres, set the two env vars
above, then `go test -race -coverprofile=coverage.out ./...`.

## Known Limitations / TODOs

- The character switcher persists a per-account pick and drives the top-bar avatar and Dashboard hero, but data stays account-wide (Destiny collections are account-scoped); character-scoped surfaces arrive with P2 loadouts
- Redis is in docker-compose but not actively used (JWT revocation and token persistence are Postgres-backed; Redis would be needed for multi-replica distributed caching)
- Search index is built in-memory — lost on restart; rebuilds automatically once manifest is ready (~30s on first start) and after each manifest update
- Xûr location is always "Unknown" — the public Bungie API does not expose vendor location
- Milestone `missing` counts are not computed yet — the field is omitted from the weekly payload and the UI hides the badge rather than implying completion
