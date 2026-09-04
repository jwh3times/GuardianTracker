# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Guardian Tracker uses the SemVer target version in `VERSION`; every merge to
`main` is stamped with an annotated version tag and GitHub Release such as
`v1.2.7`. One merge to `main` produces exactly one version, so each released
section below corresponds to a single merged pull request.

Older release notes are retained in [1.0–1.1](docs/changelog/1.0-1.1.md) and [0.x](docs/changelog/0.x.md).

## [Unreleased]

No unreleased changes.

## [1.3.5] - 2026-09-03

### Changed

- Routed every batched Manifest hash lookup through the existing signed-id
  helper and added its inverse for reading ids back, resolving the code-scanning
  finding that a user-supplied item hash reached a narrowing integer conversion.
  The signed-id encoding is unchanged, so Manifest lookups behave identically for
  every hash, including those above 2^31 that the Manifest stores as negative
  ids.

## [1.3.4] - 2026-09-03

### Changed

- Centralized weekly acquisition recommendation policy in a dedicated planner,
  with typed canonical difficulty tiers and generation-safe Efficiency index
  rebuilding that retains the previous complete recommendations during a
  Manifest replacement.

## [1.3.3] - 2026-09-03

### Added

- Added the framework-neutral browser-session client foundation with atomic
  token/user projection storage, legacy-state migration, stale-work fencing,
  cross-tab adoption, Web Locks lifecycle coordination, and production browser
  adapters; application callers remain on the existing auth path until the
  follow-up cutover.

## [1.3.2] - 2026-09-02

### Changed

- Promoted the browser E2E and accessibility job to a required pull-request
  check after its stabilization threshold, while keeping visual regression
  advisory.

## [1.3.1] - 2026-09-02

### Fixed

- Made the Windows private-workspace portability test isolate its unavailable
  1Password CLI fixture from a real `op.exe` installed later on the host PATH.

## [1.3.0] - 2026-09-02

### Added

- Added `npm run sync:main` to safely move clean public and optional private
  checkouts to `main` and fast-forward them from `origin/main`, with a
  `--skip-private` option for public-only synchronization.

## [1.2.16] - 2026-09-02

### Fixed

- Prevented in-flight Records enrichment loads from repopulating the shared
  weapon-type, exotic-weapon, or catalyst-link caches after a Manifest swap;
  raw per-membership Bungie profile records retain their existing cache
  behavior.

## [1.2.15] - 2026-09-02

### Fixed

- Preserved the Cosmetics tabpanel and its accessible tab relationship when the
  selected ownership filter has no matching items.

## [1.2.14] - 2026-09-02

### Fixed

- Removed the Node private-workspace bootstrap's temporary 1Password reference
  directory when the CLI is unavailable or authorization fails.

## [1.2.13] - 2026-09-01

### Changed

- Refreshed the pinned nginx 1.31.4 frontend runtime image digest.

## [1.2.12] - 2026-09-01

### Changed

- Updated local and container frontend tooling from Node.js 26.7.0 to 26.8.1.

## [1.2.11] - 2026-09-01

### Changed

- Updated React Router from 8.3.0 to 8.3.1.

## [1.2.10] - 2026-09-01

### Changed

- Updated `@tanstack/react-query` from 5.102.3 to 5.102.8,
  `@testing-library/react` from 16.3.2 to 16.3.3, `@types/node` from 26.3.0 to
  26.4.0, and `@vitejs/plugin-react` from 6.1.0 to 6.1.1.

## [1.2.9] - 2026-08-30

### Fixed

- Accepted whitespace-bearing 1Password secret references when their dotenv
  values and command-line arguments are double-quoted.

## [1.2.8] - 2026-08-30

### Fixed

- Omitted the Dashboard's weekly reset countdown while weekly data is loading
  or unavailable instead of displaying a fabricated zero-minute reset.

## [1.2.7] - 2026-08-28

### Added

- Added a repository policy test for local Markdown targets and heading anchors,
  and added provenance headers to every generated Markdown skill reference.
- Added a value-free Minikube Secret example that `setup.ps1` copies without
  overwriting local configuration.

### Changed

- Consolidated public documentation around one owner per subject: durable
  product intent moved from the stale PRD to `docs/product.md`, browser-test
  details moved to the frontend guide, maintainer recovery received a focused
  runbook, and old release notes moved into `docs/changelog/` archives.
- Corrected public-client OAuth, Minikube preparation, Xûr location, frontend
  preferences, route ownership, container-pin ownership, and agent guidance.

### Removed

- Removed the obsolete PRD and wireframe handoff documents after preserving
  current product principles and classifying the remaining design prototype as
  historical.

## [1.2.6] - 2026-08-28

### Changed

- Updated `@tanstack/react-query` from 5.102.2 to 5.102.3,
  `@types/node` from 26.2.0 to 26.3.0, and Oxlint from 1.79.0 to 1.80.0.

## [1.2.5] - 2026-08-27

### Changed

- Updated the pinned frontend runtime image from nginx 1.31.3 to 1.31.4 on
  Alpine 3.24.

## [1.2.4] - 2026-08-27

### Changed

- Updated `@tanstack/react-query` from 5.102.0 to 5.102.2 and
  `@types/react-dom` from 19.2.4 to 19.2.5.

## [1.2.3] - 2026-08-27

### Added

- `npm run bootstrap:private` (root `scripts/bootstrap-private.mjs`) clones the
  optional private companion repository into `private/` from any checkout or
  `git worktree`, resolving its location through 1Password. It finds the
  machine-local `.private-workspace/repository.env.ref` in the current checkout
  or in the main checkout that owns the worktree, accepts `--op-reference` or a
  credential-free `--url` override, refuses to clone over an existing
  `private/`, and keeps the resolved location out of process arguments and
  terminal output. The public-identifier policy test now covers the script.

## [1.2.2] - 2026-08-26

### Changed

- Updated `@tanstack/react-query` from 5.101.4 to 5.102.0.

## [1.2.1] - 2026-08-25

### Changed

- Updated `@testing-library/user-event` from 14.6.5 to 14.6.6.

## [1.2.0] - 2026-08-25

### Added

- Added Windows workspace-portability helpers for optional restoration of the
  independent ignored private documentation workspace, guarded 1Password-backed
  local secret restoration, and value-free public/private workspace status. The
  public setup path remains usable without private access, private branch names
  are redacted by default, and secret targets must be ignored and absent before
  restoration.
- Added public-safe VS Code settings and tasks plus Windows CI coverage for the
  portability workflow under Windows PowerShell 5.1 and PowerShell 7.

### Changed

- Removed the unused Bungie OAuth client-secret configuration and consistently
  model the application as a public OAuth client. Authorization-code grants send
  the public client ID without a `client_secret`; Bungie's access-only
  authorization is encrypted at rest and, after expiry, can be reconnected to
  the same membership without ending or rotating the Guardian Tracker browser
  session.

### Security

- Private workspace cloning and secret restoration now suppress Git tracing,
  remove all `OP_*` credentials from Git and transport-helper environments,
  evaluate the complete committed ignore-rule semantics, reject reparse-point
  paths, and require quoted Kubernetes `stringData` values before installing a
  plaintext target.

[Unreleased]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.3...HEAD
[1.3.3]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.2...v1.3.3
[1.3.2]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.16...v1.3.0
[1.2.16]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.15...v1.2.16
[1.2.15]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.14...v1.2.15
[1.2.14]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.13...v1.2.14
[1.2.13]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.12...v1.2.13
[1.2.12]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.11...v1.2.12
[1.2.11]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.10...v1.2.11
[1.2.10]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.9...v1.2.10
[1.2.9]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.8...v1.2.9
[1.2.8]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.7...v1.2.8
[1.2.7]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.6...v1.2.7
[1.2.6]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.5...v1.2.6
[1.2.5]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.4...v1.2.5
[1.2.4]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.3...v1.2.4
[1.2.3]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.2...v1.2.3
[1.2.2]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/jwh3times/GuardianTracker/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/jwh3times/GuardianTracker/compare/v1.1.2...v1.2.0
