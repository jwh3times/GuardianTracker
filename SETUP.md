# Setup Guide

Guardian Tracker is a local-first Destiny 2 companion app. It is designed for
development with Docker Compose, with Minikube kept as a deployment-manifest
validation path rather than production parity.

Production-specific infrastructure runbooks, cloud resource names, deployment
secrets, and incident notes belong under `private/`, which is gitignored.

## What You'll Need

- Docker Desktop for the recommended full-stack local environment.
- Go 1.25+ for backend development.
- Node.js 26+ for frontend development.
- A Bungie application for API and OAuth credentials.
- Minikube only if you need to validate the Kubernetes manifests.

## 1. Get Bungie Credentials

Create an application at <https://www.bungie.net/en/Application> and record:

- API key
- OAuth client ID
- OAuth client secret

For local development, set the redirect URI to:

```text
http://localhost:5273/auth/callback
```

If you use a public HTTPS tunnel for OAuth testing, add that tunnel callback URL
to the Bungie application too, then update the local environment values to match.

## 2. Create Environment Files

Run the helper from the repository root:

```powershell
./setup.ps1
```

Or copy the templates manually:

```powershell
cp .env.example .env
cp backend/api-service/.env.example backend/api-service/.env
cp frontend/.env.example frontend/.env.local
```

Fill in the required secrets:

| Variable | Purpose |
| --- | --- |
| `BUNGIE_API_KEY` | Bungie API key |
| `BUNGIE_CLIENT_ID` | Bungie OAuth client ID |
| `BUNGIE_CLIENT_SECRET` | Bungie OAuth client secret |
| `JWT_SECRET` | 32+ character signing secret |
| `DATABASE_URL` | Postgres connection string; Compose sets this for the container |
| `TOKEN_ENCRYPTION_KEY` | 32-byte base64 AES-256-GCM key for stored Bungie tokens |

Optional values:

| Variable | Purpose |
| --- | --- |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS` | Previous encryption key during key rotation |
| `ADMIN_MEMBERSHIP_IDS` | Comma-separated Bungie membership IDs pinned to admin at login |
| `CORS_ALLOWED_ORIGINS` | Explicit browser origins allowed to call the API |
| `JWT_ACCESS_TTL` | Access-token lifetime as a Go duration (default `30m`) |

Do not commit `.env`, generated secrets, manifest databases, cloud credentials, or
production runbooks.

## 3. Run the Full Stack

Docker Compose is the default path for local development:

```powershell
docker compose up --build
```

Open:

- Frontend: <http://localhost:5273>
- API: <http://localhost:8081>
- pgAdmin: <http://localhost:5150>

On first startup, the API downloads the Destiny 2 manifest SQLite database. A
matching search-index snapshot is restored when available; collections and other
manifest-dependent surfaces can still return warming responses during a new
manifest download or rebuild.

Stop the stack without deleting data:

```powershell
docker compose down
```

Stop and remove local volumes:

```powershell
docker compose down -v
```

## 4. Run Individual Services

Use this path when you are actively editing one service and want faster feedback.

```powershell
# Terminal 1 - API service
cd backend/api-service
go run .

# Terminal 2 - frontend
cd frontend
npm start
```

The Vite dev server runs on `:5273` and proxies API calls to `:8081`.

## 5. Validate Kubernetes Manifests

The `k8s/` path is for local Minikube validation. It intentionally runs in
development mode without production Postgres parity.

```powershell
cd k8s
./startup.ps1
```

See [k8s/README.md](./k8s/README.md) for script details and troubleshooting.

## Ports

Host ports are offset where useful so the stack can run beside other local
projects. Container ports stay fixed.

| Service | Internal | Host / exposed | Defined in |
| --- | --- | --- | --- |
| Frontend dev (Vite) | `5273` | `5273` | `frontend/vite.config.ts`, `frontend/Dockerfile.dev` |
| Frontend prod (nginx) | `8080` | `5273` | `frontend/nginx.conf`, `frontend/Dockerfile`, `docker-compose.yml` |
| API service | `8081` | `8081` | `backend/api-service/config/config.go`, `backend/api-service/Dockerfile`, `docker-compose.yml` |
| Postgres | `5432` | `5532` | `docker-compose.yml`, `.env.example` |
| pgAdmin | `80` | `5150` | `docker-compose.yml`, `.env.example` |

Compose mappings:

```text
postgres        ${POSTGRES_PORT:-5532}      -> 5432
pgadmin         ${PGADMIN_PORT:-5150}       -> 80
api-service     ${API_SERVICE_PORT:-8081}   -> 8081
frontend        ${FRONTEND_PORT:-5273}      -> 8080
test-postgres   ${TEST_POSTGRES_PORT:-5533} -> 5432
```

Minikube mappings:

| Object | Port | Notes |
| --- | --- | --- |
| api-service Deployment | `containerPort 8081` | liveness `/health`, readiness `/ready` |
| api-service Service | `8081 -> 8081` | `ClusterIP` |
| frontend Deployment | `containerPort 8080` | nginx |
| frontend Service | `80 -> 8080` | `NodePort` |
| `startup.ps1` port-forward | `localhost:5273 -> frontend:80` | local browser access |

## Tests and Checks

Backend:

```powershell
cd backend/api-service
go test ./...
go vet ./...
./test-local.ps1
```

Frontend:

```powershell
cd frontend
npm test
npm run type-check
npm run lint
npm run format:check
```

`./test-local.ps1` starts the test Postgres service on `:5533`, enables cgo for
SQLite-backed manifest tests, and runs the Go coverage path that most closely
matches CI.

## Deployment Notes

This repository currently validates Docker images in CI but does not publish or
deploy them. Azure and production deployment planning belongs in private
operations notes until a deployment path is accepted and implemented.
