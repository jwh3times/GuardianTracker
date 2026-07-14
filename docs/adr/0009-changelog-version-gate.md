# ADR 0009: Changelog-Version Gate and the Version Oracle

**Status:** Accepted
**Date:** 2026-07-13

## Context

ADR 0005 tags every merge to `main` with an auto-incremented build number and
states that "version tags, GitHub Releases, and changelog entries should stay
aligned," but nothing enforced that. In practice, `CHANGELOG.md` entries
accumulated under a single `[Unreleased]` section across nine tagged releases,
and three dependabot merges landed with no changelog entry at all, because
dependabot never touches `CHANGELOG.md`. The "next build number" rule also
lived only inside `.github/workflows/version.yml`, with no way for a PR-time
check or a human/agent workflow to compute the same answer ahead of the merge.

## Decision

Extract the build-number rule from `version.yml` into `scripts/next-version.sh`,
a single script that prints the version the *next* merge to `main` will mint.
Three consumers call it and none reimplement the rule: `version.yml` (tags the
merge), a new `changelog-version` CI job (verifies a PR agrees with it), and the
`/ship` skill (`.claude/skills/ship/SKILL.md`, writes the changelog section for
it before opening a PR).

Add `changelog-version` as a required PR-only status check named
`Changelog Version`. It fails a human-authored PR when `CHANGELOG.md`'s top
`## [x.y.z]` section does not equal the version `scripts/next-version.sh`
predicts. Bot-authored PRs (`github.event.pull_request.user.type == "Bot"`) are
exempt, since dependabot never edits `CHANGELOG.md`; `/ship` backfills an entry
for any tag it finds with no matching changelog section on its next run, which
is what makes the exemption safe rather than a silent gap.

## Consequences

- A merge's changelog entry is written for the version that merge will actually
  mint, not left in a permanent `[Unreleased]` bucket.
- The build-number rule has exactly one implementation; `version.yml`, CI, and
  `/ship` cannot disagree about what "next version" means.
- Bot PRs stay unblocked, but a released version can no longer go permanently
  undocumented — the next `/ship` run backfills it.
- `Changelog Version` becomes a fifth required status check once the repository
  ruleset is updated to include it (it cannot be required before the check
  exists on a run).
- Anyone editing the build-number rule must edit `scripts/next-version.sh` only
  and re-run `scripts/next-version.test.sh`; editing `version.yml`'s tagging
  step alone would desync it from the CI guard and `/ship`.
