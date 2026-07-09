# ADR 0005: Versioning with Auto-Incremented Version Tags

**Status:** Accepted
**Date:** 2026-07-08

## Context

The `main` branch is protected. A versioning workflow needs to identify merges
without pushing bump commits back to the protected branch or retriggering CI.

## Decision

Keep the target three-part version in the root `VERSION` file. On pushes to
`main`, the version workflow creates an annotated version tag of the form
`v<major>.<minor>.<build>`, such as `v0.2.0` or `v0.2.1`.

The changelog records human-readable version history. Tags identify exact merge
builds.

## Consequences

- Version stamping works with protected branches.
- CI is not retriggered by version bump commits.
- Build numbers can restart at `0` when the major/minor version changes.
- Version tags and changelog entries should stay aligned.
