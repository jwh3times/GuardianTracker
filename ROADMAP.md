# Roadmap

**Updated:** 2026-07-13

This public roadmap lists work that is not implemented yet. Completed work is
tracked in [CHANGELOG.md](./CHANGELOG.md), and durable architecture decisions are
tracked in [docs/adr](./docs/adr/README.md).

Detailed implementation handoff plans, private security analysis, deployment
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

### 2. Deeper Character-Scoped Surfaces

**Status:** Not implemented
**Gate:** Bungie response-shape verification
**Likely size:** Large

The character switcher now drives authenticated weekly vendor context, including
inventory that can vary by class. Collections remain account-wide. Future
character-scoped surfaces should use verified Bungie equipment, progression,
and activity data rather than reinterpreting account-wide collection data.

Expected shape:

- character detail endpoints
- character-scoped UI routes or panels
- clear distinction between account-wide collection data and per-character data

### 3. God-Roll and Owned-Roll Insights

**Status:** Not implemented
**Gate:** Bungie item-instance verification plus data-source decision
**Likely size:** Large

Show a player's owned weapon rolls and compare them to curated or user-defined
target rolls.

Decisions to settle:

- Which item-instance components are required.
- Whether roll recommendations are user-authored, curated, or imported from a
  third-party wishlist format.
- Licensing and freshness expectations for any external roll source.

### 4. Notifications and Digests

**Status:** Not implemented
**Gate:** Product and provider decision
**Likely size:** Large

Send opt-in reminders when weekly or vendor data contains missing or wishlisted
items.

Decisions to settle:

- Email provider and sending domain.
- User preference model and unsubscribe flow.
- Whether the app should run scheduled jobs before production deployment exists.

### 5. Shareable Collection Progress

**Status:** Not implemented
**Gate:** Security and privacy review
**Likely size:** Medium to large

Allow users to create public read-only snapshots of collection progress without
exposing private account details or authenticated endpoints.

Expected shape:

- explicit snapshot creation
- revocable share token
- unauthenticated public page with minimal data
- rate limiting and abuse controls

## Operations Backlog

### 7. Production Deployment Path

**Status:** Deferred
**Gate:** Maintainer deployment decision
**Likely size:** Large

CI currently validates tests and Docker builds. Publishing images, provisioning
cloud resources, and deploying production infrastructure should be implemented
only after the target hosting model is accepted. Environment-specific runbooks
belong in `private/` until then.

### 8. Metrics

**Status:** Not implemented
**Gate:** Production observability decision
**Likely size:** Medium

Structured request/access logging with request IDs and sanitized identifiers has
shipped (see [docs/architecture.md](./docs/architecture.md#request-logging)).
Metrics remain unimplemented; add them only after the production runtime and
collector target are known.

## Completed Work

Completed work is intentionally not duplicated here. See [CHANGELOG.md](./CHANGELOG.md)
for shipped changes.
