### Changed

- The changelog scheme moves from `per-merge` to `stamp-on-tag` (ADR 0027, superseding ADR 0009
  and part of ADR 0005). A branch no longer predicts the version its merge will mint: `/ship`
  writes its entries to its own `changelog.d/<branch-slug>.md` fragment and never edits
  `CHANGELOG.md`. After each merge to `main`, `version.yml` runs the new
  `scripts/stamp-changelog.mjs`, which turns every untagged merge's fragments (or, with none, its
  PR title) into a dated section with its compare link, pushes one
  `chore(release): v<x.y.z> [skip ci]` commit through the release GitHub App the main ruleset
  lets bypass it, tags each merge commit, and creates its GitHub Release from that section. Open
  PRs now merge in any order without renumbering, and Dependabot merges get a section without a
  backfill.
- The `Changelog Version` check keeps its required name but now verifies the branch's fragments
  and fails when a branch edits a released section, a `## [Unreleased]` entry or a fragment
  already on `main`. Dependabot PRs pass it without the old author exemption.
- The shared `ship`, `docs-updater` and `end-session` skills are reinstalled from
  `jwh3times/skills` with `stamp-on-tag` support, the canonical `check-changelog.mjs` is updated,
  and `skills-lock.json` now records the source commit of every installed skill.
