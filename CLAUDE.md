# CLAUDE.md

@AGENTS.md

## Claude Code specifics

Everything above is imported from [AGENTS.md](./AGENTS.md), which is the canonical,
tool-neutral operating guide. **Repo operating context belongs there, not here.** This
section covers only what is specific to Claude Code as a tool.

### Subagents

The agents listed in AGENTS.md's routing table are dispatchable as Claude Code
subagents via the Agent tool. Their definitions live in `.claude/agents/`.
These files are also the source for the generated Codex mirrors in `.codex/agents/`.
Editing one requires re-running `node scripts/sync-agent-configs.mjs`; CI's
`format-check` job fails otherwise. The same applies to `.claude/skills/`, which
generates `.agents/skills/`.

### Skills

Docs freshness is handled at push time by the `/ship` skill
(`.claude/skills/ship/SKILL.md`), not by a per-turn hook. `/ship` invokes the
`docs-updater` subagent against the branch diff, writes the `CHANGELOG.md` section for
the version the merge will mint, runs the fast checks, pushes, and opens or updates the
PR. Run it when a branch is ready for review.
