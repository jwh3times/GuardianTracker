# ADR 0017: Own the Browser Session Projection

- Status: Accepted — implementation sequenced in [#172](https://github.com/jwh3times/GuardianTracker/issues/172)
- Date: 2026-08-17

## Context

The browser session lifecycle is currently split across `lib/api.ts`,
`AuthContext.tsx`, and `OAuthCallback.tsx`. Each can read or mutate the access
token and user snapshot. They coordinate through separate localStorage keys, a
same-document custom event, a module-local refresh promise, and direct
navigation from transport code.

That split admits states and races which no module owns:

- a partial or corrupt token/user pair can leave React anonymous while requests
  still send the remaining token;
- refresh failure classification clears a usable projection for responses such
  as the Origin check's `403`, even though only refresh `401` proves that the
  server session ended;
- concurrent tabs can rotate the same HttpOnly refresh cookie independently,
  causing the second use to trigger server-side reuse detection;
- late callback, refresh, or logout work can overwrite a newer identity or
  resurrect a locally ended session; and
- identity-neutral React Query keys can retain one membership's data across a
  logout and subsequent login as another membership.

The browser needs one lifecycle owner, but it must not become a second session
issuer. ADR 0008 keeps the refresh credential in an HttpOnly cookie, outside
JavaScript. ADR 0012 keeps canonical session creation, rotation, revocation,
reuse detection, and logout in the server's `auth.SessionIssuer`.

## Decision

One framework-neutral `BrowserSessionClient` owns the browser's access-token and
user-snapshot projection and every transition that can replace or end it. One
production singleton is shared by React and the request adapter.

Its small public surface has these responsibilities:

```ts
interface BrowserSessionClient {
  getSnapshot(): BrowserAuthSnapshot;
  subscribe(listener: () => void): () => void;
  beginAuthorization(): Promise<AuthorizationStart>;
  completeAuthorization(input: AuthorizationCompletion): Promise<void>;
  request(input: RequestInfo | URL, init?: RequestInit): Promise<Response>;
  end(scope: "current" | "all"): Promise<void>;
}
```

`BrowserAuthSnapshot` is a tagged `anonymous | authenticated` value. The
authenticated variant exposes the Guardian Tracker user snapshot, not the
access token. The token remains private to the client's request machinery.
`apiFetch` becomes a thin JSON, empty-response, and `ApiError` adapter over
`request`; it does not own session transitions.

### Projection and hydration

The client synchronously hydrates one schema-versioned, revisioned localStorage
envelope. An authenticated envelope contains the access token and its complete
user snapshot together; an anonymous tombstone records an ended projection.
The envelope is validated before use and written as one value, so a token-only
or user-only state is not representable through the client.

On first hydration, the client migrates the current separate access-token and
user keys only when both form a valid pair. It clears partial or corrupt legacy
state and removes the obsolete refresh-token key if present. Hydration never
probes the refresh cookie and never performs network I/O.

Each accepted local transition advances the envelope revision and the client's
in-memory generation. Work captures its initiating generation. Obsolete work
cannot publish, clear, retry, or notify after that generation is retired.

The client listens for storage events and adopts a valid newer envelope from
another tab. It publishes exactly once for each accepted transition. Same-
membership token rotation may update the user snapshot without declaring an
identity boundary; becoming anonymous or changing Destiny membership does.

### Establishment and callback idempotence

`beginAuthorization` owns the authorization-URL transport.
`completeAuthorization` owns the credentialed callback exchange and publishes
the returned access token and user snapshot as one replacement. Repeated calls
with the same code and state share one in-flight operation, including calls
caused by React StrictMode or a remount. The operation publishes at most once
for its captured generation, and an obsolete completion is rejected rather
than replacing newer state.

The client owns no post-login or error-page navigation. After completion, the
OAuth callback UI observes the published snapshot or handles a typed completion
failure, then navigates declaratively through React routing.

### Authenticated requests and refresh

Only a request captured from an authenticated generation can attempt refresh.
An anonymous request neither probes the cookie nor tries to establish a
session. The client attaches the private access token, and a protected `401`
may enter the refresh path once and retry the original request once.

Refresh-cookie rotation is coordinated origin-wide with an exclusive Web Lock.
After acquiring the lock, the client re-reads and validates the persisted
envelope. If another tab already published a newer authenticated envelope for
the same membership, the request reuses that token instead of rotating the
cookie again. Callback, refresh, and logout cookie mutations use the same
origin-wide lifecycle coordination so establishment and ending cannot race.

Web Locks are required for callback establishment and authenticated refresh. If
they are unavailable, `completeAuthorization` rejects with the typed
`AUTHORIZATION_UNAVAILABLE` outcome before making the credentialed callback
request. The client preserves an existing projection and rejects an attempted
refresh with the typed, non-destructive `REFRESH_UNAVAILABLE` outcome. Logout
still durably ends the local projection but skips the unsafe remote cookie
mutation. There is no same-tab-only coordination fallback.

Only refresh `401` definitively ends the captured session. Network failures,
`429`, and `5xx` responses are retryable failures which preserve the projection;
`403` and other protocol or configuration responses are surfaced without
clearing it. A second protected `401` after a successful refresh also ends only
the captured generation. An obsolete failure cannot clear newer state.

### Logout and identity boundaries

`end("current" | "all")` makes logout locally final before attempting remote
cleanup: it publishes anonymous state, retires older work, and triggers
identity-bound cleanup. Server logout is then best effort and serialized with
other cookie-mutating lifecycle work. A stale access token may be refreshed and
retried solely to finish logout, but that result is never republished. Remote
failure never restores the ended projection, and a new establishment waits for
the departing cookie mutation to finish.

The client reports projection transitions; it does not import React Query or
feature contexts. A consumer registered at application composition cancels and
clears the QueryClient and resets identity-bound provider state when the
projection becomes anonymous or its Destiny membership changes. A same-
membership refresh retains those caches. The later data-access decision may
improve query-key ownership, but must preserve this boundary cleanup.

### Ports and adapters

The production factory supplies three small internal ports:

- an auth transport for authorization setup, callback, refresh, and current/all
  logout using the existing endpoint methods, bodies, credentials, and Origin
  behavior;
- projection persistence backed by localStorage and storage events; and
- origin-wide refresh/lifecycle coordination backed by Web Locks.

Scripted transport, in-memory persistence, and a deterministic fake coordinator
exercise the client in isolation. React's auth provider and `apiFetch` depend on
the concrete client's public interface; no speculative external session port is
introduced.

## Boundaries

- `auth.SessionIssuer` remains the sole canonical session-lifecycle owner. The
  browser client projects server results and coordinates browser work only.
- The refresh JWT remains exclusively in the host-only HttpOnly cookie scoped
  to `/api/auth`. The envelope never contains it and JavaScript never reads it.
- Callback and refresh remain credentialed and retain their current form/JSON
  bodies and exact-Origin enforcement.
- Callback and refresh continue returning complete `{token, user}` replacements;
  the browser does not merge partial identity fields.
- The user snapshot is a rendering and routing hint. Server authorization,
  including role and feature checks, remains authoritative.
- The frontend stops querying `/api/auth/profile`; callback and refresh results
  are its one user-snapshot source. The backend route remains as a compatibility
  surface outside this refactor.
- Login, callback, logout, refresh timing, and visible routing remain unchanged
  except for the verified race, classification, and identity-isolation fixes.

## Migration and test surface

Implementation replaces the split lifecycle rather than layering over it:

1. Introduce the client state machine, typed outcomes, factory ports, and one
   production singleton.
2. Add atomic envelope hydration, one-time legacy migration, generation fencing,
   storage-event adoption, and required Web Lock coordination.
3. Move authorization setup/completion, bearer injection, refresh/retry, and
   current/all logout transport into the client.
4. Make `apiFetch` a response adapter and make AuthProvider a declarative
   `getSnapshot`/`subscribe` projection with no public token, `login` mutator, or
   permanently false loading state.
5. Route OAuth callback exchange through the idempotent client while retaining
   UI-owned presentation and navigation.
6. Register identity-bound QueryClient and provider cleanup at composition and
   remove frontend `/api/auth/profile` queries.
7. Delete the duplicated localStorage writes, custom auth event, transport-level
   hard redirect, and module-local refresh promise after replacement coverage is
   green.

Client contract tests replace implementation-shaped storage and event tests.
They cover legacy migration and repair, atomic establishment, exactly-once
subscription, same-membership and identity-boundary adoption, callback
single-flight behavior, generation races, current/all logout finality, stale-
token logout retry without republication, absence of a JavaScript refresh
credential, and every refresh classification (`401`, `403`, `429`, `5xx`, and
network failure). Coordinator tests cover same-tab and cross-tab contention,
inside-lock envelope re-reading, newer-token reuse, and the unsupported-browser
outcome.

Thin integration tests survive for `apiFetch` JSON/error/empty-response behavior,
AuthProvider projection, declarative protected-route redirects, and OAuth
callback success/error presentation under StrictMode. Browser coverage adds a
two-page concurrent-`401` case proving one refresh rotation and one shared
replacement. Identity tests prove logout and membership replacement clear
cached user data while same-membership refresh does not. Existing backend
SessionIssuer, cookie, Origin, rotation, reuse-detection, current/all logout,
authorization, and audit tests remain unchanged.

## Alternatives considered

### Keep AuthContext and `apiFetch` as peer owners

This fails the deletion test: removing either owner leaves the other with a
partial lifecycle, and every transition still requires coordinated persistence,
notification, navigation, and race handling across modules.

### Put the lifecycle in React context

That would make transport and cross-tab coordination depend on React lifecycle
semantics. StrictMode and remount behavior would remain correctness concerns
instead of presentation concerns.

### Keep the same-tab refresh promise as a fallback

It cannot prevent two tabs from rotating the same cookie. Allowing refresh
without origin-wide exclusion would preserve the server reuse-detection race,
so unsupported browsers receive the explicit non-destructive outcome instead.

### Let the client own server session rules

Mirroring issuer, revocation, reuse, or authorization policy in the browser
would violate ADR 0012 and create a weaker competing authority. The accepted
client owns only the browser projection and transport orchestration.

## Consequences

- React, transport, persistence, and cross-tab behavior observe one atomic
  browser projection through one transition owner.
- Stale work cannot overwrite or clear a newer identity, concurrent tabs cannot
  independently rotate one refresh cookie, and logout cannot resurrect locally.
- Only evidence that the session ended clears authenticated state; transient and
  configuration failures remain visible without destroying it.
- Identity-bound frontend data is cleared at membership boundaries even before
  query-key ownership is redesigned.
- Callback establishment and authenticated refresh require Web Locks;
  unsupported browsers cannot establish or renew a session and must use a
  compatible browser. Logout remains locally final without attempting an
  uncoordinated cookie mutation.
- Implementation is sequenced by the [#172](https://github.com/jwh3times/GuardianTracker/issues/172) handoff and proceeds slice by slice.
