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

**Status:** Partially implemented — storage, validation, and DIM-format import only
**Gate:** Owner approval of the player-facing surface
**Likely size:** Large

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
  saved rolls, and only an identical roll is refused. There is no HTTP route yet
  and no way for a player to reach this from the app.
- A target names perks by display name rather than by the game's plug
  identifier. An owner capture established that a perk's base and enhanced
  variants are two identifiers sharing one name, linked by no manifest field, so
  an identifier would stop matching as soon as an enhanced copy dropped.
- Import parsing for a DIM-format file is implemented
  (`services/rolltargets/dimfile.go`, `dimimport.go`): every line of an
  uploaded file is parsed and, where it names a real weapon and resolvable
  perks, saved, with a per-line report of what happened and why. It is not
  reachable from the app yet — see below.

Still to settle:

- The endpoint(s) that expose roll targets and DIM import to the frontend.
- Where matches appear for the owner, and how an unmatched or unresolved perk is
  presented honestly.

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
