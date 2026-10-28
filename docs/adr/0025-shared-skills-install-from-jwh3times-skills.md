# ADR 0025: Shared Skills Install from jwh3times/skills

- Status: Accepted
- Date: 2026-10-27

## Context

`end-session`, `handoff`, `lets-go` and `ship` were hand-copied into each of the
owner's repositories, and every copy had drifted: a fix made in one reached none
of the others. The `docs-updater` subagent carried this repository's document
ownership inside its own prompt, in the same way.

## Decision

**`end-session`, `handoff`, `lets-go`, `ship` and `docs-updater` install from the
shared [`jwh3times/skills`](https://github.com/jwh3times/skills) repository and
are never edited here.** What they need to know about this repository lives in
settings files under `docs/agents/` (`workspace.md`, `issue-tracker.md`,
`human-follow-up.md`, `docs.md`, `releases.md`, `ship.md`, `handoffs.md`), which
each skill reads when it runs and proposes on its first run. Re-running the
install is the update:

```bash
npx skills add jwh3times/skills -s end-session -s handoff -s lets-go -s ship -s docs-updater -a codex --copy -y
npm run sync:agents
```

- `docs-updater` becomes a skill; its documents table and triggers become the
  `Doc map` in `docs/agents/docs.md`.
- `docs/agents/human-followups.md` becomes the `human-follow-up.md` settings
  file; the follow-up procedure itself ships with the skills.
- `scripts/sync-agents.mjs`, `scripts/lint-agent-neutrality.mjs` and their tests
  are copies of the canonical ones in `jwh3times/skills`; fix them there, then
  copy them here. `npm run lint:agents` runs in **Format Check**.
- The `mattpocock/skills` set stays vendored as `skills-lock.json` records it.

## Consequences

- A local edit to one of the five skills is lost on the next install.
  Repository-specific behavior goes into a setting; a fix to a skill goes to
  `jwh3times/skills`.
- `.prettierignore` lists the five skills, so the root Markdown check never
  reformats installed bytes.
- [ADR 0024](./0024-committing-to-work-creates-the-issue.md) holds for the
  shared `end-session` through its `New work` setting, `draft item` here.
