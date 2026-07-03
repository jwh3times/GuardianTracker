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
- [Formatting](#formatting)
- [Tests & coverage](#tests--coverage)
- [CI gates](#ci-gates)
- [Reporting bugs & requesting features](#reporting-bugs--requesting-features)
- [Security issues](#security-issues)

## Ways to contribute

- **Report a bug** or **request a feature** via [GitHub Issues](https://github.com/jwh3times/GuardianTracker/issues)
  (use the provided templates).
- **Improve documentation** — fixes to `README.md`, `CLAUDE.md`, this guide, or
  inline comments are always welcome.
- **Submit code** — pick up an open issue or propose a change. For anything large,
  open an issue first so we can align on the approach before you invest time.

## Development setup

### Prerequisites

- Docker Desktop (for Docker Compose or Minikube)
- Go 1.25+ and Node.js 26+ (for running services individually — CI pins Go 1.25.x and Node 26)
- A [Bungie API application](https://www.bungie.net/en/Application) for OAuth
  credentials (API key, client ID, client secret)

### 1. Configure environment variables

```powershell
./setup.ps1   # copies every .env.example into place
```

Then fill in the required `BUNGIE_*`, `JWT_SECRET`, and `TOKEN_ENCRYPTION_KEY`
secrets. See the [README](./README.md#2-configure-environment-variables) for the
full table and [SECURITY.md](./SECURITY.md) for guidance on handling secrets.

### 2. Run the stack

The fastest path for full-stack work is Docker Compose:

```powershell
docker compose up --build
```

- Frontend: <http://localhost:5273>
- API: <http://localhost:8081>

For single-service work with hot reload, run services individually (Vite for the
frontend, [Air](https://github.com/air-verse/air) for the API). See the
[README](./README.md#3-start-services) and [CLAUDE.md](./CLAUDE.md#running-services)
for all three run options (Compose, Minikube, individual).

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

For architecture, the auth/token flow, key files, and common tasks, read
**[CLAUDE.md](./CLAUDE.md)** — it is the deep-dive developer guide.

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

## Formatting

Formatting is enforced in CI (`format-check`). Run the formatters before pushing:

```powershell
# Frontend (from frontend/)
npm run format          # writes Prettier formatting
npm run format:check    # verifies without writing (what CI runs)

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

See [CLAUDE.md → Full Go coverage locally](./CLAUDE.md#full-go-coverage-locally-matches-ci)
for the toolchain details (a C compiler is required for the cgo tests).

## CI gates

Every PR must pass these GitHub Actions jobs before it can merge:

| Check                   | What it does                                                                                                               |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| **Format Check**        | Prettier (frontend) + `gofmt` (Go) — fails if anything is unformatted                                                      |
| **Test Frontend**       | type-check, lint, Vitest with coverage thresholds, production build                                                        |
| **Test Go Services**    | `go vet`, `govulncheck`, `go test -race` with the coverage gate; DB integration tests against a Postgres service container |
| **Build Docker Images** | builds both production Docker images (build validation; no push)                                                           |

CodeQL also scans the repo on every PR via GitHub's default setup; for human PRs it
is enforced through a code-scanning merge rule rather than as a required status
check. See [CLAUDE.md → CI/CD](./CLAUDE.md#cicd) for the full explanation, including
how Dependabot PRs are handled.

## Reporting bugs & requesting features

Use the issue templates:

- **Bug report** — steps to reproduce, expected vs. actual, environment.
- **Feature request** — the problem you're trying to solve and your proposed solution.

Search existing issues first to avoid duplicates.

## Security issues

**Do not open a public issue for security vulnerabilities.** Follow the process in
[SECURITY.md](./SECURITY.md) — email the maintainer directly. You can expect a
response within 72 hours.
