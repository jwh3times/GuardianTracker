# Environment

Environment files, the optional private workspace, `main` synchronization, the Node pin and the
runtime values the API reads. `AGENTS.md` ("Environment Setup") keeps the setup command and the
rules that never change; this file holds the rest.

## Environment setup

Public-only development requires no private workspace. Authorized maintainers
can optionally restore the ignored `private/` directory as an independent Git
repository with `./scripts/bootstrap-private-workspace.ps1 -PrivateFromPrompt`,
or through 1Password with `-PrivateFromOnePassword` after creating the ignored
machine-local `.private-workspace/repository.env.ref` file containing only
`GUARDIAN_PRIVATE_REPOSITORY_URL=op://<vault>/<item>/<field>`.
`npm run bootstrap:private` (from the repo root) is the Node equivalent for new
git worktrees; it reuses the main checkout's reference file. The Node helper
requires untracked, committed-ignore-protected private/reference paths, stages
new clones before publication, and suppresses raw Git/1Password diagnostics.
For its process-argument limits and failure behavior, read
[Node bootstrap safety](../maintainers/workspace-recovery.md#node-bootstrap-safety-and-diagnostics)
when restoring private workspaces or changing this helper.

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

Frontend tooling uses the exact Node.js 26 patch in the root `.nvmrc`.
`frontend/package.json` accepts the Node 26 line only; CI and all three frontend
Dockerfiles use the exact `.nvmrc` patch. Node 26 remains Current until its
scheduled LTS transition on October 28, 2026; this project intentionally accepts
that short-term churn because Node is frontend development/build tooling and the
deployed frontend runtime is nginx.

## Required runtime values

| Variable                                | Purpose                                                                                                                                                |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `GO_ENV`                                | Required; exactly `development` or `production`                                                                                                        |
| `LOG_LEVEL`                             | `debug`, `info`, `warn`, or `error`; defaults to `info`                                                                                                |
| `LOG_FORMAT`                            | `text` or `json`; defaults to text in development and JSON in production                                                                               |
| `BUNGIE_API_KEY`                        | From <https://www.bungie.net/en/Application>                                                                                                           |
| `BUNGIE_CLIENT_ID`                      | Public Bungie OAuth client identifier; no client secret is used                                                                                        |
| `AUTH_REDIRECT_URI`                     | OAuth callback URL                                                                                                                                     |
| `JWT_SECRET`                            | 32+ char random string (`openssl rand -base64 32`)                                                                                                     |
| `DATABASE_URL`                          | Postgres connection string (local development uses `sslmode=disable`; production guidance: [PostgreSQL TLS](../../SETUP.md#production-postgresql-tls)) |
| `TOKEN_ENCRYPTION_KEY`                  | 32-byte base64 key for Bungie authorization encryption (`openssl rand -base64 32`)                                                                     |
| `TOKEN_ENCRYPTION_KEY_VERSION`          | Positive version written for the current key (start at `1`)                                                                                            |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS`         | (optional) previous key for rotation during key migration                                                                                              |
| `TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION` | Exact positive version for the previous key                                                                                                            |
| `CORS_ALLOWED_ORIGINS`                  | Exact browser origins allowed to call the API                                                                                                          |
| `ADMIN_MEMBERSHIP_IDS`                  | (optional) comma-separated Bungie membership IDs pinned to admin role at login                                                                         |
