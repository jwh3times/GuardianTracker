# Development flow

The flow itself is `.agents/skills/ship/settings/dev-flow.md`, and
`AGENTS.md` summarizes it. This file holds what this repository adds.

## Docs and agent config

- `docs/`
- `.github/ISSUE_TEMPLATE/`

## Board items

Every board item is an issue
([ADR 0026](../adr/0026-every-board-item-is-an-issue.md)), but an issue is not
automatically ready. It passes the gate once it is labelled `ready-for-agent`, its
body is a spec, ticket or triage brief, and it is not labelled `blocked`.

## Exemption details

- **Dependency bumps** also cover the `npm run sync:image-pins` commit a Node or
  `@playwright/test` bump needs, and a toolchain or module pin moved to clear a
  `govulncheck` advisory.
- **Release prep** also covers `VERSION` and changelog backfills.
