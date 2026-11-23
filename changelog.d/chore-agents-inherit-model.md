### Changed

- Every Claude Code agent definition in `.claude/agents/` now sets `model: inherit`, so a subagent
  runs on the model of the session that launched it instead of a pinned Sonnet or Opus. The Codex
  mirrors are unchanged, because the generator drops `model:`. No application behavior changes.
