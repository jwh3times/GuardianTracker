# ADR 0003: Manifest-Derived Data and Verify-First Changes

**Status:** Accepted
**Date:** 2026-07-08

## Context

Much of Guardian Tracker depends on Bungie's Destiny manifest and live API
response shapes. Those shapes can be surprising, incomplete, or different from
assumptions made from in-game UI.

Incorrect assumptions about manifest fields or weekly data can ship misleading
collection, availability, or recommendation behavior.

## Decision

Changes that depend on Bungie manifest structure, live API response shape, or
game-data semantics must be verified against real data before implementation
when the fact is load-bearing.

Public docs may describe current behavior and known gates, but they should not
promise future Bungie-derived features until the required data signal has been
verified.

## Consequences

- Roadmap items that depend on Bungie data include explicit verification gates.
- Code should prefer manifest-backed or API-backed facts over hardcoded guesses.
- When no reliable signal exists, the UI should omit or qualify the claim rather
  than present false certainty.
