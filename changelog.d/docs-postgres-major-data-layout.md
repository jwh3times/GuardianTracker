### Fixed

- The agent notes on PostgreSQL major upgrades (`docs/agents/dependabot.md`, and the reason text of
  the `docker compose down -v` rule in `docs/agents/guard.md`) no longer say that a new major
  refuses to start on the old data directory or that the volume must be wiped. The 18+ images keep
  each major's cluster in its own subdirectory of the `postgres-data` volume, so a new major starts
  on an empty cluster beside the old one, which stays intact. The upgrade step is now to log in and
  re-enter the local data worth keeping, never `docker compose down -v`. No application behavior
  changes.
