# Roadmap

This public roadmap lists work that is not implemented yet. Completed work is
tracked in [CHANGELOG.md](./CHANGELOG.md), and durable architecture decisions are
tracked in [docs/adr](./docs/adr/README.md).

Detailed implementation handoffs live on GitHub Issues and the Project board
(see `AGENTS.md`'s Work Tracking section). Private security analysis, deployment
runbooks, and environment-specific operations notes belong under `private/`.

## How to Use This Roadmap

- Treat each product item below as requiring its own design or implementation
  spec before build.
- Verify Bungie API and manifest assumptions against real data before coding.
- Keep authorization checks server-side.
- Add or update tests with every behavior change.
- Update `CHANGELOG.md` when an item ships.
- Add an ADR when a change creates or supersedes a durable architecture or
  operating decision.

## Product Backlog

### God-Roll and Owned-Roll Insights

**Status:** Partially implemented — storage, validation, DIM-format import,
matching saved targets against the owned weapons a fresh Bungie profile read
reports, a REST surface, the Roll targets page (`/rolls`, behind the
`god-roll` flag), a per-weapon item-drawer section, and closest-owned-copy
progress for still-chasing targets all exist
**Gate:** none — the owner approved the player-facing surface 2026-09-23 (#350)
**Likely size:** Medium (remaining scope)

Show owned weapon rolls for the selected Destiny membership and compare them to
target rolls.

Decided:

- The feature is called **roll targets**, so it stays distinct from the existing
  wish list of items to acquire.
- Targets are authored by the user, and the user can import a wish list file in
  DIM's text format that they supply. Guardian Tracker bundles and fetches no
  third-party roll data.
- It works against the signed-in owner's own profile and needs no public hosting.
- Roll targets have storage and validation: a target names the perks that must
  all be present, on a specific weapon Item hash or on no weapon at all (DIM's
  any-item wildcard), and is either wanted or unwanted (DIM's undesirable/"trash"
  roll). A named weapon's perks are checked against that weapon's own perk pool
  before the target saves (`services/rolltargets`); a weapon may hold several
  saved rolls, and only an identical roll is refused. `GET/POST /api/rolltargets`,
  `PATCH/DELETE /api/rolltargets/:id`, `POST /api/rolltargets/bulk` (delete
  named ids or every target the membership owns), and `POST
/api/rolltargets/import` expose this over REST, membership-scoped through
  the caller's JWT alone and gated behind the `god-roll` feature flag (alpha
  tier).
- A target names perks by display name rather than by the game's plug
  identifier. An owner capture established that a perk's base and enhanced
  variants are two identifiers sharing one name, linked by no manifest field, so
  an identifier would stop matching as soon as an enhanced copy dropped.
- Import parsing for a DIM-format file is implemented
  (`services/rolltargets/dimfile.go`, `dimimport.go`): every line of an
  uploaded file is parsed and, where it names a real weapon and resolvable
  perks, saved, with a per-line report of what happened and why. The Roll
  targets page's file picker and paste box both reach it, rendering the report
  inline.
- Matching saved targets against what the player owns is implemented
  (`services/ownedrolls`, `services/rolltargets/matching.go`). A new package,
  `services/ownedrolls`, reads the membership's vault, character inventories,
  and equipped items (profile components 102/201/205) together with each
  instanced item's socket states (component 305) and resolves the currently
  seated plug in every perk column to its Manifest display name — the same
  spelling and sorting a saved target uses, so the two compare directly.
  Component 310 (which plugs could be swapped in) is deliberately not
  requested; it answers a different question and was a large share of the
  profile response the owner capture measured. `rolltargets.Matches` joins
  that owned-roll read to the membership's saved targets: a target matches an
  owned weapon when every perk it names is present on that weapon (extra
  perks do not prevent a match), an any-weapon target is tested against every
  owned weapon, and a target nothing satisfies is still reported rather than
  dropped. For each unmatched target, the service also selects the applicable
  owned copy matching the most target perk columns (at least one but fewer than
  all), preserving deterministic owned-copy order for equal scores. `GET
/api/rolltargets/matches` exposes that best copy and its matched-perk subset
  with the joined result, and the
  Roll targets page (`/rolls`, behind the `god-roll` flag, alpha tier) reads
  it: unmatched targets render as "Still chasing" (a goal, never absence,
  filtered by default to acquired/wishlisted weapons plus any-weapon targets,
  with a "Show all" toggle and a search box), including "Your best copy has X
  of Y target perks" when a partial copy exists; wanted matches render as "You
  have it" (grouped by target, each owned copy labelled "Copy 1", "Copy 2", …
  with the target's own perks highlighted), and unwanted matches render
  collapsed and informational. A match-report failure never hides the
  player's own saved targets — every one still lists, neutrally, with a
  Reconnect or Retry banner depending on the failure.
- A per-weapon roll-target section in the Collections item detail drawer is
  implemented (#371, roll targets slice 6), reusing the same
  `data/rolltargets.ts` queries and cache entry as the page. Opening a
  weapon's drawer shows: which of its own targets an owned copy currently
  satisfies (grouped by owned copy, since one weapon-scoped drawer can have a
  copy that satisfies more than one saved target — an any-weapon target
  included), which of its own targets are still unmatched ("Still chasing",
  with closest-copy progress when available),
  and any match on a roll marked unwanted, collapsed and informational. A
  match-report failure falls back to the same neutral, Reconnect/Retry
  treatment as the page, scoped to that weapon's own saved targets. The
  section is read-only — no target creation, editing, or deletion from the
  drawer — and always links to `/rolls` for management. Below alpha tier it
  shows a locked upsell instead of a farming promise.

Still to settle:

- Recording which DIM file a target came from, deferred as its own card.
- The mockup's "Best farm: {source}" line (joining acquisition data to
  targets) is out of scope for the drawer section that shipped in #371 and
  remains its own decision.
- Manual perk authoring (saving a target without a DIM import) is deferred.
- How an unmatched or unresolved perk is presented honestly to the owner.

### Notifications and Digests

**Status:** Not implemented
**Gate:** Public deployment decision, then product and provider decision
**Likely size:** Large

Guardian Tracker currently runs only as a local application, so this waits on
[Production Deployment Path](#production-deployment-path).

Send opt-in reminders when weekly or vendor data contains missing or wishlisted
items.

Decisions to settle:

- Email provider and sending domain.
- User preference model and unsubscribe flow.
- How scheduled jobs run once a production deployment exists.

### Shareable Collection Progress

**Status:** Not implemented
**Gate:** Public deployment decision, then security and privacy review
**Likely size:** Medium to large

A public share page needs a public host, so this waits on
[Production Deployment Path](#production-deployment-path).

Allow users to create public read-only snapshots of collection progress without
exposing private account details or authenticated endpoints.

Expected shape:

- explicit snapshot creation
- revocable share token
- unauthenticated public page with minimal data
- rate limiting and abuse controls

## Operations Backlog

### Production Deployment Path

**Status:** Deferred — Guardian Tracker is a local-only application by
maintainer decision
**Gate:** Maintainer decision to publish
**Likely size:** Large

Notifications and Digests, Shareable Collection Progress, and Metrics all wait on
this decision.

CI currently validates tests and Docker builds. Publishing images, provisioning
cloud resources, and deploying production infrastructure should be implemented
only after the target hosting model is accepted. Environment-specific runbooks
belong in `private/` until then.

### Metrics

**Status:** Not implemented
**Gate:** Production deployment path, then observability decision
**Likely size:** Medium

Structured request/access logging with request IDs and sanitized identifiers has
shipped (see [docs/architecture.md](./docs/architecture.md#request-logging)).
Metrics remain unimplemented; add them only after the production runtime and
collector target are known.
