# Guardian Tracker

Guardian Tracker is a Destiny 2 companion app that helps players understand
their collection gaps and decide what to chase next. Players sign in with Bungie
OAuth; the app analyzes collections, wishlist items, weekly data, catalysts,
crafting patterns, triumphs, seals, and cosmetics through a Go API and React
frontend.

## Core Features

- **Bungie OAuth login** with HMAC-signed CSRF state and Guardian Tracker JWTs.
- **Collection analysis** for weapons, armor, exotics, and cosmetics using the
  Bungie manifest and player collection state.
- **Wishlist management** with priority, notes, and availability surfacing.
- **This Week** with milestones, Xur inventory and best-effort Tower location,
  active-Guardian vendor context, class-aware armor labels, daily actions, reset
  timing, and recommendation ranking.
- **Catalysts, crafting, triumphs, and seals** from Bungie records data.
- **Cosmetics gallery** for emblems, shaders, ghosts, ships, sparrows, emotes,
  ornaments, and finishers.
- **Account-backed onboarding tour** that introduces the Dashboard, This Week,
  and Collections once per Bungie account.
- **Global item search** over a manifest-derived index.
- **Roles, feature flags, admin console, and audit log** for controlled rollout
  and administration.

## Tech Stack

| Concern               | Choice                                                             |
| --------------------- | ------------------------------------------------------------------ |
| Frontend              | React 19, TypeScript, Vite, React Router, TanStack Query           |
| Backend               | Go, Gin                                                            |
| User data             | PostgreSQL                                                         |
| Manifest data         | Bungie manifest SQLite database                                    |
| Local runtime         | Docker Compose                                                     |
| Kubernetes validation | Minikube manifests under `k8s/`                                    |
| CI                    | GitHub Actions format, Staticcheck, test, coverage, browser, and Docker validation |

## Architecture

```text
Frontend (React/TS :5273)
    -> API Service (Go/Gin :8081)
        -> Bungie API
        -> Postgres user data
        -> SQLite Destiny manifest
```

See [docs/architecture.md](./docs/architecture.md) for the public architecture
overview and [docs/adr](./docs/adr/README.md) for durable decisions.

## Quick Start

Detailed setup, ports, environment variables, and test commands live in
[SETUP.md](./SETUP.md).

```powershell
./setup.ps1
docker compose up --build
```

Open:

- Frontend: <http://localhost:5273>
- API: <http://localhost:8081>

The first API startup downloads the Destiny 2 manifest, so some data surfaces can
briefly show warming states.

## Common Commands

```powershell
# Full local stack
docker compose up --build

# Backend tests
cd backend/api-service
go test ./...
go run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...

# Frontend tests
cd frontend
npm test

# Frontend checks
npm run type-check
npm run lint
npm run format:check
```

For CI-equivalent Go coverage with cgo and test Postgres, run:

```powershell
cd backend/api-service
./test-local.ps1
```

Browser tests are hermetic: Playwright starts the test-only fake Bungie command,
the real API, and Vite; the suite never calls Bungie.net. Start only its isolated
database, set the deterministic fixture clock, then run the functional/axe or
visual project:

```powershell
docker compose --profile e2e up -d --wait e2e-postgres
$env:E2E_FIXED_TIME="2026-07-18T18:00:00Z"
cd frontend
npm run e2e
npm run e2e:visual
cd ..
docker compose --profile e2e down -v
```

Every pull request requires Format Check, Test Frontend, Test Go Services,
Build Docker Images, and Changelog Version. The Go job includes the pinned
Staticcheck command above; see
[CONTRIBUTING.md](./CONTRIBUTING.md#ci-gates) for the complete gate details.
The separate Browser E2E + Axe and Browser Visual Regression jobs do not use
`continue-on-error`, but remain non-required during stabilization. Promote E2E
and axe after ten consecutive clean runs; visual regression remains optional.

## Project Layout

```text
frontend/                 React + TypeScript SPA
backend/api-service/      Go API service
database/init/            Postgres bootstrap SQL
k8s/                      Minikube validation manifests and scripts
docs/                     Public architecture docs and ADRs
private/                  Gitignored private plans, runbooks, and research
.github/workflows/        CI and version-release workflows
```

## Documentation

- [SETUP.md](./SETUP.md) - local setup, environment, ports, tests.
- [docs/architecture.md](./docs/architecture.md) - implemented architecture.
- [docs/README.md](./docs/README.md) - public/private documentation boundary.
- [ROADMAP.md](./ROADMAP.md) - not-yet-implemented work and gates.
- [CHANGELOG.md](./CHANGELOG.md) - shipped changes by version.
- [SECURITY.md](./SECURITY.md) - security model, reporting, checklist.
- [CONTRIBUTING.md](./CONTRIBUTING.md) - contribution workflow and CI gates.
- [AGENTS.md](./AGENTS.md) - canonical AI agent operating guide (all tools).
- [CLAUDE.md](./CLAUDE.md) - Claude Code specifics; imports AGENTS.md.
- [frontend/README.md](./frontend/README.md) - frontend-specific guide.
- [k8s/README.md](./k8s/README.md) - Minikube validation guide.

This is a public repository. Keep secrets, production runbooks, private security
reviews, raw research dumps, and detailed implementation handoffs under
`private/`, which is gitignored.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](./CONTRIBUTING.md) for workflow,
formatting, tests, and PR expectations. Future work is tracked in
[ROADMAP.md](./ROADMAP.md).

By participating you agree to the [Code of Conduct](./CODE_OF_CONDUCT.md). For
support, see [SUPPORT.md](./SUPPORT.md). Report vulnerabilities privately per
[SECURITY.md](./SECURITY.md).

## License

MIT License - see [LICENSE](./LICENSE).
