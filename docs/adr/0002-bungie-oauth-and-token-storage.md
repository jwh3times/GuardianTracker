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
server-side in Postgres encrypted with AES-256-GCM. Issue Guardian Tracker JWT
access tokens and rotating per-device refresh sessions.

Refresh sessions are backed by Postgres and include reuse detection. Sign-out
everywhere bumps the user's token version. Authorization reads current role and
revocation state from the DB-backed revocation cache rather than trusting JWT
role hints.

## Consequences

- Browser code never receives Bungie OAuth tokens.
- OAuth state survives process restarts and multiple API replicas.
- Token encryption key rotation is supported through previous-key reads.
- Revocation and role changes propagate through the cache window rather than
  instantly.
- The API depends on Postgres for production-grade token persistence.
