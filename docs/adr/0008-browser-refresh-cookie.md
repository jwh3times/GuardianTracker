# ADR 0008: Browser Refresh Credential Cookie

**Status:** Accepted
**Date:** 2026-07-13

## Context

Guardian Tracker needs a long-lived per-device refresh credential, but exposing
that credential to JavaScript through localStorage increases the impact of an
XSS bug. The access token remains useful to the SPA and has a short default
lifetime.

## Decision

Keep the 30-minute access JWT and non-secret user snapshot in localStorage. Send
the rotating refresh JWT only in a host-only `guardian_refresh_token` cookie
with `HttpOnly`, `SameSite=Lax`, and `Path=/api/auth`; add `Secure` in
production. Callback and refresh responses return only `{token,user}`.

Require an exact allowlisted `Origin` on the callback and refresh endpoints.
Clients send credentialed requests, refresh with an empty JSON body, and expire
the cookie on logout, logout-all, and definitive refresh failures. Do not add a
legacy localStorage bridge; existing browser sessions authenticate once again.

## Consequences

- JavaScript cannot read the long-lived refresh credential.
- Per-device rotation and reuse detection remain backed by Postgres.
- The frontend and API must be same-site for the `SameSite=Lax` design. A future
  cross-site deployment must adopt `SameSite=None; Secure` and revisit CSRF
  protection before launch.
- Access tokens remain exposed to JavaScript, but their default lifetime is
  limited to 30 minutes.
