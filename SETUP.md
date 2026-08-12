# Setup Guide

Guardian Tracker is a local-first Destiny 2 companion app. It is designed for
development with Docker Compose, with Minikube kept as a deployment-manifest
validation path rather than production parity.

Production-specific infrastructure runbooks, cloud resource names, deployment
secrets, and incident notes belong under `private/`, which is gitignored.

## What You'll Need

- Docker Desktop for the recommended full-stack local environment.
- Go 1.26+ for backend development.
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

| Variable               | Purpose                                                         |
| ---------------------- | --------------------------------------------------------------- |
| `BUNGIE_API_KEY`       | Bungie API key                                                  |
| `BUNGIE_CLIENT_ID`     | Bungie OAuth client ID                                          |
| `BUNGIE_CLIENT_SECRET` | Bungie OAuth client secret                                      |
| `JWT_SECRET`           | 32+ character signing secret                                    |
| `DATABASE_URL`         | Postgres connection string; Compose sets this for the container |
| `TOKEN_ENCRYPTION_KEY` | 32-byte base64 AES-256-GCM key for stored Bungie tokens         |

Set the required runtime mode. Keep the current key version explicit in new
environment files; it defaults to `1` only to preserve existing version-1 rows:

| Variable                       | Purpose                                                                                                  |
| ------------------------------ | -------------------------------------------------------------------------------------------------------- |
| `GO_ENV`                       | Exactly `development` or `production`; there is no implicit default                                      |
| `TOKEN_ENCRYPTION_KEY_VERSION` | Positive `SMALLINT` version for the current encryption key (start at `1`; omitted value defaults to `1`) |

Optional values:

| Variable                                | Purpose                                                                        |
| --------------------------------------- | ------------------------------------------------------------------------------ |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS`         | Previous encryption key during key rotation                                    |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION` | Exact positive version for the previous encryption key; set only with that key |
| `ADMIN_MEMBERSHIP_IDS`                  | Comma-separated Bungie membership IDs pinned to admin at login                 |
| `CORS_ALLOWED_ORIGINS`                  | Explicit browser origins allowed to call the API                               |
| `JWT_ACCESS_TTL`                        | Access-token lifetime as a Go duration (default `30m`)                         |
| `LOG_LEVEL`                             | `debug`, `info`, `warn`, or `error`; defaults to `info`                        |
| `LOG_FORMAT`                            | `text` or `json`; defaults to `text` in development and `json` in production   |

Do not commit `.env`, generated secrets, manifest databases, cloud credentials, or
production runbooks.

See [SECURITY.md](./SECURITY.md#token-encryption-key-rotation) before rotating an
encryption key. The key and its version must move together.

Invalid `LOG_LEVEL` or `LOG_FORMAT` values fail startup. Application logs carry
a server-generated request UUID and use route templates rather than raw URLs.
They do not record query strings, request/response bodies, authorization
headers, User-Agent values, or routine client IPs. Security audit rows remain a
separate Postgres trail with the exact identifiers needed for forensics.

## 3. Run the Full Stack

Docker Compose is the default path for local development:

```powershell
docker compose up --build
```

The checked-in example sets `GO_ENV=development` explicitly. Compose refuses to
render the API service when `GO_ENV` is missing, preventing an accidental
implicit degraded-mode startup.

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

### Refreshing pinned container images

The nginx, PostgreSQL, and API runtime images use patch-qualified tags pinned to
multi-platform OCI index digests. This makes amd64 and arm64 builds reproducible.
Dependabot advances the tag and digest together when a newer tagged release is
available, but it may not report an upstream republish of the same tag. During an
image refresh, compare the registry's reported `Digest` with the committed
`@sha256` value before updating it:

```powershell
docker buildx imagetools inspect nginxinc/nginx-unprivileged:1.31.3-alpine3.24
docker buildx imagetools inspect postgres:18.4-alpine3.24
docker buildx imagetools inspect postgres:18.4
docker buildx imagetools inspect alpine:3.24.1
```

After pulling an image update, rebuild and recreate existing local containers so
Docker does not keep serving an older cached layer or container:

```powershell
docker compose --profile test --profile e2e pull postgres test-postgres e2e-postgres
docker compose build --pull --no-cache api-service frontend
docker compose --profile test --profile e2e up -d --force-recreate --wait postgres test-postgres e2e-postgres api-service frontend
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

The startup script rebuilds both application images with `--pull --no-cache` in
Minikube's Docker daemon. If applying a Deployment did not change its pod
template, the script restarts it so a reused local tag cannot leave old pods
running. Newly created deployments and pod-template changes already consume the
rebuilt images and are not restarted a second time.

See [k8s/README.md](./k8s/README.md) for script details and troubleshooting.

## Ports

Host ports are offset where useful so the stack can run beside other local
projects. Container ports stay fixed.

| Service               | Internal | Host / exposed | Defined in                                                                                     |
| --------------------- | -------- | -------------- | ---------------------------------------------------------------------------------------------- |
| Frontend dev (Vite)   | `5273`   | `5273`         | `frontend/vite.config.ts`, `frontend/Dockerfile.dev`                                           |
| Frontend prod (nginx) | `8080`   | `5273`         | `frontend/nginx.conf`, `frontend/Dockerfile`, `docker-compose.yml`                             |
| API service           | `8081`   | `8081`         | `backend/api-service/config/config.go`, `backend/api-service/Dockerfile`, `docker-compose.yml` |
| Postgres              | `5432`   | `5532`         | `docker-compose.yml`, `.env.example`                                                           |
| pgAdmin               | `80`     | `5150`         | `docker-compose.yml`, `.env.example`                                                           |
| E2E Postgres          | `5432`   | `5534`         | `docker-compose.yml`, `.env.example`                                                           |

Compose mappings:

```text
postgres        127.0.0.1:${POSTGRES_PORT:-5532}      -> 5432
pgadmin         127.0.0.1:${PGADMIN_PORT:-5150}       -> 80
api-service     ${API_SERVICE_PORT:-8081}   -> 8081
frontend        ${FRONTEND_PORT:-5273}      -> 8080
test-postgres   127.0.0.1:${TEST_POSTGRES_PORT:-5533} -> 5432
e2e-postgres    127.0.0.1:${E2E_POSTGRES_PORT:-5534}  -> 5432
```

Postgres, pgAdmin, and both disposable test databases bind only to loopback. The
frontend and API remain reachable on the configured host interfaces for local
browser and tunnel workflows. `e2e-postgres` has no data volume; it mounts the
normal database initializer read-only and resets when its container is removed.

Minikube mappings:

| Object                     | Port                            | Notes                                  |
| -------------------------- | ------------------------------- | -------------------------------------- |
| api-service Deployment     | `containerPort 8081`            | liveness `/health`, readiness `/ready` |
| api-service Service        | `8081 -> 8081`                  | `ClusterIP`                            |
| frontend Deployment        | `containerPort 8080`            | nginx                                  |
| frontend Service           | `80 -> 8080`                    | `NodePort`                             |
| `startup.ps1` port-forward | `localhost:5273 -> frontend:80` | local browser access                   |

## Tests and Checks

Backend:

```powershell
cd backend/api-service
go test ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...
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

Browser quality gates:

```powershell
# From the repository root
docker compose --profile e2e up -d --wait e2e-postgres
$env:E2E_FIXED_TIME="2026-07-18T18:00:00Z" # deterministic Saturday/Xur fixture

cd frontend
npm ci
npx playwright install chromium
npm run e2e          # functional journeys + axe + destructive auth last
npm run e2e:visual   # local preview; CI owns canonical Linux comparison

cd ..
docker compose --profile e2e down -v
```

The Playwright configuration launches `go run ./cmd/fake-bungie` from
`backend/api-service`, the real API, and Vite, then waits for fake health, API
readiness, and the asynchronous search index. The fake generates its tiny
SQLite manifest at runtime and provides loopback-only scenario controls; the
suite never contacts the real Bungie API. Tests use one worker because account
and scenario state are shared.

`.github/workflows/browser.yml` publishes reports and failure evidence for 14
days. Browser E2E + Axe and visual regression are advisory initially and do not
hide failures with `continue-on-error`. After ten consecutive clean E2E/axe
runs, add `Browser E2E + Axe` to branch protection. Keep `Browser Visual
Regression` optional.

`./test-local.ps1` starts the test Postgres service on `:5533`, enables cgo for
SQLite-backed manifest tests, and runs the Go coverage path that most closely
matches CI.

## Deployment Notes

This repository currently validates Docker images in CI but does not publish or
deploy them. Azure and production deployment planning belongs in private
operations notes until a deployment path is accepted and implemented.
