# ADR 0020: Own Frontend Data Access

- Status: Accepted — implementation sequenced in [#172](https://github.com/jwh3times/GuardianTracker/issues/172)
- Date: 2026-08-20

## Context

Frontend data access is currently spread across twelve feature and context
modules. `lib/queries.ts` is a partial seam: it owns Collections (two variants)
and Items, and it already documents the intended pattern — shared query keys so
variants share one cache entry, and `select` adapting the payload at the seam so
"no feature module ever sees the wire shape."

Every other resource declares its own. The consequences are visible in the
current tree:

- The `refreshMutation` in `features/collections/Collections.tsx` and
  `features/settings/Settings.tsx` is duplicated verbatim — same endpoint, same
  six-family invalidation fan-out — differing only in how membership is sourced.
- Four resources are declared by more than one owner: `characters` by
  `CharacterContext` and Settings, `weekly` by Dashboard and This Week,
  `wishlist` by Wish list, Collections, and Dashboard, and `currentUser` by
  Collections and Dashboard. The two `characters` declarations derive membership
  through different expressions that happen to agree today.
- Wish list owns four hand-rolled optimistic mutations, each repeating
  `cancelQueries`, `setQueryData`, rollback, and settle, and each writing the
  wire type into the cache while its consumer projects on read.
- Modules invalidate keys they do not own: Collections and Settings invalidate
  six families across ownership lines, and Admin invalidates `["flags"]`, which
  belongs to `FlagsContext`.

The failure this produces is not a rendering bug. It is that query identity,
wire projection, mutation behavior, optimistic coordination, and refresh effects
have no owner, so a change to any of them must be found by grep and applied in
every copy. Adding a seventh membership-scoped resource requires remembering two
unrelated feature files.

Three accepted decisions already constrain the answer. ADR 0018 requires that a
shared frontend membership-refresh adapter exist and keep invalidating both
Collections variants plus Characters, Weekly, Catalysts, Crafting, and Seals.
ADR 0019 assigns this decision query identity, projections, mutation helpers,
optimistic coordination, and cross-feature invalidation. ADR 0017 requires
identity-bound QueryClient cleanup registered at composition and states that
this decision may improve query-key ownership but must preserve that boundary.

## Decision

Frontend data access is divided into **data-access modules**, one per domain
resource, in `frontend/src/data/<resource>.ts`. A data-access module owns its
query identity, its endpoint paths, its projection to domain types, its
mutations and their optimistic coordination, and its own invalidation. Ownership
follows the resource, not the page, because four resources already have two or
three consumers each.

The decomposition is by resource — Collections, Wish list, Weekly, Characters,
Items, Search, Flags, Admin, Catalysts, Crafting, Seals — and explicitly not one
broad data-access interface for the whole application, and not per-feature
colocation, which reproduces the current duplication the moment a second page
needs the same resource.

### Public surface

A data-access module exports **hooks**. Query-option factories remain the
internal composition unit and are module-private except where React Query's own
APIs require an options object from a caller. React Query does not appear in a
feature module.

This matches the seam ADR 0017 established for the session: a framework-neutral
owner with a small public surface, React kept outside it.

### Query identity

Query keys are private to their module and are never exported as literals. No
module names another module's keys.

### Projection

Every resource projects to a domain type at the seam, including resources whose
wire and domain shapes are identical today. No module under `features/` or
`components/` imports from `types/api`. A one-to-one passthrough type is the
cost of preventing the wire shape from silently becoming the domain shape.

The projections that currently sit in consumers — `toCharacter` in
`CharacterContext` and Settings, `toWishlistEntry` in Wish list — move to their
modules.

### Cache shape

The cache stores the **wire** shape. `select` projects on read. This preserves
the shared-base pattern that lets `collectionsQuery` and `collectionsFullQuery`
share one cache entry, and it preserves React Query's per-observer `select`
memoization, which those two variants depend on.

The consequence is that optimistic writes are expressed in wire terms. That cost
lands inside the module that already owns the wire type, which is where it
belongs.

### Membership arguments

Membership-scoped hooks read the current membership from the session seam
themselves and take no membership arguments. Explicit parameters remain only for
genuine parameters: item hash, search text, audit type.

This makes it impossible for a page to query a membership other than the current
one, and removes the class of defect visible today, where two owners derive the
same membership through different expressions.

Because these hooks read the session seam, ADR 0017's client is a prerequisite
for this ADR's implementation.

### Cross-resource invalidation

A membership-refresh data-access module owns the refresh endpoint and the
fan-out. It imports each membership-scoped resource module and calls that
module's own invalidation entry point; it does not name another module's keys.

The membership-scoped set stays explicit and greppable in one file. A registry
that modules join at import time was rejected: it buys "impossible to forget" at
the price of import-order and side-effect fragility under Vite.

### Mutations and optimistic coordination

A data-access module owns the entire cache half of a mutation — cancel,
snapshot, optimistic write, rollback, settle — and surfaces a typed error. The
feature owns the user-visible copy.

Failure copy is presentation and legitimately differs by context. The two
existing delete-failure messages differ today; unifying them is a behavior
change and is out of scope here.

### Identity cleanup

ADR 0017's cleanup is unchanged: one composition-registered consumer clears the
whole QueryClient when the projection becomes anonymous or the Destiny
membership changes, and a same-membership refresh retains the caches.

Per-module reset was rejected. It is precisely what a newly added module can
silently omit, and the failure mode is one membership's data surviving into
another's session. A full clear is correct by construction.

### Enforced boundary

The import boundary is machine-checked, not documented. ESLint
`no-restricted-imports` zones bar `features/**` and `components/**` from
importing `@tanstack/react-query` and `types/api`; only `src/data/**` may. This
fails `npm run lint` in the `Test Frontend` job.

The repository has made this call twice before — the workflow-action pin policy
and the Node-version policy exist because prose comments did not hold, and the
PostgreSQL image policy was added for the same reason in v1.0.7. An
architecture boundary that is not checked decays between pull requests.

## Boundaries

- The session and user-snapshot projection remains ADR 0017's
  `BrowserSessionClient`. Data-access modules read the current membership from
  that seam; they do not own authentication, token lifecycle, or identity
  cleanup.
- Preferences synchronization — membership reset, stale-work fencing, write
  serialization, and visible failure state — remains the separate decision in
  the Preferences synchronization ticket.
- `apiFetch` remains the transport and response adapter. Data-access modules
  compose it; they do not replace it.
- Backend ownership is unchanged. ADRs 0016, 0018, and 0019 own their service
  contracts; this ADR does not alter the REST wire.
- Toast copy, page layout, and navigation remain feature concerns.

## Migration and test surface

Implementation is a resource-by-resource strangler, one resource per pull
request: create the data-access module, move every consumer to it, delete the
inline query, and only then move to the next resource. Order by consumer count —
Wish list, Collections, Weekly, Characters first — so the highest-duplication
resources are consolidated earliest.

ADR 0017's client lands before the membership-scoped modules, because implicit
membership reads depend on it.

Authorized during migration, as pure consolidation with no behavior change:

1. Deduplicating the verbatim `refreshMutation` into the membership-refresh
   module.
2. Moving `toCharacter` and `toWishlistEntry` from consumers to their seams.

Explicitly not authorized: unifying the two differing delete-failure messages.
That is a behavior change and requires its own decision.

Test surface:

- Each data-access module gets contract tests owning key identity and shape,
  projection, and optimistic rollback and settle behavior.
- The membership-refresh module's test asserts the invalidation set and fails
  when a membership-scoped module is added without being wired into the fan-out.
  This is the failure ADR 0018 cares about.
- Feature tests drop wire-shape assertions and retain visible behavior only.
- Shared MSW fixtures remain in `src/test/testServer.ts`.
- The ADR 0018 survivors are retained unchanged: `CollectionsSummaryView` and
  `CollectionsView` tests, query-variant tests, availability filters and badges,
  Cosmetics parity, deep links, Dashboard summary behavior, and
  privacy/warming/error states.

## Alternatives considered

### Grow `lib/queries.ts` into one data-access module

Rejected. One module for every resource is the broad generic interface this
decision exists to avoid; it would own unrelated identities, projections, and
invalidation policies with no boundary between them.

### Colocate hooks inside each feature folder

Rejected. It reproduces the current duplication whenever two features share a
resource, and four resources already have multiple consumers.

### Export query-option objects rather than hooks

Rejected as the public surface, retained as the internal unit. Options objects
compose well but leave React Query visible in every feature module, which is the
coupling this decision removes.

### Store projected domain objects in the cache

Rejected. It breaks the shared-base pattern behind the two Collections variants
and pushes projection into every write path.

### Let the refresh owner name raw query keys

Rejected. It defeats private key ownership and reinstates the cross-ownership
reach that Collections, Settings, and Admin exhibit today.

### Per-module cache reset on identity change

Rejected. See "Identity cleanup".

### Document the import boundary instead of enforcing it

Rejected. See "Enforced boundary".

## Consequences

Query identity, projection, mutation behavior, optimistic coordination, and
refresh effects each gain exactly one owner, and the boundary that keeps them
there is checked by CI rather than by review.

Adding a resource means adding one module and, if it is membership-scoped,
wiring it into the refresh fan-out — a step whose omission now fails a test
instead of silently shipping a stale page.

The costs are real and accepted: one-to-one projection types for thin resources,
optimistic recipes written against wire types, a coupling from membership-scoped
modules to the session seam, and a migration that cannot begin until ADR 0017's
client exists.
