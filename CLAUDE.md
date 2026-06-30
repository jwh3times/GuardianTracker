# Guardian Tracker — Claude Code Guide

## Project Overview

Guardian Tracker is a Destiny 2 collection tracker web app. Players log in via Bungie OAuth; the app analyzes their collections to surface missing items with acquisition difficulty ratings, wish-list management, and weekly recommendations.

## Ground Rules

- **Don't build on unverified assumptions — ask.** When a task depends on a fact you can't confirm from the code, the docs, or a quick check — especially **external or domain facts** (Bungie API response shapes, the manifest's presentation-node structure, third-party behavior, game data) — stop and ask before designing against a guess. If ground truth is _obtainable_ (a real Bungie manifest is a public CDN download needing only `BUNGIE_API_KEY` — no OAuth; a running service; a sample response), ask to get it and verify **before** writing the implementation, not as a manual step deferred to the end.
- A sensible default for a genuinely low-stakes choice is fine — state it and proceed. The bar: would being wrong force a rework or ship something incorrect? If yes, it's load-bearing — ask.

## Architecture

Two services: a Go API backend and a React frontend that calls it directly over REST.

```text
Frontend (React/TS :5273)
    └─► API Service (Go/Gin :8081)  — OAuth, JWT, manifest, collections
```

For the full port map — Docker Compose, Kubernetes, dev/cross-service wiring — see **[Ports in README.md](./README.md#ports)**.

## Running Services

- **Docker Compose** (Option A) — one command for the full stack; best default for local dev and integration testing.
- **Minikube** (Option B) — validates Kubernetes manifests; dev-validation only (no Postgres).
- **Individual services** (Option C) — fast hot reload during active single-service development.

### Option A: Docker Compose (full stack)

```powershell
cp .env.example .env      # fill in BUNGIE_* secrets
docker compose up --build
```

- Frontend `http://localhost:5273`, API `http://localhost:8081`, pgAdmin `http://localhost:5150`
- Postgres `:5532`, Redis `:6379` (Redis in compose but not actively used)
- Bungie manifest persists in the `manifest-data` named volume.
- `database/init/01-init.sql` auto-loads into Postgres on first run.

```powershell
docker compose down        # stop (keeps volumes)
docker compose down -v     # stop and wipe Postgres/Redis/manifest volumes
```

### Option B: Kubernetes (Minikube)

```powershell
cd k8s
./startup.ps1
```

Dev-validation only — runs `GO_ENV: development` with no Postgres (in-memory token store, no wishlist/preferences persistence).

### Option C: Individual services

```powershell
# API Service (from backend/api-service/)
go run .        # or: air   (hot reload via .air.toml)

# Frontend (from frontend/)
npm start       # Vite dev server on :5273
```

### Exposing via ngrok (public HTTPS)

To test Bungie OAuth, tunnel the frontend with ngrok. Add the ngrok URL to `CORS_ALLOWED_ORIGINS` in the root `.env`, rebuild api-service, set `AUTH_REDIRECT_URI` to the ngrok callback, and register that URL in your Bungie app at https://www.bungie.net/en/Application. Free ngrok subdomains change on each restart — update all three every time.

## Environment Setup

```powershell
./setup.ps1     # copies all .env.example files at once
```

Or copy manually: root `.env`, `backend/api-service/.env`, `frontend/.env.local`.

### Required secrets

| Variable                        | Purpose                                                                                               |
| ------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `BUNGIE_API_KEY`                | From https://www.bungie.net/en/Application                                                            |
| `BUNGIE_CLIENT_ID`              | Bungie app settings                                                                                   |
| `BUNGIE_CLIENT_SECRET`          | Bungie app settings                                                                                   |
| `JWT_SECRET`                    | 32+ char random string (`openssl rand -base64 32`)                                                    |
| `DATABASE_URL`                  | Postgres connection string (`postgres://guardian_app:...@host:5532/guardian_tracker?sslmode=disable`) |
| `TOKEN_ENCRYPTION_KEY`          | 32-byte base64 key for Bungie token encryption (`openssl rand -base64 32`)                            |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS` | (optional) previous key for rotation during key migration                                             |
| `ADMIN_MEMBERSHIP_IDS`          | (optional) comma-separated Bungie membership IDs pinned to admin role at login                        |

## CI/CD

GitHub Actions (`.github/workflows/ci-cd.yml`) — four required jobs:

1. **format-check** — `npm run format:check` (Prettier) + `gofmt`. Fix: `npm run format` from `frontend/` or `gofmt -w .` from `backend/api-service/`.
2. **test-frontend** — type-check, lint, Vitest coverage (≥70% lines, ≥65% branches), build
3. **test-go-services** — `go vet`, `govulncheck`, `go test -race` + Postgres container; statement coverage ≥60%
4. **build-docker-images** — build validation only (no push configured)

CodeQL runs on PRs via default setup; gated through the code-scanning merge rule (not as a required status check — requiring CodeQL `Analyze` contexts blocks Dependabot PRs which never produce them).

### Branch protection (`main`)

Ruleset id `17717600` (Settings → Rules): PR required, 0 approvals (self-merge once green), required status checks (`Format Check`, `Test Frontend`, `Test Go Services`, `Build Docker Images`), code-scanning gate (errors+warnings / medium+), no bypass actors.

To change the gate: `gh api repos/jwh3times/GuardianTracker/rulesets/17717600` or GitHub UI. Required check names must match CI job `name:` exactly.

## Agent delegation

For specialized work, invoke the appropriate subagent:

| Task                                                                                                                 | Agent                       |
| -------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| Go backend — Gin handlers, JWT/auth, Bungie OAuth, manifest, collections, records, weekly, search, roles/flags/admin | `go-services`               |
| React frontend — pages, components, design system (`gt-*`), React Query, AuthContext, FlagsContext, Vitest tests     | `react-frontend`            |
| PostgreSQL schema, migrations, token store, audit log; SQLite manifest queries                                       | `postgres-specialist`       |
| Kubernetes manifests, Minikube, kubectl, secrets, configmaps                                                         | `kubernetes-infrastructure` |
| Dockerfiles, image builds, layer caching                                                                             | `docker-containers`         |
| Security testing — OAuth, JWT, CSRF, data isolation, XSS, CORS, admin endpoints                                      | `penetration-tester`        |
| Code review — correctness, security, pattern violations                                                              | `code-reviewer`             |
| Documentation sync — CLAUDE.md, README.md, SECURITY.md, all agent files                                              | `docs-updater`              |

### Running tests

```powershell
# Go (from backend/api-service/)
go test ./...
./test-local.ps1          # full CI-equivalent: cgo + Postgres (see go-services agent for flags)

# Frontend (from frontend/)
npm test
```

## Known Limitations

- Character switcher drives display only; collection data is account-wide (character-scoped surfaces are P2)
- Redis is in docker-compose but not actively used (JWT revocation and token persistence are Postgres-backed)
- Search index is in-memory — lost on restart; rebuilds automatically (~30s after manifest is ready)
- Xûr location is always "Unknown" — the public Bungie API does not expose vendor location
- Raid and dungeon milestones carry a real missing count; non-raid/dungeon milestones still omit the field (no manifest reward→collectible signal)
