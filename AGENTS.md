# AGENTS.md

This is the canonical operating guide for AI coding agents working in Guardian Tracker.
It is tool-neutral: agents that read `AGENTS.md` natively (Codex and others) get what every
session needs from this file, and each section names the `docs/agents/` file to read before a
given kind of work. Codex reads only the first 32 KiB of this file, so reference detail lives
in those files, not here.

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

## Development flow

<!-- shared:development-flow -->

The flow skills are this repository's binding process for changing code. Read
`.agents/skills/open-pr/settings/dev-flow.md` (the routing, exemptions and issue linking) and
`docs/agents/dev-flow.md` (what this repository adds) before the first edit of a change, and whenever
no ready issue exists. `/ask-matt` maps the flow itself.

- **The gate.** Every code edit starts from a ready issue: an open `ready-for-agent` issue whose body
  is a `/to-spec` spec or a `/to-tickets` ticket or that carries a `/triage` brief, or a `bug` issue
  worked through `/diagnosing-bugs`, which needs the number and no spec. With none, stop before the
  first edit and route the user. Reading code, running tests and answering questions need no issue.
- **Private work.** A ready issue in the private tracker counts. Its number stays out of the branch,
  the PR, commits and public issues, and it is closed by hand after the merge.
- **The human starts every phase.** `/wayfinder`, `/grill-with-docs`, `/to-spec`, `/to-tickets`,
  `/implement` and `/triage` stay user-invoked. At a phase boundary, name the next command, say in
  one line why, and wait. Inside a phase, use `/tdd`, `/code-review`, `/diagnosing-bugs`,
  `/domain-modeling`, `/codebase-design`, `/prototype` and `/research` whenever they fit.
- **Exempt from the gate:** dependency bumps, changes touching only docs and agent config, release
  prep, and an explicit user waiver recorded in the PR body as `Flow waived by the user: <reason>`.
- **Linking.** Branches are `<type>/<issue#>-<slug>`; the PR body opens with `Closes #N` or
  `Part of #N`.
- **The flow ends at `/open-pr`.** `/implement` stops at a commit; `/open-pr` refreshes the docs,
  writes the changelog and the PR body, and opens the PR (Claude Code's built-in `/ship` is a
  different command, outside this flow). When the user says to merge it, `/land` merges it
  and confirms its release. `/dependabot` reviews the open Dependabot PRs and hands the ones the
  user names to `/land`; nothing merges a Dependabot PR on its own. `/ask-matt` also routes to
  `/pr`, `/retro` and `/implement-spec`, which are not installed here.
- **Tests are proven, reds are read.** Before `/implement` commits a change that adds or changes
  tests, and in `/code-review`, run `/test-prover`: in a scratch worktree it reverts the fix and
  breaks its guards, and every new test must go red. `/open-pr` runs it too. `/land` and
  `/dependabot` read a red check with `/ci-triage`, which never re-runs anything.
  `/records-audit` checks the board, the trackers, the follow-up wiki and the changelog against
  each other; run it weekly, and `/what-next` says when it is due.
- **`/handoff` is the shared skill.** It parks the session for the next one, on either machine or
  harness, closes it out with `/end-session`, and `/lets-go` resumes it. It does not write the
  portable file `/ask-matt` describes, so a `/prototype` detour stays in the session or uses a
  branch.
- **A wizard is not the record.** A step only a human can take follows the `## Procedure` in
  `.agents/skills/open-pr/settings/human-follow-up.md`, whose instructions stand without the script.
- **Research goes where it is safe to keep.** `/research` findings that would expose private material
  go to the private companion, where there is one, never to an ignored folder with no repository
  behind it. Details of an unpublished vulnerability are a draft advisory, never a research file.

<!-- /shared:development-flow -->

## Architecture

Two services: a Go API backend and a React frontend that calls it directly over REST.

```text
Frontend (React/TS :5273)
    └─► API Service (Go/Gin :8081)  — OAuth, JWT, manifest, collections
```

For the full port map — Docker Compose, Kubernetes, dev/cross-service wiring — see **[Ports in SETUP.md](./SETUP.md#ports)**.

### Key directories

- `backend/api-service/` — Go API: `api/router.go` (the route table; `main.go` is a composition root and registers no routes), `api/handlers/` (Gin handlers), `auth/` (JWT issue/verify, middleware, HMAC-signed OAuth state, roles, revocation, encrypted token store, `SessionIssuer` owning login/reconnect/refresh/logout), `db/` (Postgres stores + embedded migrations, audit log, users/roles/flags/wishlist/roll targets/prefs/digest; `Stores` fields are interfaces backed by degraded implementations — never nil — when there is no database; `db/adapters/` translates stores into consumer-side interfaces, including the membership-keyed Preferences, Wish list, Roll targets, and Digest repositories), `services/` (bungie client, manifest, manifeststate, membershipstate, sharedread, collections, records, weekly, recommendations, search, items, characters, efficiency, sources, preferences, wishlist, rolltargets, ownedrolls, digest), `config/`, `cache/`.
- `frontend/src/` — React app: `features/` (pages), `components/`, `contexts/` (AuthContext, FlagsContext), `data/` (ADR 0020 data-access modules, one per domain resource — owns query identity, projection, mutations, and invalidation; ADR 0020's eleven resources have all landed — Wish list, Collections, Weekly, Characters, Items, Search, Flags, Admin, Catalysts, Crafting, and Seals — and Digest and Roll targets joined later, for thirteen — plus a `membershipRefresh` module owning the cross-resource cache-refresh fan-out over the eight membership-scoped ones (Catalysts, Characters, Collections, Crafting, Digest, Roll targets, Seals, Weekly) and a completeness test that fails if a membership-scoped module skips it, and `preferences.ts`, ADR 0021's framework-neutral membership-keyed preferences client, which deliberately uses no React Query and is that fan-out's one named exemption; `features/**` and `components/**` are barred by an oxlint `no-restricted-imports` zone from importing `@tanstack/react-query` or `types/api`, so only `data/` may), `lib/`, `types/`.
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

Bungie OAuth login uses a public client with stateless, HMAC-signed v2 CSRF
`state` bound to an independent HttpOnly browser transaction cookie.
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

Before changing auth, sessions, cookies, origins, roles, membership-scoped routes or request
logging, read [docs/agents/security-and-logging.md](./docs/agents/security-and-logging.md): the
behavior it lists must be preserved.

## Running Services

- **Docker Compose** (Option A) — one command for the full stack; best default for local dev and integration testing.
- **Minikube** (Option B) — validates Kubernetes manifests; dev-validation only (no Postgres).
- **Individual services** (Option C) — fast hot reload during active single-service development.

`SETUP.md` is the human-facing setup guide and remains the fuller reference. The
agent-facing subset — ports and test commands — is duplicated inline below on
purpose; the environment table is in `docs/agents/environment.md`.

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
Before changing or debugging `k8s/startup.ps1`, read [k8s/README.md](./k8s/README.md#start-and-stop)
(what it builds and restarts, and when it stops) and its
[troubleshooting](./k8s/README.md#troubleshooting).

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

**Never commit real secrets. Use `.env` locally and keep generated/private files out of git.**
Do not put the private repository location or real 1Password identifiers in public docs.

Before restoring the private workspace or secrets, running `npm run sync:main`, adding or
reading a runtime environment variable, or moving the Node pin, read
[docs/agents/environment.md](./docs/agents/environment.md).

## CI/CD

GitHub Actions runs five required jobs in `.github/workflows/ci-cd.yml` (`Format Check`,
`Test Frontend`, `Test Go Services`, `Build Docker Images`, `Changelog Version`) and the required
`Browser E2E + Axe` in `.github/workflows/browser.yml`; `version.yml` tags every merge to `main`.
Before changing a workflow, a policy test, the ruleset or versioning, or when a required check
fails and you need to know what it runs, read [docs/agents/ci.md](./docs/agents/ci.md).

## Testing and Validation

Use the narrowest relevant test first, then run broader checks when the change crosses module boundaries.

```powershell
# Windows workspace portability (from the repo root)
npm run test:workspace-portability

# Docker context isolation (repo root; Docker Buildx + running builder required)
npm run test:docker-context

# nginx response headers and caching (repo root; running Docker required)
npm run test:nginx-headers

# Repository policy tests and agent-config checks (repo root; Format Check runs all three)
npm run test:policy
npm run sync:agents:check
npm run lint:agents

# Go (from backend/api-service/)
go test ./...
GOTOOLCHAIN=go1.27.4 go run honnef.co/go/tools/cmd/staticcheck@2026.2.2 ./...   # pin required; see docs/agents/local-runs.md
go tool govulncheck ./...
./test-local.ps1          # full CI-equivalent: cgo + Postgres (see go-services agent for flags)

# Frontend (from frontend/)
npm run type-check
npm run lint
npm test -- --coverage
npm run build

# Browser (start from repo root, then run scripts from frontend/)
$env:GO_ENV="development"                  # only in a worktree with no root .env
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

Before running Staticcheck or measuring Go coverage locally, or when a Node,
`@playwright/test` or Go bump moves an image pin, read
[docs/agents/local-runs.md](./docs/agents/local-runs.md): it explains the Staticcheck
`GOTOOLCHAIN` pin, why a plain `go test ./...` under-reports coverage, and
`npm run sync:image-pins`.

## Documentation Rules

Before adding, moving, renaming or deleting a doc, or deciding whether something belongs in a
public doc or in `private/`, read the Documentation rules and the Doc map in
[docs/agents/docs.md](./docs/agents/docs.md).

- Keep private audits, exploratory notes, and sensitive operational details in
  `private/`. Execution sequencing and slice order are **not** private documents —
  they live on the board (see [Work Tracking](#work-tracking) and
  [ADR 0022](./docs/adr/0022-github-owns-task-status.md)). Do not recreate a
  private status file.
- Do not move private operational detail into public docs or expose its remote location through
  the public repository.

## Work Tracking

Task status lives on GitHub, not in this repository's files. See
[ADR 0022](./docs/adr/0022-github-owns-task-status.md).

- **Board:** the private user-level Project `Guardian Tracker`
  (<https://github.com/users/jwh3times/projects/4>), linked to both the public
  repository and the private companion. Fields: `Chain`, `Order`, `Blocked By`,
  `Gate`, `ADR`, `Status`, `Triage`, `Priority`, `Due`.
- **Issues go in the public repository by default.** File in the private
  companion only when the body would need a credential, a real
  provider/cost/account identifier, or exploitable security detail.
- **Every board item is an issue** ([ADR 0026](./docs/adr/0026-every-board-item-is-an-issue.md)):
  new work is filed as an issue and added to the board, with no draft items.
  Readiness is the triage label; running `/to-spec`, `/to-tickets`, or
  `/wayfinder` is still what makes work `ready-for-agent`
  ([ADR 0024](./docs/adr/0024-committing-to-work-creates-the-issue.md)). An issue
  waiting on something outside itself also carries `blocked`; a blocker that is
  another issue is a native `blocked_by` edge. Every open item has a `Priority`
  (`P1 — Next` to `P4 — Parked`).
- **The ready frontier is computed, not asserted:** open `ready-for-agent`
  issues without `blocked`, with no open native blocker and no assignee, in
  `Priority` order.
- Record point-in-time findings as comments on the issue they informed. Do not
  create standing status documents; that is the drift ADR 0022 removes.

### Human follow-ups from completed work

Before reporting completed agent work, create or update a private-companion
follow-up issue for every required human action, add it to the shared project,
label it `ready-for-human`, and publish linked step-by-step instructions in the
private wiki. This overrides the public-by-default rule for
these follow-ups. Whenever completed work leaves a human action, read
[the human-follow-up settings](./docs/agents/human-follow-up.md) (destination, label,
board, wiki) and the `## Procedure` they point to, which ships with the shared skills
([`.agents/skills/end-session/settings/human-follow-up.md`](./.agents/skills/end-session/settings/human-follow-up.md)):
it defines deduplication, evidence, privacy, and handoff completion. Engineering tasks
the agent can perform remain agent work.

### Rules for every architecture-deepening slice

Before working one of the remaining behavior-preserving ownership migrations that ADRs
0014–0021 sequence, read
[its rules in docs/agents/dev-flow.md](./docs/agents/dev-flow.md#rules-for-every-architecture-deepening-slice).

## Agent Routing

Specialized agent definitions live in `.claude/agents/`. If your tool supports
subagents, dispatch them; if not, read the file for the relevant area as repo notes —
they are useful either way.

| Task                                                                                                                 | Agent                       | Definition                                    |
| -------------------------------------------------------------------------------------------------------------------- | --------------------------- | --------------------------------------------- |
| Go backend — Gin handlers, JWT/auth, Bungie OAuth, manifest, collections, records, weekly, search, roles/flags/admin | `go-services`               | `.claude/agents/go-services.md`               |
| React frontend — pages, components, design system (`gt-*`), React Query, AuthContext, FlagsContext, Vitest tests     | `react-frontend`            | `.claude/agents/react-frontend.md`            |
| PostgreSQL schema, migrations, token store, audit log; SQLite manifest queries                                       | `postgres-specialist`       | `.claude/agents/postgres-specialist.md`       |
| Kubernetes manifests, Minikube, kubectl, secrets, configmaps                                                         | `kubernetes-infrastructure` | `.claude/agents/kubernetes-infrastructure.md` |
| Dockerfiles, image builds, layer caching                                                                             | `docker-containers`         | `.claude/agents/docker-containers.md`         |
| Security testing — OAuth, JWT, CSRF, data isolation, XSS, CORS, admin endpoints                                      | `penetration-tester`        | `.claude/agents/penetration-tester.md`        |
| Code review — correctness, security, pattern violations                                                              | `code-reviewer`             | `.claude/agents/code-reviewer.md`             |

Documentation sync is not a subagent: it is the `docs-updater` skill (see
[Agent skills](#agent-skills)), which `/open-pr` and `/end-session` run in a subagent.

`.claude/agents/` and `.agents/skills/` are the **source of truth** — the former is
the Claude Code convention; the latter is where skill installers write (and where a
skill written by hand would live), so installing or updating a skill stays a
one-way operation with no manual copying. The generated mirrors — `.codex/agents/*.toml`
and `.claude/skills/**` — are produced from them by `scripts/sync-agents.mjs`
(via `npm run sync:agents`) and committed. The mirror covers a skill's entire
directory, not just `SKILL.md` — reference docs, `scripts/*.sh`, `agents/*.yaml`, all
of it. Never edit a generated file: change the authored source and re-run the sync.
`format-check` fails on drift.

**Never symlink `.claude/skills/<name>` to `.agents/skills/<name>` (or vice versa)
instead of running the sync.** Git for Windows sets the per-clone `core.symlinks` to
`false`, and on such a clone `git add` on a symlinked directory silently stages the real file
contents instead of a link — a commit would duplicate every file rather than
recording one. A link would also let a write to the mirror land in the authored
source. So `npm run sync:agents` removes any link it finds under `.codex/agents/` or
`.claude/skills/` (`npx skills update` creates exactly that link) before it writes
the real copies, and `npm run sync:agents:check` reports a link as orphaned;
the authored trees may hold no links at all. Regenerate instead.

Codex has no per-agent tool allowlist, so the `tools:`, `model:`, and `color:`
frontmatter keys are dropped in the generated TOML rather than translated. The one
fact that survives is read-only access: an agent whose `tools:` list names no
file-writing tool (`Write`, `Edit`, `MultiEdit`, `NotebookEdit`) gets
`sandbox_mode = "read-only"`.

## Agent skills

### Shared skills

<!-- shared:shared-skills -->

`end-session`, `handoff`, `lets-go`, `open-pr`, `land`, `dependabot`, `docs-updater`, `what-next`,
`test-prover`, `ci-triage`, `records-audit` and `tldr` install from `jwh3times/skills`, and are
never edited here: reinstalling overwrites a local edit.
A fix goes upstream, and what they need to know about this repository goes in the settings files
under `docs/agents/`, which each skill proposes on first run.

The other shared skills come from one `mattpocock/skills` snapshot (v1.3.2, pinned by commit), the
same in every repository, and are never edited here either: what they need to know goes in this
file or `docs/agents/`. Moving to a newer snapshot is a change in `jwh3times/skills`. Keep
`.agents/skills/` out of every formatter: `scripts/check-skills.mjs` fails when an installed copy
differs from `skills-lock.json`, or the lock from these commands.

Re-running the install is the update; never run `npx skills update`, which leaves `.claude/skills/`
linked to its source instead of copied:

```bash
npx skills add jwh3times/skills -s end-session -s handoff -s lets-go -s open-pr -s land -s dependabot -s docs-updater -s what-next -s test-prover -s ci-triage -s records-audit -s tldr -a codex --copy -y
npx skills add mattpocock/skills#d1478fcb140db0b93c0c8df17bcfd311844749d0 \
  -s ask-matt -s code-review -s codebase-design -s diagnosing-bugs -s domain-modeling -s grill-me \
  -s grill-with-docs -s grilling -s implement -s improve-codebase-architecture -s prototype \
  -s research -s setup-matt-pocock-skills -s tdd -s teach -s to-questionnaire -s to-spec \
  -s to-tickets -s triage -s wait-what -s wayfinder -s wizard -s writing-for-agents \
  -a codex --copy -y
node scripts/sync-agents.mjs
```

`scripts/sync-agents.mjs`, `lint-agent-neutrality.mjs`, `check-docs.mjs`, `check-skills.mjs`,
`guard.mjs`, and `sync-main.mjs`, `next-version.mjs`, `check-changelog.mjs` and
`stamp-changelog.mjs` where this repository carries them, are canonical in `jwh3times/skills` with
their tests: fix them there, then copy them here.
`sync-agents.mjs` also writes the marked blocks in this file: change them in `jwh3times/skills`, not
here. It registers `guard.mjs` in `.claude/settings.json`, which stops the commands and edits these
rules forbid; this repository's own additions go in `docs/agents/guard.md`.

<!-- /shared:shared-skills -->

The decision is [ADR 0025](./docs/adr/0025-shared-skills-install-from-jwh3times-skills.md).

`.prettierignore` excludes `.agents/skills/` and the generated `.claude/skills/`,
so the root Markdown check never reformats installed bytes. The sections below
configure the `mattpocock/skills` set.

### Issue tracker

Issues live in GitHub Issues (`jwh3times/GuardianTracker`), managed via the `gh`
CLI. See `docs/agents/issue-tracker.md` for `gh` conventions and
[Work Tracking](#work-tracking) for what belongs on the board.

### Triage labels

Default canonical vocabulary (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`), unchanged. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout: root `CONTEXT.md` + existing `docs/adr/`. See `docs/agents/domain.md`.

## MCP servers

The shared set is `context7` and `playwright`, declared in `.mcp.json` and `.codex/config.toml`.
Before using one for the first time, or changing either file or the Playwright pin, read
[docs/agents/mcp.md](./docs/agents/mcp.md).

GitHub work goes through `gh`, which the flow skills already require; a GitHub MCP server
is deliberately not in the set. Personal servers in your user-level config are fine
alongside the shared ones, provided git and GitHub operations still go through `git` and
`gh`. Secrets never pass through an MCP server: credential restoration uses the scripts
in [docs/agents/environment.md](./docs/agents/environment.md).

## Known Limitations

Before working on character scoping, per-Guardian progression, the search index, Xûr location
or milestone missing counts, or treating a gap there as a bug, read Known Limitations in
[docs/architecture.md](./docs/architecture.md#known-limitations).
