# Contributing to Guardian Tracker

Thanks for your interest in contributing! This guide covers how to get a local
environment running, the conventions we follow, and what the CI gates expect so
your pull request lands smoothly.

By participating in this project you agree to abide by our
[Code of Conduct](./CODE_OF_CONDUCT.md).

## Table of Contents

- [Ways to contribute](#ways-to-contribute)
- [Development setup](#development-setup)
- [Project layout](#project-layout)
- [Branching & pull requests](#branching--pull-requests)
- [Commit messages](#commit-messages)
- [Versioning](#versioning)
- [Formatting](#formatting)
- [Tests & coverage](#tests--coverage)
- [CI gates](#ci-gates)
- [Reporting bugs & requesting features](#reporting-bugs--requesting-features)
- [Security issues](#security-issues)

## Ways to contribute

- **Report a bug** or **request a feature** via [GitHub Issues](https://github.com/jwh3times/GuardianTracker/issues)
  (use the provided templates).
- **Improve documentation** — fixes to `README.md`, `SETUP.md`, `docs/`, this
  guide, or inline comments are always welcome. Keep private runbooks, security
  reviews, raw research, and implementation handoffs under gitignored `private/`.
- **Submit code** — pick up an open issue or propose a change. For anything large,
  open an issue first so we can align on the approach before you invest time.

## Development setup

### Prerequisites

- Docker Desktop (for Docker Compose or Minikube)
- Go 1.26+ and the exact Node.js 26 patch listed in `.nvmrc` (for running
  services individually). The frontend package accepts only Node 26; CI and the
  frontend Dockerfiles use the exact `.nvmrc` patch, and npm rejects other Node
  lines.
- A [Bungie API application](https://www.bungie.net/en/Application) for OAuth
  configuration (API key and public client ID; public clients have no client
  secret or refresh token)

### 1. Configure environment variables

```powershell
./setup.ps1   # copies every .env.example into place
```

Then fill in the required `BUNGIE_*`, `JWT_SECRET`, and `TOKEN_ENCRYPTION_KEY`
secrets. See [SETUP.md](./SETUP.md#2-create-environment-files) for the full
table and [SECURITY.md](./SECURITY.md) for guidance on handling secrets.

### 2. Run the stack

The fastest path for full-stack work is Docker Compose:

```powershell
docker compose up --build
```

- Frontend: <http://localhost:5273>
- API: <http://localhost:8081>

For single-service work with hot reload, run services individually (Vite for the
frontend, [Air](https://github.com/air-verse/air) for the API). See
[SETUP.md](./SETUP.md) and [AGENTS.md](./AGENTS.md#running-services) for all
three run options (Compose, Minikube, individual).

> On first run the API service downloads the ~100MB Destiny 2 manifest. The
> collections endpoint returns `503` until that completes.

## Project layout

```text
frontend/                 # React 19 + TypeScript SPA (Vite, TanStack Query)
backend/api-service/      # Go + Gin — OAuth, JWT, Bungie API, manifest, collections
database/init/            # PostgreSQL bootstrap SQL
k8s/                      # Kubernetes manifests + Minikube scripts
.github/workflows/        # CI/CD pipeline
```

For public architecture, read [docs/architecture.md](./docs/architecture.md).
For agent-specific operating context, read **[AGENTS.md](./AGENTS.md)**.

## Branching & pull requests

`main` is protected: **direct pushes are blocked**, and all changes land via pull
request once CI is green.

1. Branch off the latest `main`:

   ```bash
   git switch main && git pull
   git switch -c <type>/<short-description>   # e.g. feat/wishlist-sort, fix/oauth-redirect
   ```

2. Make your change, keeping commits focused.
3. Run the formatters and tests locally (see below) before pushing.
4. Open a PR against `main`. Fill out the PR template — describe what changed, why,
   and how you verified it.
5. Ensure all required status checks pass. The branch must be **up to date with
   `main`** (strict checks) before it can merge.

Keep PRs reasonably small and single-purpose; they are far easier to review and
much more likely to merge quickly.

## Commit messages

Write clear, imperative-mood subject lines that describe the change:

```text
Add wishlist priority sorting to the Collections drawer
Fix OAuth callback dropping the state parameter on retry
```

- Keep the subject to ~72 characters; add a body when the _why_ isn't obvious.
- One logical change per commit where practical.

## Versioning

The root `VERSION` file is the project version source of truth. It must contain a
plain three-part SemVer value in `<major>.<minor>.<build>` form, such as
`0.2.0`.

Every merge to `main` creates an annotated tag and GitHub Release named
`v<major>.<minor>.<build>`. The workflow auto-increments the build number within
the current major/minor line. When bumping major or minor, build `0` is valid: if
`VERSION` is `1.0.0` and no `v1.0.*` tag exists yet, the release remains
`v1.0.0` instead of being forced to `v1.0.1`.

## Formatting

Formatting is enforced in CI (`format-check`). Run the formatters before pushing:

```powershell
# Frontend (from frontend/)
npm run format          # writes Prettier formatting
npm run format:check    # verifies without writing (what CI runs)

# Repo markdown — README, SETUP, docs/, .claude/, k8s/ (from the repo root)
# The frontend-scoped run above cannot reach these files.
./frontend/node_modules/.bin/prettier --write "**/*.md"
./frontend/node_modules/.bin/prettier --check "**/*.md"   # what CI runs

# Go service (from backend/api-service/)
gofmt -w .
```

## Tests & coverage

```powershell
# Frontend (from frontend/)
npm test                # Vitest
npm run type-check
npm run lint

# Go service (from backend/api-service/)
go test ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...
```

Coverage thresholds are enforced in CI:

- **Frontend**: lines ≥70%, branches ≥65% (Vitest).
- **Go services**: statement coverage ≥60% (with the race detector).

A plain `go test ./...` on a fresh Windows checkout reports lower coverage because
the SQLite (cgo) and Postgres-integration tests self-skip. To reproduce CI's full
coverage locally, use the helper script (it spins up a throwaway Postgres):

```powershell
cd backend/api-service
./test-local.ps1          # all tests + total coverage
./test-local.ps1 -Html    # also open the per-line HTML report
```

See [AGENTS.md → Full Go coverage locally](./AGENTS.md#full-go-coverage-locally-matches-ci)
for the toolchain details (a C compiler is required for the cgo tests).

For full-browser validation, start the isolated database and use the fake Bungie
fixtures; never point automated tests at the live Bungie API:

```powershell
docker compose --profile e2e up -d --wait e2e-postgres
$env:E2E_FIXED_TIME="2026-07-18T18:00:00Z"
cd frontend
npm run e2e
npm run e2e:visual
```

## CI gates

Every PR must pass these GitHub Actions jobs before it can merge:

| Check                   | What it does                                                                                                                                             |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Format Check**        | Prettier (frontend) + Prettier (repo markdown) + `gofmt` (Go) — fails if anything is unformatted                                                         |
| **Test Frontend**       | type-check, lint, Vitest with coverage thresholds, production build                                                                                      |
| **Test Go Services**    | `go vet`, Staticcheck 2026.1, declared `govulncheck` v1.6.0 via `go tool`, `go test -race` with the coverage gate; DB integration tests against Postgres |
| **Build Docker Images** | builds both production Docker images (build validation; no push)                                                                                         |
| **Changelog Version**   | PR-only; fails if `CHANGELOG.md`'s top version doesn't match the tag the merge will mint (`scripts/next-version.sh`); exempt for bot-authored PRs        |

Third-party actions in `.github/workflows/` must use a reviewed 40-character
commit SHA followed by a `# vX.Y.Z` release comment. The format job tests this
policy, and the `github-actions` Dependabot configuration updates the SHA and
comment together when a newer allowed release is available.

CodeQL also scans the repo on every PR via GitHub's default setup; for human PRs it
is enforced through a code-scanning merge rule rather than as a required status
check. See [AGENTS.md → CI/CD](./AGENTS.md#cicd) for the full explanation, including
how Dependabot PRs are handled.

The separate browser workflow reports failures normally but is not initially in
branch protection. Promote `Browser E2E + Axe` after ten consecutive clean runs.
`Browser Visual Regression` remains advisory.

## Reporting bugs & requesting features

Use the issue templates:

- **Bug report** — steps to reproduce, expected vs. actual, environment.
- **Feature request** — the problem you're trying to solve and your proposed solution.

Search existing issues first to avoid duplicates.

## Security issues

**Do not open a public issue for security vulnerabilities.** Follow the process in
[SECURITY.md](./SECURITY.md) — email the maintainer directly. You can expect a
response within 72 hours.
