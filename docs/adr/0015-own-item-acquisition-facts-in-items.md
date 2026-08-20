# ADR 0015: Own Item Acquisition Facts in Items

- Status: Accepted — implementation sequenced in [#172](https://github.com/jwh3times/GuardianTracker/issues/172)
- Date: 2026-08-16

## Context

Collections and the Wish List independently join Manifest item definitions to
every linked Collectible, then project the same name, icon, rarity, item type,
and acquisition-source union. The projections have already drifted: Collections
uses slot-specific armor types while the Wish List reports generic `Armor`.
Wish List enrichment also performs separate item and Collectible reads and
silently converts Manifest failures into plausible-looking unknown or empty
facts.

`services/manifest` owns raw SQLite access, and `services/sources` already owns
the meaning of an individual acquisition source and its difficulty and
raid/dungeon facets. Neither should own Guardian Tracker's complete,
user-independent Item projection. `services/items` already owns Manifest-derived
Item detail, bounded caches, and Manifest-observer invalidation, making it the
natural seam for the canonical facts.

## Decision

`services/items` owns **Item acquisition facts** through one named
`AcquisitionFactsReader` interface:

```go
type AcquisitionFactsReader interface {
	Lookup(ctx context.Context, hashes []uint32) (map[uint32]AcquisitionFacts, error)
	Catalog(ctx context.Context) ([]AcquisitionFacts, error)
}
```

`Lookup` resolves any inventory Item, including non-collectible deep links.
`Catalog` returns every distinct Item linked to at least one Collectible. It is
ordered by Item hash. The interface has no single-item convenience method,
projection flags, open-ended query language, or presentation-tree result.

`AcquisitionFacts` contains only Manifest-derived, user-independent facts:

- Item hash, name, description, and icon;
- slot-specific item type, tier type, rarity, exotic status, and collection
  category;
- every linked Collectible hash, unique and ascending;
- the deterministic acquisition-source union produced by
  `services/sources.DescribeAll`;
- the existing representative-Collectible `FarmOnly` result.

It exposes no raw Bungie definitions, representative Collectible identity,
membership ownership, wish-list metadata, live availability, presentation-tree
policy, or wire-format decisions. `services/sources` remains the owner of each
source's text, difficulty, and raid/dungeon facets.

### Result and error contract

- A known Item with no linked Collectibles is successful and has allocated empty
  Collectible and acquisition-source slices.
- An unknown hash is absent from the successful `Lookup` result.
- Empty lookup input returns an allocated empty map without Manifest access.
- Manifest unavailability, context cancellation, corrupt data, and query failure
  return errors and no partial result. Unavailability must never masquerade as an
  unknown Item or empty sources.
- Existing wish-list entries whose Item disappeared may render an explicit
  unknown-item tombstone. Adding an unknown Item remains invalid.
- Returned values are immutable by contract and copied where needed so callers
  cannot mutate reusable state.

### Coherence and publication

Items owns one immutable full-catalog publication per Manifest generation and
retains a 4,096-entry bounded cache for selected facts, with no TTL. If every
requested hash is cached, `Lookup` may serve the current-generation values. If
any hash misses, the complete requested batch is reloaded coherently rather than
combining cached facts from one generation with queried facts from another.

Private combined repository reads hold the existing SQLite read lock across item
definitions and Collectibles. Raw Manifest types do not cross the Items seam.
Catalog and batch publication use the owner-local contract from ADR 0014; an
obsolete load may finish for its initiating caller but cannot republish reusable
state. Errors and unknown hashes are not cached.

## Ownership boundaries

- **Collections** consumes `Catalog` and retains presentation-tree construction,
  membership ownership, collected/missing derivation, summaries, counts, and
  live-availability overlays. It remains a Manifest observer until its complete
  seam is decided separately.
- **Wish List** consumes one `Lookup` and retains persisted priority, notes,
  ordering, timestamps, tombstones, and live availability.
- **Item HTTP handling** consumes `Lookup` and adapts the canonical facts to its
  existing wire response.
- **Perks and catalysts** remain neighboring Items capabilities outside
  `AcquisitionFactsReader`.

Implementation removes duplicated source grouping, Item projection, handler-local
raw Manifest interfaces, and obsolete implementation-shaped tests. The
[#172](https://github.com/jwh3times/GuardianTracker/issues/172) handoff sequences
it, and the real-Manifest verification below remains a gate the handoff runs
before this ADR's slice opens.

## Alternatives considered

### Keep projection in Collections and Wish List

This preserves the current package layout but fails the deletion test: Item
meaning, Manifest joins, error policy, and parity tests remain duplicated and
continue to drift.

### Put canonical facts in services/manifest

This would mix raw Bungie storage with Guardian Tracker's item type, category,
source-union, and farm-only meanings. Storage changes and product-language
changes would share the wrong locality.

### Add an opaque query language

A single query operation could admit new selectors without new methods, but it
would expose storage-shaped policy and invite projection masks that make known
empty values ambiguous.

### Return a collection-ready tree from Items

Moving the presentation tree would make Collections' dominant call trivial, but
would pre-decide its separate ownership decision and broaden Items beyond Item
facts.

## Consequences

- Collections, Wish List, and deep-link Item handling receive one canonical
  static projection and one explicit failure policy.
- Slot-specific armor types become consistent across Collections and Wish List.
- Live availability and user-authored state remain visibly separate joins.
- Interface tests replace duplicated projection tests; source-classification,
  Manifest-multiplicity, Collections ownership/tree/count, Wish List metadata,
  public wire, and frontend behavior tests survive at their existing seams.
- New Items tests cover known-empty, unknown, unavailable, deterministic source
  union, representative-only farm-only, defensive copies, bounded caching,
  coherent batches, and ADR 0014 stale-publication races.
- Before implementation, ADR 0003 requires a current real-Manifest comparison of
  multi-Collectible Items, acquisition-source parity, catalog cardinality, and
  representative ordering. If a deterministic representative order changes any
  farm-only outcome, implementation stops for a separate decision.
