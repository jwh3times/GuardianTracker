### Changed

- The workspace settings' Dev stack no longer lists `docker compose down -v`
  as never run, beside the volume deletes the guard denies. It is a command an
  agent runs only when the user asks for it by name, never as a way of
  stopping the stack, matching the guard's Ask rule; a PostgreSQL major does
  not need it. `AGENTS.md`'s Compose commands mark it the same way and point at
  the Dev stack rules before stopping the stack or wiping a volume.
