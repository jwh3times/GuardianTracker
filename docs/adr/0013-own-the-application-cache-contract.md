# ADR 0013: Own the Application Cache Contract

**Status:** Accepted
**Date:** 2026-08-12

## Context

The application cache was a four-method adapter over
`github.com/patrickmn/go-cache`: get, set, delete, and clear. Guardian Tracker
did not use the dependency's broader API for persistence, counters, callbacks,
or item enumeration. The dependency remained concurrency-safe and had no known
vulnerability, but its latest release predated Go modules and its periodic
cleanup worker could only be stopped indirectly by a runtime finalizer.

The behavior behind the adapter is part of the application contract. A positive
TTL is an absolute deadline from the most recent set; reading an item does not
extend it. An expired item is immediately a cache miss even when periodic cleanup
has not yet reclaimed its memory. A zero per-item TTL uses the cache default,
while a non-positive default or a negative per-item TTL means no expiration.
Writes are synchronously visible, missing-key deletion is harmless, and all four
operations are safe for concurrent use. The cache is unbounded; introducing an
eviction policy would require separate configuration and production evidence.

## Considered Options

- **Keep `patrickmn/go-cache`.** This has the lowest migration cost and already
  matches the required data behavior. It has no identified vulnerability, but
  its pre-module release remains a direct dependency and callers cannot
  deterministically stop its cleanup worker; shutdown depends on a runtime
  finalizer.
- **Adopt `ttlcache/v3`.** This is the closest maintained general-purpose
  alternative, but reads extend TTL by default and its loader, event, metric,
  and eviction APIs are unnecessary for Guardian Tracker's four-operation
  boundary. Configuring away those defaults would add a deeper adapter without
  adding a capability the application needs.
- **Adopt Ristretto.** Its admission and write-buffer design serves bounded,
  high-throughput caches, but writes may be rejected or dropped and are not
  immediately visible without an additional wait. That conflicts with the
  existing synchronous set/get contract.
- **Own the narrow implementation.** A mutex, map, expiration timestamps, and
  one optional cleanup worker are enough to preserve the complete contract and
  make worker shutdown explicit. The repository assumes responsibility for the
  implementation and its concurrency tests.

## Decision

Guardian Tracker owns a small in-process implementation behind the existing
`cache.Cache` interface. It uses a mutex-protected map with absolute expiration
deadlines, lazy expiration checks on reads, and periodic reclamation at the
configured cleanup interval. It deliberately implements no admission policy,
capacity limit, sliding expiration, loader, or callback system.

The concrete memory cache exposes an idempotent `Close` method that stops and
joins its optional cleanup worker. The composition root owns that lifecycle and
closes the single production cache during shutdown. Consumer interfaces remain
unchanged, the implementation does not use a runtime finalizer, and tests that
do not exercise cleanup construct caches without a worker.

## Consequences

- Cache keys, values, TTL configuration, invalidation ownership, and service
  behavior remain unchanged.
- The legacy `patrickmn/go-cache` module is no longer part of the backend's
  dependency graph.
- Expiration, overwrite, deletion, clearing, concurrent access, cleanup, and
  shutdown are covered at the `cache.Cache` boundary. Time-sensitive tests use
  a controlled clock instead of scheduler-dependent sleeps.
- The implementation and its concurrency correctness are now maintained in this
  repository. Any future replacement must preserve the contract tests or record
  an intentional behavior change in a superseding decision.
- Memory remains unbounded. Adding a capacity or cost policy is a separate
  decision because cached values vary in shape and no meaningful limit is
  currently configured.
