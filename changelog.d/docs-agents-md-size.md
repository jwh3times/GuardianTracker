### Changed

- `AGENTS.md` fits Codex's 32 KiB instruction budget again (52,347 bytes down to 27,808), so a
  Codex session reads the whole file instead of silently losing its last third. Reference detail
  moved to the doc that owns its topic: CI/CD, branch protection and the policy
  tests to `docs/agents/ci.md`; the private workspace, `main` sync, Node pin and runtime values
  table to `docs/agents/environment.md`; the shared MCP servers to `docs/agents/mcp.md`; the auth
  behavior to preserve and application logging to `docs/agents/security-and-logging.md`; the
  "Rules for every architecture-deepening slice" to `docs/agents/dev-flow.md`; the Staticcheck and
  Go coverage notes to `docs/agents/local-runs.md`; the documentation rules to
  `docs/agents/docs.md`; and the known limitations to `docs/architecture.md`. `AGENTS.md` keeps a
  pointer naming when to read each one, and the Doc map records the 30,000-byte limit.
  ADR 0028 records the decision and supersedes ADR 0022 in part. No application behavior changes.
