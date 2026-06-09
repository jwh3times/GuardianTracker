# Guardian Tracker

A web-based app for Destiny 2 players that integrates Bungie APIs to analyze collections, identify missing items with acquisition difficulty ratings, and manage wish lists.

## Core Features

- **Bungie OAuth Login**: Secure login via Bungie.net OAuth with CSRF protection
- **Collection Analysis**: Automatically identify missing weapons, armor, and exotics — classified by acquisition difficulty (Easy / Moderate / Challenging)
- **Wish List Management**: Track and prioritize desired items
- **Destiny 2 Manifest**: Auto-downloads and updates the full Bungie item manifest (SQLite)
- **Weekly Reset Notifications**: Placeholder — planned for future releases

## Architecture

```text
Frontend (React/TS :3000)
    └─► API Service (Go/Gin :8081)  — OAuth, JWT, manifest, collections
```

### Frontend

- **Framework**: React 19 + TypeScript (Vite)
- **Data fetching**: TanStack React Query + `apiFetch` (REST)
- **UI**: Custom "Guardian Tracker" design system — oklch design tokens + `gt-*` CSS classes (persistent sidebar shell, rarity-driven theming)
- **Routing**: React Router v7

### Backend

- **API Service**: Go + Gin — Bungie OAuth, JWT access/refresh tokens, in-memory token store, manifest download, collection analysis, rate-limited Bungie API client

### Data Storage

- **PostgreSQL**: User data schema (defined, not yet wired to service)
- **SQLite**: Bungie Destiny 2 manifest database (downloaded automatically on startup)
- **In-memory cache**: Collection results with configurable TTL

### Infrastructure

- **Local**: Docker Compose (recommended) or Minikube
- **CI/CD**: GitHub Actions — lint, test, Docker build on every push

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

Set the OAuth redirect URI to `http://localhost:3000/auth/callback` (or your ngrok HTTPS URL for Bungie's HTTPS requirement).

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

### 3. Start Services

#### Option A: Docker Compose (recommended)

```powershell
docker compose up --build
```

Frontend: <http://localhost:3000> — API: <http://localhost:8081>

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
