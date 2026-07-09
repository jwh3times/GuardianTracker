# ADR 0005: Versioning with Auto-Incremented Releases

**Status:** Accepted
**Date:** 2026-07-08

## Context

The `main` branch is protected. A versioning workflow needs to identify merges
without pushing bump commits back to the protected branch or retriggering CI.

## Decision

Keep the target three-part SemVer version in the root `VERSION` file. Guardian
Tracker treats the third numeric component as the project build number, so
versions are written as `<major>.<minor>.<build>`.

On pushes to `main`, the version workflow creates an annotated version tag and a
GitHub Release of the form `v<major>.<minor>.<build>`, such as `v0.2.0` or
`v0.2.1`.

For a major or minor bump, build `0` is a valid release. If `VERSION` is changed
to `1.0.0` and no `v1.0.*` tag exists, the workflow releases `v1.0.0`; it only
increments to `v1.0.1` after a `v1.0.0` tag exists.

The changelog records human-readable version history. Tags identify exact merge
builds, and GitHub Releases provide the release artifact in GitHub.

## Consequences

- Version stamping works with protected branches.
- CI is not retriggered by version bump commits.
- Build numbers can restart at `0` when the major/minor version changes.
- Version tags, GitHub Releases, and changelog entries should stay aligned.
