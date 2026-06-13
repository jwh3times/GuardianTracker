# Guardian Tracker

A web-based app for Destiny 2 players that integrates Bungie APIs to analyze collections, identify missing items with acquisition difficulty ratings, and manage wish lists.

## Core Features

- **Bungie OAuth Login**: Secure login via Bungie.net OAuth with CSRF protection
- **Collection Analysis**: Automatically identify missing weapons, armor, and exotics — classified by acquisition difficulty (Easy / Moderate / Challenging)
- **Wish List Management**: Track and prioritize desired items
- **Destiny 2 Manifest**: Auto-downloads and updates the full Bungie item manifest (SQLite)
- **This Week**: Real weekly milestones, Xûr inventory with missing/wishlist flags, daily actions, and reset countdowns
- **Catalysts, Crafting & Seals**: Exotic catalyst progress, crafting pattern unlocks, and triumph/seal completion from the Bungie records API

## Architecture

```text
Frontend (React/TS :5273)
    └─► API Service (Go/Gin :8081)  — OAuth, JWT, manifest, collections
```

### Frontend

- **Framework**: React 19 + TypeScript (Vite)
- **Data fetching**: TanStack React Query + `apiFetch` (REST)
- **UI**: Custom "Guardian Tracker" design system — oklch design tokens + `gt-*` CSS classes (persistent sidebar shell, rarity-driven theming)
- **Routing**: React Router v7

### Backend

- **API Service**: Go + Gin — Bungie OAuth, JWT access/refresh tokens with revocation, DB-backed encrypted Bungie token store, manifest download, collection analysis, rate-limited Bungie API client

### Data Storage

- **PostgreSQL**: Users, wishlist, preferences, and AES-256-GCM-encrypted Bungie OAuth tokens (schema applied by the migration runner at startup; runs in degraded in-memory mode when `DATABASE_URL` is unset)
- **SQLite**: Bungie Destiny 2 manifest database (downloaded automatically on startup)
- **In-memory cache**: Collection results with configurable TTL

### Infrastructure

- **Local**: Docker Compose (recommended) or Minikube
- **CI/CD**: GitHub Actions — lint, test, Docker build on every push

### Ports

Every port across the config files, by environment. This is the single source of
truth for the project's ports (CLAUDE.md links here).

**Core services**

Host ports are GuardianTracker's **fleet lane** so its stack runs alongside other
local projects (ApexRacers owns the canonical `5432`/`5050`/`8080`; LeaseBook uses
the `5632`/`8082`/`5373` lane). Only the host side moves — container ports never
change. Each is overridable via the env var shown in the Compose mappings below.

| Service | Internal (container/process) | Host / exposed | Defined in |
| --- | --- | --- | --- |
| Frontend (dev — Vite) | `5273` | `5273` | `frontend/vite.config.ts`, `frontend/Dockerfile.dev` |
| Frontend (prod — nginx) | `8080` | `5273` (mapped) | `frontend/nginx.conf`, `frontend/Dockerfile`, `docker-compose.yml` |
| API Service (Go/Gin) | `8081` | `8081` | `backend/api-service/config/config.go`, `backend/api-service/Dockerfile`, `docker-compose.yml` |
| Postgres | `5432` | `5532` | `docker-compose.yml`, `.env.example` |
| pgAdmin | `80` | `5150` | `docker-compose.yml`, `.env.example` |
| Redis | `6379` | `6379` | `docker-compose.yml`, `.env.example` |

**Docker Compose mappings** (`HOST:CONTAINER`)

```text
postgres        ${POSTGRES_PORT:-5532}      -> 5432
pgadmin         ${PGADMIN_PORT:-5150}       -> 80
redis           ${REDIS_PORT:-6379}         -> 6379
api-service     ${API_SERVICE_PORT:-8081}   -> 8081   (PORT env = 8081)
frontend        ${FRONTEND_PORT:-5273}      -> 8080   # host 5273 hits nginx:8080
test-postgres   ${TEST_POSTGRES_PORT:-5533} -> 5432   # "test" profile only; not started by a plain `up`
```

**Kubernetes (Minikube — `k8s/`)**

| Object | Port | Notes |
| --- | --- | --- |
| api-service Deployment | `containerPort 8081` | liveness `/health`, readiness `/ready` on 8081 |
| api-service Service | `8081 → 8081` | `ClusterIP` |
| frontend Deployment | `containerPort 8080` | `NGINX_PORT=8080` env |
| frontend Service | `80 → 8080` | `NodePort` |
| `startup.ps1` port-forward | `localhost:5273 → frontend:80` | dev access (matches the docker frontend port) |

**Dev / cross-service wiring**

- Vite proxy → `http://localhost:8081` (`vite.config.ts`)
- `VITE_API_URL` → `http://localhost:8081` (`frontend/.env.example`)
- OAuth redirect / CORS → `http://localhost:5273/auth/callback` — this host port doubles as the Bungie OAuth redirect origin, so the same URI must be registered in your [Bungie application settings](https://www.bungie.net/en/Application)
- nginx CSP `connect-src` allows `http://localhost:8081` (+ Bungie, ngrok)
- Test Postgres (`backend/api-service/test-local.ps1`) → host `5533 → 5432`; a `test`-profiled Compose service grouped under the `guardiantracker` project in Docker Desktop

## Project Structure

```text
guardian-tracker/
├── frontend/                    # React + TypeScript SPA
├── backend/
│   └── api-service/             # Go — OAuth, JWT, Bungie API, manifest
├── database/
│   └── init/01-init.sql         # PostgreSQL schema
├── k8s/                         # Kubernetes manifests + startup scripts
└── .github/workflows/ci-cd.yml  # CI/CD pipeline
```

## Getting Started

### Prerequisites

- Docker Desktop (for Docker Compose or Minikube deployment)
- Go 1.21+ and Node.js 20+ (for running services individually)
- Minikube (for Kubernetes deployment only)

### 1. Get Bungie API Credentials

Create an application at <https://www.bungie.net/en/Application> to obtain:

- API Key
- OAuth Client ID
- OAuth Client Secret

Set the OAuth redirect URI to `http://localhost:5273/auth/callback` (or your ngrok HTTPS URL for Bungie's HTTPS requirement).

### 2. Configure Environment Variables

Run the setup script to copy all `.env.example` files:

```powershell
./setup.ps1
```

Or copy manually:

```powershell
cp .env.example .env
cp backend/api-service/.env.example backend/api-service/.env
cp frontend/.env.example frontend/.env.local
```

Required secrets:

| Variable | Description |
| --- | --- |
| `BUNGIE_API_KEY` | Your Bungie API key |
| `BUNGIE_CLIENT_ID` | OAuth client ID |
| `BUNGIE_CLIENT_SECRET` | OAuth client secret |
| `JWT_SECRET` | 32+ char random string (`openssl rand -base64 32`) |
| `DATABASE_URL` | Postgres connection string (compose sets it automatically) |
| `TOKEN_ENCRYPTION_KEY` | 32-byte base64 key for Bungie token encryption (`openssl rand -base64 32`) |

### 3. Start Services

#### Option A: Docker Compose (recommended)

```powershell
docker compose up --build
```

Frontend: <http://localhost:5273> — API: <http://localhost:8081>

#### Option B: Kubernetes (Minikube)

```powershell
cd k8s
./startup.ps1
```

See [k8s/README.md](./k8s/README.md) for details.

#### Option C: Individual services

```powershell
# Terminal 1 — API Service
cd backend/api-service && go run .

# Terminal 2 — Frontend
cd frontend && npm start
```

> **Note:** On first run, the API Service downloads the Destiny 2 manifest database (~100MB). The collections endpoint returns 503 until the download completes.

## Development

For detailed development guidance, code structure, token flow, and common tasks, see [CLAUDE.md](./CLAUDE.md).

### Running Tests

```powershell
# Go service
cd backend/api-service && go test ./...

# Frontend
cd frontend && npm test
```

### Code Quality

```powershell
# Frontend
npm run type-check
npm run lint

# Go service
go vet ./...
```

## CI/CD

GitHub Actions runs on every push to `main` and `develop`:

1. Type-check, lint, and test all services
2. Build Docker images (build validation; push not yet configured)

## Security

See [SECURITY.md](./SECURITY.md) for credential management, environment setup, and the production security checklist.

## Documentation

- [CLAUDE.md](./CLAUDE.md) — developer guide: architecture, token flow, common tasks
- [Frontend Guide](./frontend/README.md) — frontend-specific setup and component docs
- [Kubernetes Deployment](./k8s/README.md) — Minikube setup and troubleshooting
- [SECURITY.md](./SECURITY.md) — credential management and security guide

## License

MIT License — see [LICENSE](./LICENSE) for details.
