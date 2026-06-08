# Guardian Tracker — Claude Code Guide

## Project Overview

Guardian Tracker is a Destiny 2 collection tracker web app. Players log in via Bungie OAuth, and the app analyzes their in-game collections to surface missing items with acquisition difficulty ratings, wish-list management, and weekly recommendations.

## Architecture

Microservices deployed on Kubernetes (Minikube locally, designed for cloud). Four main components communicate via REST and GraphQL:

```text
Frontend (React/TS :3000)
    └─► GraphQL Service (Apollo :4000)
            ├─► Auth Service (Go/Gin :8081)  — OAuth, JWT, token store
            └─► Bungie Service (Go/Gin :8082) — manifest, collections
                    └─► auth-service /internal — Bungie OAuth tokens
```

### Service Ports

| Service | Port | Language |
| --- | --- | --- |
| Frontend | 3000 | React + TypeScript |
| GraphQL Service | 4000 | Node.js + TypeScript |
| Auth Service | 8081 | Go + Gin |
| Bungie Service | 8082 | Go + Gin |

## Running Services

Pick the option that matches what you're doing:

- **Docker Compose** (Option A) — one command to run the whole stack. Best default for local dev, onboarding, and integration testing.
- **Minikube** (Option B) — for validating the Kubernetes manifests / deployment parity. Overkill for everyday work.
- **Individual services** (Option C) — for actively developing a single service with fast hot reload (Air / Vite / nodemon).

### Option A: Docker Compose (full stack)

Runs all four services plus Postgres and Redis from their production Dockerfiles.

```powershell
cp .env.example .env      # fill in BUNGIE_* secrets for real OAuth
docker compose up --build
```

- Frontend `http://localhost:3000`, GraphQL `http://localhost:4000/graphql`, Auth `:8081`, Bungie `:8082`
- Postgres `:5432`, Redis `:6379`
- Backend-to-backend calls use compose DNS (`http://auth-service:8081`, etc.); the frontend's `VITE_` URLs stay on `localhost` because they run in the browser.
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

### Option C: Individual services

```powershell
# Auth Service
cd backend/auth-service
go run .

# Bungie Service
cd backend/bungie-service
go run .

# GraphQL Service
cd backend/graphql-service
npm run dev

# Frontend
cd frontend
npm start
```

### Hot Reload (Air)

Both Go services have `.air.toml` configured. Use `air` instead of `go run .` for hot reload during development.

### Exposing via ngrok (public HTTPS)

To test Bungie OAuth against a public HTTPS URL (or share a running instance), tunnel
the frontend with ngrok. After auth, Bungie redirects back to the ngrok domain, so the
browser page runs with the ngrok origin (e.g. `https://<sub>.ngrok-free.dev`).

Two things must know about that origin:

1. **CORS** — add the ngrok URL to `CORS_ALLOWED_ORIGINS` in the root `.env` (keep
   `http://localhost:3000` too), then rebuild graphql-service so the compiled CORS
   list picks it up. The GraphQL service validates the request `Origin` against this
   list in all environments:

   ```powershell
   # root .env
   CORS_ALLOWED_ORIGINS=http://localhost:3000,https://<sub>.ngrok-free.dev

   docker compose up -d --build graphql-service
   ```

   (auth-service already allows any origin in dev via a `*` fallback, so only
   graphql-service needs this.)

2. **OAuth redirect** — set `AUTH_REDIRECT_URI` to the ngrok callback
   (`https://<sub>.ngrok-free.dev/auth/callback`) and add the same URL to your Bungie
   app's redirect settings at <https://www.bungie.net/en/Application>.

**Caveats:**

- The frontend calls the backend at the `localhost` URLs baked in at build time, so
  this only works when the browser runs on the **same machine** as Docker. To use the
  ngrok URL from another device, tunnel the backend too and rebuild the frontend with
  `VITE_GRAPHQL_URL` / `VITE_AUTH_SERVICE_URL` pointing at public URLs.
- Free ngrok subdomains change on each restart — update `CORS_ALLOWED_ORIGINS`,
  `AUTH_REDIRECT_URI`, and the Bungie app redirect each time, or use a reserved domain.

## Environment Setup

Every service has a `.env.example`. Copy and fill each one before running:

```powershell
# Root
cp .env.example .env

# Auth service
cd backend/auth-service
cp .env.example .env

# Bungie service
cd backend/bungie-service
cp .env.example .env

# GraphQL service
cd backend/graphql-service
cp .env.example .env

# Frontend
cd frontend
cp .env.example .env.local
```

### Required secrets (all services share these)

- `BUNGIE_API_KEY` — from <https://www.bungie.net/en/Application>
- `BUNGIE_CLIENT_ID` — from Bungie app settings
- `BUNGIE_CLIENT_SECRET` — from Bungie app settings
- `JWT_SECRET` — 32+ char random string (`openssl rand -base64 32`)
- `INTERNAL_API_KEY` — shared between auth-service and bungie-service for service-to-service auth

## Key Files

### Auth Service (`backend/auth-service/`)

| File | Purpose |
| --- | --- |
| `main.go` | Gin router, OAuth flow, CSRF state machine, wishlist stubs |
| `jwt.go` | JWT generation and validation (access 24h, refresh 30d) |
| `middleware.go` | `AuthMiddleware` and `OptionalAuthMiddleware` Gin handlers |
| `tokenstore.go` | In-memory Bungie OAuth token store with auto-refresh |

**Endpoints:**

- `GET /api/auth/bungie` — initiate OAuth, returns auth URL + CSRF state
- `POST /api/auth/bungie/callback` — exchange code for JWT tokens
- `POST /api/auth/refresh` — rotate access + refresh tokens
- `GET /api/auth/validate` — validate JWT (protected)
- `GET /api/auth/profile` — current user profile (protected)
- `GET/POST/DELETE /api/wishlist` — wish list CRUD (stubs, JWT protected)
- `GET /internal/bungie-token/:membershipId` — internal: fetch Bungie access token
- `GET /health` and `GET /ready` — health/readiness probes

### Bungie Service (`backend/bungie-service/`)

| File | Purpose |
| --- | --- |
| `main.go` | Gin router, dependency wiring, manifest startup |
| `config/config.go` | Typed config with env var parsing helpers |
| `services/bungie/client.go` | HTTP client with rate limiting + retry |
| `services/bungie/manifest.go` | Manifest download, version tracking, SQLite extraction |
| `services/bungie/types.go` | All Bungie API types, constants, helpers |
| `services/collections/service.go` | Collection analysis + difficulty classification |
| `services/manifest/repository.go` | SQLite read-only queries against manifest DB |
| `services/auth/client.go` | Client that talks to auth-service internal API + validates JWTs |
| `cache/cache.go` | In-memory cache (and no-op cache interface) |
| `api/handlers/collections.go` | HTTP handler for collections, validates JWT + membership |
| `api/handlers/health.go` | Health, ready, and manifest status endpoints |

**Endpoints:**

- `GET /api/collections/:membershipType/:membershipId` — user collections (JWT protected)
- `POST /api/collections/:membershipType/:membershipId/refresh` — invalidate cache
- `GET /api/manifest/status` — manifest version and readiness
- `GET /api/weekly/recommendations` — placeholder
- `GET /api/items/search` — placeholder
- `GET /health` and `GET /ready`

**Bungie manifest flow:**

1. On startup, service fetches manifest metadata from `https://www.bungie.net/Platform/Destiny2/Manifest/`
2. Downloads the English `.content` (SQLite) ZIP file from Bungie CDN
3. Extracts and stores at `./data/manifest.sqlite`
4. Background goroutine checks for updates every hour (configurable)

### GraphQL Service (`backend/graphql-service/src/`)

| File | Purpose |
| --- | --- |
| `server.ts` | Express + Apollo Server 4, rate limiting, security middleware |
| `schema.ts` | Full GraphQL schema — all types, queries, mutations |
| `resolvers.ts` | Resolver implementations proxying to Go services |
| `context.ts` | Request context — JWT extraction + user hydration |
| `services/AuthService.ts` | Auth service HTTP client |
| `services/BungieService.ts` | Bungie service HTTP client |
| `utils/auth.ts` | `requireAuth` helper |
| `utils/validation.ts` | Zod schemas for input validation |

**GraphQL operations:**

- Queries: `currentUser`, `userCollections`, `searchItems`, `weeklyRecommendations`, `wishList`
- Mutations: `login`, `addToWishList`, `removeFromWishList`, `updateWishListItem`, `refreshUserData`, `logout`

### Frontend (`frontend/src/`)

The UI was fully redesigned (the "Guardian Tracker" design system): a custom dark
theme built on oklch design tokens and `gt-*` CSS classes (not Tailwind utilities for
new work). Layout is a persistent sidebar + top bar shell. See `frontend/design/` for
the source design and `frontend/README.md` for the full component map.

| Path | Purpose |
| --- | --- |
| `App.tsx` | Router, lazy-loaded pages, `ProtectedLayout` (AppShell + auth gate) |
| `index.tsx` | App root; imports `styles/{tokens,kit,app}.css` |
| `contexts/AuthContext.tsx` | Auth state, localStorage persistence, token refresh |
| `contexts/PreferencesContext.tsx` | User prefs (card style, "for you" badges); localStorage `guardian_prefs` |
| `lib/apollo.ts` | Apollo Client with auth link |
| `lib/mockData.ts` | Mock data for backend-less screens + fallbacks (typed port of `design/src/data.js`) |
| `lib/adapters.ts` | GraphQL `DestinyItem`/`WishListItem` → design `GTItem`/`WishlistEntry` |
| `styles/{tokens,kit,app}.css` | Design tokens + component/shell styles (plain CSS) |
| `components/AppShell.tsx` | Sidebar + top bar + mobile nav; global search; character switcher |
| `components/Brand.tsx` | Logo mark |
| `components/kit/` | Design component kit: `Icon`, primitives, `ItemCard`, composites (`Panel`, `CategoryTree`, `ItemDetailDrawer`, `SealCard`, …) |
| `components/ui/` | Legacy primitives still used by shell: `LoadingSpinner`, `Toast`, plus `ErrorBoundary` (Tailwind) |
| `pages/Login.tsx` | Bungie OAuth initiation (redesigned) |
| `pages/OAuthCallback.tsx` | Handles `/auth/callback` — exchanges code, stores tokens |
| `pages/Dashboard.tsx` | Completion hero + "do this today"; real collection totals, mock weekly |
| `pages/Collections.tsx` | Category tree + filterable item grid/list + detail drawer; real data, mock fallback |
| `pages/WishList.tsx` | Wishlist management; real GraphQL with mock fallback |
| `pages/ThisWeek.tsx` | Weekly recommendations / Xûr / milestones (mock — no backend yet) |
| `pages/Catalysts.tsx` | Catalysts & crafting patterns (mock — no backend yet) |
| `pages/Triumphs.tsx` | Triumphs & seals (mock — no backend yet) |
| `pages/Settings.tsx` | Account info + appearance preferences + sign out |
| `graphql/queries.ts` | Apollo queries |
| `graphql/mutations.ts` | Apollo mutations |
| `types/design.ts` | Design-system domain types (`GTItem`, `Seal`, `Weekly`, …) |

### Database (`database/init/`)

- `01-init.sql` — PostgreSQL schema (used in CI, not yet wired to running services)

### Kubernetes (`k8s/`)

- Individual service YAML manifests
- `startup.ps1` / `shutdown.ps1` — Minikube lifecycle scripts
- `auth-service-configmap.yaml` — configmap with OAuth redirect URI (update for ngrok if needed)

## Development Notes

### Manifest Database

The Bungie manifest is a ~100MB SQLite file. On first run the bungie-service downloads it (~10–30s). The version is tracked in `./data/manifest_version.txt`. The service gracefully starts without the manifest (collections endpoint returns 503 until ready).

### Token Flow

```text
1. Frontend → GET /api/auth/bungie (auth-service) → returns authUrl + state
2. User → Bungie.net → redirects to /auth/callback?code=...&state=...
3. Frontend → POST /api/auth/bungie/callback → gets JWT access + refresh tokens
4. Frontend stores tokens in localStorage (guardian_token, guardian_refresh_token)
5. Apollo Client injects Authorization: Bearer <token> on every GraphQL request
6. GraphQL service forwards Authorization header to downstream services
7. Bungie service: validates JWT locally, then calls auth-service /internal/bungie-token/:id
8. auth-service: returns stored Bungie OAuth token (auto-refreshed if expired)
```

### Authentication Security

- CSRF: state parameter stored server-side with 10-min TTL, consumed on use
- JWT: HS256, access=24h, refresh=30d, token-type claim prevents refresh tokens being used as access tokens
- Internal API: `X-Internal-API-Key` header on service-to-service calls
- Rate limiting: 100 req/15min/IP in production (GraphQL), 1000 in dev

### Wishlist (placeholder)

The wishlist endpoints in auth-service return hardcoded mock data. The GraphQL mutations route to auth-service but there is no database persistence layer yet.

### Weekly Recommendations (placeholder)

The bungie-service `/api/weekly/recommendations` returns empty arrays. Full implementation is planned.

## CI/CD

GitHub Actions (`.github/workflows/ci-cd.yml`):

1. **test-frontend** — type-check, lint, test, build
2. **test-graphql** — type-check, lint, test, build
3. **test-go-services** — go vet, go test with race detector
4. **build-docker-images** — builds all 4 Docker images; pushes on `main` branch
5. **deploy-staging** — triggered on `develop` branch (stub — add commands)
6. **deploy-production** — triggered on `main` branch (stub — add commands)

Docker images pushed to Docker Hub as `guardiantracker/*`.

## Common Tasks

### Add a new GraphQL query

1. Add type + query to `backend/graphql-service/src/schema.ts`
2. Add resolver in `backend/graphql-service/src/resolvers.ts`
3. Add Zod validation schema in `utils/validation.ts` if needed
4. Add query in `frontend/src/graphql/queries.ts`
5. Use with `useQuery` in frontend component

### Update Bungie manifest item types

Edit `backend/bungie-service/services/bungie/types.go` — constants and helpers live there.

### Add a new protected endpoint to auth-service

Wrap route handler with `AuthMiddleware()`:

```go
api.GET("/my-endpoint", AuthMiddleware(), func(c *gin.Context) {
    membershipID, _ := c.Get("membership_id")
    // ...
})
```

### Run tests

```powershell
# Go services (from each service dir)
go test ./...

# GraphQL service
cd backend/graphql-service && npm test

# Frontend
cd frontend && npm test
```

## Known Limitations / TODOs

- Wishlist has no database persistence (hardcoded mock data)
- Weekly recommendations endpoint returns empty data
- `refreshUserData` mutation is a stub
- `logout` mutation does not blacklist the JWT
- `updateWishListItem` mutation is a stub
- PostgreSQL schema exists but is not wired to any running service
- Redis is configured in graphql-service but not actively used
- The debug `DataSourceBanner` was removed in the redesign; Collections now uses the `DataFreshnessChip` instead
- The This Week, Catalysts & Crafting, and Triumphs & Seals pages render mock data from `lib/mockData.ts` — their backends don't exist yet
- The character switcher and global search in the app shell operate on mock data (no character/search backend yet)
