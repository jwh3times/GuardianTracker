### Added

- `docs/agents/guard.md` holds this repository's own rules for the shared guard hook
  (`scripts/guard.mjs`), on top of the rules it applies everywhere. It denies removing or pruning
  Docker volumes, a profiled `docker compose --profile test|e2e down` (which takes the main stack
  down too) and `npx skills update`. It asks before `docker compose down -v`, stopping the main
  stack, `k8s/shutdown.ps1` and `minikube delete`, and protects `private/`, the plaintext env and
  secret targets, keys and certificates, the machine-local 1Password reference and bootstrap
  folders, local Compose and editor settings, and `node_modules/`. The docs map lists the new file.
  No application behavior changes.
