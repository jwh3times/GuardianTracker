# Development flow

The flow itself is `.agents/skills/ship/settings/dev-flow.md`, and
`AGENTS.md` summarizes it. This file holds what this repository adds.

## Docs and agent config

- `docs/`
- `.github/ISSUE_TEMPLATE/`

## Board drafts

A project-board draft item is not an issue. Running `/to-spec` or `/to-tickets`
on it, or claiming it, makes it one
([ADR 0024](../adr/0024-committing-to-work-creates-the-issue.md)); it is ready
once its body meets the gate.

## Exemption details

- **Dependency bumps** also cover the `npm run sync:image-pins` commit a Node or
  `@playwright/test` bump needs, and a toolchain or module pin moved to clear a
  `govulncheck` advisory.
- **Release prep** also covers `VERSION` and changelog backfills.
