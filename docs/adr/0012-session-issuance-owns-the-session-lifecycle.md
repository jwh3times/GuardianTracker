# ADR 0012: Session Issuance Owns the Session Lifecycle

**Status:** Accepted
**Date:** 2026-08-11

## Context

Logging in, refreshing, and logging out were implemented as Gin handlers rather
than as a module those handlers called. `BungieCallback` and `RefreshToken`
together performed the OAuth code exchange, the primary Destiny membership lookup, the user
upsert, the Bungie-token store write, the JWT mint, the session create or rotate
with reuse detection, the audit event, the cookie write, and the JSON response —
nine collaborators, all reached through `*gin.Context`.

Every leaf was unit-tested and the composition was not: the callback's success
path had no Go test at all, so the token-version fallback, the admin pinning,
and the rule that a failed session write must fail the login were only ever
exercised by the browser suite.

Two consequences had already shipped:

- The `user` object was hand-built at four endpoints. Two of the four omitted
  `role`, and nothing could notice.
- After [ADR 0006](./0006-roles-feature-flags-and-admin-authorization.md)'s
  stores became never-nil interfaces, the handlers' `userStore != nil` guards
  became permanently true. A deployment without Postgres — the Minikube path in
  [ADR 0004](./0004-local-development-and-minikube-scope.md) — got
  `ErrUnavailable` from `CreateSession` and returned **500 on every login**.
  "There is no database" and "the write failed" were the same error.

## Decision

`auth.SessionIssuer` owns the browser session lifecycle: `AuthorizeURL`,
`Login`, `Refresh`, `EndSession`, `EndAllSessions`. It decides what a session is
and what invalidates one. The Gin handlers keep only what is HTTP-shaped — status
codes, the refresh cookie, audit events, and response bodies.

Failures cross that boundary as `auth.SessionError`, carrying a `Reason` that is
also the audit reason string, plus the membership and session ids. The handler
maps each `Reason` through one of two tables to a status, a message, an audit
event, and whether the refresh cookie is cleared. An unmapped reason returns 500
rather than falling through to a success.

`auth.ErrUnavailable`, translated from `db.ErrUnavailable` at the db adapter,
separates the two conditions the degraded-login defect conflated. A session write
that fails must fail the login, because the access token is rejected on its next
request when no live session row exists. A deployment with no database is the
one exception: there is no row to write and nothing that reads one, so the login
proceeds session-less.

The `user` payload has one builder, used by all four endpoints that return it.

## Consequences

- The composed rules are reachable without gin: the login success path, the
  upsert-failure fallback, admin pinning, reuse detection, legacy-session
  adoption, and the degraded path are all tested against the issuer directly.
- Login works without Postgres again. Sessions, revocation, and reuse detection
  remain Postgres-backed wherever it is configured;
  [ADR 0008](./0008-browser-refresh-cookie.md)'s guarantees are unchanged for
  every deployment that has one.
- Adding a failure mode means naming a `Reason` and mapping it, rather than
  writing another branch that constructs its own response.
- `role` now appears on all four responses. It remains a display hint;
  `GET /api/flags` stays authoritative, per ADR 0006.
- The Bungie OAuth token-endpoint call has one implementation for both grants,
  so the 90-day `refresh_expires_in` fallback is written once.
- `auth` still does not import `db`. The `db/adapters` package carries the
  sentinel translation, as it already does for Bungie token storage.
