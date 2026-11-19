# ADR 0027: Stamp Each Merge's Changelog Section When It Is Tagged

**Status:** Accepted
**Date:** 2026-11-19
**Supersedes in part:** [ADR 0005](./0005-versioning-with-build-tags.md) and
[ADR 0009](./0009-changelog-version-gate.md); see the last paragraph of the decision.

## Context

ADR 0009 made `Changelog Version` fail a PR whose top `CHANGELOG.md` section did not name the
version `scripts/next-version.mjs` predicted for its merge. The prediction was made at ship time,
so every merge invalidated every open PR's section: branches landed one at a time, two open PRs
predicting the same version renumbered and conflicted under `## [Unreleased]`, and the versions
Dependabot merges minted needed a backfill on the next ship, because bot PRs were exempt from the
check.

## Decision

A branch no longer names the version its merge will mint. `/ship` writes the branch's entries to
its own `changelog.d/<branch-slug>.md` fragment and never edits `CHANGELOG.md`. After the merge,
`.github/workflows/version.yml` runs `scripts/stamp-changelog.mjs`, which turns each untagged
merge's fragments (or, with none, its PR title) into a dated section, keeps the compare-link
footer current, commits the sections with `[skip ci]`, and tags the merge commit. The version is
still `scripts/next-version.mjs`'s computation over the root `VERSION` file (ADR 0005's build
number rule), now made once, at tag time, over the merges in the order they landed.

The stamp commit goes straight to `main`, which the main ruleset otherwise forbids, so
`version.yml` pushes it with a token from a release GitHub App listed as the ruleset's one bypass
actor (`GITHUB_TOKEN` cannot be one). `Changelog Version` keeps its name, which the ruleset
requires, and now checks the fragments instead of a predicted version, with no bot exemption. The
scheme and its operating rules are `stamp-on-tag` in `jwh3times/skills`, which holds the canonical
scripts.

This supersedes ADR 0009's gate (a PR's top section must name the predicted version, and bot PRs
are exempt and backfilled), and the parts of ADR 0005 that say the workflow never commits to
`main`, needs no bypass actor, and never uses `[skip ci]`. ADR 0005's version format and build
numbering, and ADR 0009's single implementation of that rule in `scripts/next-version.mjs`, stand.

## Consequences

- Open PRs merge in any order without renumbering, and a Dependabot merge gets its section
  without a backfill.
- Every first-parent commit on `main` is a release, so rebase and squash merging stay off, and
  nobody pushes several commits straight to `main`.
- `[skip ci]` keeps the stamp commit from starting the CI/CD Pipeline, Browser Quality or
  `version.yml` itself; CodeQL default setup still analyzes it.
- Each repository that uses the scheme installs the App, lists it as an "Always allow" bypass
  actor on its `main` ruleset, and holds its client ID and private key as the
  `RELEASE_APP_CLIENT_ID` variable and `RELEASE_APP_PRIVATE_KEY` secret.
