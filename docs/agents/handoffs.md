# Handoffs

## Map key

GuardianTracker

## Extra document content

- Machine-local files the other machine must have, named never valued: `.env`,
  `backend/api-service/.env`, `frontend/.env.local`, `k8s/api-service-secret.yaml` and
  `.private-workspace/repository.env.ref` — restored there with
  `./scripts/restore-private-secrets.ps1` then `./setup.ps1`, and the companion with
  `npm run bootstrap:private`.
- The companion appears only as its state (clean, ahead/behind) and commit hash — never its
  remote, branch name or file contents.
