# CI/CD

What the workflows run, the policy tests behind **Format Check**, the pins, the required checks
and versioning. What to do once a PR is open, and the reds that aren't the code, are in
[merging](merging.md).

GitHub Actions (`.github/workflows/ci-cd.yml`) — five required jobs. Both this
workflow and `.github/workflows/browser.yml` provision Node from the root
`.nvmrc` and Go from `backend/api-service/go.mod` (`setup-go`'s
`go-version-file`):

1. **format-check** — Prettier over `frontend/`, Prettier over repo markdown, and `gofmt`. Fix: `npm run format` from `frontend/`; `./frontend/node_modules/.bin/prettier --write "**/*.md"` from the repo root; `gofmt -w .` from `backend/api-service/`. The frontend-scoped run cannot reach markdown outside `frontend/`, which is why the root markdown step exists — editing `README.md`, `SETUP.md`, `docs/`, or `.claude/` requires the root command.
   It also runs `npm run test:policy` (the `node --test` list of every policy test in the root
   `package.json`), which exercises the generator's and the tool-neutrality lint's own logic
   and enforces the repository's workflow-action, Go security-tool, Go-toolchain, Node-version,
   PostgreSQL-image, workspace-portability, safe main-branch synchronization, local documentation-link, and
   changelog-footer policies, and tests the parsing behind `npm run board:blockers`
   (which itself reads the private board locally, not in CI). It also runs
   `k8s/startup.ps1`'s cluster start-up against fake `minikube`, `docker` and
   `kubectl` executables, since no CI job runs Minikube. The Go-toolchain policy
   (`scripts/go-toolchain-policy.test.mjs`) holds the `golang` builder tag in
   `backend/api-service/Dockerfile` and every `GOTOOLCHAIN=` pin to the one full `X.Y.Z` release
   `backend/api-service/go.mod` names, requires every `setup-go` step to read it with
   `go-version-file`, and requires the `docker` Dependabot entry to ignore `golang` minor and major
   updates, so the shipped binary is compiled by the Go that CI tested (#544, #564). The Node policy keeps
   `.nvmrc`, both workflows, all three frontend Dockerfiles, package engine metadata, and Node ambient
   types on the Node 26 line, with one exact patch for local, CI, and container tooling. The
   PostgreSQL policy keeps the `Test Go Services` service container on the same `major.minor` as
   the three Compose PostgreSQL services — Dependabot's `docker-compose` ecosystem does not see
   workflow service images — and fails on any retired `postgres:<version>` reference left behind
   in a tracked file outside the release history (`CHANGELOG.md`, `changelog.d/`,
   `docs/changelog/`). The changelog-footer
   policy keeps `CHANGELOG.md`'s reference-link definitions in step with its version
   headings — every released section defined, no orphans, each comparing from the section
   below it, and `[Unreleased]` comparing from the newest release. `version.yml` keeps
   the footer current when it stamps a release, and this test catches a hand edit that
   breaks it. The job also runs
   `npm run sync:agents:check`, which fails if
   `.codex/agents/` (generated from `.claude/agents/`) or `.claude/skills/`
   (generated from `.agents/skills/`) is out of sync with its source. Fix:
   `npm run sync:agents`. Run `./frontend/node_modules/.bin/prettier --write "**/*.md"`
   first if you edited a skill's markdown — regenerating before formatting mirrors
   unformatted content and drifts again on the next format pass. After the sync check it runs
   `npm run lint:agents` (`scripts/lint-agent-neutrality.mjs`), which rejects agent and skill
   sources that read wrongly from their mirrored copies: a bare `$0`–`$9` in a `SKILL.md`
   (Claude Code substitutes the invocation's arguments), a relative link that breaks under the
   mirror's directory, or a blind `claude`→`Codex` substitution artifact.
2. **test-frontend** — type-check, lint, Vitest coverage (≥70% lines, ≥65% branches), build
3. **test-go-services** — `go vet`, Staticcheck 2026.2.2, declared `govulncheck` tool v1.6.0 via `go tool govulncheck`, `go test -race` + Postgres container; statement coverage ≥60%
4. **build-docker-images** — `npm run test:docker-context` verifies both application `.dockerignore` policies with synthetic fixtures and a real Docker scratch `COPY` probe. `npm run test:nginx-headers` verifies security-header inheritance and cache behavior over HTTP using the pinned nginx runtime and synthetic assets. The job then validates application image builds (no push configured). Neither probe reads local environment files.
5. **changelog-version** (check name `Changelog Version`, kept because the ruleset
   requires it) — verifies the branch's `changelog.d/` fragments parse and that it
   edits no released `CHANGELOG.md` section, no `## [Unreleased]` entry and no
   fragment already on `main` (`scripts/check-changelog.mjs`). A PR with no fragment,
   such as Dependabot's, passes; `version.yml` writes its section from the PR title.

The workflow also runs **Test Workspace Portability (Windows)** as a non-required
validation job. It executes the portability suite and the Minikube startup test on
`windows-latest` under both Windows PowerShell 5.1 and PowerShell 7; the
portability job is not a protected branch check.

`.github/workflows/browser.yml` adds **Browser E2E + Axe**, a required check
promoted after exceeding ten consecutive clean runs, and the advisory **Browser
Visual Regression** job. Both report failures normally (no `continue-on-error`)
and retain reports/evidence for 14 days; visual stays optional.

Every third-party workflow `uses:` entry is pinned to a reviewed 40-character
release commit with a readable `# vX.Y.Z` comment. The `github-actions`
Dependabot ecosystem advances both the SHA and comment.

CodeQL runs on PRs via default setup; gated through the code-scanning merge rule (not as a required status check — requiring CodeQL `Analyze` contexts blocks Dependabot PRs which never produce them).

**Versioning** (`.github/workflows/version.yml`): every merge (push) to `main` tags the merge commit with a three-part version from the root `VERSION` file (`v<major>.<minor>.<build>`, e.g. `v1.18.26`). The third component is the auto-incrementing build number. When the major/minor version is bumped, build `0` is allowed. `Scheme: stamp-on-tag` (ADR 0027): branches never edit `CHANGELOG.md`; each writes its entries to `changelog.d/<branch-slug>.md`, and after the merge `scripts/stamp-changelog.mjs` turns that merge's fragments (or, with none, its PR title) into its dated section, deletes them, and pushes one `chore(release): v<x.y.z> [skip ci]` commit plus the tag on the merge commit, then creates the GitHub Release from the section. The push uses a token from the release GitHub App, the main ruleset's one bypass actor (`GITHUB_TOKEN` cannot bypass a ruleset), and `[skip ci]` keeps it from starting CI/CD or Browser Quality.

## Branch protection (`main` and `release/**`)

Repository rules (Settings -> Rules), one active ruleset, `Main/Release branch rules`, covering the default branch and `release/**`: PR required, 0 approvals (self-merge once green), every review thread resolved before merging, merge commits the only merge method, required status checks (`Format Check`, `Test Frontend`, `Test Go Services`, `Build Docker Images`, `Changelog Version`, `Browser E2E + Axe`) on a branch up to date with its base branch (strict), code-scanning gate (errors+warnings / medium+), a code-quality rule at severity `warnings`, Copilot code review on every push (draft PRs excluded), and no branch deletion or force push. One bypass actor: the release GitHub App, "Always allow", so `version.yml` can push the stamp commit. `Browser Visual Regression` remains optional. What they mean when landing a PR is in merging: the up-to-date rule under [Before merging](merging.md#before-merging), the merge-commit rule under [Merging](merging.md#merging).

To change the gate, update the repository ruleset through GitHub UI or the GitHub API. Required check names must match CI job `name:` exactly.
