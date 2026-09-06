# AGENTS.md

This is the canonical operating guide for AI coding agents working in Guardian Tracker.
It is tool-neutral and self-contained: agents that read `AGENTS.md` natively (Codex and
others) get the complete picture from this file alone.

`CLAUDE.md` imports this file via `@AGENTS.md` and adds only Claude Code-specific
mechanics. **Repo operating context belongs here, not there.**

## Project Overview

Guardian Tracker is a Destiny 2 collection tracker web app. Players log in via Bungie OAuth; the app analyzes their collections to surface missing items with source-specific acquisition guidance, wish-list management, and weekly recommendations.

Primary stack:

- Frontend: React, TypeScript, Vite, global CSS with `gt-*` design-token system
- Backend: Go, Gin, PostgreSQL
- Local orchestration: Docker Compose
- Dev-validation orchestration: Kubernetes manifests for Minikube

## Ground Rules

- **Don't build on unverified assumptions — ask.** When a task depends on a fact you can't confirm from the code, the docs, or a quick check — especially **external or domain facts** (Bungie API response shapes, the manifest's presentation-node structure, third-party behavior, game data) — stop and ask before designing against a guess. If ground truth is _obtainable_ (a real Bungie manifest is a public CDN download needing only `BUNGIE_API_KEY` — no OAuth; a running service; a sample response), ask to get it and verify **before** writing the implementation, not as a manual step deferred to the end.
- A sensible default for a genuinely low-stakes choice is fine — state it and proceed. The bar: would being wrong force a rework or ship something incorrect? If yes, it's load-bearing — ask.
- Keep public and private documentation separate. This is a public GitHub repo, so public docs must not contain secrets, private deployment details, internal-only notes, or sensitive audit data.
- Prefer existing project patterns over introducing new abstractions.
- Keep implementation changes scoped to the request. Do not rewrite unrelated code or docs.
- Do not remove user-authored work unless explicitly asked.
- When changing user-facing behavior, update the relevant public docs in the same change.

## Architecture

Two services: a Go API backend and a React frontend that calls it directly over REST.

```text
Frontend (React/TS :5273)
    └─► API Service (Go/Gin :8081)  — OAuth, JWT, manifest, collections
```

For the full port map — Docker Compose, Kubernetes, dev/cross-service wiring — see **[Ports in SETUP.md](./SETUP.md#ports)**.

### Key directories

- `backend/api-service/` — Go API: `api/router.go` (the route table; `main.go` is a composition root and registers no routes), `api/handlers/` (Gin handlers), `auth/` (JWT issue/verify, middleware, HMAC-signed OAuth state, roles, revocation, encrypted token store, `SessionIssuer` owning login/reconnect/refresh/logout), `db/` (Postgres stores + embedded migrations, audit log, users/roles/flags/wishlist/prefs; `Stores` fields are interfaces backed by degraded implementations — never nil — when there is no database; `db/adapters/` translates stores into consumer-side interfaces, including the membership-keyed Preferences repository), `services/` (bungie client, manifest, collections, records, weekly, recommendations, search, items, characters, efficiency, sources, preferences), `config/`, `cache/`.
- `frontend/src/` — React app: `features/` (pages), `components/`, `contexts/` (AuthContext, FlagsContext), `lib/`, `types/`.
- `database/init/01-init.sql` — Postgres bootstrap for Docker Compose; `k8s/` — Minikube manifests.
- `frontend/e2e/` — Playwright functional, accessibility, and visual browser tests.
- `backend/api-service/cmd/fake-bungie/` — Test-only Bungie/manifest fixture service.

### Acquisition-source invariants

Read [CONTEXT.md](./CONTEXT.md) for the canonical acquisition-source, difficulty,
raid/dungeon, availability, and farm-only vocabulary. Preserve these implementation
invariants:

- Collection and wishlist item projections expose `acquisitionSources`, the
  deterministic union contributed by every linked collectible. Manifest lookups by
  item hash therefore return all matching collectibles, not one representative row.
- Difficulty and raid/dungeon are source facets. Items have no aggregate difficulty;
  live `availableNow` remains a separate vendor-derived join.
- Efficiency counts each item hash once per source bucket and once across a matched
  milestone's bucket union. Weekly recommendation difficulty remains scoped to its
  source/action.
- Efficiency publishes ordered, capped ranked facts behind an explicit
  cold/ready result and retains its previous complete index while a replacement
  builds. Recommendations owns wording, emphasis, difficulty, and ranked-versus-
  fallback selection; Weekly calls it once and only adapts the complete outcomes
  to its response shape.
- Farm-only classification retains the current representative-collectible behavior.
  Its multi-collectible semantics are unresolved and require a separate decision.

### Auth & token flow

Bungie OAuth login uses a public client with stateless, HMAC-signed CSRF `state`.
The API sends no Bungie client secret; Bungie returns an expiring access token
without a refresh token. The access-only Bungie authorization is stored against
the tracked Destiny membership, **AES-256-GCM encrypted** in Postgres with
explicit current/previous key versions. When it expires, authenticated users
reconnect the same Bungie membership through `POST /api/auth/bungie/reconnect`;
that replaces only the Bungie authorization and does not mint or rotate a
Guardian Tracker session.

Guardian Tracker separately returns a short-lived access JWT for localStorage
and sends its per-device rotating refresh JWT only in a host-only HttpOnly cookie
scoped to `/api/auth`. The callback, reconnect, and refresh endpoints require an
exact allowlisted browser origin; the cookie policy assumes the frontend and API
are same-site. Refresh sessions retain Postgres-backed revocation and reuse
detection. Single-device logout preserves the membership-wide Bungie
authorization; logout-all evicts it. Role tiers (standard / beta / alpha / admin)
and feature flags gate endpoints; `ADMIN_MEMBERSHIP_IDS` pins admins at login.
Security details and the credential-rotation runbook live in
[SECURITY.md](./SECURITY.md).

## Running Services

- **Docker Compose** (Option A) — one command for the full stack; best default for local dev and integration testing.
- **Minikube** (Option B) — validates Kubernetes manifests; dev-validation only (no Postgres).
- **Individual services** (Option C) — fast hot reload during active single-service development.

`SETUP.md` is the human-facing setup guide and remains the fuller reference. The
agent-facing subset — ports, environment table, test commands — is duplicated
inline below on purpose, because tool-neutral agents cannot follow links out of
this file.

### Option A: Docker Compose (full stack)

```powershell
cp .env.example .env      # fill in BUNGIE_* secrets
docker compose up --build
```

- Frontend `http://localhost:5273`, API `http://localhost:8081`, pgAdmin `http://localhost:5150`
- Postgres `:5532`; Postgres and pgAdmin bind only to `127.0.0.1`
- Bungie manifest persists in the `manifest-data` named volume.
- `database/init/01-init.sql` auto-loads into Postgres on first run.

```powershell
docker compose down        # stop (keeps volumes)
docker compose down -v     # stop and wipe Postgres/manifest volumes
```

### Option B: Kubernetes (Minikube)

```powershell
./setup.ps1     # first run: creates the ignored Secret without overwriting
cd k8s
# Fill api-service-secret.yaml placeholders before startup.
./startup.ps1
```

Dev-validation only — runs `GO_ENV: development` with no Postgres (in-memory token store, no wishlist/preferences persistence).
The startup script rebuilds both application images in Minikube with
`--pull --no-cache`; because the Deployments use local images with
`imagePullPolicy: Never`, it restarts only an existing Deployment whose applied
pod template did not already trigger a rollout.

### Option C: Individual services

```powershell
# API Service (from backend/api-service/)
go run .

# Frontend (from frontend/)
npm start       # Vite dev server on :5273
```

### Exposing via ngrok (public HTTPS)

To test Bungie OAuth, tunnel the frontend with ngrok. Add the ngrok URL to `CORS_ALLOWED_ORIGINS` in the root `.env`, rebuild api-service, set `AUTH_REDIRECT_URI` to the ngrok callback, and register that URL in your Bungie app at <https://www.bungie.net/en/Application>. Free ngrok subdomains change on each restart — update all three every time.

## Environment Setup

```powershell
./setup.ps1     # creates missing env files and the ignored Minikube Secret
```

Or copy manually: root `.env`, `backend/api-service/.env`,
`frontend/.env.local`, and `k8s/api-service-secret.yaml` from their committed
example files.

Public-only development requires no private workspace. Authorized maintainers
can optionally restore the ignored `private/` directory as an independent Git
repository with `./scripts/bootstrap-private-workspace.ps1 -PrivateFromPrompt`,
or through 1Password with `-PrivateFromOnePassword` after creating the ignored
machine-local `.private-workspace/repository.env.ref` file containing only
`GUARDIAN_PRIVATE_REPOSITORY_URL=op://<vault>/<item>/<field>`. Do not put the
private repository location or real 1Password identifiers in public docs.
`npm run bootstrap:private` (from the repo root) is the Node equivalent for new
git worktrees; it reuses the main checkout's reference file.

From the repo root, `npm run sync:main` preflights the public checkout and the
optional independent `private/` repository for uncommitted changes, fetches and
prunes `origin`, switches each checkout to `main`, and fast-forwards it from
`origin/main`. A failed preflight prevents either repository from being changed;
diverged history fails without a merge commit or history rewrite. A missing
`private/` repository is skipped. Use `npm run sync:main -- --skip-private` to
update only the public checkout.

The complete value-free recovery, verification, worktree, and backup handoff is
in `docs/maintainers/workspace-recovery.md`. The restored private repository's
`README.md` is the sole index for its internal file layout.

When private restoration templates are available, run
`./scripts/restore-private-secrets.ps1` before `./setup.ps1`; both helpers refuse
to overwrite existing environment files, so `setup.ps1` must not create example
copies first. The restore helper supports `-Target root,api,frontend,k8s` and
writes only targets protected by committed ignore rules. Run
`./scripts/workspace-status.ps1` for value-free public/private status and target
protection checks. It redacts the private branch by default; ahead/behind values
use local tracking refs without fetching. `-IncludePrivateBranch` opts into
showing the private branch name.

**Never commit real secrets. Use `.env` locally and keep generated/private files out of git.**

Frontend tooling uses the exact Node.js 26 patch in the root `.nvmrc`.
`frontend/package.json` accepts the Node 26 line only; CI and both frontend
Dockerfiles use the exact `.nvmrc` patch. Node 26 remains Current until its
scheduled LTS transition on October 28, 2026; this project intentionally accepts
that short-term churn because Node is frontend development/build tooling and the
deployed frontend runtime is nginx.

### Required runtime values

| Variable                                | Purpose                                                                                               |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `GO_ENV`                                | Required; exactly `development` or `production`                                                       |
| `LOG_LEVEL`                             | `debug`, `info`, `warn`, or `error`; defaults to `info`                                               |
| `LOG_FORMAT`                            | `text` or `json`; defaults to text in development and JSON in production                              |
| `BUNGIE_API_KEY`                        | From <https://www.bungie.net/en/Application>                                                          |
| `BUNGIE_CLIENT_ID`                      | Public Bungie OAuth client identifier; no client secret is used                                       |
| `AUTH_REDIRECT_URI`                     | OAuth callback URL                                                                                    |
| `JWT_SECRET`                            | 32+ char random string (`openssl rand -base64 32`)                                                    |
| `DATABASE_URL`                          | Postgres connection string (`postgres://guardian_app:...@host:5532/guardian_tracker?sslmode=disable`) |
| `TOKEN_ENCRYPTION_KEY`                  | 32-byte base64 key for Bungie authorization encryption (`openssl rand -base64 32`)                    |
| `TOKEN_ENCRYPTION_KEY_VERSION`          | Positive version written for the current key (start at `1`)                                           |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS`         | (optional) previous key for rotation during key migration                                             |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION` | Exact positive version for the previous key                                                           |
| `CORS_ALLOWED_ORIGINS`                  | Exact browser origins allowed to call the API                                                         |
| `ADMIN_MEMBERSHIP_IDS`                  | (optional) comma-separated Bungie membership IDs pinned to admin role at login                        |

### Auth and security behavior to preserve

- OAuth state is HMAC signed.
- The access-only Bungie authorization is encrypted at rest with AES-256-GCM and exact current/previous key versions; expiry requires an authenticated reconnect of the same Bungie membership.
- The shared browser session client atomically persists access JWT/user state in the versioned `guardian_browser_session` localStorage envelope and owns authenticated transport. Web Locks coordinate callback, refresh, and logout across tabs; callback completion and refresh require them. `AuthProvider` only subscribes to the public user/authenticated snapshot. The rotating refresh JWT is only in the host-only HttpOnly `guardian_refresh_token` cookie.
- Refresh token revocation is backed by PostgreSQL.
- Callback, authenticated Bungie reconnect, and refresh require an exact allowlisted `Origin`; the cookie design assumes the frontend and API are same-site.
- Single-device logout preserves Bungie authorization; logout-all evicts it.
- Admin access is controlled by explicit membership ID configuration.

See `SECURITY.md` for public security posture and reporting guidance.

### Application logging

The API uses `log/slog` and assigns every request a server-generated UUID exposed
as `X-Request-ID`. Access records use Gin route templates and include method,
status, duration, and response bytes. They never include raw URLs/query strings,
bodies, authorization headers, User-Agent values, or routine client IPs.
Membership, session, user, and character identifiers are deterministic 24-hex
pseudonyms in application logs; exact values remain only in the PostgreSQL audit
trail. Successful app requests log at info, 4xx at warn, 5xx at error, and
successful health probes at debug.

## CI/CD

GitHub Actions (`.github/workflows/ci-cd.yml`) — five required jobs. Both this
workflow and `.github/workflows/browser.yml` provision Node from the root
`.nvmrc`:

1. **format-check** — Prettier over `frontend/`, Prettier over repo markdown, and `gofmt`. Fix: `npm run format` from `frontend/`; `./frontend/node_modules/.bin/prettier --write "**/*.md"` from the repo root; `gofmt -w .` from `backend/api-service/`. The frontend-scoped run cannot reach markdown outside `frontend/`, which is why the root markdown step exists — editing `README.md`, `SETUP.md`, `docs/`, or `.claude/` requires the root command.
   It also runs `node --test scripts/sync-agent-configs.test.mjs scripts/workflow-pins.test.mjs scripts/node-version-policy.test.mjs scripts/postgres-pin-policy.test.mjs scripts/workspace-portability.test.mjs scripts/sync-main.test.mjs scripts/documentation-links.test.mjs`,
   which exercises the generator's own logic and enforces the repository's workflow-action,
   Go security-tool, Node-version, PostgreSQL-image, workspace-portability, safe
   main-branch synchronization, and local documentation-link policies. The Node policy keeps
   `.nvmrc`, both workflows, both frontend Dockerfiles, package engine metadata, and Node ambient
   types on the Node 26 line, with one exact patch for local, CI, and container tooling. The
   PostgreSQL policy keeps the `Test Go Services` service container on the same `major.minor` as
   the three Compose PostgreSQL services — Dependabot's `docker-compose` ecosystem does not see
   workflow service images — and fails on any retired `postgres:<version>` reference left behind
   in a tracked file. The job also runs
   `npm run sync:agents -- --check`, which fails if
   `.codex/agents/` (generated from `.claude/agents/`) or `.claude/skills/`
   (generated from `.agents/skills/`) is out of sync with its source. Fix:
   `npm run sync:agents`. Run `./frontend/node_modules/.bin/prettier --write "**/*.md"`
   first if you edited a skill's markdown — regenerating before formatting mirrors
   unformatted content and drifts again on the next format pass.
2. **test-frontend** — type-check, lint, Vitest coverage (≥70% lines, ≥65% branches), build
3. **test-go-services** — `go vet`, Staticcheck 2026.1, declared `govulncheck` tool v1.6.0 via `go tool govulncheck`, `go test -race` + Postgres container; statement coverage ≥60%
4. **build-docker-images** — build validation only (no push configured)
5. **changelog-version** — verifies `CHANGELOG.md`'s top version equals the tag the
   merge will mint (`scripts/next-version.sh`, the same oracle `version.yml` uses).
   Bot-authored PRs are exempt; `/ship` backfills their entries.

The workflow also runs **Test Workspace Portability (Windows)** as a non-required
validation job. It executes the portability suite on `windows-latest` under both
Windows PowerShell 5.1 and PowerShell 7; the portability job is not a protected
branch check.

`.github/workflows/browser.yml` adds **Browser E2E + Axe**, a required check
promoted after exceeding ten consecutive clean runs, and the advisory **Browser
Visual Regression** job. Both report failures normally (no `continue-on-error`)
and retain reports/evidence for 14 days; visual stays optional.

Every third-party workflow `uses:` entry is pinned to a reviewed 40-character
release commit with a readable `# vX.Y.Z` comment. The `github-actions`
Dependabot ecosystem advances both the SHA and comment. `actions/setup-node`
remains at v7.0.0 until the fix tracked by upstream issue 1596 reaches a v7
release; take the first patched v7 update proposed by Dependabot.

CodeQL runs on PRs via default setup; gated through the code-scanning merge rule (not as a required status check — requiring CodeQL `Analyze` contexts blocks Dependabot PRs which never produce them).

**Versioning** (`.github/workflows/version.yml`): every merge (push) to `main` tags the merge commit with a three-part version from the root `VERSION` file (`v<major>.<minor>.<build>`, e.g. `v0.2.0`). The third component is the auto-incrementing build number. When the major/minor version is bumped, build `0` is allowed. Tag-based because `main` is protected with no bypass actors — a workflow can't push a bump commit.

### Branch protection (`main`)

Repository rules (Settings -> Rules): PR required, 0 approvals (self-merge once green), required status checks (`Format Check`, `Test Frontend`, `Test Go Services`, `Build Docker Images`, `Changelog Version`, `Browser E2E + Axe`), code-scanning gate (errors+warnings / medium+), no bypass actors. `Browser Visual Regression` remains optional.

To change the gate, update the repository ruleset through GitHub UI or the GitHub API. Required check names must match CI job `name:` exactly.

## Testing and Validation

Use the narrowest relevant test first, then run broader checks when the change crosses module boundaries.

```powershell
# Windows workspace portability (from the repo root)
npm run test:workspace-portability

# Go (from backend/api-service/)
go test ./...
go run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...
go tool govulncheck ./...
./test-local.ps1          # full CI-equivalent: cgo + Postgres (see go-services agent for flags)

# Frontend (from frontend/)
npm run type-check
npm run lint
npm test -- --coverage
npm run build

# Browser (start from repo root, then run scripts from frontend/)
docker compose stop frontend api-service   # 5273/8081 would be silently reused
docker compose --profile e2e up -d --wait e2e-postgres
$env:E2E_FIXED_TIME="2026-07-18T18:00:00Z"
cd frontend
npm run e2e
npm run e2e:visual
```

`npm run type-check`, `npm run lint`, `npm test -- --coverage` (Vitest coverage), and `npm run build`
are the same steps the `Test Frontend` CI job runs. Playwright reuses any server
already on 5273/8081 outside CI, so a running Compose app stack silently replaces
the hermetic one and the suite dies at login with `Failed to fetch`. Visual
baselines are Linux renderings and must be regenerated inside the
`mcr.microsoft.com/playwright` image whose tag matches the installed
`@playwright/test` — never commit snapshots produced on
Windows. Both procedures, including baseline regeneration, are in
[frontend/README.md](./frontend/README.md#browser-tests).

### Full Go coverage locally (matches CI)

A plain `go test ./...` under-reports coverage (~52% vs CI's ~63%) because two test groups
self-skip: sqlite-backed manifest/search tests need **cgo**, and the `db` package integration tests
need a reachable Postgres via **`TEST_DATABASE_URL`** (distinct from `DATABASE_URL` — unit tests
must still exercise the degraded no-DB paths). `./test-local.ps1` (from `backend/api-service/`)
closes both gaps: it starts the throwaway `test-postgres` Compose service on port **5533** (so it
won't collide with the main Postgres on 5532), exports `CGO_ENABLED=1` + `TEST_DATABASE_URL`, and
runs `go test -race -coverprofile`. Flags: `-Html` opens the HTML coverage report; `-Down` removes
the test container afterwards. Migrations are embedded and applied automatically by the harness.

## Documentation Rules

Public docs:

- `README.md` - project overview, quickstart, and doc index
- `SETUP.md` - local setup and troubleshooting
- `CONTEXT.md` - project glossary: the domain vocabulary and the seam names, one
  owner per term. Read it before naming a concept in code, a test, or an issue.
- `docs/architecture.md` - current implemented architecture
- `ROADMAP.md` - public future-looking roadmap
- `SECURITY.md` - public security posture and reporting
- `CHANGELOG.md` - notable shipped changes
- `docs/adr/` - public architecture decisions
- `AGENTS.md` - canonical, tool-neutral agent operating guide (this file)
- `CLAUDE.md` - thin `@AGENTS.md` importer plus Claude Code-specific mechanics
- `docs/agents/` - per-repo configuration for third-party engineering skills
  (issue tracker, triage labels, domain docs); see `## Agent skills` below
- `docs/maintainers/workspace-recovery.md` - value-free private-workspace and
  machine-local configuration recovery runbook for authorized maintainers

Private docs:

- `private/README.md` - sole index for operations, residual
  security risks, reference evidence, and retired work. Private paths may be
  reorganized without duplicating that internal layout in this public guide.

Rules:

- Keep public docs factual and safe for a public repository.
- Keep private audits, exploratory notes, and sensitive operational details in
  `private/`. Execution sequencing and slice order are **not** private documents —
  they live on the board (see [Work Tracking](#work-tracking) and
  [ADR 0022](./docs/adr/0022-github-owns-task-status.md)). Do not recreate a
  private status file.
- Archive shipped private planning content instead of leaving duplicate active plans.
- When deleting or consolidating private docs, scrub references to the removed file.

Public committed docs describe implemented behavior, local setup, durable decisions, security
model, shipped changes, and gated future work. `private/` is gitignored and, for authorized
maintainers, can be an independent private documentation repository. It holds
deployment runbooks, private security reviews, raw Bungie/API research, and
environment-specific operations notes. Public contributors do not need it. Do not move private
operational detail into public docs or expose its remote location through the public repository.

## Work Tracking

Task status lives on GitHub, not in this repository's files. See
[ADR 0022](./docs/adr/0022-github-owns-task-status.md).

- **Board:** the private user-level Project `Guardian Tracker`
  (<https://github.com/users/jwh3times/projects/4>), linked to both the public
  repository and the private companion. Fields: `Chain`, `Order`, `Blocked By`,
  `Gate`, `ADR`, `Status`.
- **Issues go in the public repository by default.** File in the private
  companion only when the body would need a credential, a real
  provider/cost/account identifier, or exploitable security detail.
- **Unclaimed work is a project draft item, not an issue.** An issue is created
  when a slice is claimed, so the open-issue list reads as live status. Convert
  the draft, then wire native `blocked_by` edges — drafts have no number, so
  their ordering is carried by the `Blocked By` field.
- **The ready frontier is computed, not asserted:** open items with no open
  blocker and no assignee.
- Record point-in-time findings as comments on the issue they informed. Do not
  create standing status documents; that is the drift ADR 0022 removes.

### Rules for every architecture-deepening slice

The remaining slices sequenced by ADRs 0014–0021 are behavior-preserving
ownership migrations. For each one:

- Land the replacement, the deletion of the old path, tests, and affected
  documentation in **one green PR**. Avoid dual owners and cleanup-only
  follow-ups.
- Follow the controlling ADR's migration and acceptance steps. When its final
  slice ships, update the ADR and the index status to "implemented in `vX.Y.Z`".
- Preserve ADR 0003's verify-first rule for Manifest assumptions and ADR 0008's
  HttpOnly same-site refresh-cookie design.
- Preserve the participant/observer split, Item acquisition-facts ownership,
  typed Collections outcomes, explicit Wish list tombstones, and Preferences'
  atomic membership-keyed synchronization contract.
- Do not decide farm-only multi-collectible semantics, collapse source-string and
  milestone-name vocabularies, or assume deterministic Efficiency buckets
  without separate real-Manifest evidence.
- Keep roadmap features out of these migrations.

A slice is ready only when its dependencies are complete, its replacement and
deletion boundaries are explicit, its load-bearing external facts have current
evidence, and its tests exercise the new seam rather than duplicating its data.

## Agent Routing

Specialized agent definitions live in `.claude/agents/`. If your tool supports
subagents, dispatch them; if not, read the file for the relevant area as repo notes —
they are useful either way.

| Task                                                                                                                          | Agent                       | Definition                                    |
| ----------------------------------------------------------------------------------------------------------------------------- | --------------------------- | --------------------------------------------- |
| Go backend — Gin handlers, JWT/auth, Bungie OAuth, manifest, collections, records, weekly, search, roles/flags/admin          | `go-services`               | `.claude/agents/go-services.md`               |
| React frontend — pages, components, design system (`gt-*`), React Query, AuthContext, FlagsContext, Vitest tests              | `react-frontend`            | `.claude/agents/react-frontend.md`            |
| PostgreSQL schema, migrations, token store, audit log; SQLite manifest queries                                                | `postgres-specialist`       | `.claude/agents/postgres-specialist.md`       |
| Kubernetes manifests, Minikube, kubectl, secrets, configmaps                                                                  | `kubernetes-infrastructure` | `.claude/agents/kubernetes-infrastructure.md` |
| Dockerfiles, image builds, layer caching                                                                                      | `docker-containers`         | `.claude/agents/docker-containers.md`         |
| Security testing — OAuth, JWT, CSRF, data isolation, XSS, CORS, admin endpoints                                               | `penetration-tester`        | `.claude/agents/penetration-tester.md`        |
| Code review — correctness, security, pattern violations                                                                       | `code-reviewer`             | `.claude/agents/code-reviewer.md`             |
| Documentation sync — README.md, SETUP.md, docs/architecture.md, ROADMAP.md, CHANGELOG.md, SECURITY.md, AGENTS.md, agent files | `docs-updater`              | `.claude/agents/docs-updater.md`              |

`.claude/agents/` and `.agents/skills/` are the **source of truth** — the former is
the Claude Code convention; the latter is where third-party skill installers write
(and where a human edits a skill by hand), so installing or updating a skill stays a
one-way operation with no manual copying. The generated mirrors — `.codex/agents/*.toml`
and `.claude/skills/**` — are produced from them by `scripts/sync-agent-configs.mjs`
(via `npm run sync:agents`) and committed. The mirror covers a skill's entire
directory, not just `SKILL.md` — reference docs, `scripts/*.sh`, `agents/*.yaml`, all
of it. Never edit a generated file: change the authored source and re-run the sync.
`format-check` fails on drift.

**Never symlink `.claude/skills/<name>` to `.agents/skills/<name>` (or vice versa)
instead of running the sync.** This repo has `git config core.symlinks` set to
`false`, so `git add` on a symlinked directory silently stages the real file
contents instead of a link — a commit would duplicate every file rather than
recording one. A directory-walking generator also can't see into a symlink
(`entry.isDirectory()` is false for a symlink), so it would treat the linked-to
skill as untracked source and, in write mode, delete the mirrored copy as an
orphan. Both failure modes have happened in this repo; regenerate instead.

Codex has no per-agent tool allowlist, so the `tools:` and `model:` frontmatter keys
are dropped in the generated TOML rather than translated.

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`jwh3times/GuardianTracker`), managed via the `gh`
CLI. See `docs/agents/issue-tracker.md` for `gh` conventions and
[Work Tracking](#work-tracking) for what belongs on the board.

### Triage labels

Default canonical vocabulary (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`), unchanged. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout: root `CONTEXT.md` (created lazily) + existing `docs/adr/`. See `docs/agents/domain.md`.

## Known Limitations

- Collections data remains membership-wide. The character switcher scopes
  authenticated weekly vendor inventory, today actions, availability ranking,
  and Xûr location to the selected character; deeper character surfaces remain P2.
- Search index snapshots persist beside the manifest by version; a missing or new-version snapshot rebuilds automatically (~30s after the manifest is ready). A build that fails is retried by the next search request (throttled to one attempt per 30s) instead of waiting for the next hourly manifest swap.
- Xûr location is best-effort: the authenticated character-vendor component's
  location index resolves through the manifest to "The Tower"; failures omit the field.
- Raid and dungeon milestones carry a real missing count; non-raid/dungeon milestones
  omit it because verified current reward definitions contain no collectible-linked items.
