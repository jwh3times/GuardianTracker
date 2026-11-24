# Guard

The commands and paths `scripts/guard.mjs` stops in this repository, on top of the rules it applies
everywhere. Each heading is a setting defined in `.agents/skills/open-pr/settings/guard.md`.

## Deny commands

- `^docker (volume (rm|prune)|system prune\b.*\s--volumes)\b` — removing or pruning Docker volumes
  wipes `postgres-data` and `manifest-data`. The throwaway test databases have no data volume:
  stop them with `docker compose --profile test rm -fsv test-postgres`.
- `^(docker compose|docker-compose)\b.*\s--profile[ =](test|e2e)\b.*\sdown\b` — a profiled
  `down` takes the main stack down too. Stop a test database with
  `docker compose --profile test rm -fsv test-postgres` or
  `docker compose --profile e2e rm -fsv e2e-postgres`.
- `^((npx|bunx|npm exec|pnpm (dlx|exec))( -y| --yes)? )?skills(@\S+)? update\b` — `npx skills update`
  leaves `.claude/skills/` linked to its source instead of copied. Re-run the `npx skills add`
  commands in `AGENTS.md`, then `npm run sync:agents`.

## Ask commands

- `^(docker compose|docker-compose)\b.*\sdown\b.*\s(-[a-zA-Z]*v[a-zA-Z]*|--volumes)\b` — `docker compose down -v`
  wipes `postgres-data` and `manifest-data` (a ~290MB Bungie re-download). A PostgreSQL major
  bump does not need it: the new major starts beside the old cluster (`docs/agents/dependabot.md`).
  Run it only on the user's word.
- `^(docker compose|docker-compose)\b.*\s(stop|down|kill)\b(?!(\s+(frontend|api-service))+$)` — the
  main stack (`postgres`, `pgadmin`, `api-service`, `frontend`) stays running unless the user asks.
  Stopping only `frontend api-service` before a browser run passes.
- `(^shutdown|(^|[\s\\/])shutdown\.ps1)(\s|$)` — `k8s/shutdown.ps1` tears the Minikube stack down;
  run it only on the user's request.
- `^minikube delete\b` — deletes the local cluster and its profile; only with the user's approval.

## Protected paths

- `^private/` — the companion, an independent repository.
- `(^|/)\.env(\.(?!example$)[^/]+)?$` — plaintext env targets restored from 1Password, not
  regenerable from the repository; the tracked `.env.example` files are the templates.
- `^k8s/api-service-secret\.yaml$` — a plaintext secret target restored from 1Password.
- `^k8s/secrets/` — plaintext secret targets restored from 1Password.
- `\.secrets\.yaml$` — plaintext secret targets restored from 1Password.
- `\.(pem|key|crt|p12|pfx)$` — keys and certificates, not regenerable from the repository.
- `^\.private-workspace/` — the machine-local 1Password reference file.
- `^\.private-bootstrap-[^/]*/` — a bootstrap's staged clone, which may hold the companion.
- `^docker-compose\.override\.yml$` — the contributor's local Compose overrides.
- `^\.claude/settings\.local\.json$` — the contributor's own Claude Code settings.
- `^\.vscode/` — the editor settings and tasks.
- `^(frontend/)?node_modules/` — the root Markdown Prettier check runs from them.
