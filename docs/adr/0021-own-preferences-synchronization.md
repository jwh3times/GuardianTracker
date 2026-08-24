# ADR 0021: Own Preferences Synchronization

- Status: Accepted — implementation sequenced in [#172](https://github.com/jwh3times/GuardianTracker/issues/172)
- Date: 2026-08-20
- Supersedes in part: [ADR 0019](./0019-own-wish-list-and-preferences.md); see
  [Supersession](#supersession)
- Implementation note (2026-08-24): the backend provenance slice is complete.
  `preferences.Service` now distinguishes authoritative reads from degraded
  defaults, and `PreferencesHandler` serializes the additive `persisted` field
  while preserving GET's `200` and PUT's `503`. The frontend Preferences client
  and synchronization slice E15 remains pending, so this ADR remains Accepted
  with sequenced implementation rather than Implemented.

## Context

[ADR 0019](./0019-own-wish-list-and-preferences.md) gave `services/preferences`
the backend ownership of defaults, validation, atomic partial patches, and
irreversible onboarding completion, and explicitly deferred membership reset,
stale-work fencing, write serialization, and visible failure state to this
decision. [ADR 0020](./0020-own-frontend-data-access.md) repeated that carve-out
and omitted Preferences from its resource list.

The browser side is `contexts/PreferencesContext.tsx`, and it admits states no
module owns:

- `guardian_prefs` is one global localStorage key. It is not membership-scoped
  and is never cleared at logout, so one Destiny membership's values hydrate
  another's first paint.
- `syncedMembershipId` is assigned in a `finally`, so a **failed** read still
  reports preferences ready while `onboardedAt` still holds the previous
  membership's value. A genuinely new account can be silently denied the
  onboarding tour because a previous account completed it.
- Both setters call `apiFetch(...).catch(() => {})`. ADR 0019 has PUT return
  `503` when persistence is unavailable, so a toggle can report success that the
  server never stored.
- Both setters perform network I/O inside a `setPrefs` updater function, which
  React StrictMode double-invokes, issuing each PUT twice.
- Nothing fences a late PUT response, and nothing listens for the storage events
  those preference writes emit, so two tabs diverge until reload.
- `onboardedAt: undefined` means both "not resolved yet" and "could not be
  resolved".

Two structural facts shape the answer. Every preference consumer — Collections,
Settings, and OnboardingTour — sits inside `ProtectedLayout`, so no anonymous
surface reads a preference. And `cardStyle` and `personalize` appear in no query
key; they are render input to Collections, so preferences participate in no
cross-resource invalidation.

The wire also discards a distinction the backend is already required to make.
ADR 0019 requires the persistence adapter to distinguish a genuinely new
account, for which defaults are the domain value, from persistence
unavailability — but `GET /api/preferences` returns `200` with defaults for
both. The browser therefore cannot separate saved settings from a degraded read,
cannot fail the onboarding gate closed, and cannot explain a failing write.

## Decision

One framework-neutral `PreferencesClient` owns the browser preference projection
and every transition that can replace it. One production singleton is shared by
its React consumers.

### Placement and public surface

The client lives in `frontend/src/data/preferences.ts` and exports hooks. React
Query is not used: preferences are one small object with no shared cache base,
no request deduplication across consumers, and no invalidation fan-out, while
they do require synchronous pre-paint hydration, storage-event adoption, a
single-flight coalescing write queue, and membership re-keying.

The module nonetheless lives inside ADR 0020's `src/data/**` zone, so its
`types/api` import satisfies the existing ESLint boundary with no exception, and
consumers import a hook exactly as they would from any data-access module.
Whether a data-access module is implemented with React Query or with a neutral
client is an implementation detail behind the module boundary.

### Projection and hydration

The client synchronously hydrates one schema-versioned, revisioned localStorage
envelope holding a **single slot keyed by the current Destiny membership**. A
membership change replaces the slot; becoming anonymous removes it. The envelope
is a cache of the last server-confirmed value and is never evidence of a stored
state.

Anonymous performs no read, accepts no write, holds no envelope, and projects
defaults. No local anonymous value is migrated into an account at login, so no
merge policy between a local and a stored value is required.

### Observable state

The snapshot exposes two orthogonal unions with `values` always present, so
Collections renders with no loading branch:

```ts
resolution:
  | { status: "anonymous" }
  | { status: "unresolved"; source: "cache" | "defaults" }
  | {
      status: "resolved";
      persisted: boolean;
      onboarding: "required" | { completedAt: string };
    };
save:
  | { status: "idle" }
  | { status: "saving" }
  | { status: "failed"; error: PreferenceError };
```

They are orthogonal because a write is accepted while the read is still in
flight; a combined union would need the saving-by-resolution cross-product to
express a state this decision deliberately allows.

`onboarding` is constructible only inside `resolved`, so the fail-closed
onboarding gate — `resolved && persisted && onboarding === "required"` — is a
type-level guarantee rather than a remembered condition. `persisted: false` and
`onboarding: "required"` legitimately co-occur, because a degraded read returns
defaults with no completion timestamp; that combination must not show the tour.

### Provenance on the wire

`GET /api/preferences` gains an additive `persisted: boolean` and keeps
returning `200`. `persisted: true` is authoritative and covers a genuinely new
account, whose defaults are the domain value with `onboardedAt: null`.
`persisted: false` marks unstored defaults returned because persistence is
unavailable.

`preferences.Service` exposes the provenance its adapter is already required to
compute; the handler serializes it. GET does not become `503`: that would put an
error state across the whole Minikube dev-validation environment, which runs
without PostgreSQL by design under
[ADR 0004](./0004-local-development-and-minikube-scope.md) and works today.

There is no third "fresh account" provenance. A new account is
`persisted: true` with a null completion timestamp.

### Reads

The client reads exactly once per resolved membership, plus cross-tab adoption.
There is no focus, visibility, or timer refetch: this client is the origin's
sole writer of preferences, so adoption already covers every change that can
occur while a session is open.

A session that reads while degraded stays degraded for its lifetime. A
successful write is the recovery path that upgrades provenance; a read-only
session has nothing to lose by not observing that persistence returned.

### Writes

Writes are accepted immediately, including before the read settles, because ADR
0019's `Apply` is a true field-presence partial patch: writing `cardStyle`
before having read cannot clobber `personalize`.

A write applies optimistically, and on failure **rolls back** and surfaces a
typed error. The rendered state therefore always equals the last state the
server confirmed. The client owns cancel, snapshot, optimistic write, rollback,
and settle; the feature owns the user-visible copy, matching ADR 0020's split.

Writes are **single-flight with coalescing**: at most one PUT is in flight, and
further field changes coalesce into one queued patch that fires on settle. One
coalesced patch is exactly one atomic `Apply`, which makes "the last value the
user chose is the value stored" true rather than probable.

A write is attempted even when the last read reported `persisted: false`. This
keeps one failure origin rather than two that must agree, and it is
self-correcting when persistence has recovered since the read.

There is no automatic retry. A failed patch rolls back immediately and the user
retries by changing the control again. This deliberately breaks symmetry with
ADR 0017's refresh classification, and the asymmetry is the point: refresh
classifies retryable failures because wrongly ending a live session is
catastrophic and not user-recoverable, whereas a reverted display toggle costs
one click. Retrying would also lengthen the window in which the interface shows
a value the server does not have, which is the dishonesty this decision exists
to remove.

### Cross-tab convergence

The client adopts newer **same-membership** envelopes from storage events and
ignores others. There is no Web Lock: preferences carry no rotating credential,
so none of the server-side reuse-detection hazard that makes origin-wide
exclusion mandatory for refresh in
[ADR 0017](./0017-own-the-browser-session-projection.md) applies here, and
last-write-wins across tabs is acceptable.

A monotonic envelope revision orders all three publishers — an adopted envelope,
the settling initial read, and the single-flight patch response. The newest
revision wins regardless of source. This converges because a patch response is
the server's post-patch state, so an adopted value arriving in between is at
worst a brief flicker and never a durable wrong value.

### Identity boundaries and invalidation

ADR 0017's composition-registered cleanup consumer resets this module alongside
its QueryClient clear. ADR 0017 owns the definition of an identity boundary —
becoming anonymous or changing Destiny membership, with same-membership refresh
retaining state — and nothing re-derives that rule. A test asserts the consumer
resets Preferences, because the registration is otherwise forgettable.

Preferences is **explicitly exempt** from ADR 0020's membership-refresh fan-out:
a Bungie data refresh cannot change a Guardian Tracker user setting. The
exemption is recorded as a named exclusion in the fan-out test rather than
achieved by omission, so a reader can tell a deliberate exemption from forgotten
wiring — which is the failure that test exists to catch.

## Boundaries

- `services/preferences` remains the backend owner of defaults, validation,
  atomic partial patches, and irreversible server-stamped onboarding completion
  under ADR 0019. The browser client projects those results; it does not own
  preference policy.
- `BrowserSessionClient` remains the owner of the session projection and of the
  identity-boundary definition under ADR 0017. The preferences client reads the
  current membership from that seam and registers for its cleanup signal.
- ADR 0020 continues to own query identity, projection, mutation helpers,
  optimistic coordination, and cross-feature invalidation for its resources.
  This module honors ADR 0020's import boundary and hook surface without using
  React Query.
- `apiFetch` remains the transport and response adapter.
- Onboarding completion remains irreversible and server-stamped. The browser
  never supplies, replaces, clears, or infers the completion timestamp.
- Toast and error copy, the Settings layout, and tour presentation remain
  feature concerns.
- Routes, methods, PUT semantics, PUT's `503`, GET's `200`, and every existing
  response field are unchanged. The only wire change is the additive `persisted`
  field.

## Supersession

This is the repository's first ADR supersession.

ADR 0019's Boundaries state that it "does not change the existing REST wire."
That statement is superseded **only** for the additive `persisted` field on
`GET /api/preferences`. Everything else in ADR 0019 remains in force, including
every Wish list decision, the atomic `Apply` contract, PUT's `503`, and GET's
`200`. ADR 0019 carries a forward-pointing status line to this record.

The convention this establishes for later supersessions is: **narrow** — name
the exact statement rather than the document; **named** — name the field or
behavior that changes; and **bidirectional** — the superseded ADR points forward
and the superseding ADR points back. Do not reissue an ADR wholesale when one
statement changes.

## Migration and test surface

Implementation replaces the split rather than layering over it:

1. Add the `persisted` provenance to `preferences.Service` and serialize it from
   the handler, keeping GET at `200`.
2. Introduce the client state machine, typed outcomes, and factory ports: an
   auth-adapted transport, projection persistence backed by localStorage and
   storage events, and the membership source read from ADR 0017's client.
3. Add synchronous single-slot hydration, revision ordering, same-membership
   adoption, and the single-flight coalescing write queue.
4. Move defaults, read, patch, and onboarding completion behind the module and
   expose hooks to Settings, Collections, and OnboardingTour.
5. Register the module's reset in ADR 0017's composition cleanup consumer and
   record its membership-refresh exemption in ADR 0020's fan-out test.
6. Delete `contexts/PreferencesContext.tsx`, the global `guardian_prefs` key,
   the `syncedMembershipId` readiness flag, and the `undefined` onboarding
   sentinel after replacement coverage is green.

Client contract tests over in-memory persistence and scripted transport replace
the implementation-shaped context tests. They cover single-slot hydration and
membership re-keying, the anonymous state holding no envelope, provenance
handling for both `persisted` values, write acceptance before the read settles,
optimistic rollback with a typed error, single-flight coalescing into one patch,
revision ordering across adopted envelopes and late read and patch responses,
same-membership adoption with foreign envelopes ignored, absence of automatic
retry, and the fail-closed onboarding gate in every unresolved, failed, and
degraded state.

Two boundary tests are required because their wiring is otherwise forgettable:
ADR 0017's cleanup consumer resets Preferences, and ADR 0020's fan-out test
records the Preferences exemption as deliberate.

Thin React tests survive for Settings' visible control behavior and the tour
gate. No browser case is added: storage-event adoption and revision ordering are
fully deterministic against a fake persistence port, and a contract test runs in
the required `Test Frontend` job rather than the advisory browser workflow,
which is the wrong place for a race's regression signal.

Backend tests add provenance coverage to the existing GET degraded-mode cases.
Existing ADR 0019 preferences service tests for defaults, validation, atomic
patching, and idempotent server-stamped onboarding are retained.

## Alternatives considered

### Build Preferences as a React Query data-access module

Rejected. The React Query features that justify ADR 0020 — a shared wire cache
with per-observer `select`, request deduplication across consumers, and
cross-resource invalidation — are all inapplicable to one small object with no
query key and no fan-out, while the four behaviors this decision requires are
ones React Query does not natively provide.

### Place the client outside `src/data/**` with an ESLint exception

Rejected. ADR 0020 argued the import boundary must be machine-checked precisely
because prose boundaries decay in this repository. Spending an exception on a
directory-naming preference is a poor trade when the permitted zone already
fits.

### Keep localStorage as the local source of truth

Rejected. It is what makes a stale value indistinguishable from a confirmed one,
and what lets one membership's settings hydrate another's session.

### Make GET return `503` when persistence is unavailable

Rejected. It is a breaking change that would surface an error state across the
Minikube dev-validation environment, which has no PostgreSQL by design and works
today. The additive `persisted` field conveys the same fact without breaking a
client.

### Retain the value and show "not saved" instead of rolling back

Rejected. It leaves the interface displaying a value the server does not have,
which is the current defect with an indicator attached.

### Retry a failed patch before rolling back

Rejected. See the write section: the blast radius does not justify the extra
state or the longer window of displayed-but-unstored value.

### Coordinate preference writes with a Web Lock

Rejected. Locks exist in ADR 0017 to prevent concurrent rotation of one refresh
cookie. Preferences have no such credential, and requiring Web Locks would make
preferences unavailable on browsers lacking them for no correctness gain.

### Let the client derive the identity boundary itself

Rejected. It duplicates a rule ADR 0017 already owns, and the failure mode of
the two definitions disagreeing is one membership observing another's settings —
the exact defect this decision removes.

### Include Preferences in the membership-refresh fan-out

Rejected. A Bungie data refresh has no causal relationship to a user setting,
and including it would teach that "membership-scoped" implies "refreshable".

## Consequences

- One owner holds the browser preference projection, its persistence, its
  transport, and every transition that can replace it.
- A membership can no longer observe another membership's preferences or
  onboarding state, and a failed read can no longer masquerade as a resolved
  one.
- A preference change either reaches the server or visibly reverts; it can no
  longer silently fail.
- Concurrent changes across controls and tabs converge on one revision order,
  and one coalesced patch maps to one atomic backend `Apply`.
- The onboarding tour appears only on positive evidence and is suppressed in
  every unresolved, failed, and degraded state.
- The REST wire gains one additive field, and ADR 0019 is superseded on exactly
  one narrow statement.
- A degraded read is visible rather than indistinguishable from saved defaults,
  at the cost of a read-only degraded session not observing recovery.
- Implementation is sequenced by the [#172](https://github.com/jwh3times/GuardianTracker/issues/172) handoff and proceeds slice by slice.
