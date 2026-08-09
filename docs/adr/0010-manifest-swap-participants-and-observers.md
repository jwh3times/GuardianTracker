# ADR 0010: Manifest Swap Participants and Observers

**Status:** Accepted
**Date:** 2026-08-09

## Context

The manifest file swap (download a new SQLite database, close open handles,
`os.Rename` it over the live file, reopen) previously ran through a single
`RegisterSwapHooks(before, after)` pair per consumer. That mechanism conflated
two unrelated lifecycles under one callback shape:

- Modules holding an OS-level handle on the manifest file (`manifest.Provider`,
  and `search.Service`, which opens its own SQLite connection) MUST close that
  handle before the rename or the rename fails outright on Windows, and on
  Linux the module keeps serving a deleted inode.
- Modules holding manifest-derived state (caches, indexes) only need to know
  once a version has genuinely changed, so they can invalidate or rebuild.

Treating both as one `before`/`after` pair meant every consumer's `after` hook
ran identically on both the success path and the rollback path (a failed
rename, old file still in place), even though invalidating a cache or
rebuilding an index on a rollback is pure waste — the version did not change.
It also left ordering and "must register before the first download" as tribal
knowledge rather than something the type system enforced.

## Decision

Split the swap seam into two interfaces on `*bungie.ManifestService`:

- `SwapParticipant { CloseForSwap(); Reopen() error }`, enrolled via
  `RegisterParticipant(p)`, for a module holding an OS handle on the manifest
  file.
- `ManifestObserver { OnVersionChanged(version string) error }`, enrolled via
  `RegisterObserver(o)`, for a module holding manifest-derived state.

The swap sequence is: close every participant, `os.Rename`, reopen every
participant, then — **only if the rename succeeded** — notify every observer.
`Reopen` therefore means "reopen against whatever manifest is now live," not
"a new version was installed"; it runs on the rollback path too, but observers
do not, since nothing they own actually went stale. Registration order is
enrollment order, and `manifest.Provider` is registered first because
observers may query the manifest through it. A failing participant or observer
is logged (with its `%T` type) and the swap continues rather than aborting,
since aborting would strand later registrants closed. Both registrations must
happen before the first download can fire.

`main.go` wires two participants (`manifest.Provider`, `search.Service`) and
six observers (`records`, `weekly`, `collections`, `items`, `search`,
`efficiency`), each owning what "a new version landed" means for its own
state — `main.go` no longer knows or needs to know the specifics.

## Consequences

- Any future module that opens its own handle on the manifest file must
  register as a `SwapParticipant`, or the rename runs under its open handle.
  Any module holding manifest-derived state should register as a
  `ManifestObserver` rather than reusing an ad hoc eviction call wired through
  `main.go`.
- Observers never fire on a rollback, so a failed download can no longer
  trigger a wasted cache eviction or index rebuild for data that never
  changed.
- A consumer that used to rely on an exported cache-key constant for external
  eviction (records' weapon-type/exotic-weapon/catalyst-links keys, weekly's
  public-payload key) now owns and evicts that key itself from its own
  `OnVersionChanged`, so those constants are unexported.
- `code-reviewer` and `go-services` agent docs describe the contract so new
  manifest-touching code is registered the same way rather than reinventing a
  bespoke hook.
