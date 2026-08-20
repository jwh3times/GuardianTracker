# ADR 0018: Own Complete Membership Collections

- Status: Accepted — implementation sequenced in [#172](https://github.com/jwh3times/GuardianTracker/issues/172)
- Date: 2026-08-17

## Context

The Collections service currently returns an incomplete result. It owns the
Bungie collectible-profile read, Item-level ownership, presentation-tree
overlay, summary counts, and missing-item derivation, but the Gin handler still
chooses the summary or full projection, asks Weekly for live availability,
intersects that map with tracked Items, and mutates the result. The refresh
handler also knows that a manual refresh must invalidate Collections,
Characters, and Records.

This split hides important behavior at an HTTP boundary. A full collection is
not complete until canonical Item facts, membership ownership, and live
availability agree by Item hash. A membership data refresh is broader than the
route name: narrowing it to the Collections cache would let subsequent
Characters and Records requests reuse stale backend data.

The current invalidation is also insufficient under concurrency. Collections,
Characters, and Records can begin an upstream load, have their cache entry
deleted by refresh, and then publish the older result after refresh returns.
Manual refresh can therefore claim that the next request will fetch fresh data
while allowing pre-refresh work to become reusable again.

Moving Weekly directly into the existing Collections constructor would create a
runtime construction cycle. Weekly requires Collections' missing-item reader,
while a complete Collections result requires Weekly's established live-
availability capability. The design must break that cycle without a late setter,
optional dependency, or handler-supplied join.

ADR 0015 also changes the Manifest side of this flow: Collections will consume
canonical Item acquisition facts from Items while retaining presentation-tree
and membership-ownership policy. That introduces a cross-observer ordering
constraint which the complete seam must make explicit.

## Decision

`services/collections` has two construction stages within one owning package.

`MembershipAnalysis` is constructed first. It owns the reusable membership and
Manifest analysis needed to derive ownership, tree counts, summaries, and
missing Item hashes. Weekly consumes this concrete core only through its
existing consumer-side `weekly.MissingItemReader` interface.

The outer `collections.Service` is constructed after Weekly. It owns the
complete handler-facing use cases and receives:

- the required `MembershipAnalysis` core;
- a required consumer-side `LiveAvailabilityReader`, satisfied in production by
  Weekly's existing narrow availability capability; and
- required Characters and Records membership-refresh participants.

There is no setter, late-bound closure, nil-degraded production path, or
per-request availability callback. The resulting production dependency graph is
Items → MembershipAnalysis → Weekly → complete Collections Service → Gin.

### Public operations

The complete service exposes three explicit operations:

```go
type MembershipRequest struct {
	MembershipType int
	MembershipID   string
	AccessToken    string
}

type Membership struct {
	MembershipType int
	MembershipID   string
}

func (s *Service) GetSummary(ctx context.Context, req MembershipRequest) (Summary, error)
func (s *Service) GetFull(ctx context.Context, req MembershipRequest) (Full, error)
func (s *Service) RefreshMembership(ctx context.Context, membership Membership) error
```

`GetSummary` and `GetFull` are different capabilities rather than a boolean
projection flag or a result with optional meaning. Both return the same ordered
presentation roots, tree counts, four-category summary, and ownership-profile
`FetchedAt`. `Summary` has no Item surface and performs no availability work.

`Full` adds node Item hashes and Collections-owned `CollectionItem` values. Each
`CollectionItem` combines:

- the canonical, immutable Item acquisition facts from ADR 0015;
- the membership's Item-hash-scoped collected state; and
- optional `AvailableFrom` vendor text for the independent live-availability
  fact.

Ownership remains true when any Collectible linked to the Item is acquired.
Tree leaves remain deduplicated by Item hash within each node while permitting
the same Item under distinct nodes; summary totals remain globally deduplicated
by Item hash. Root, child, leaf, and first-seen Item order remain observable
contract behavior.

The HTTP adapter mechanically normalizes a `Full` result into the existing
`items`, `collectedHashes`, and `availableNow` fields. It does not reconstruct
or reinterpret completeness. The current default and exact `?include=all` wire
shapes remain unchanged.

### Analysis ownership

After ADR 0015, `MembershipAnalysis` consumes
`items.AcquisitionFactsReader.Catalog` and a presentation-node reader. Raw Item
definitions and Collectible rows no longer cross from Manifest into
Collections. The analysis maps Catalog facts through their linked Collectible
hashes to the presentation tree, then overlays Bungie's profile- and character-
level collected Collectibles to derive Item ownership.

The core retains the current immutable, copy-on-write split:

- rate-limited Bungie collected state and its `FetchedAt` survive an ordinary
  Manifest swap; and
- Manifest-derived Catalog/tree/owned projections rebuild coherently for the
  new generation.

A cached read during a mid-swap Manifest gap may retain the prior coherent
Manifest projection as specified by ADR 0014. A cold read still reports
Manifest unavailable rather than returning empty success.

### Live availability

Collections declares a required consumer-side `LiveAvailabilityReader`. Its
production adapter preserves the current verified item-hash-to-vendor map,
empty-character fallback, character-scoped vendor behavior, exact vendor names,
and Xûr-last tie precedence.

Only `GetFull` invokes it, after the core result succeeds. Collections intersects
the returned map with tracked Items and applies it to a fresh per-request result;
it never mutates or caches availability inside the longer-lived membership
analysis. Availability remains best effort: unavailable live data produces an
empty overlay and does not fail an otherwise valid Collections result. The
result's `FetchedAt` remains the ownership-profile fetch time, not the
availability-join time.

This decision keeps the established narrow capability in Weekly for the scoped
migration. A future decision may extract a shared availability owner, but this
implementation must not duplicate Bungie calls or vendor caches to simulate one.

### Authorization and HTTP boundary

Gin retains route/query binding, HTTP authentication, Bungie-token resolution,
error-to-status mapping, and JSON serialization. Before token lookup or service
access, it compares both membership type and membership ID from the access JWT
with the requested route. A Destiny membership is the pair, so the current
ID-only check is replaced as a verified correctness fix.

The complete service receives an already-authorized membership and resolved
Bungie access token. It owns collected-state and projection semantics, not JWT,
token-store, or HTTP policy. Existing public error statuses, codes,
`retryAfter`, and messages remain unchanged.

### Membership data refresh

`RefreshMembership` remains an invalidation command, not an eager Bungie fetch.
It owns exactly the existing participant set:

1. Collections membership analysis;
2. Characters; and
3. Records.

Each owner implements a required membership-refresh participant contract and
owns a monotonic generation per membership. A loader captures its owner's
generation before work begins and may publish to the cache only if that
generation remains current. Refresh advances the generation and invalidates the
owner's cache entry as one owner-local transition.

The complete service advances all three participants synchronously before
returning. Pre-refresh work may finish for the request that initiated it, but
cannot become reusable after refresh returns. The coordination does not change
ADR 0013's general cache API, capacities, TTLs, or eviction policy.

Weekly public/vendor caches, Wish List, Preferences, shared Manifest state, and
Items are not refresh participants. Weekly-facing client queries may refetch and
observe the newly derived missing set, but the server's vendor caches retain
their existing daily/reset policies.

### Manifest publication order

Items must advance its Manifest observer before Collections. Once Items owns
the Catalog, notifying Collections first could let Collections pair new
presentation nodes with a still-published old Catalog and publish that mixture
under its new generation.

The composition root therefore registers Items before Collections. Collections
captures one ADR 0014 publication attempt across Catalog access,
presentation-node access, tree construction, and reusable analysis publication.
The Manifest provider remains reopened before any observer runs, as required by
ADR 0010.

## Boundaries

- Items owns user-independent Item acquisition facts, linked Collectible hashes,
  source unions, category, and representative-only farm classification.
- Collections owns presentation-tree policy, membership ownership, collected
  and missing derivation, counts, summaries, complete collection outcomes, and
  the live-availability intersection.
- Weekly remains the current owner of the narrow live-availability capability
  and consumes only the missing-item interface from MembershipAnalysis.
- Characters and Records own their cache keys and publication fences;
  Collections invokes their refresh contracts without learning those keys.
- The frontend's `CollectionsSummaryView` and `CollectionsView` remain the
  complete presentation adapters. Feature modules continue to avoid the raw
  wire shape.
- Farm-only multi-Collectible semantics, broad vendor features, source-string
  consolidation, and presentation-root policy changes remain out of scope.

## Migration and test surface

Implementation replaces the split rather than wrapping it indefinitely:

1. Implement ADR 0015's Item Catalog and real-Manifest verification gate.
2. Extract the current reusable analysis into `MembershipAnalysis`, consuming
   Catalog plus presentation nodes, and apply the ADR 0014 publication fence.
3. Keep Weekly's `MissingItemReader` pointed at MembershipAnalysis.
4. Add the outer Service, typed Summary/Full outcomes, complete CollectionItem,
   live-availability join, and current refresh participants.
5. Move projection choice, availability overlay, and refresh fan-out from Gin to
   the complete service; delete `Lightweight`, handler overlay logic, and direct
   handler dependencies on Characters, Records, and Weekly after replacement
   coverage is green.
6. Add owner-local per-membership publication fencing to Collections,
   Characters, and Records without changing the general cache.
7. Register Items before Collections and capture one Collections publication
   generation across Catalog/tree analysis.
8. Strengthen route ownership validation to the complete membership pair while
   preserving the existing HTTP wire and error contract.

Collections contract tests replace implementation-shaped handler storage,
overlay, and invalidation tests. They cover Summary and Full separately,
full-only availability, tracked-Item intersection, best-effort empty
availability, complete immutable CollectionItems, core-error short-circuit,
profile/character ownership union, any-linked-Collectible ownership,
deterministic hashes/order, and exact refresh participation.

Deterministic concurrency tests block each participant's load, refresh the
membership, release the old work, and prove the old result is not reused. ADR
0014 tests block Catalog/tree work across a Manifest advance and prove obsolete
analysis cannot publish. An integration test pins Items-before-Collections
observer ordering.

Tree dominance, nesting, nameless/redacted branches, cycle handling, per-node
deduplication, global summary deduplication, missing-item filtering, retained
profile state across swaps, and `FetchedAt` behavior survive under the analysis
interface. Item projection/source/farm tests move to Items as required by ADR 0015. Handler tests retain membership-pair authorization, binding, exact
summary/full serialization, refresh response, and error mapping only.

Frontend `CollectionsSummaryView`/`CollectionsView` tests, query-variant tests,
availability filters and badges, Cosmetics parity, deep links, Dashboard
summary behavior, and privacy/warming/error states survive. The shared frontend
membership-refresh adapter must continue invalidating both Collections variants
and the existing Characters, Weekly, Catalysts, Crafting, and Seals query
families; page tests retain visible pending behavior.

The ADR 0015 real-Manifest verification of multi-Collectible parity, Catalog
cardinality, source unions, and representative ordering remains a gate before
implementation. If that evidence changes farm-only behavior or presentation
cardinality, implementation stops for the separate decision required by ADR 0003.

## Alternatives considered

### Keep completion and refresh coordination in Gin

This fails the deletion test. Removing the handler redistributes projection
selection, availability intersection, refresh participants, and concurrency
guarantees instead of leaving one complete Collections capability.

### Inject Weekly into the original Collections service

Weekly already requires Collections for missing Items, creating a construction
cycle. Setter injection, nil guards, and late-bound closures replace the cycle
with a temporally invalid service and are not accepted.

### Pass resolved availability through the handler

This lets Collections perform the final map intersection but leaves the HTTP
adapter coordinating domain inputs and deciding when availability applies. The
seam would still be incomplete.

### Extract a standalone availability module in this decision

Availability has multiple consumers and may eventually justify that owner, but
its current implementation shares Weekly's Xûr, character resolution, and raw
vendor caches. Extracting all of those correctly would materially broaden #166.
The two-stage Collections construction resolves the immediate cycle without
duplicating calls or pre-deciding that larger migration.

### Delete cache entries without publication fencing

This preserves the current code shape but lets older in-flight work republish
after refresh. Cache deletion is an event, not a freshness guarantee, unless the
owning loader also participates in the generation protocol.

## Consequences

- Gin receives one complete Collections use case instead of coordinating domain
  completion and sibling cache behavior.
- Summary reads stay cheap, while Full reads always return a coherent ownership
  and best-effort availability outcome.
- Weekly can continue reading missing Items without a construction cycle or a
  broad dependency on the complete service.
- Manual refresh has an enforceable post-return freshness boundary across its
  current three backend participants.
- Item and Collections Manifest publications cannot be observed in the wrong
  advancement order.
- The complete membership-pair authorization check closes the platform-type
  mismatch before credentials or cached data are touched.
- Implementation is sequenced by the [#172](https://github.com/jwh3times/GuardianTracker/issues/172) handoff and proceeds slice by slice.
