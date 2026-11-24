# MCP servers

The shared MCP servers: when to reach for each, enabling them, and their pin. The rule that git,
GitHub and secrets never go through an MCP server is in `AGENTS.md` ("MCP servers").

The shared set is committed twice, once per tool: `.mcp.json` (Claude Code) and
`.codex/config.toml` (Codex). Keep the two files declaring the same servers with the same
arguments.

| Server       | Reach for it when                                                                                                                        |
| ------------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `context7`   | A task depends on a library, framework, or CLI's current API — Gin, React Query, Vite, Playwright, oxlint. Bungie facts are not in it.   |
| `playwright` | A change needs eyes on the running app: drive `localhost:5273`, read console and network, screenshot. The e2e suite stays the test gate. |

Enabling them is a one-time step per person:

- **Claude Code** asks you to approve each project server on the first session in this
  checkout. If you already have the claude.ai Context7 connector, declining the project
  `context7` avoids duplicate tools.
- **Codex** reads `.codex/config.toml` only once the checkout is a trusted project.
- **Playwright's browser** is installed separately:
  `npx @playwright/mcp@<pinned version> install-browser chrome-for-testing`. Every
  `@playwright/mcp` release builds on an alpha `playwright-core`, so its browser revision
  never matches the one `frontend/`'s `@playwright/test` installs, and the server runs
  `--browser chromium` rather than an installed Chrome that contributors may not have.
  Bumping the pin needs the same command again.

Nothing updates the pin: Dependabot does not read either file. Bump both files together,
then re-run the browser install. The server runs `--isolated`, so no browser profile or
sign-in persists between sessions; it writes snapshots into `.playwright-mcp/`, which is
ignored.
