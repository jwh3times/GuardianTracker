# Roadmap

**Updated:** 2026-07-11

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

### 1. E2E, Accessibility, and Visual Regression

**Status:** Not implemented
**Gate:** Test infrastructure spec
**Likely size:** Large

Add Playwright coverage for the main user journeys, then layer automated
accessibility checks and optional visual snapshots.

Initial target flows:

- login callback and protected routing
- collections browsing and item drawer
- wishlist CRUD
- search deep links
- This Week, catalysts, crafting, triumphs, settings, and admin smoke paths

### 2. Character-Scoped Surfaces

**Status:** Not implemented
**Gate:** Bungie response-shape verification
**Likely size:** Large

The character switcher currently drives display context. Future character-scoped
surfaces should use verified Bungie character, equipment, progression, and
activity data rather than assuming account-wide collection data can be safely
reinterpreted per character.

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

### 8. Structured Logging and Metrics

**Status:** Not implemented
**Gate:** Production observability decision
**Likely size:** Medium

Move from ad hoc logs to structured logs with request IDs, sanitized identifiers,
and deploy-target-appropriate aggregation. Add metrics only after the production
runtime is known.

## Completed Work

Completed work is intentionally not duplicated here. See [CHANGELOG.md](./CHANGELOG.md)
for shipped changes.
