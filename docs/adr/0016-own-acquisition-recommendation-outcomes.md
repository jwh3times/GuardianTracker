# ADR 0016: Own Acquisition Recommendation Outcomes

- Status: Accepted — implementation sequenced in [#172](https://github.com/jwh3times/GuardianTracker/issues/172)
- Date: 2026-08-17

## Context

Efficiency currently returns an implementation-shaped `ScoredAction` containing
bucket identity, raw source text, string action kind, counts, score, wording,
and explanation. Weekly then reinterprets that result: it chooses the visible
emphasis, classifies difficulty from the raw source again, and owns a separate
Xûr and weekly-reset fallback path. The two paths have drifted. Ranked actions
serialize title-case difficulty values such as `Challenging`, while fallbacks
serialize lowercase values; the frontend casts both directly to a lowercase-only
design type. Ranked difficulty badges consequently lose their intended label
lookup and color without a type error.

Efficiency ranking and an acquisition recommendation are different domain
concepts. The former scores source buckets and also supports milestone missing
counts; the latter is the complete player-facing action, explanation, source,
difficulty, emphasis, and fallback policy. Absorbing all recommendation behavior
into Efficiency would mix Xûr and generic weekly guidance into the Manifest
index owner. Leaving the policy in Weekly would preserve the split that caused
the correctness defect.

## Decision

`services/recommendations` owns complete recommendation outcome types and
behavior. Weekly owns the consumer-side seam it needs:

```go
type AcquisitionRecommender interface {
	Recommend(input recommendations.Input) []recommendations.Recommendation
}
```

`Input` contains already-resolved, read-only facts: missing Item hashes, wish-list
Item hashes, live availability by Item hash, active milestone names, whether Xûr
is present, and the ordered Xûr Item facts needed by the existing fallback. The
module owns no Bungie, Manifest, or Postgres port and `Recommend` performs no
I/O. It reads only the Efficiency snapshot returned by its internal Ranker.

`Recommendation` contains exactly the outcome Weekly needs to serialize:

- stable ID, action text, explanation, and time estimate;
- typed action kind: `activity`, `vendor`, or `weekly`;
- optional source text, absent only for the source-less weekly-reset fallback;
- typed difficulty, source-derived when source text is present and action-scoped
  as `Moderate` for the preserved source-less fallback; and
- typed emphasis expressing the existing Activity, Vendor, Available now,
  Wishlist, Xûr, or Weekly presentation choice.

When source text is present, difficulty equals `sources.Difficulty(sourceText)`;
Xûr fallback actions therefore carry the explicit Xûr source and `Easy`. The
only source-less outcome is the generic weekly-reset fallback, whose kind is
`weekly`, difficulty is `Moderate`, and emphasis is Weekly. This exception
preserves current action-scoped behavior without inventing a Manifest source.

Raw score, source hash, ranking label, missing and wish-list counts, availability
flags, and featured flags do not cross the AcquisitionRecommender seam. They
remain internal facts used to choose and explain the result. Returned
recommendations are ordered, allocated, and immutable by contract. `Recommend`
never returns an error. When ranking cannot produce an action, the module
preserves the current behavior: up to five ordered Xûr wish-list or missing-item
actions, otherwise one generic weekly-reset fallback. It therefore returns at
least one recommendation.

### Ownership and adapters

`services/efficiency` retains the Manifest-derived source-bucket index, scoring,
score-descending order with source-hash tie-break, the six-action ranked limit,
and `MissingForMilestone`. Its candidate result is an internal input to
Recommendations rather than a type Weekly interprets. Recommendations preserves
the ranked order and owns final selection between ranked results and fallbacks,
action wording, explanation construction, source/difficulty coherence, emphasis
precedence, and fallback selection.

The internal ranking seam returns an explicit state rather than overloading an
empty slice:

```go
type Ranker interface {
	Rank(input RankInput) RankResult
}

type RankResult struct {
	State      RankState // cold | ready
	Candidates []RankedCandidate
}
```

`RankInput` is the missing-item, wish-list, live-availability, and active-milestone
subset of the public recommendation input; Xûr fallback facts do not enter the
ranking seam. `Candidates` is allocated even when empty, and its slice order is
the deterministic final rank order defined below.

`cold` means no complete index has ever been published. `ready` with no
candidates means the current snapshot genuinely produced no actionable match.
While a replacement index builds, the prior complete index remains `ready` in
accordance with ADR 0014. Both states select the same preserved weekly fallback
when there is no candidate, but their distinction survives inside Efficiency and
Recommendations tests so unavailable data cannot silently become empty data.
`RankedCandidate` carries source hash, label, source text, typed source kind,
missing and wish-list counts, and availability/featured flags; it carries no
score, final wording, explanation, emphasis, or difficulty. Candidate order is
part of the Ranker contract: Efficiency applies the existing score order,
source-hash tie-break, and six-action cap before returning them. Recommendations
preserves that order while turning the facts into complete outcomes.

Efficiency retains its existing non-blocking, retry-on-request index lifecycle.
This decision does not introduce a second lifecycle operation or move index-build
coordination into Weekly or Recommendations.

The external seam has two justified adapters: the production
`*recommendations.Planner` and a deterministic fake used by Weekly tests.
Recommendations' internal ranking seam likewise has the production Efficiency
engine and a fake used by Recommendations tests. These are in-process
dependencies; no remote port is introduced. Weekly continues gathering facts
because it already needs the same Bungie, wish-list, milestone, and Xûr data for
the rest of its response. It calls `Recommend` once and performs field-for-field
wire assembly only.

### Difficulty wire contract

`services/sources` remains the sole owner of difficulty classification and gains
a named difficulty type so action and source tiers cannot degrade to unrelated
strings inside the backend. The weekly wire uses the same canonical title-case
values as acquisition sources: `Easy`, `Moderate`, `Challenging`, and `Unrated`.

The frontend introduces one canonical raw union shared by acquisition sources
and weekly recommendations:

```ts
type APIDifficulty = "Easy" | "Moderate" | "Challenging" | "Unrated";
```

Raw weekly response types use that union and adapt it to the lowercase design
vocabulary through the same exhaustive difficulty adapter used for acquisition
sources. During migration the runtime adapter accepts both canonical title-case
and legacy lowercase spellings; an unknown value becomes the explicit `unrated`
state. Feature modules never cast weekly JSON directly to design types. This is
the planned correctness fix; recommendation content and ordering remain otherwise
unchanged.

## Boundaries

- Sources continues classifying source difficulty, raid/dungeon, and action-kind
  facets. Recommendations does not duplicate those tables.
- Weekly continues owning the complete This Week response, reset timing,
  milestones, Xûr inventory, today actions, data fetching, and degraded-mode
  behavior. It does not own recommendation policy.
- Efficiency continues owning `MissingForMilestone`; Weekly consumes it through
  a separate narrow `MilestoneMissingCounter` interface instead of depending on
  the concrete engine.
- Efficiency remains the owner-local Manifest observer for its index and retains
  ADR 0014's deliberate previous-complete-index fallback while a new generation
  builds. Recommendations owns no Manifest publication.
- No new broad vendor or featured-activity signal is introduced; ADR 0007 still
  limits inputs to verified data paths.
- This decision does not make source-bucket representative selection deterministic
  across unordered Manifest rows. That separate finding still requires the
  real-Manifest verification excluded from this Wayfinder map.

## Migration and test surface

Implementation replaces rather than layers the existing split:

1. Add the typed source difficulty, Recommendations outcome types, and the
   consumer-side `weekly.AcquisitionRecommender` interface.
2. Change Efficiency's exposed ranked value to the explicit-state, ordered,
   capped candidate result Recommendations needs, while preserving its current
   non-blocking index self-heal.
3. Move action wording, explanation, emphasis precedence, and both fallbacks
   from Efficiency/Weekly behind `Recommend`; change Efficiency's exposed ranked
   value so score remains internal and only ranked facts cross the internal seam.
4. Inject the AcquisitionRecommender into Weekly and delete `mapEngineActions`,
   `rankRecommended`, and `buildRecommended` after their behavior is covered at
   the new interface.
5. Replace Weekly's concrete Efficiency dependency for milestone counts with the
   consumer-side `MilestoneMissingCounter` interface.
6. Add raw frontend weekly types and the shared tolerant difficulty adapter,
   then route Dashboard and This Week through that projection.

Recommendations interface tests replace implementation-shaped Weekly mapping and
fallback tests. They cover deterministic selection, wording, explanation,
source/kind/difficulty coherence, ranked availability-over-wish-list emphasis
precedence, Xûr and generic fallback parity, stable IDs, and input immutability.
Efficiency's source-bucket deduplication, scoring, deterministic ordering,
tie-break, six-action cap, explicit cold-versus-ready state, existing index
lifecycle, and milestone union tests survive at its own interface. Weekly
composition tests use fake AcquisitionRecommender and MilestoneMissingCounter
adapters and assert verbatim assembly; one integration test combines the real
Recommendations and Efficiency implementations. Backend wire tests pin all four
title-case tiers. Frontend adapter tests cover the raw union, both spellings, and
unknown-to-`unrated`, and a This Week regression test proves a ranked
`Challenging` action receives the challenging badge color. Existing handler,
Dashboard, persistence, error-state, functional browser, and accessibility tests
survive; the corrected badge color may require a Linux visual-baseline update.

## Alternatives considered

### Keep interpretation and fallback policy in Weekly

This keeps the package graph small but fails the deletion test: recommendation
meaning remains split, Weekly must understand Efficiency internals, and tier or
emphasis changes still require coordinated edits and tests in multiple modules.

### Expand Efficiency into the complete recommendation owner

This avoids a new package, but conflates the glossary-distinct ranking and
recommendation concepts. Xûr fallback copy and generic weekly guidance do not
belong to the Manifest-derived index and milestone-count owner.

### Add a thin Recommendations wrapper

A pass-through around `Efficiency.Rank` would be shallow. The accepted module
earns its seam by owning the behavior currently split across Efficiency and
Weekly; it does not merely rename or forward `ScoredAction`.

## Consequences

- Efficiency has one contract for ranked order. Recommendations preserves that
  order and is the sole owner of final ranked-versus-fallback selection, wording,
  explanation, action kind, source, difficulty, emphasis, and fallback policy.
- Weekly becomes an assembler rather than a second recommendation implementation.
- The verified backend/frontend tier drift is fixed through a typed backend
  value, one canonical wire vocabulary, and one explicit frontend projection.
- Existing recommendation and fallback behavior remains stable apart from the
  corrected difficulty representation and badge styling.
- Implementation is sequenced by the [#172](https://github.com/jwh3times/GuardianTracker/issues/172) handoff and proceeds slice by slice.
