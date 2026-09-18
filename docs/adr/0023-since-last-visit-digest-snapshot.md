# ADR 0023: Snapshot-Owned Since-Last-Visit Digest

- Status: Accepted (not implemented)
- Date: 2026-09-18

## Context

Guardian Tracker shows current state well — `This Week` already renders
milestones and activities, recommendations, Xûr, and reset countdowns — but it
can answer no question of the form "what changed while I was away". Every
surface is a projection of the present.

The board's **Notifications and digests** item covers the same user value, but
its premise is scheduled email, which depends on an always-on public host.
Guardian Tracker is local-only by owner decision (2026-09-08, reconfirmed
2026-09-17), and a local application is not running during precisely the
interval a notification would describe. That item stays parked. This decision
covers the in-app portion, which needs no host.

Two facts constrain the design, both verified against primary sources rather
than assumed.

**Bungie exposes no acquisition timestamp.** In the Bungie.Net API
`openapi.json`, `Destiny.Components.Collectibles.DestinyCollectibleComponent`
declares exactly one property, `state`, an `int32` bitmask. The repository's
`bungie.CollectibleComponent` is therefore a complete model of it, and nothing
in a collectible says when it was acquired. "What is new" cannot be derived from
a single read.

**The game's own recency lists are not a substitute.**
`DestinyProfileCollectiblesComponent` carries `recentCollectibleHashes` and
`newnessFlaggedCollectibleHashes`, both described as collectibles "determined by
the game as having been 'recently' acquired". Neither defines the window, and
the specification says of the second that "the game client itself actually
controls this data, so I personally question whether anyone will get much use
out of this: because we can't edit this value through the API". They run on the
game's clock, not the application's, and cannot anchor a digest keyed to a visit
boundary.

The application also persists no history. Seven migrations define `users`,
`bungie_tokens`, `feature_flags`, `refresh_sessions`, `audit_log`, `role_audit`,
`user_preferences`, and `wishlist_items`. Collections are fetched live and
cached in memory only. There is nothing to diff against.

## Decision

A new membership-keyed store owns a **snapshot of the collected set** and the
**visit clock**. The digest is the difference between a fresh read and that
snapshot.

### Visit boundary

A visit begins on a page load or login occurring more than **two hours** after
the previous recorded activity. Two timestamps are required:

- `last_activity_at` — written on every page load and login.
- `visit_started_at` — the anchor of the current visit.

On activity at time `T`:

1. If `T - last_activity_at > 2h`, or no record exists, a new visit begins: the
   digest is computed, the snapshot is replaced, and `visit_started_at = T`.
2. `last_activity_at = T` unconditionally.

The digest is therefore **computed once per visit and frozen for its duration**.
Navigating away and back within a visit re-renders the same digest rather than
an empty one. Items acquired while the application is open appear in the next
visit's digest, which is the correct reading of "since last visit".

### Diff is additive only

Destiny 2 collections are monotonic: a collectible, once acquired, is not lost.
The digest therefore has exactly one section — acquired — computed as the set
difference `current − snapshot`. Loss is never surfaced, and no reconciliation
path exists for it.

### The snapshot write is gated on privacy

`profileCollectibles` carries a `Privacy` field
(`None = 0`, `Public = 1`, `Private = 2`). A `Private` read returns an empty
collectibles map. Because losses are never displayed, an empty read looks
harmless — but writing it as the snapshot baselines the account at zero, and the
following visit reports every collectible as newly acquired.

The snapshot is written **only** when `Privacy == Public` and the read
succeeded. A failed or private read yields no digest and no write; the previous
snapshot stands. Under a successful public read a shrinking set is anomalous
rather than impossible — deleting all characters is the owner-identified case —
so the new baseline is accepted and logged, which lets that situation resolve
itself instead of wedging the account permanently.

### Availability stays live

Vendor availability changes on a rotation schedule the application already
knows, not in response to anything the player did. It is read live and is never
part of the snapshot. The snapshot holds acquisition only.

### Freshness requires no forced refresh

`CACHE_TTL_COLLECTIONS` defaults to five minutes, and the cache is in-process. A
visit requires a gap of two hours, and nothing warms the cache while the
application is closed, so the first collections read of a visit is always past
its expiry and fetched fresh. No refresh is forced.

This is a coupling, not a coincidence: it holds while the collections cache TTL
stays below the visit-gap threshold. Changing either value without the other
reintroduces the possibility of diffing stale state.

## Boundaries

- The store owns the snapshot, the visit clock, and nothing else. It does not
  own presentation, wording, or ordering.
- The digest is a read of Collections; it does not re-derive acquisition facts,
  which remain owned by Items per
  [ADR 0015](./0015-own-item-acquisition-facts-in-items.md).
- Preferences do not own the visit clock.
  [ADR 0021](./0021-own-preferences-synchronization.md) scopes the preferences
  client to a small object of user choices, exempt from the membership-refresh
  fan-out because a Bungie refresh cannot change it. A visit marker is observed
  state tied to data freshness and does not belong there.
- Membership scope is the **pair** — platform type and identifier — consistent
  with every other membership-scoped route.
- This decision does not unpark **Notifications and digests**. Email delivery,
  sending domain, unsubscribe, and scheduled jobs remain gated on the public
  deployment decision.

## Migration and test surface

- Migration `0008` adds `digest_state`, keyed on
  `(membership_type, membership_id)`, holding `last_activity_at`,
  `visit_started_at`, `snapshot`, and `snapshot_taken_at`. One row per
  membership, replaced atomically; no history table and no row per collectible.
- The store is an interface with a degraded implementation, never nil, so the
  no-database development mode continues to boot.
- `bungie.ProfileResponse` gains `recentCollectibleHashes`. Component 800 is
  already requested and the field already arrives on the wire, where the decoder
  currently discards it. It is captured as a corroborating signal only; no
  behavior keys off it.
- Tests must cover: a private read writing no snapshot; a failed read writing no
  snapshot; the first visit rendering "tracking starts now" rather than an empty
  digest or a full-collection dump; a within-visit reload returning the same
  frozen digest; and a shrinking public read accepting the baseline without
  surfacing loss.

## Alternatives considered

**Derive newness from `recentCollectibleHashes` alone.** Rejected: the window is
undefined and client-controlled, as the specification itself cautions. It cannot
express "since your last visit".

**Store the visit clock in preferences.** Rejected: it would widen ADR 0021's
deliberately narrow contract and place freshness-dependent state inside the one
module exempted from the membership-refresh fan-out.

**Keep the clock in browser storage.** Rejected for correctness: it is per-device
and lost when site data is cleared. Attractive because it needs no migration,
but the snapshot needs a durable home regardless, so it saves nothing.

**A thin digest needing only a timestamp** — reset and Xûr arrival, derived from
the known cadence with no snapshot. Rejected as not worth shipping alone: the
information is already visible on `This Week`, so it restates what the player can
see rather than telling them what changed.

**Snapshot availability as well as acquisition.** Rejected: availability changes
on a schedule rather than through player action, so a stored diff would persist
state that is already computable live.

## Consequences

- The application gains its first persisted record of prior Bungie-derived
  state. Until now every projection was of the present, and this introduces a
  durable baseline that can disagree with reality if written from a bad read —
  which is why the privacy gate is on the write rather than the display.
- The digest is only as good as visit boundaries. A player who opens the
  application every hour never crosses the two-hour gap and never sees a digest.
  This is accepted: such a player has not been away.
- Collections cache TTL and the visit gap become coupled values.
- **Notifications and digests** remains parked and is not partially satisfied by
  this work. If public deployment is ever reconsidered, the snapshot here is a
  usable foundation for a delivered digest, but no part of that is decided now.
