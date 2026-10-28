# Docs

## Doc map

- `AGENTS.md` — the canonical, self-contained, tool-neutral operating guide: architecture and key
  directories, acquisition-source invariants, auth and token flow, the inline ports and
  environment table, CI/CD, branch protection, test commands, documentation rules, work tracking,
  agent routing, known limitations. It is the only home of repo operating context. **When:**
  - an endpoint or auth/security behavior changes (`backend/api-service/api/router.go`,
    `backend/api-service/auth/`): Auth & token flow, "Auth and security behavior to preserve";
  - env, ports or Compose change (`backend/api-service/config/config.go`, `docker-compose.yml`,
    `k8s/`): `## Running Services` and `## Environment Setup` — the only copy tool-neutral agents
    read;
  - CI or the ruleset changes (`.github/workflows/`): `## CI/CD` — job names, the policy tests,
    coverage gates, action and Go pins, required checks, versioning;
  - a service, data-access module or key directory is added (`backend/api-service/services/`,
    `frontend/src/data/`): Key directories;
  - an agent or skill is added, removed or replaced (`.claude/agents/`, `.agents/skills/`,
    `skills-lock.json`): Agent Routing and Agent skills;
  - a known limitation is resolved — only once the code confirms it.
- `CLAUDE.md` — a thin `@AGENTS.md` importer plus Claude Code-only mechanics. **When:** only the
  subagent roster, skills or hooks mechanics change; never operating context.
- `README.md` — overview, feature summary, quick start, doc index. **When:** a top-level feature
  (`frontend/src/features/`), a quick-start command, or the doc index changes.
- `SETUP.md` — local setup, env files, ports, Compose, Minikube, test commands, troubleshooting.
  **When:** a prerequisite, command, port, env var, migration command or deployment behavior
  actually changes; never otherwise. Its Staticcheck `GOTOOLCHAIN` and PostgreSQL wording are held
  by `scripts/go-toolchain-policy.test.mjs` and `scripts/postgres-pin-policy.test.mjs`.
- `docs/README.md` — the documentation map and the public/private boundary. **When:** a document
  is added, removed, renamed, or changes owner.
- `docs/product.md` — durable product goals, users and principles; never status. **When:** a
  durable product decision changes.
- `docs/architecture.md` — implemented runtime shape, data flow, route groups, data model,
  security posture. **When:** a route group's surface (`api/router.go`), the data model
  (`backend/api-service/db/migrations/`), or the runtime/infrastructure shape changes.
- `docs/adr/*.md` — accepted decisions. `frozen`: report a contradiction between an accepted ADR
  and the code instead of editing either. An ADR's `Status` line and its index row move to
  "implemented in `vX.Y.Z`" in the change that ships its final slice, not here.
- `docs/adr/README.md` — the ADR index and status vocabulary. **When:** an ADR is added.
- `ROADMAP.md` — public work not yet shipped, with gates and likely size. **When:** a branch
  implements a roadmap item (remove or revise it), or a gate changes.
- `CHANGELOG.md` — Keep a Changelog, one dated section per merge, compare-link footer.
  `ship-owned`.
- `docs/changelog/*.md` — release notes archived from `CHANGELOG.md`. `frozen`.
- `SECURITY.md` — reporting process, controls, security model, production checklist, credential
  rotation. **When:** auth, role, flag, admin, audit, cookie/origin or other security behavior
  changes.
- `CONTRIBUTING.md` — contribution workflow, branch and PR rules, formatting, CI gates,
  versioning. **When:** the development flow, a formatter, a required check or a coverage gate
  changes.
- `SUPPORT.md` — public support and issue-reporting guidance. **When:** the issue templates or
  support route change.
- `CODE_OF_CONDUCT.md`, `LICENSE` — `owner-only`.
- `frontend/README.md` — frontend structure, scripts, and the browser and visual-test procedure.
  **When:** `frontend/package.json` scripts, `frontend/playwright.config.ts`, or the
  `frontend/src/` layout changes.
- `k8s/README.md` — the Minikube validation workflow. **When:** `k8s/*.yaml`, the startup/shutdown
  scripts, or a prerequisite changes.
- `docs/maintainers/workspace-recovery.md` — the value-free private-workspace recovery runbook.
  **When:** the public recovery contract changes (`scripts/bootstrap-private*`,
  `scripts/restore-private-secrets.ps1`, `scripts/workspace-status.ps1`, `scripts/sync-main.mjs`).
- `docs/agents/*.md` — conventions and settings the skills read; `dev-flow.md` holds the gate,
  exemptions and linking rules. **When:** a convention, route or setting changes.
- `.claude/agents/*.md` — area notes for the subagents, mirrored to `.codex/agents/`. **When:**
  `go-services.md`: endpoints, JWT, OAuth, manifest flow; `react-frontend.md`: structure, query
  patterns, pages; `postgres-specialist.md`: schema and migrations; `penetration-tester.md`: the
  attack surface; `code-reviewer.md`: review rules; `docker-containers.md`: Dockerfiles and the
  build-context policy; `kubernetes-infrastructure.md`: only after a confirmed manifest or script
  change, never from local cluster state.
- `.agents/skills/*/**` — skills, mirrored to `.claude/skills/`. The `mattpocock/skills` set in
  `skills-lock.json`, and `end-session`, `handoff`, `lets-go`, `ship` and `docs-updater` from
  `jwh3times/skills` (ADR 0025): `frozen` here, fixed upstream. **When:** a skill is added,
  removed or replaced: AGENTS.md's Agent skills section and `CLAUDE.md`'s skills paragraph.
- `.github/PULL_REQUEST_TEMPLATE.md` — the PR body's sections. **When:** the development flow's
  linking rule changes.
- `private/README.md` — the companion's sole index: every private file classified as living,
  reference snapshot or historical evidence, and its `**Updated:**` line. **When:** a private file
  is added, finished, moved or retired.
- `private/**/*.md` — durable private material, laid out as `private/README.md` says: deployment
  and infrastructure decisions, the accepted residual-risk register, Bungie API research,
  bootstrap procedures, an archive of shipped private outcomes; never status (ADR 0022).
  **When:** shipped or retired private work moves to the archive; a production-infrastructure
  decision is deferred, made or regated; a residual risk is accepted or resolved (auth,
  authorization, cookie/origin, audit, deployment topology, revocation cache); a Bungie fact is
  verified. Dated evidence under `private/evidence/` is `amend-only`.

**Never in public docs:** the companion's location, issue and wiki links, and real 1Password
identifiers; deployment runbooks, cloud resource names and incident notes; private security
reviews, exploit-level analysis and unpublished vulnerability details; oversized or operational
Bungie API research; audit evidence and retired planning content; credential-handling runbooks
and rotation records; credential values; any backlog, next action or status document (GitHub owns
task status, ADR 0022).

## Drift probes

- API endpoints: `(GET|POST|PUT|DELETE|PATCH)\(` in `backend/api-service/api/router.go`
- JWT claims: `JWTClaims|json:"` in `backend/api-service/auth/jwt.go`
- Env vars: `get[A-Za-z0-9]*Env\(|getJWTAccessTTL|Getenv` in
  `backend/api-service/config/config.go` — every `get*Env` helper
- Base images: `^FROM` in the Dockerfiles; `image:` in `docker-compose.yml` and the workflows
- Ports: `ports:` in `docker-compose.yml`
- K8s topology: tracked `k8s/*.yaml`, `k8s/startup.ps1`, `k8s/shutdown.ps1`,
  `k8s/api-service-secret.yaml.example` — never the ignored local Secret
- Frontend features: `frontend/src/features/**/*.tsx`
- Frontend data-access modules: `frontend/src/data/*.ts`
- Frontend queries: `apiFetch|useQuery|useMutation` in `frontend/src`
- Migrations: `backend/api-service/db/migrations/*.sql`
- Coverage gates: `thresholds` in `frontend/vite.config.ts`; the `Enforce Go coverage gate` step
  in `.github/workflows/ci-cd.yml`
- Go and Node pins: `GO_VERSION` in both workflows; `.nvmrc`
- CI jobs: the keys under `jobs:` in `.github/workflows/ci-cd.yml` and `browser.yml`, then each
  one's `name:` — never a bare `name:` grep, which counts steps too
- Policy tests: `test:policy` in `package.json`
- npm scripts: `scripts` in `package.json` and `frontend/package.json`
- Required status checks: `gh api repos/jwh3times/GuardianTracker/rules/branches/main`

## Docs validation

From the repository root:

- `./frontend/node_modules/.bin/prettier --check "**/*.md"` (fix with `--write`) — the frontend
  run cannot reach Markdown outside `frontend/`
- `npm run test:docs-links` — local links and heading anchors in every Markdown file
- `node --test scripts/go-toolchain-policy.test.mjs scripts/postgres-pin-policy.test.mjs` — the
  policies that read `AGENTS.md`, `SETUP.md` and the agent files
