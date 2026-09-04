# Contributing to Guardian Tracker

Thanks for your interest in contributing! This guide covers contribution
conventions and the CI contract. [SETUP.md](./SETUP.md) owns local installation,
environment, ports, and runnable validation commands.

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
  reviews, and raw research under gitignored `private/`. Detailed
  implementation handoffs live on GitHub Issues and the Project board instead
  (see `AGENTS.md`'s Work Tracking section).
- **Submit code** — pick up an open issue or propose a change. For anything large,
  open an issue first so we can align on the approach before you invest time.

## Development setup

Follow [SETUP.md](./SETUP.md) for prerequisites and all development options. The
recommended full-stack path is:

```powershell
./setup.ps1   # creates missing local configuration from committed examples
```

```powershell
docker compose up --build
```

Use only an API key and public OAuth client ID from the Bungie application;
Guardian Tracker does not use a client secret. See
[SECURITY.md](./SECURITY.md#credential-management) for secret-handling policy.

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

Coverage thresholds are enforced in CI:

- **Frontend**: lines ≥70%, branches ≥65% (Vitest).
- **Go services**: statement coverage ≥60% (with the race detector).

Run the relevant local commands from
[SETUP.md → Tests and Checks](./SETUP.md#tests-and-checks). Browser tests have
additional process-isolation and Linux-baseline requirements documented in
[frontend/README.md → Browser Tests](./frontend/README.md#browser-tests); that
guide is their canonical owner.

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
