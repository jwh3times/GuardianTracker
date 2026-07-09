# ADR 0001: Public and Private Documentation Boundary

**Status:** Accepted
**Date:** 2026-07-08

## Context

Guardian Tracker is a public GitHub repository, but some project knowledge is
not appropriate for public docs: deployment runbooks, private security review
detail, cloud resource names, raw API research dumps, and implementation
handoffs that can drift quickly.

The repo previously mixed public documentation with a deep public implementation
plan. That made the docs useful for execution but harder to audit as public
project documentation.

## Decision

Committed docs describe implemented behavior, local setup, durable decisions,
security model, and gated future work. Detailed implementation handoffs and
private operating notes belong under `private/`, which is gitignored.

Public roadmap items list status, gate, and likely size. They do not duplicate
completed work or include step-by-step private execution plans.

## Consequences

- Public docs are safer to publish and easier to audit.
- Deep plans remain available locally without becoming public sources of truth.
- Contributors can still build and test the project from committed docs.
- When a private plan changes a durable decision, the public result must be
  captured in an ADR.
