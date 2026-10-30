# Workspace

## Private companion

`private/` — an independent Git repository (not a submodule), ignored by this one and restored by
authorized maintainers with `npm run bootstrap:private` or
`./scripts/bootstrap-private-workspace.ps1`. Repository:
`discover: git -C private remote get-url origin` — its location stays out of public files;
`docs/maintainers/workspace-recovery.md` restores it when the checkout is absent.
`private/README.md` is the sole index of its layout.

## Workspace status command

`git -C private fetch -q --prune origin` (when `private/.git` exists), then
`pwsh -NoProfile -File scripts/workspace-status.ps1` — value-free status of both repositories
(the private branch name redacted) and of the four plaintext env targets; it exits 1 on an invalid
companion or an unprotected target. Its ahead/behind counts come from local tracking refs, hence
the fetch. Where only Windows PowerShell 5.1 exists, run it with `powershell` instead of `pwsh`.

## Companion commit checks

- When the index changed, set `private/README.md`'s `**Updated:**` line to today.

## Sync with main

`npm run sync:main` — preflights both checkouts for uncommitted changes, then fast-forwards each
clean one from `origin/main`; a missing `private/` is skipped. `npm run sync:main -- --skip-private`
updates only the public checkout.

## Safe-to-remove paths

- `frontend/playwright-report/`, `frontend/test-results/`, `frontend/playwright/.auth/`
- `frontend/dist/`
- `backend/api-service/.e2e/` — the manifest and search index the E2E API writes
- `backend/api-service/coverage` (extensionless profile), `coverage/`, `*.coverprofile`, `*.out`,
  `*.test`, `*.exe`, `*.tsbuildinfo` — ignored matches only
- stray scratch files at the repository root or under `private/`, named one by one

## Never-remove paths

- `private/`, in any form — an independent repository; `git clean -X` would take it
- `.env`, `.env.*` at any depth (`backend/api-service/.env`, `frontend/.env.local`) and
  `k8s/api-service-secret.yaml`, `k8s/secrets/`, `*.secrets.yaml` — plaintext targets restored
  from 1Password, not regenerable from the repository
- `*.pem`, `*.key`, `*.crt`, `*.p12`, `*.pfx`
- `.private-workspace/` — the machine-local 1Password reference file
- `.private-bootstrap-*/` — a bootstrap's staged clone; one left behind may hold the companion
- `docker-compose.override.yml`, `.claude/settings.local.json`, `.vscode/`
- `.claude/worktrees/` — removing one is the user's decision; it may hold unpushed commits
- `node_modules/`, `frontend/node_modules/` — the root Markdown Prettier check runs from it

## Dev stack

- **Show:** `docker compose --profile test --profile e2e ps`; `minikube status` where Minikube is
  installed.
- **Stop the throwaway test databases** (no data volume; tests recreate them):
  `docker compose --profile test rm -fsv test-postgres` and
  `docker compose --profile e2e rm -fsv e2e-postgres`.
- **Main stack** (`postgres`, `pgadmin`, `api-service`, `frontend`): leave it running unless the
  user asks; stop it with `docker compose stop` or `docker compose down`. Minikube:
  `./k8s/shutdown.ps1` (interactive), only on request.
- **Never run:** `docker compose down -v` or any `docker volume rm`/`prune` — it wipes
  `postgres-data` and `manifest-data` (a ~290MB Bungie re-download);
  `docker compose --profile test down` or `docker compose --profile e2e down` — either takes the
  main stack down too.

## Generated-config check

`npm run sync:agents:check` (inside the required **Format Check**), regenerated with
`./frontend/node_modules/.bin/prettier --write "**/*.md"` then `npm run sync:agents` — always: it is quick, and it also catches an orphaned mirror.

## Final portability check

`pwsh -NoProfile -File scripts/workspace-status.ps1`, after fetching both repositories: it must
exit 0, report both trees clean, and show both ahead counts at 0.
