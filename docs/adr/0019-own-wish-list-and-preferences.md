# ADR 0019: Own Wish List and Preferences

- Status: Accepted — implementation sequenced in [#172](https://github.com/jwh3times/GuardianTracker/issues/172)
- Date: 2026-08-17
- Superseded in part by
  [ADR 0021](./0021-own-preferences-synchronization.md): the Boundaries
  statement that this ADR "does not change the existing REST wire" no longer
  holds for the additive `persisted` field on `GET /api/preferences`. Every
  other decision here, including PUT's `503` and GET's `200`, remains in force.
- Implementation note (2026-08-24): the Preferences backend slice is complete.
  `services/preferences`, its membership-keyed database adapter, the atomic
  storage `Apply`, and `PreferencesHandler` now implement the Preferences
  decisions below. Wish list ownership slices B3/B5 remain pending, so this ADR
  remains Accepted with sequenced implementation rather than Implemented.

## Context

Wish list behavior is currently split across Gin handlers, database stores,
Manifest access, Weekly availability, and Weekly personalization. Handlers know
PostgreSQL-shaped rows and errors, validate operations, resolve Item details,
join live vendors, and assemble the public result. Weekly reaches the database
through its own adapter to read Wish list hashes. Deleting the handler would
therefore redistribute the meaning of a Wish list entry instead of leaving one
complete capability.

Preferences have a related but distinct problem. Their handler owns defaults,
validation, onboarding completion, and a sequence of field-level writes.
Concurrent partial updates can overwrite one another because the store has no
single atomic patch operation. Persistence unavailability is also intentionally
asymmetric on the current wire: a read returns defaults, while a write fails.

A complete Wish list read requires three facts with different owners:

- persisted user-authored entry metadata;
- canonical, user-independent Item acquisition facts from ADR 0015; and
- independent, best-effort live availability.

Weekly already consumes Wish list Item hashes and currently supplies the live
availability capability. Constructing one complete Wish list service before
Weekly would create a runtime cycle; constructing it only after Weekly would
leave Weekly without its required hash reader. The design must break that cycle
without a setter, optional dependency, handler callback, or direct database
escape hatch.

The current note limit also contains a verified correctness defect. Go validates
the byte length while PostgreSQL constrains character length, so valid Unicode
notes can be rejected before persistence. The domain limit is 500 Unicode code
points.

## Decision

Wish list and Preferences become separate backend owners:

- `services/wishlist` owns persisted Wish list entries, Item validation,
  completion with canonical facts and availability, mutations, and the narrow
  hash view consumed by Weekly.
- `services/preferences` owns preference defaults, validation, atomic partial
  updates, and irreversible onboarding completion.

They remain separate because a Wish list entry is an Item-oriented aggregate
with live completion, while a preference is a small Guardian Tracker user
setting. Combining them would produce a broad “user data” service with no
cohesive invariant.

### Two-stage Wish list construction

`wishlist.Entries` is constructed first around a required Wish list repository
and the ADR 0015 Item lookup capability. It owns persistence, mutation
validation, Item-existence rules, and the membership-scoped Item-hash view.

Weekly declares the one method it consumes:

```go
type WishListReader interface {
	ListItemHashes(ctx context.Context, membershipID string) ([]uint32, error)
}
```

The production adapter is `wishlist.Entries`; Weekly tests use a fake. Weekly
retains its current fail-open behavior when Wish list personalization cannot be
read, because recommendations remain usable without that optional signal.

After Weekly is constructed, the outer `wishlist.Service` is constructed around
the required Entries core, a required consumer-side `LiveAvailabilityReader`,
and a best-effort Bungie credential reader. The resulting dependency order is:

Items → Wish list Entries → Weekly → complete Wish list Service → Gin.

There is no setter, late-bound closure, nil-degraded production dependency, or
Gin-supplied completion callback. Database adapters do not bypass Entries to
serve Weekly.

### Persistence ports and adapters

Wish list declares a consumer-side, membership-keyed repository. Its methods
return Wish list domain stored records and typed domain errors. The adapter in
`db/adapters` owns:

- resolving the internal Guardian Tracker user ID from the Destiny membership;
- translating PostgreSQL rows into stored entry records; and
- translating unavailable, duplicate, and not-found storage failures into the
  Wish list error vocabulary.

`db.WishlistItem`, `pgx`, `pgconn`, constraint names, and SQL error codes do
not cross into Wish list services or Gin. A degraded repository returns a typed
unavailable error rather than an empty list.

The repository preserves creation order. Returned slices and acquisition-source
collections are allocated by the owning service, so callers cannot mutate
cached or adapter-owned state.

### Complete Wish list operations

The handler-facing service exposes explicit operations:

```go
List(ctx, membershipID)
Add(ctx, membershipID, AddCommand)
Update(ctx, membershipID, EntryID, UpdateCommand)
Remove(ctx, membershipID, EntryID)
DeleteMany(ctx, membershipID, []EntryID)
SetPriorityMany(ctx, membershipID, []EntryID, Priority)
```

Gin translates route parameters and the existing bulk action string into these
typed operations, maps typed errors, and mechanically serializes the existing
JSON. It does not perform repository calls, Item lookups, availability joins,
or domain validation.

`List`, `Add`, and `Update` return complete entries and therefore perform the
necessary Item and live-availability completion. `Remove`, `DeleteMany`, and
`SetPriorityMany` do not fetch Item facts, credentials, or vendors. An empty
List result short-circuits Item lookup, credential resolution, and availability.

Add and Update return the same complete entry shape as List. Add rejects an Item
absent from a successful current Item lookup and performs no write. A lookup
failure is an explicit error and also performs no write.

Update first reads the stored row and resolves its current Item state before
mutating it. This ordering distinguishes an established unknown-Item tombstone
from a transient lookup failure: metadata may be updated for a confirmed
tombstone, while a lookup failure causes no write.

### Entry outcome and Item completion

The service returns repository-ordered `[]Entry`. Each Entry includes:

- typed saved entry ID;
- Item hash;
- typed priority;
- notes;
- creation time;
- optional best-effort availability; and
- a tagged `KnownItem | UnknownItemTombstone` Item state.

A KnownItem carries the canonical ADR 0015 acquisition facts with an allocated
acquisition-source collection. List resolves all stored hashes through one
batched Item lookup rather than one Manifest call per entry.

Three Item outcomes remain distinct:

1. a successful lookup containing the hash produces KnownItem;
2. a successful lookup without a persisted hash produces
   UnknownItemTombstone; and
3. a lookup failure fails the whole operation.

List never drops a stored row or returns a partially completed success because
Item facts are unavailable. A tombstone retains the user's metadata and the
existing visible fallback projection: name “Unknown Item,” type “Item,” rarity
“Common,” and empty icon and acquisition sources.

Live availability remains independent of provenance and priority. The outer
service asks its required availability reader once for the completed Item set.
Credential lookup is best effort: failure supplies an empty Bungie token so
public Xûr availability can still survive. Availability failure produces an
empty overlay and does not fail an otherwise valid Wish list operation.

The existing wire remains exact: string IDs, uppercase priorities, title-case
rarity, UTC RFC3339 `dateAdded`, `acquisitionSources`, and independent
`availableNow`/`availableFrom`. No aggregate Item difficulty or legacy
`sources` field is reintroduced.

### Validation and bulk commands

Wish list owns these invariants before persistence:

- priority is one of Low, Medium, High, or Urgent;
- Add defaults omitted priority to Medium;
- Update is a partial patch;
- notes contain at most 500 Unicode code points;
- after duplicate input IDs are removed, bulk commands contain one through 100
  unique entry IDs; and
- the priority in SetPriorityMany is valid.

The service accepts duplicate input IDs and deduplicates them before validation,
persistence, and skipped-count calculation. Bulk results retain the existing
`{Updated, Skipped}` semantics:
missing or foreign entries count as skipped, valid owned entries succeed, and
their absence does not roll back the rest of the command.

The Unicode code-point rule replaces byte-length validation and aligns the Go
domain check with PostgreSQL `char_length`. This is a correctness fix; the
nominal 500-character product limit is unchanged.

### Preferences service

`preferences.Service` uses a consumer-side repository with an atomic
`Apply(ctx, membershipID, initial Values, Patch)` operation. The service passes
its defaults as the initial values for a missing row; the database adapter
translates them to `PreferenceInitial`, so persistence does not define domain
defaults. Patch represents field presence separately from the field value and
updates only supplied fields in one transactional statement. It replaces the
current sequence of field-level writes, preventing concurrent partial patches
from restoring stale values.

The service owns:

- defaults of framed display density, personalized recommendations enabled, and
  onboarding not completed;
- validation of display density as framed or compact;
- partial-patch semantics; and
- irreversible onboarding completion.

An onboarding-complete command is idempotent. Persistence stamps its server time
only when no completion timestamp exists; clients cannot supply, replace, or
clear that timestamp.

The persistence adapter distinguishes a genuinely new account, for which
defaults are the domain value, from persistence unavailability. The current
HTTP asymmetry remains during this decision: GET `/api/preferences` maps
unavailable persistence to defaults and 200, while PUT returns 503. The later
browser-preference synchronization decision may revise failure visibility, but
must not bypass this service or weaken atomic patches.
[ADR 0021](./0021-own-preferences-synchronization.md) resolved this: both status
codes stay exactly as described, and the provenance this adapter already
computes is exposed as an additive `persisted` field so the browser can tell a
genuinely new account from unavailable persistence.

`PreferencesHandler` performs authentication, binding, typed error mapping, and
serialization only. It does not own defaults, validation, write ordering, or
onboarding policy.

## Boundaries

- Items owns canonical Item acquisition facts and the meaning of a successful
  lookup with an absent hash. Wish list owns how a persisted entry represents
  that absence.
- Sources continues owning acquisition-source difficulty and raid/dungeon
  facets. Wish list does not invent an Item-level difficulty.
- Weekly owns its complete This Week response and preserves fail-open optional
  Wish list personalization. It consumes only Item hashes from Entries.
- Weekly remains the scoped production adapter for the narrow live-availability
  capability. Wish list owns when and how that best-effort fact completes an
  entry.
- Database stores own SQL and transactions; adapters translate storage
  identity and errors; domain services own behavior.
- ADR 0017 identity-bound cleanup must clear Wish list data and reset
  Preferences when the browser becomes anonymous or changes Destiny membership.
  Same-membership token refresh retains both.
- The frontend data-access decision owns query identity, projections, mutation
  helpers, optimistic coordination, and cross-feature invalidation.
- The frontend Preferences synchronization decision owns membership reset,
  stale-work fencing, write serialization, and visible failure state.
- This ADR does not implement those frontend decisions and does not change the
  existing REST wire. **Superseded in part:**
  [ADR 0021](./0021-own-preferences-synchronization.md) adds the `persisted`
  provenance field to `GET /api/preferences`. That is the only wire change; the
  status codes, methods, and every other field described here are unchanged.

## Migration and test surface

Implementation replaces the split rather than layering another facade:

1. Implement ADR 0015's Item Lookup and real-Manifest verification gate.
2. Add Wish list domain types, repository ports, typed errors, and database
   adapters which hide internal user IDs and PostgreSQL details.
3. Introduce Entries, move persistence and Item validation behind it, and point
   Weekly's consumer-side hash reader at Entries.
4. Construct the complete Service after Weekly, move batched Item completion,
   tombstones, credential handling, availability, and operation validation into
   it, then delete handler-owned equivalents.
5. Replace byte-length note validation with Unicode code-point validation.
6. Add the Preferences atomic Apply repository operation and service, migrate
   defaults, validation, partial patches, and onboarding policy, then replace
   the combined handler path with PreferencesHandler.
7. Keep the public routes and JSON mechanically unchanged.

Wish list service tests replace implementation-shaped handler and storage tests.
They cover:

- priority, patch, note, and bulk validation, including Unicode boundary cases;
- Add default priority and no write when Item validation fails;
- Update of known Items and confirmed tombstones, with no write on lookup
  failure;
- one batched facts lookup, repository ordering, complete allocated facts, and
  the distinction between tombstone and lookup failure;
- empty-List short-circuit and the absence of Item/token/vendor calls for
  Remove and bulk operations;
- credential failure with public availability preserved, availability failure
  as an empty overlay, and exact vendor enrichment;
- duplicate, not-found, unavailable, and bulk updated/skipped typed outcomes;
  and
- preservation of the exact wire projection through thin handler tests.

Preferences service tests cover defaults, validation, atomic partial patches,
idempotent server-stamped onboarding, fresh-account versus unavailable
persistence, and the existing GET/PUT degraded-mode asymmetry.

Database integration tests retain membership ownership, creation order,
uniqueness, foreign-entry skipping, bulk transaction behavior, constraints, and
atomic concurrent preference patches. Weekly tests retain a fake one-method
WishListReader and assert fail-open personalization. Existing frontend Wish
list and Settings behavior remains unchanged at this stage.

## Alternatives considered

### Keep Wish list completion in Gin

This fails the deletion test. Removing Gin would redistribute Item completion,
tombstone policy, validation, availability degradation, and storage-error
translation across adapters and consumers.

### Construct one complete Wish list service before Weekly

Weekly needs Wish list hashes, while the complete service needs Weekly's current
availability capability. A direct dependency creates a runtime cycle. Setters,
late closures, optional readers, and nil degradation make service validity
depend on composition timing and are not accepted.

### Let Weekly read the database adapter directly

That breaks the owner boundary and gives two consumers independent meanings for
membership resolution, storage errors, and whether an unavailable Wish list is
empty. Entries is the narrow reusable core.

### Treat absent Item facts as an error or silently drop the row

A previously saved Item can disappear from a later Manifest. Failing or dropping
it would hide user-authored metadata. The explicit tombstone preserves that
state while keeping a transient lookup failure visibly different.

### Retain field-by-field preference writes

Serializing them in Gin would still leave persistence without one atomic patch
invariant and would not protect against concurrent callers. The repository must
apply supplied fields together.

### Combine Wish list and Preferences into one user-data service

They share membership scoping but not behavior, dependencies, lifecycle, or
domain invariants. The combined interface would be broad and shallow.

## Consequences

- Gin receives complete Wish list and Preferences outcomes and no longer knows
  persistence rows, SQL failures, Item joins, availability policy, or
  preference write ordering.
- Weekly keeps a narrow, early-constructed hash reader without a construction
  cycle or database bypass.
- Stored Wish list entries survive absence from a successful current Item lookup
  explicitly, while real lookup failures cannot masquerade as tombstones or
  partial success.
- Bulk semantics and the exact REST wire remain stable.
- Unicode notes are validated according to the product's character limit rather
  than their encoded byte count.
- Concurrent preference patches cannot restore stale fields, and onboarding
  completion remains irreversible and server-stamped.
- Implementation is sequenced by the [#172](https://github.com/jwh3times/GuardianTracker/issues/172) handoff and proceeds slice by slice.
