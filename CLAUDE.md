# CLAUDE.md

@AGENTS.md

## Claude Code specifics

Everything above is imported from [AGENTS.md](./AGENTS.md), which is the canonical,
tool-neutral operating guide. **Repo operating context belongs there, not here.** This
section covers only what is specific to Claude Code as a tool.

### Subagents

The agents listed in AGENTS.md's routing table are dispatchable as Claude Code
subagents via the Agent tool. Their definitions live in `.claude/agents/`, which
is also the source for the generated Codex mirror in `.codex/agents/` — see the
sync contract in AGENTS.md.

### Skills

Docs freshness is handled at push time by the `/ship` skill
(`.claude/skills/ship/SKILL.md`), not by a per-turn hook. `/ship` invokes the
`docs-updater` subagent against the branch diff, classifies the release as major,
minor, or build-only, updates `VERSION` for major and minor releases, writes the
`CHANGELOG.md` section for the version the merge will mint, runs the fast checks,
pushes, and opens or updates the PR. Run it when a branch is ready for review.
