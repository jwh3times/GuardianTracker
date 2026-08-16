# ADR 0014: Own Manifest-Derived Publication

- Status: Accepted — implementation pending final Wayfinder handoff
- Date: 2026-08-16

## Context

Manifest observers invalidate reusable state after a successful Manifest swap, as
defined by [ADR 0010](./0010-manifest-swap-participants-and-observers.md). That
invalidation alone does not prevent stale publication:

1. a request begins loading data from Manifest generation A;
2. generation B is installed and the owning observer clears its state;
3. the old request finishes and republishes its generation-A result into the
   now-current cache or index.

The race exists in the manifest-derived state owned by Items, Records, Weekly,
Collections, and Efficiency. Collections can additionally label an old result
with a newer version if it reads the version after its Manifest queries. Search
does not have this race: as a swap participant, it drains or aborts an active
build before the Manifest file is replaced.

The general cache contract in [ADR 0013](./0013-own-the-application-cache-contract.md)
is intentionally unaware of Manifest generations. Extending `cache.Cache` with
generation or loader coordination would weaken that reusable application seam.
The participant/observer split in ADR 0010 must also remain intact: publication
correctness belongs to observers and must not make every observer a swap
participant.

## Decision

Introduce an owner-local `manifeststate.Publication` coordination seam. Each
affected Manifest observer owns one publication and uses it to guard only the
reusable state derived from the Manifest.

The seam has three operations:

- `Begin` captures an opaque attempt for the current monotonically increasing
  generation.
- `Attempt.Publish` executes a supplied commit only if the attempt's generation
  is still current, and reports whether it published.
- `Advance` validates the installed Manifest version, advances the generation,
  and runs the owner's infallible invalidation callback as one linearized state
  transition.

An empty version is rejected. Repeating the current version is an idempotent
no-op. A later return to a previously seen version string still creates a new
generation, so abandoned work can never become current again.

`Begin`, `Publish`, and `Advance` are serialized by the publication. The
invalidation callback runs within `Advance`'s critical section and therefore must
be bounded, non-blocking, and non-reentrant. The callback cannot fail; observer
errors are limited to validating the version before the transition. This makes
generation advancement and owner invalidation atomic from the perspective of
loaders.

A request whose old attempt loses the race may still return the coherent result
it loaded to the request that initiated it. It must not install that result as
reusable state. The next request loads or builds against the current generation.

Common `manifeststate.Load` and `manifeststate.LoadIf` helpers will compose this
protocol with the existing `cache.Cache` contract. They centralize the
begin/load/conditional-publish sequence without changing `cache.Cache`,
`cache.Load`, or ADR 0013.

### Owner scopes

- **Items:** advance clears the three bounded Manifest lookup caches; their
  stores are publication-guarded.
- **Records:** advance deletes only the three fixed Manifest lookup keys. Other
  cache entries remain untouched.
- **Weekly:** advance deletes only the global public Manifest-derived payload.
  Version-addressed authenticated vendor rows remain naturally isolated.
- **Collections:** advance clears the shared tree and invalidates rebuilt
  analysis publication while retaining Bungie-collected state. Tree and analysis
  loads publish under one coherent captured generation. A broader Collections
  state split remains a separate Wayfinder decision.
- **Efficiency:** a current-generation build may start while an obsolete build
  finishes. Only the current build may publish; the prior complete index remains
  an intentional fallback until the current build succeeds.
- **Search:** unchanged. Its existing swap-participant lifecycle already prevents
  an old build from publishing after file replacement.

## Alternatives considered

### Add generation checks directly at every store

This is mechanically small but makes correctness depend on every caller
remembering a multi-step protocol. The affected modules already use several
different cache and index shapes, so duplicated fencing is likely to drift.

### Address every value by Manifest version or generation

Generation-addressed keys make obsolete writes unreachable, but push generation
plumbing through cache keys and call sites and leave retired state to expire or be
collected. They also do not by themselves coordinate compound publications such
as Collections analysis or Efficiency indexes.

### Introduce a high-level Manifest cache registry

A central registry could own invalidation and loading for every module, but would
absorb module-specific state meanings and future Wayfinder decisions. Owner-local
publications preserve the existing observer ownership boundary.

## Consequences

- In-flight work can no longer repopulate current reusable state after its
  Manifest generation is retired.
- Existing owner-specific invalidation meanings, cache capacities, TTLs, and
  non-Manifest entries are preserved.
- The general cache API and Manifest swap protocol remain stable.
- Owners must keep invalidation callbacks short and must route all relevant
  publication through the shared helper or attempt contract.
- Tests must deterministically block a load or build, advance the Manifest,
  release the old work, and prove that the old result was returned at most to its
  initiating caller but was not reused. Existing ADR 0010 ordering/rollback and
  ADR 0013 cache contract tests remain part of the surviving test surface.
- This ADR selects the contract only. Implementation sequencing and pull-request
  slices belong to the final Wayfinder handoff.
