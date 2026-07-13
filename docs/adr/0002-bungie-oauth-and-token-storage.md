# ADR 0002: Bungie OAuth and Token Storage

**Status:** Accepted
**Date:** 2026-07-08

## Context

Guardian Tracker needs Bungie OAuth access to read player profile, collection,
record, character, and vendor-related data. The app also needs its own session
model so the frontend can call the API without sending Bungie tokens to the
browser.

## Decision

Use Bungie OAuth with stateless HMAC-signed CSRF state. Store Bungie OAuth tokens
server-side in Postgres encrypted with AES-256-GCM. Persist an exact positive
key version with every encrypted row; new writes use the current key/version and
reads accept only an exact current or configured previous match. Issue Guardian
Tracker JWT access tokens and rotating per-device refresh sessions. Browser
delivery of the refresh credential is defined by
[ADR 0008](./0008-browser-refresh-cookie.md).

Refresh sessions are backed by Postgres and include reuse detection. Sign-out
everywhere bumps the user's token version. Authorization reads current role and
revocation state from the DB-backed revocation cache rather than trusting JWT
role hints.

## Consequences

- Browser code never receives Bungie OAuth tokens.
- OAuth state survives process restarts and multiple API replicas.
- Token encryption key rotation is supported through exact previous-key/version
  reads; unknown versions fail.
- Revocation and role changes propagate through the cache window rather than
  instantly.
- The API depends on Postgres for production-grade token persistence.
