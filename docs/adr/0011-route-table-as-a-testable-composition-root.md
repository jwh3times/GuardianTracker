# ADR 0011: Route Table as a Testable Composition Root

**Status:** Accepted
**Date:** 2026-08-09

## Context

The whole route table lived in `package main`, which nothing can import and no
test can construct. Two rules that decide whether an endpoint is safe were
therefore unassertable, and both fail silently when broken:

- **Authentication.** `jwtHelper.Middleware(revoker)` was repeated on all 24
  authenticated route registrations. Omitting it on a new route publishes that
  endpoint to unauthenticated callers with no error and no log — the mistake
  looks exactly like the 23 correct lines around it.
- **Flag gating.** `authz.RequireFlag(enforceFlags, key)` paired a route with a
  flag key by hand at five sites. `RequireFlag` deliberately fails open on a key
  it cannot resolve (ADR 0006: flags are rollout controls, not a security
  boundary), so a mistyped or wrong key gates nothing at all.

Two adapters translating `db` types into the `auth` and `weekly` consumer-side
interfaces also lived in `main.go`, purely to break an import cycle. One of them
carries a load-bearing sentinel translation: `auth.TokenStore`'s CAS
reconciliation branches on "definitively absent" versus "the read failed", so
reporting a transient error as `ErrTokensNotFound` would let it overwrite a row
it never read. Nothing tested it. The two background pruners were unreachable
for the same reason.

## Decision

`package api` (`api/router.go`) owns the route table behind
`NewRouter(Deps) *gin.Engine`. `main.go` keeps configuration, service
construction, manifest-swap enrolment, and the server lifecycle — it is a
composition root and nothing more.

Authentication is applied **once per group**, not per route:

```go
authed := api.Group("", d.JWT.Middleware(d.Revoker))
```

Every authenticated route registers on `authed` (or on the `admin` subgroup
inside it). Registering on the wrong group is now the only way to publish an
unauthenticated endpoint, and it is visible in the diff as a changed receiver.

`api/router_test.go` asserts the invariants against the table gin actually
built, walking `Engine.Routes()` so a route added later is covered without
anyone extending the test:

- every `/api` route not in an explicit allowlist answers 401 without a bearer;
- the four public routes are still reachable, so the gate cannot creep onto
  login;
- disabling one flag at a time blocks exactly the routes gated on that flag;
- admin routes 403 a non-admin and 503 in a degraded build.

**The public-route allowlist lives in the test, not in the router.** Adding a
route outside the authenticated group fails the suite until someone edits the
allowlist deliberately. If the router owned the list it could grant itself the
exemption, which is the review moment worth keeping.

The adapters move to `db/adapters`, and the pruners to `db/pruner.go` with an
injectable interval. `corsMiddleware` becomes `middleware.CORS` beside the
`OriginAllowed` helper it already shared.

## Consequences

- A new authenticated endpoint registers on `authed`; a new **public** `/api`
  endpoint additionally requires an allowlist entry in `api/router_test.go`,
  which is the intended friction.
- `RequireFlag`'s fail-open behaviour is unchanged — ADR 0006 still holds — but
  a wrong key is now caught by a test instead of shipping as a silently
  ungated route.
- The three admin handlers were nil-safe only because `RequireAdmin` ran first
  (see the B4 work in v0.3.60). That is now asserted on the real route table
  rather than being true by coincidence of two `main.go` expressions agreeing.
- `main_test.go` is gone; its CORS cases moved to
  `api/middleware/middleware_test.go`. `package main` has no tests because it
  no longer holds anything to test.
- `db.Stores.Available()` keeps its remaining caller (`auth.NewAuthz`) and the
  pruner gate, unchanged from ADR-less B4 reasoning: a background loop that can
  only ever fail is noise, not resilience.
