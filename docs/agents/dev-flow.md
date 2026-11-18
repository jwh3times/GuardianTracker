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

## Before implementing

- **Survey the card first.** About half the cards surveyed had a false premise. Re-derive the work
  from the code and report a contradiction before building on it
  ([verification](verification.md#board-cards-are-hypotheses)).
- **Prefer the current, non-legacy approach** for config and tooling, even when the legacy fix is
  smaller, and say what the trade-off is.
- **A compatibility workaround goes through the library's documented extension point**, never by
  mirroring its internals (Vitest's empty `Matchers<R, T>` interface exists for jest-dom-style
  libraries). Present the real options with a recommendation, including what waiting costs and
  whether the dependency earns its place. A shim that must exist is tied to the condition that
  justifies it by a policy test, as `scripts/jest-dom-shim-policy.test.mjs` does.
- **A change that supersedes a documented decision corrects the doc in the same change.** Grep for
  the claim, not the filename: one retired rule survived as five paraphrases in five files. When
  retiring a doc, scrub live references in `.agents/skills/` too, which can recreate it.

## Exemption details

- **Dependency bumps** also cover the `npm run sync:image-pins` commit a Node or
  `@playwright/test` bump needs, and a toolchain or module pin moved to clear a
  `govulncheck` advisory.
- **Release prep** also covers `VERSION` and changelog backfills.
