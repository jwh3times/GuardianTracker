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

### Option A: Kubernetes (Minikube)

```powershell
cd k8s
./startup.ps1
```

### Option B: Individual services

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

| Path | Purpose |
| --- | --- |
| `App.tsx` | Router, lazy-loaded pages, `ProtectedRoute` |
| `contexts/AuthContext.tsx` | Auth state, localStorage persistence, token refresh |
| `lib/apollo.ts` | Apollo Client with auth link |
| `pages/Login.tsx` | Bungie OAuth initiation |
| `pages/OAuthCallback.tsx` | Handles `/auth/callback` — exchanges code, stores tokens |
| `pages/Dashboard.tsx` | Overview page |
| `pages/Collections.tsx` | Missing items by category with `DataSourceBanner` |
| `pages/WishList.tsx` | Wish list management |
| `graphql/queries.ts` | Apollo queries |
| `graphql/mutations.ts` | Apollo mutations |
| `components/ui/` | Button, Card, LoadingSpinner, Toast |

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
- The `DataSourceBanner` in Collections.tsx is a debug component and should be removed or gated before production
