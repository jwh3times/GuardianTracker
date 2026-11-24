### Fixed

- The agent notes on PostgreSQL major upgrades (`docs/agents/dependabot.md`, and the reason text
  of the `docker compose down -v` rule in `docs/agents/guard.md`) again say that a new major
  refuses to start on the volume. v1.18.128 wrongly said it starts beside the old cluster. The
  18+ image's entrypoint exits when it finds an older major's `PG_VERSION` under
  `/var/lib/postgresql/*/docker`, which happened on the move to PostgreSQL 19. The upgrade step is
  now to stop `postgres`, move the old cluster to `<X>/docker.pre-<N>` inside the volume (intact
  and reversible), start the stack, and re-enter the local data worth keeping. No application
  behavior changes.
