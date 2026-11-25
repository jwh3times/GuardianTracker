# Docs

## Doc map

- `AGENTS.md` — the canonical, tool-neutral operating guide, holding what every session needs:
  overview, ground rules, the development flow, architecture and key directories,
  acquisition-source invariants, auth and token flow, ports and the essential commands, work
  tracking, agent routing, agent skills and the must-never rules. Reference detail lives in the
  `docs/agents/` files it points to, each pointer saying when to read it. Codex reads only the
  first 32 KiB of `AGENTS.md`, so keep it under 30,000 bytes (`wc -c AGENTS.md`): new reference
  detail goes to a `docs/agents/` file and a one-line pointer, not into `AGENTS.md`. Never move it
  into a nested `AGENTS.md`, which Codex reads into the same budget. **When:**
  - auth or token behavior changes (`backend/api-service/auth/`): Auth & token flow;
  - ports or Compose change (`docker-compose.yml`, `k8s/`): `## Running Services`;
  - a service, data-access module or key directory is added (`backend/api-service/services/`,
    `frontend/src/data/`): Key directories;
  - an agent or skill is added, removed or replaced (`.claude/agents/`, `.agents/skills/`,
    `skills-lock.json`): Agent Routing and Agent skills.
- `docs/agents/environment.md` — the private workspace and `main` sync helpers, the Node pin, and
  the runtime values table. **When:** an env var (`backend/api-service/config/config.go`), the
  `.nvmrc` pin, or a bootstrap, restore, status or sync helper changes.
- `docs/agents/security-and-logging.md` — the auth and security behavior to preserve, and what
  application logs record. **When:** an endpoint or auth/security behavior changes
  (`backend/api-service/api/router.go`, `backend/api-service/auth/`), or request logging changes.
- `docs/agents/ci.md` — CI/CD job names, the policy tests, coverage gates, action and Go pins,
  required checks, branch protection, versioning. **When:** CI or the ruleset changes
  (`.github/workflows/`), or a policy test is added to `test:policy`.
- `docs/agents/mcp.md` — the shared MCP servers, enabling them, and their pin. **When:** a shared
  MCP server or its pin changes (`.mcp.json`, `.codex/config.toml`, which must declare the same
  servers).
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
  security posture, known limitations. **When:** a route group's surface (`api/router.go`), the
  data model (`backend/api-service/db/migrations/`), or the runtime/infrastructure shape changes;
  Known Limitations only once the code confirms a limitation is resolved.
- `CONTEXT.md` — the domain glossary, written through `domain-modeling`. `frozen`: report a term the
  code contradicts instead of editing it.
- `docs/adr/*.md` — accepted decisions. `frozen`: report a contradiction between an accepted ADR
  and the code instead of editing either. An ADR's `Status` line and its index row move to
  "implemented in `vX.Y.Z`" in the change that ships its final slice, not here.
- `docs/adr/README.md` — the ADR index and status vocabulary. **When:** an ADR is added.
- `ROADMAP.md` — public work not yet shipped, with gates and likely size. **When:** a branch
  implements a roadmap item (remove or revise it), or a gate changes.
- `CHANGELOG.md` — Keep a Changelog, one dated section per merge, compare-link footer, stamped by
  `version.yml` after each merge. `frozen`: a branch edits only the policy text at the top.
- `changelog.d/*.md` — the branch's changelog fragment, one per branch (`changelog.d/README.md`).
  `ship-owned`.
- `VERSION` — the version file `open-pr` bumps. `ship-owned`.
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
- `THIRD_PARTY_NOTICES.md` — the license notices of third-party files kept here. `owner-only`: report
  the change when a skill's source changes.
- `frontend/README.md` — frontend structure, scripts, and the browser and visual-test procedure.
  **When:** `frontend/package.json` scripts, `frontend/playwright.config.ts`, or the
  `frontend/src/` layout changes.
- `frontend/design/README.md` — the historical design prototype, kept as design history. `frozen`.
- `frontend/design/icons/README.md` — the icon pack: its colors, files and sizes. **When:** an icon
  file in `frontend/public/` is added, removed or regenerated.
- `k8s/README.md` — the Minikube validation workflow. **When:** `k8s/*.yaml`, the startup/shutdown
  scripts, or a prerequisite changes.
- `docs/maintainers/workspace-recovery.md` — the value-free private-workspace recovery runbook.
  **When:** the public recovery contract changes (`scripts/bootstrap-private*`,
  `scripts/restore-private-secrets.ps1`, `scripts/workspace-status.ps1`, `scripts/sync-main.mjs`).
- `docs/agents/*.md` — conventions and settings the skills read; `dev-flow.md` holds what this
  repository adds to the shared flow. **When:** a convention, route or setting changes.
- `docs/agents/guard.md` — the commands and paths `scripts/guard.mjs` stops here, on top of its
  shared rules. **When:** a **Never run** command, a dev-stack command or a never-remove path in
  `workspace.md` changes.
- `docs/agents/merging.md`, `docs/agents/verification.md`, `docs/agents/local-runs.md` — from open
  PR to finished merge, including Dependabot's red-on-arrival bumps; proving a card's premise, a
  test or a gate; the development machine's traps. **When:** a lesson is learned that no script or
  policy test enforces, or a command, check or pin they quote changes.
- `docs/agents/dependabot.md` — what a Dependabot review needs here: the bumps reviewed in full,
  CI's install, where each dependency is used, and the known fix for each bump that arrives red.
  **When:** `.github/dependabot.yml`, a pin a policy test ties together,
  `scripts/sync-image-pins.mjs` or a Dependabot fix changes.
- `.claude/agents/*.md` — area notes for the subagents, mirrored to `.codex/agents/`. **When:**
  `go-services.md`: endpoints, JWT, OAuth, manifest flow; `react-frontend.md`: structure, query
  patterns, pages; `postgres-specialist.md`: schema and migrations; `penetration-tester.md`: the
  attack surface; `code-reviewer.md`: review rules; `docker-containers.md`: Dockerfiles and the
  build-context policy; `kubernetes-infrastructure.md`: only after a confirmed manifest or script
  change, never from local cluster state.
- `.agents/skills/*/**` — skills, mirrored to `.claude/skills/`. The `mattpocock/skills` set in
  `skills-lock.json`, and the skills from `jwh3times/skills` that the `shared-skills` block in
  `AGENTS.md` installs (ADR 0025): `frozen` here, fixed upstream. **When:** a skill is added,
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

## Documentation rules

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
- `AGENTS.md` - canonical, tool-neutral agent operating guide
- `CLAUDE.md` - thin `@AGENTS.md` importer plus Claude Code-specific mechanics
- `docs/agents/` - per-repo conventions and settings the installed skills read
  (issue tracker, triage labels, domain docs, development flow, and the shared
  skills' settings files); see `AGENTS.md`'s `## Agent skills`
- `docs/agents/dev-flow.md` - what this repository adds to the shared development
  flow: board items, what to do before implementing, the rules for every
  architecture-deepening slice, and exemption details
- `docs/agents/merging.md` - once a PR is open: whether CI's answer is for your
  commit, reds that aren't the code, merging, and Dependabot's red-on-arrival bumps
- `docs/agents/verification.md` - before trusting a card's premise, a test or a gate
- `docs/agents/local-runs.md` - before handing a tree to a reviewer, or running the
  stack and browser tests on a development machine
- `docs/maintainers/workspace-recovery.md` - value-free private-workspace and
  machine-local configuration recovery runbook for authorized maintainers

Private docs:

- `private/README.md` - sole index for operations, residual
  security risks, reference evidence, and retired work. Private paths may be
  reorganized without duplicating that internal layout in this public guide.

Rules:

- Keep public docs factual and safe for a public repository.
- Archive shipped private planning content instead of leaving duplicate active plans.
- When deleting or consolidating private docs, scrub references to the removed file.

Public committed docs describe implemented behavior, local setup, durable decisions, security
model, shipped changes, and gated future work. `private/` is gitignored and, for authorized
maintainers, can be an independent private documentation repository. It holds
deployment runbooks, private security reviews, raw Bungie/API research, and
environment-specific operations notes. Public contributors do not need it.

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
- Go and Node pins: the `toolchain` (else `go`) line in `backend/api-service/go.mod`; `.nvmrc`
- CI jobs: the keys under `jobs:` in `.github/workflows/ci-cd.yml` and `browser.yml`, then each
  one's `name:` — never a bare `name:` grep, which counts steps too
- Policy tests: `test:policy` in `package.json`
- npm scripts: `scripts` in `package.json` and `frontend/package.json`
- Required status checks: `gh api repos/jwh3times/GuardianTracker/rules/branches/main`

## Docs validation

From the repository root:

- `node scripts/check-docs.mjs` — local links and anchors, and `Doc map` coverage
- `./frontend/node_modules/.bin/prettier --check "**/*.md"` (fix with `--write`) — the frontend
  run cannot reach Markdown outside `frontend/`
- `node --test scripts/go-toolchain-policy.test.mjs scripts/postgres-pin-policy.test.mjs` — the
  policies that read `AGENTS.md`, `SETUP.md` and the agent files
