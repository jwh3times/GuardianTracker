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
    └─► GraphQL Service (Apollo :4000)
            ├─► Auth Service (Go :8081)    — OAuth, JWT, Bungie token store
            └─► Bungie Service (Go :8082)  — manifest, collection analysis
```

### Frontend

- **Framework**: React 18 + TypeScript
- **GraphQL Client**: Apollo Client
- **UI Toolkit**: Tailwind CSS + Radix UI (shadcn/ui)
- **Routing**: React Router v6

### Backend

- **GraphQL Layer**: Apollo Server 4 (Node.js/TypeScript)
- **Auth Service**: Go + Gin — Bungie OAuth, JWT access/refresh tokens, in-memory Bungie token store
- **Bungie Service**: Go + Gin — manifest download/caching, collection analysis, rate-limited Bungie API client

### Data Storage

- **PostgreSQL**: User data schema (defined, not yet wired to services)
- **SQLite**: Bungie Destiny 2 manifest database (downloaded automatically on startup)
- **In-memory cache**: Collection results with configurable TTL

### Infrastructure

- **Local**: Kubernetes via Minikube (startup/shutdown scripts included)
- **CI/CD**: GitHub Actions — lint, test, Docker build, and push on `main`

## Project Structure

```text
guardian-tracker/
├── frontend/                        # React + TypeScript SPA
├── backend/
│   ├── graphql-service/             # Apollo Server (Node.js/TypeScript)
│   ├── auth-service/                # Go — Bungie OAuth + JWT service
│   └── bungie-service/              # Go — Bungie API + manifest service
├── database/
│   └── init/01-init.sql             # PostgreSQL schema
├── k8s/                             # Kubernetes manifests + startup scripts
└── .github/workflows/ci-cd.yml      # CI/CD pipeline
```

## Getting Started

### Prerequisites

- Node.js 18+
- Go 1.21+
- Docker Desktop (for Kubernetes deployment)
- Minikube (for local Kubernetes)

### 1. Get Bungie API Credentials

Create an application at <https://www.bungie.net/en/Application> to obtain:

- API Key
- OAuth Client ID
- OAuth Client Secret

Set the OAuth redirect URI to `https://<your-ngrok-domain>/auth/callback` (Bungie requires HTTPS).

### 2. Configure Environment Variables

Each service has a `.env.example`. Copy and fill each:

```bash
cp .env.example .env
cp backend/auth-service/.env.example backend/auth-service/.env
cp backend/bungie-service/.env.example backend/bungie-service/.env
cp backend/graphql-service/.env.example backend/graphql-service/.env
cp frontend/.env.example frontend/.env.local
```

Required secrets across all services:

| Variable | Description |
| --- | --- |
| `BUNGIE_API_KEY` | Your Bungie API key |
| `BUNGIE_CLIENT_ID` | OAuth client ID |
| `BUNGIE_CLIENT_SECRET` | OAuth client secret |
| `JWT_SECRET` | 32+ char random string (`openssl rand -base64 32`) |
| `INTERNAL_API_KEY` | Shared key for service-to-service communication |

### 3. Install Dependencies

```bash
# Frontend
cd frontend && npm install && cd ..

# GraphQL Service
cd backend/graphql-service && npm install && cd ../..

# Go services
cd backend/auth-service && go mod download && cd ../..
cd backend/bungie-service && go mod download && cd ../..
```

### 4. Start Services

#### Option A: Kubernetes (Minikube)

```powershell
cd k8s
./startup.ps1
```

See [k8s/README.md](./k8s/README.md) for details.

#### Option B: Individual services

```bash
# Terminal 1 — Auth Service
cd backend/auth-service && go run .

# Terminal 2 — Bungie Service
cd backend/bungie-service && go run .

# Terminal 3 — GraphQL Service
cd backend/graphql-service && npm run dev

# Terminal 4 — Frontend
cd frontend && npm start
```

### 5. Service URLs

| Service | URL |
| --- | --- |
| Frontend | <http://localhost:3000> |
| GraphQL Playground | <http://localhost:4000/graphql> |
| Auth Service | <http://localhost:8081> |
| Bungie Service | <http://localhost:8082> |

> **Note:** On first run, the Bungie Service will download the Destiny 2 manifest database (~100MB). The collections endpoint returns 503 until the download completes.

## Development

For detailed development guidance, code structure, token flow, and common tasks, see [CLAUDE.md](./CLAUDE.md).

### Running Tests

```bash
# Go services (from each service directory)
go test ./...

# GraphQL service
cd backend/graphql-service && npm test

# Frontend
cd frontend && npm test
```

### Code Quality

```bash
# GraphQL + Frontend
npm run type-check
npm run lint

# Go services
go vet ./...
```

## CI/CD

GitHub Actions runs on every push to `main` and `develop`:

1. Type-check, lint, and test all services
2. Build Docker images
3. Push images to Docker Hub on `main` (requires `DOCKER_USERNAME` and `DOCKER_PASSWORD` secrets)

## Security

See [SECURITY.md](./SECURITY.md) for credential management, environment setup, and the production security checklist.

## Documentation

- [CLAUDE.md](./CLAUDE.md) — developer guide: architecture, token flow, common tasks
- [Frontend Guide](./frontend/README.md) — frontend-specific setup and component docs
- [Kubernetes Deployment](./k8s/README.md) — Minikube setup and troubleshooting
- [SECURITY.md](./SECURITY.md) — credential management and security guide

## License

MIT License — see [LICENSE](./LICENSE) for details.
