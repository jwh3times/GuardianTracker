# Guardian Tracker — Claude Code Guide

## Project Overview

Guardian Tracker is a Destiny 2 collection tracker web app. Players log in via Bungie OAuth, and the app analyzes their in-game collections to surface missing items with acquisition difficulty ratings, wish-list management, and weekly recommendations.

## Architecture

Two services: a Go API backend and a React frontend that calls it directly over REST.

```text
Frontend (React/TS :3000)
    └─► API Service (Go/Gin :8081)  — OAuth, JWT, manifest, collections
```

### Service Ports

| Service | Port | Language |
| --- | --- | --- |
| Frontend | 3000 | React + TypeScript |
| API Service | 8081 | Go + Gin |

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

- Frontend `http://localhost:3000`, API `http://localhost:8081`
- Postgres `:5432`, Redis `:6379`
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
   CORS_ALLOWED_ORIGINS=http://localhost:3000,https://<sub>.ngrok-free.dev

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

## Key Files

### API Service (`backend/api-service/`)

| File | Purpose |
| --- | --- |
| `main.go` | Gin router, dependency wiring, manifest startup |
| `config/config.go` | Typed config with env var parsing helpers |
| `auth/jwt.go` | JWT generation and validation (access 24h, refresh 30d) |
| `auth/middleware.go` | JWT middleware for protected routes |
| `auth/tokenstore.go` | In-memory Bungie OAuth token store with auto-refresh |
| `api/handlers/auth.go` | OAuth flow, token refresh, profile endpoints |
| `api/handlers/characters.go` | HTTP handler for characters |
| `api/handlers/collections.go` | HTTP handler for collections |
| `api/handlers/wishlist.go` | Wishlist CRUD stubs |
| `api/handlers/health.go` | Health, ready, manifest status endpoints |
| `services/bungie/client.go` | HTTP client with rate limiting + retry |
| `services/bungie/manifest.go` | Manifest download, version tracking, SQLite extraction |
| `services/bungie/types.go` | All Bungie API types, constants, helpers |
| `services/collections/service.go` | Collection analysis + difficulty classification |
| `services/characters/service.go` | Character fetching |
| `services/manifest/repository.go` | SQLite read-only queries against manifest DB |
| `cache/cache.go` | In-memory cache (and no-op cache interface) |

**Endpoints:**

- `GET /api/auth/bungie` — initiate OAuth, returns auth URL + CSRF state
- `POST /api/auth/bungie/callback` — exchange code for JWT tokens
- `POST /api/auth/refresh` — rotate access + refresh tokens
- `GET /api/auth/validate` — validate JWT (protected)
- `GET /api/auth/profile` — current user profile (protected)
- `GET/POST/DELETE /api/wishlist` — wish list CRUD (stubs, JWT protected)
- `GET /api/characters/:membershipType/:membershipId` — user characters (JWT protected)
- `GET /api/collections/:membershipType/:membershipId` — user collections (JWT protected)
- `POST /api/collections/:membershipType/:membershipId/refresh` — invalidate cache
- `GET /api/manifest/status` — manifest version and readiness
- `GET /api/weekly/recommendations` — placeholder
- `GET /api/items/search` — placeholder
- `GET /health` and `GET /ready` — health/readiness probes

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
| `index.tsx` | App root; imports `styles/{tokens,kit,app}.css` |
| `contexts/AuthContext.tsx` | Auth state, localStorage persistence, token refresh |
| `contexts/PreferencesContext.tsx` | User prefs (card style, "for you" badges); localStorage `guardian_prefs` |
| `lib/api.ts` | `apiFetch` helper + `QueryClient` — all REST calls go through here |
| `lib/mockData.ts` | Mock data for backend-less screens + fallbacks |
| `lib/adapters.ts` | API response types → design `GTItem`/`WishlistEntry` |
| `styles/{tokens,kit,app}.css` | Design tokens + component/shell styles (plain CSS) |
| `components/AppShell.tsx` | Sidebar + top bar + mobile nav; global search; character switcher |
| `components/Brand.tsx` | Logo mark |
| `components/kit/` | Design component kit: `Icon`, primitives, `ItemCard`, composites |
| `components/ui/` | Legacy primitives: `LoadingSpinner`, `Toast`, `ErrorBoundary` |
| `pages/Login.tsx` | Bungie OAuth initiation |
| `pages/OAuthCallback.tsx` | Handles `/auth/callback` — exchanges code, stores tokens |
| `pages/Dashboard.tsx` | Completion hero + "do this today"; real collection totals, mock weekly |
| `pages/Collections.tsx` | Category tree + filterable item grid/list + detail drawer |
| `pages/WishList.tsx` | Wishlist management; real API with mock fallback |
| `pages/ThisWeek.tsx` | Weekly recommendations / Xûr / milestones (mock — no backend yet) |
| `pages/Catalysts.tsx` | Catalysts & crafting patterns (mock — no backend yet) |
| `pages/Triumphs.tsx` | Triumphs & seals (mock — no backend yet) |
| `pages/Settings.tsx` | Account info + appearance preferences + sign out |
| `types/api.ts` | API response types (`APIUser`, `AuthTokenResponse`, etc.) |
| `types/design.ts` | Design-system domain types (`GTItem`, `Seal`, `Weekly`, …) |

### Database (`database/init/`)

- `01-init.sql` — PostgreSQL schema (defined, not yet wired to running service)

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
3. Frontend → POST /api/auth/bungie/callback → gets JWT access + refresh tokens
4. Frontend stores tokens in localStorage (guardian_token, guardian_refresh_token)
5. lib/api.ts apiFetch injects Authorization: Bearer <token> on every request
6. React Query hooks call apiFetch for all data fetching
7. API service validates JWT on protected routes
8. API service uses stored Bungie OAuth token (auto-refreshed if expired) for Bungie API calls
```

### Authentication Security

- CSRF: state parameter stored server-side with 10-min TTL, consumed on use
- JWT: HS256, access=24h, refresh=30d, token-type claim prevents refresh tokens being used as access tokens
- Rate limiting: Bungie API client — 10 req/s, burst 20

### Wishlist (placeholder)

The wishlist endpoints return hardcoded mock data. No database persistence yet.

### Weekly Recommendations (placeholder)

The `/api/weekly/recommendations` endpoint returns empty arrays. Full implementation is planned.

## CI/CD

GitHub Actions (`.github/workflows/ci-cd.yml`):

1. **test-frontend** — type-check, lint, test, build
2. **test-go-services** — go vet, govulncheck, go test with race detector
3. **build-docker-images** — builds both Docker images; build validation only (no push configured yet)

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

## Known Limitations / TODOs

- Wishlist has no database persistence (hardcoded mock data)
- Weekly recommendations endpoint returns empty data
- `logout` does not blacklist the JWT (token stays valid up to 24h server-side)
- PostgreSQL schema exists but is not wired to any running service
- Redis is in docker-compose but not actively used by the API service
- The This Week, Catalysts & Crafting, and Triumphs & Seals pages render mock data — their backends don't exist yet
- The character switcher and global search operate on mock data (no search backend yet)
- Dashboard cosmetics category is hardcoded (no cosmetics analysis in collections service yet)
