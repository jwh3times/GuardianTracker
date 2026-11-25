# Dependabot

What reviewing a Dependabot pull request takes here, read by the `dependabot` skill in a session
and in the scheduled review. These facts apply to this repository only. What to do once a bot PR
is open (pushing a fix onto its branch, never `@dependabot rebase`) is in
[merging](merging.md#dependency-pull-requests).

## Stack

One Go service, `backend/api-service` (Gin, pgx, golang-jwt, go-sqlite3 with cgo), and one npm
frontend, `frontend/` (Vite, React, TypeScript, Vitest, oxlint, Tailwind, Playwright). The root
`package.json` holds scripts only and has no lockfile. `.github/dependabot.yml` updates five
ecosystems: `gomod` in `/backend/api-service`, `npm` in `/frontend`, `github-actions`, `docker`
(the three frontend Dockerfiles and `backend/api-service/Dockerfile`) and `docker-compose`
(`docker-compose.yml`: postgres and pgadmin). Each groups its minor and patch bumps into one PR.
In npm, the majors of four peer-locked families arrive as one PR each: `react`, `vite-ecosystem`,
`testing-library` and `oxlint-and-typescript`.

Workflows: `.github/workflows/ci-cd.yml` (CI), `browser.yml` (Browser E2E + Axe, Browser Visual
Regression) and `version.yml` (the release after a merge).

## Review triggers

- **`postgres` in `docker-compose.yml`, any tag change** (Dependabot calls 18.6 to 18.7 a minor).
  The `docker-compose` ecosystem does not see the `Test Go Services` service container in
  `ci-cd.yml`, so `scripts/postgres-pin-policy.test.mjs` fails `Format Check` until it is
  realigned. The same test fails on any retired PostgreSQL tag left in a tracked text file outside
  the release history (`CHANGELOG.md`, `changelog.d/`, `docs/changelog/`). A digest-only bump on
  the same tag passes.
- **`@playwright/test`, any bump.** The Playwright image pin in `frontend/Dockerfile.playwright`
  is in the docker ecosystem, so the npm PR cannot move it: `Format Check` and
  `Browser Visual Regression` go red.
- **`node` in the frontend Dockerfiles, any bump, a digest too.** `.nvmrc` owns the Node patch,
  and Dependabot edits only `frontend/Dockerfile` and `frontend/Dockerfile.dev`.

## Resolution

- Node: the version in the root `.nvmrc` (CI's `setup-node` reads it with `node-version-file`),
  then `( cd frontend && npm ci )`.
- Go: the release `backend/api-service/go.mod` names (CI's `setup-go` reads it with
  `go-version-file`), then
  `( cd backend/api-service && go mod download && go build ./... && go vet ./... )` and
  `( cd backend/api-service && go mod verify )`. CI's `Test Go Services` also runs `go mod tidy`,
  Staticcheck, `go tool govulncheck ./...` and `go test -race`.

## Where to look

- The `with:` inputs in all three workflows.
- PostgreSQL: the three Compose services in `docker-compose.yml` (`postgres` mounts its data
  volume at `/var/lib/postgresql` and `database/init/` as its init directory), the
  `Test Go Services` service image in `ci-cd.yml` (the Debian image, a second pin), and the tag
  quoted in `.claude/agents/postgres-specialist.md` (mirrored to `.codex/agents/`).
- Node: `.nvmrc`, the three frontend Dockerfiles, `engines` in `frontend/package.json` and
  `@types/node`; `scripts/node-version-policy.test.mjs` holds them to one Node 26 patch.
- Go: the import paths under `backend/api-service/`, the `tool` line in `go.mod` (govulncheck),
  the Staticcheck version in `ci-cd.yml`, the `toolchain` line in `go.mod`, and the `golang`
  builder image in `backend/api-service/Dockerfile`, which `npm run sync:image-pins` derives from
  it.
- Frontend config: `frontend/vite.config.ts` (Vitest and its coverage thresholds),
  `frontend/tsconfig.json`, `frontend/.oxlintrc.json`, `frontend/postcss.config.cjs`,
  `frontend/tailwind.config.cjs`, `frontend/playwright.config.ts`.

## Grounding facts

- Nothing deploys from this repository: CI builds the Docker images but does not publish or deploy
  them (`SETUP.md`, "Deployment Notes"), and there is no hosted database here. A runtime image bump
  reaches local Compose, Minikube and CI only.
- The schema comes from this repository's own migration runner: `db.Migrate`
  (`backend/api-service/db/migrate.go`, embedding `db/migrations/*.sql`), called from `main.go` at
  startup, plus `database/init/01-init.sql` on a fresh Compose volume. There is no ORM or
  migration tool to bump.
- A Go module major changes the import path to a new `/vN` suffix, so it is an INHERENT-BREAK case:
  the import paths in code and the `require` in `go.mod` move together. A Go framework or
  infrastructure module major is always NEEDS EFFORT, with the steps grounded in the files that
  import it.
- A PostgreSQL major is an INHERENT-BREAK case: the new major refuses to start on a volume holding
  an older major's cluster. The 18+ images keep each major's cluster in its own subdirectory
  (`PGDATA=/var/lib/postgresql/<major>/docker`), and the `postgres` service mounts the volume's
  root at `/var/lib/postgresql`. Before it initializes an empty `PGDATA`, the image's
  `docker-entrypoint.sh` looks for `PG_VERSION` under `/var/lib/postgresql/*/docker`, and when it
  finds one it exits with "there appears to be PostgreSQL data in: /var/lib/postgresql/18/docker",
  so Compose reports `postgres` unhealthy. That happened on 2026-11-24 after #537 (18 to 19). It
  is always NEEDS EFFORT and needs no external confirmation, with the steps grounded in this
  repository.
- A `golang` builder bump in `backend/api-service/Dockerfile` fails `Format Check` on its own:
  `scripts/go-toolchain-policy.test.mjs` requires the builder tag to equal the release `go.mod`
  names, because CI tests on that release and only `Build Docker Images` compiles with the builder
  image (#544). Dependabot never moves `go.mod`'s `toolchain` line (dependabot-core #13520). Such
  a PR is NEEDS EFFORT with a mechanical fix on its branch: `go mod edit -toolchain=go<X.Y.Z>` in
  `backend/api-service/` with the release it proposed, then `npm run sync:image-pins`, which also
  rewrites the `GOTOOLCHAIN=` pins; then `npm run sync:agents`. The `docker` entry ignores
  `golang` minor and major updates, so these PRs are patches of the current Go minor: Dependabot
  proposes only the highest tag, and a new minor would otherwise hide the next patch. Moving to a
  new Go minor is deliberate, and also needs a Staticcheck release that supports it (#564).
- Every job runs on GitHub-hosted `ubuntu-latest` except the non-required
  `Test Workspace Portability (Windows)` job in `ci-cd.yml`, on `windows-latest`.
- `scripts/node-version-policy.test.mjs` compares versions and tags, never digests, so a
  digest-only `node` bump goes green while it can leave `frontend/Dockerfile.playwright` stale.
- `Changelog Version` passes a PR with no `changelog.d/` fragment, and `version.yml` writes a
  fragment-less merge's release section from its PR title (`scripts/stamp-changelog.mjs`). A bot PR
  needs no changelog edit.

## Playbook

- **`@playwright/test` bump** (in the npm group): run `npm run sync:image-pins` from the repo root.
  It needs network access and no Docker daemon, and rewrites `ARG PLAYWRIGHT_IMAGE` in
  `frontend/Dockerfile.playwright` to the lockfile's version and its current digest. It turns
  `Format Check` (`scripts/node-version-policy.test.mjs`, whose failure reads "Image pin is stale.
  Fix: npm run sync:image-pins") and `Browser Visual Regression` (which failed with
  `browserType.launch: Executable doesn't exist at /ms-playwright/...`) green. Confirm with
  `npm run test:policy`. `Browser E2E + Axe` staying green proves nothing here: it installs
  browsers on the runner. The script can also move the Playwright digest with no version change;
  say so in the PR, since that image renders the visual baselines. Named in the verdicts on #516,
  #519 and #524; the same re-pin landed in commits `4a74422` and `bf7d2f4`.
- **`node` image bump** (in the docker group): set `.nvmrc` to the new patch, run
  `npm run sync:image-pins`, then `node --test scripts/node-version-policy.test.mjs`. On a
  digest-only `node` bump, green or not, run `npm run sync:image-pins -- --check`. Worked on #457
  and #503.
- **`postgres` tag bump** (in the compose group): move the `Test Go Services` service image in
  `ci-cd.yml` to the same `major.minor` (the Debian tag, with its digest), update the tag in
  `.claude/agents/postgres-specialist.md`, run `npm run sync:agents`, then
  `node --test scripts/postgres-pin-policy.test.mjs`. `sync:image-pins` does not own this pin.
  The retired-tag sweep skips the release history, whose released sections quote the tag each
  release ran and which `Changelog Version` does not let a branch edit (#515 was red on those).
- **PostgreSQL major** (postgres X to N): moving the tag does not migrate data. After the merge,
  stop the crash-looping service (`docker compose stop postgres`). Then move the old cluster
  aside, where the entrypoint's `*/docker` check no longer finds it, keeping it intact and
  reversible:
  `docker run --rm -v guardiantracker_postgres-data:/v alpine:3.22 mv /v/<X>/docker /v/<X>/docker.pre-<N>`
  (Git Bash needs `MSYS_NO_PATHCONV=1`). After that, `docker compose up -d` initializes the new
  major's empty cluster (`database/init/01-init.sql`, then `db.Migrate` at api-service startup),
  and the user logs in and re-enters the local data worth keeping. Keeping all the data instead
  means a `pg_dump` from the old major's pinned image, run on the moved cluster with `PGDATA` set
  to it and no ports published, restored before api-service starts. Avoid
  `docker compose down -v`: it works, but it also destroys `manifest-data` and leaves no copy.
  Removing `<X>/` once the new major is confirmed deletes the old data for good, so it is the
  user's step, filed as a human follow-up. The throwaway test database follows with
  `./test-local.ps1 -Fresh` from `backend/api-service/`; `test-postgres` and `e2e-postgres` have
  no data volume. Verify: the app boots, migrations apply, and login and a collections fetch work.
  Plus the `postgres` tag bump steps above. Worked on #537 (18 to 19).
- **Go module or `golang` image major** (only once step 5 confirmed it): a Go major changes the
  import path to a new `/vN`, so code imports and `go.mod` move together, never drop-in. Update the
  import paths, then from `backend/api-service/` run `go mod tidy`, `go build ./...` and
  `./test-local.ps1` (cgo plus Postgres, as CI runs it), and cite the specific breaking change from
  the release notes (gin, pgx, golang-jwt and go-sqlite3 are the load-bearing ones). Staticcheck
  needs the `GOTOOLCHAIN=` value AGENTS.md's Staticcheck line carries. Moving the Go version itself
  is `go mod edit -toolchain=go<X.Y.Z>`, then `npm run sync:image-pins` and `npm run sync:agents`
  (`npm run test:go-toolchain-policy` checks every copy). Close Dependabot's Go release-candidate
  proposals for the builder image.
- **Build-critical npm major** (frontend): from `frontend/`, run `npm run type-check`,
  `npm run lint`, `npm test -- --coverage` (the gate in `vite.config.ts` is lines 70% and branches
  65%) and `npm run build`, the steps `Test Frontend` runs, and cite the specific change.
