# AGENTS.md

This file gives AI coding agents repo-specific operating context for GuardianTracker. `CLAUDE.md` contains Claude-specific workflow notes; this file is the general agent guide.

## Project Overview

GuardianTracker is a Destiny 2 collection tracker. It authenticates users through Bungie OAuth, ingests profile and manifest data from the Bungie API, tracks collection gaps, supports wishlist planning, and surfaces weekly acquisition recommendations.

Primary stack:

- Frontend: React, TypeScript, Vite, CSS modules
- Backend: Go, Gin, PostgreSQL
- Local orchestration: Docker Compose
- Dev-validation orchestration: Kubernetes manifests for Minikube

## Ground Rules

- Do not build on unverified assumptions about Bungie API behavior, manifest shape, Destiny game data, or OAuth flows. Verify against code, docs, fixtures, or live/local behavior before making load-bearing changes.
- Keep public and private documentation separate. This is a public GitHub repo, so public docs must not contain secrets, private deployment details, internal-only notes, or sensitive audit data.
- Prefer existing project patterns over introducing new abstractions.
- Keep implementation changes scoped to the request. Do not rewrite unrelated code or docs.
- Do not remove user-authored work unless explicitly asked.
- When changing user-facing behavior, update the relevant public docs in the same change.

## Architecture

```text
React/Vite frontend (:5273) -> Go/Gin API (:8081) -> PostgreSQL
                               |
                               +-> Bungie OAuth and Bungie API
```

Key directories:

| Path                        | Purpose                                                            |
| --------------------------- | ------------------------------------------------------------------ |
| `backend/api-service/`      | Go API service, auth, Bungie integration, collection logic         |
| `frontend/src/`             | React application source                                           |
| `database/init/01-init.sql` | Local database initialization schema                               |
| `k8s/`                      | Minikube validation manifests                                      |
| `docs/`                     | Public project documentation                                       |
| `private/`                  | Local/private planning and archived notes; should remain untracked |
| `.claude/agents/`           | Claude-specific specialized agent instructions                     |
| `frontend/e2e/`             | Playwright functional, accessibility, and visual browser tests      |
| `backend/api-service/cmd/fake-bungie/` | Test-only Bungie/manifest fixture service                |

## Running Locally

Use `SETUP.md` as the source of truth for setup and troubleshooting.

Common commands:

```bash
# Full local stack
docker compose up -d

# Backend only
cd backend/api-service
go run .

# Frontend only
cd frontend
npm install
npm run dev
```

Default local ports:

| Service    | Port   |
| ---------- | ------ |
| Frontend   | `5273` |
| API        | `8081` |
| PostgreSQL | `5432` |

Minikube manifests are for development validation only. Do not treat them as production deployment guidance.

## Environment and Secrets

Never commit real secrets. Use `.env` locally and keep generated/private files out of git.

Important local variables:

| Variable                                | Purpose                                                              |
| --------------------------------------- | -------------------------------------------------------------------- |
| `GO_ENV`                                | Required runtime mode: exactly `development` or `production`         |
| `LOG_LEVEL`                             | `debug`, `info`, `warn`, or `error` (`info` default)                  |
| `LOG_FORMAT`                            | `text` or `json` (development text; production JSON by default)       |
| `BUNGIE_CLIENT_ID`                      | Bungie OAuth client ID                                               |
| `BUNGIE_CLIENT_SECRET`                  | Bungie OAuth client secret                                           |
| `BUNGIE_API_KEY`                        | Bungie API key                                                       |
| `AUTH_REDIRECT_URI`                     | OAuth callback URL                                                   |
| `JWT_SECRET`                            | JWT signing secret                                                   |
| `TOKEN_ENCRYPTION_KEY`                  | AES-256-GCM token encryption key                                     |
| `TOKEN_ENCRYPTION_KEY_VERSION`          | Positive version written for the current encryption key              |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS`         | Optional previous decryption key during rotation                     |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION` | Exact positive version for the previous key                          |
| `DATABASE_URL`                          | Postgres connection string                                           |
| `CORS_ALLOWED_ORIGINS`                  | Exact browser origins allowed to call the API                        |
| `ADMIN_MEMBERSHIP_IDS`                  | Optional comma-separated Bungie membership IDs with admin privileges |

Auth and security behavior to preserve:

- OAuth state is HMAC signed.
- Bungie tokens are encrypted at rest with AES-256-GCM and exact current/previous key versions.
- Access JWTs and the user snapshot are stored in localStorage; the rotating refresh JWT is only in the host-only HttpOnly `guardian_refresh_token` cookie.
- Refresh token revocation is backed by PostgreSQL.
- Callback and refresh require an exact allowlisted `Origin`; the cookie design assumes the frontend and API are same-site.
- Admin access is controlled by explicit membership ID configuration.

See `SECURITY.md` for public security posture and reporting guidance.

Logging behavior to preserve:

- Every request receives a server-owned UUID returned as `X-Request-ID` and attached to the request context.
- Access logs use route templates, method, status, duration, and response bytes; health-probe successes log at debug.
- Application logs pseudonymize membership, session, user, and character identifiers as deterministic 24-hex values. Exact identifiers belong only in the PostgreSQL audit trail.
- Never log query strings, bodies, authorization headers, User-Agent values, or routine client IPs.

## Testing and Validation

Use the narrowest relevant test first, then run broader checks when the change crosses module boundaries.

Common commands:

```powershell
# Backend tests (from the repository root)
cd backend/api-service
go test ./...
go run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...
cd ../..

# Frontend checks
cd frontend
npm run lint
npm run build
cd ..

# Hermetic browser checks (start the e2e Compose profile first)
docker compose --profile e2e up -d --wait e2e-postgres
$env:E2E_FIXED_TIME="2026-07-18T18:00:00Z"
cd frontend
npm run e2e
npm run e2e:visual
cd ..

# Local CI-equivalent backend coverage script
cd backend/api-service
./test-local.ps1
```

Required CI checks are documented in `README.md` and workflow files under `.github/workflows/`.
Browser E2E + Axe is advisory until ten consecutive clean runs; visual regression remains optional.

## Documentation Rules

Public docs:

- `README.md` - project overview, quickstart, and doc index
- `SETUP.md` - local setup and troubleshooting
- `docs/architecture.md` - current implemented architecture
- `ROADMAP.md` - public future-looking roadmap
- `SECURITY.md` - public security posture and reporting
- `CHANGELOG.md` - notable shipped changes
- `docs/adr/` - public architecture decisions
- `AGENTS.md` - general AI agent operating guide
- `CLAUDE.md` - Claude-specific operating guide

Private docs:

- `private/IMPLEMENTATION_PLAN.md` - detailed private implementation planning
- `private/ARCHIVE.md` - shipped private history, retired audits, and durable decisions

Rules:

- Keep public docs factual and safe for a public repository.
- Keep implementation plans, private audits, exploratory notes, and sensitive operational details in `private/`.
- Archive shipped private planning content instead of leaving duplicate active plans.
- When deleting or consolidating private docs, scrub references to the removed file.

## Agent Routing

If your agent environment supports specialized subagents, use the local instructions in `.claude/agents/` as task-specific guidance. These files are Claude-specific but still useful as repo notes.

| Agent file                                    | Use for                                   |
| --------------------------------------------- | ----------------------------------------- |
| `.claude/agents/go-services.md`               | Go API, auth, Bungie API, backend tests   |
| `.claude/agents/react-frontend.md`            | React, TypeScript, Vite, frontend UX      |
| `.claude/agents/postgres-specialist.md`       | Schema, migrations, SQL, data integrity   |
| `.claude/agents/kubernetes-infrastructure.md` | Kubernetes and Minikube validation        |
| `.claude/agents/docker-containers.md`         | Docker Compose and container build issues |
| `.claude/agents/penetration-tester.md`        | Security review and threat modeling       |
| `.claude/agents/code-reviewer.md`             | Focused code review                       |
| `.claude/agents/docs-updater.md`              | Documentation freshness and doc ownership |

## Known Limitations

- Collections remain account-wide; the character switcher scopes authenticated
  weekly vendor inventory, daily actions, availability ranking, and Xûr location
  to the selected Guardian.
- Search index snapshots persist beside the manifest by version; a missing or new-version snapshot rebuilds asynchronously.
- Xur location is resolved best-effort from the authenticated character-vendor
  component and manifest destination data; it is omitted when unavailable.
- Missing-count summaries intentionally omit non-raid/dungeon categories because
  current Bungie milestone and activity rewards do not expose collectible-linked items.
