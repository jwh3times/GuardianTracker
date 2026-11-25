# Development flow

The flow itself is `.agents/skills/open-pr/settings/dev-flow.md`, and
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
  justifies it by a policy test that fails once the condition clears, so the shim is deleted
  rather than left to rot (the jest-dom Vitest 5 matcher shim went this way in #467).
- **A change that supersedes a documented decision corrects the doc in the same change.** Grep for
  the claim, not the filename: one retired rule survived as five paraphrases in five files. When
  retiring a doc, scrub live references in `.agents/skills/` too, which can recreate it.

## Rules for every architecture-deepening slice

The remaining slices sequenced by ADRs 0014–0021 are behavior-preserving
ownership migrations. For each one:

- Land the replacement, the deletion of the old path, tests, and affected
  documentation in **one green PR**. Avoid dual owners and cleanup-only
  follow-ups.
- Follow the controlling ADR's migration and acceptance steps. When its final
  slice ships, update the ADR and the index status to "implemented in `vX.Y.Z`".
- Preserve ADR 0003's verify-first rule for Manifest assumptions and ADR 0008's
  HttpOnly same-site refresh-cookie design.
- Preserve the participant/observer split, Item acquisition-facts ownership,
  typed Collections outcomes, explicit Wish list tombstones, and Preferences'
  atomic membership-keyed synchronization contract.
- Do not decide farm-only multi-collectible semantics, collapse source-string and
  milestone-name vocabularies, or assume deterministic Efficiency buckets
  without separate real-Manifest evidence.
- Keep roadmap features out of these migrations.

A slice is ready only when its dependencies are complete, its replacement and
deletion boundaries are explicit, its load-bearing external facts have current
evidence, and its tests exercise the new seam rather than duplicating its data.

## Exemption details

- **Dependency bumps** also cover the `npm run sync:image-pins` commit a Node,
  `@playwright/test` or `golang` image bump needs (with its `go mod edit -toolchain`), and a
  toolchain or module pin moved to clear a `govulncheck` advisory.
- **Release prep** also covers a `VERSION` bump.
