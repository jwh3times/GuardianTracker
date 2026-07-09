# ADR 0006: Roles, Feature Flags, and Admin Authorization

**Status:** Accepted
**Date:** 2026-07-08

## Context

Guardian Tracker has early-access feature flags and an admin console. Role
changes affect authorization and need to be auditable. UI-only gating is not a
security boundary.

## Decision

Roles are stored in Postgres as ordered tiers: `standard`, `beta`, `alpha`, and
`admin`. Admin membership can be pinned by configured Bungie membership IDs at
login or granted by an existing admin.

Admin-only API routes are enforced server-side. Feature flags may hide or show
frontend surfaces, but protected backend behavior must use server-side
authorization. Role and feature-flag changes write audit events, and demoting the
last admin is blocked.

## Consequences

- JWT role claims are display hints, not authorization truth.
- Admin and tier checks belong in middleware or handlers, not only in React.
- Every new privileged endpoint must define its server-side authorization rule.
- Changes to roles, flags, or audit semantics require docs and tests.
