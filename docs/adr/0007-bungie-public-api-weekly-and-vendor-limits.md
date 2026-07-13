# ADR 0007: Bungie Public API Weekly and Vendor Limits

**Status:** Accepted
**Date:** 2026-07-08

## Context

Guardian Tracker uses Bungie's public Destiny APIs and manifest data to surface weekly
activity, vendor, and collection-availability signals.

Some player-visible weekly facts are not exposed as reliable public API data. Previous
work verified that broad "live this week" panels and general vendor-rotation cards can
look plausible in code while producing empty, redundant, or misleading UI from live data.

Known limitations include:

- Xur inventory is usable from the public API, but location is available only from the
  authenticated character-vendor component's `vendorLocationIndex` plus manifest data.
- Trials, Iron Banner, Nightfall, and comparable featured activity rotations are not
  reliably represented as public milestone definitions.
- Character vendor component data can be useful for item-level availability, but it is
  not a reliable source for a broad browsable vendor-rotation panel because many rotating
  rewards are gated behind focusing or other game systems.

## Decision

Do not build broad weekly activity or vendor panels from assumed Bungie public API
signals. Weekly and vendor features must use verified data paths and should omit claims
the API cannot support.

The app may continue to use verified item-level availability signals, such as Xur
inventory and the established live-vendor availability map, when those signals are
presented narrowly and best-effort.

Xur location may use the authenticated character-vendor component 400. Resolve its
`vendorLocationIndex` through `DestinyVendorDefinition.locations` and the destination
manifest definition. Present destination `1737926756` as the user-facing label
`The Tower`; omit the field when the character response or manifest lookup is unavailable.

If a future feature needs Trials, Iron Banner, Nightfall, featured raid/dungeon rotation,
or broad vendor browsing, it must first verify a reliable source. Static
rotation tables or third-party sources require an explicit follow-up decision covering
maintenance, freshness, licensing, and failure behavior.

## Consequences

- Xur remains the only weekly vendor module with a dedicated public UI unless another
  reliable data source is accepted.
- Xur's Tower location is best-effort and disappears rather than displaying an unknown
  or guessed value when authenticated vendor data cannot be resolved.
- "Available now" item badges can remain best-effort and item-scoped.
- Roadmap and implementation plans should not reintroduce broad weekly/vendor panels
  without a verify-first spike.
- Missing or unverifiable data should be hidden or qualified rather than guessed.
