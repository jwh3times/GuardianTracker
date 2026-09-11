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

## [1.3.36] - 2026-09-11

### Added

- One command, `npm run sync:image-pins`, now updates the container images the
  browser tests are built from. Those images are pinned to an exact version and
  fingerprint, and the version is decided by a different file than the pin lives
  in — so an automated dependency update could only ever change one of the two
  and always arrived with the build already failing. Finishing it meant looking
  up a fingerprint by hand and pasting it in. The command does that lookup and
  rewrites every affected pin instead. Nothing about the shipped app changes;
  routine dependency updates simply stop arriving broken.

### Changed

- The check that catches a mismatched image pin now names the command that fixes
  it, rather than leaving the reader to work out which file and which fingerprint
  it meant.

## [1.3.35] - 2026-09-10

### Fixed

- The difficulty badge on This Week's recommended actions now shows its proper
  colour. Top-ranked suggestions were sending the difficulty in one spelling and
  the page expected another, so those badges quietly rendered in plain grey
  instead of the green, amber or red that tells you at a glance how demanding an
  activity is. Fallback suggestions were unaffected, which is why the page only
  looked wrong some of the time.
- The "Challenging" colour itself is now lighter. Once the badge started drawing
  in that colour, it turned out not to have enough contrast against the row
  behind it to meet the WCAG AA accessibility standard — it was the only one of
  the three difficulty colours that fell short. It stays the darkest and hottest
  of them, and every other place the colour is used simply reads a little
  brighter.

### Changed

- This Week and the Dashboard now read the weekly data through one shared
  translation step rather than each trusting the server's response to already be
  in the shape the page wanted. Both pages already shared a single request; now
  they also share a single interpretation of it, so a difficulty value the app
  does not recognise resolves to an honest "Unrated" instead of silently
  reaching the page as something it cannot display. No content, ordering or
  timing changes.
- This completes the acquisition-recommendation ownership split begun in
  v1.3.4; the recommendation vocabulary is now defined in one place on the
  server and translated in one place in the browser.

## [1.3.34] - 2026-09-10

### Changed

- The This Week page's per-raid "missing" counts are now requested through a
  single narrow question — "how many of my missing items drop here?" — instead
  of the weekly page holding on to the whole recommendation-ranking engine to
  ask it. No behaviour changes: every milestone badge, including the "0 missing"
  badge on a raid you have fully collected, is identical. This is the last
  backend step of the acquisition-recommendation ownership split; only the
  frontend portion remains.
- When that count is unavailable — while the game's item index is still being
  built after startup, or for any milestone that is not a raid or dungeon — the
  page continues to render the milestone with no badge rather than a wrong one,
  exactly as before.

## [1.3.33] - 2026-09-10

### Changed

- Updated the Node.js version used to build and test the frontend from 26.8.1
  to 26.8.2, a patch release. This is build tooling only — the deployed
  frontend is served by nginx and does not run Node.

## [1.3.32] - 2026-09-10

### Fixed

- Very large item lookups against the game's item database can no longer fail
  outright. One of the internal batch reads asked for every item in a single
  query instead of splitting the request into batches like its siblings did, so
  a big enough request could exceed the database's limit on how many values one
  query may carry. It now batches the same way everything else does.

### Changed

- The ten places that read batches of definitions out of the game's item
  database now share one implementation instead of each carrying its own copy of
  the batching, and four internal read methods that nothing called were removed.
  No behaviour changes; this is groundwork that makes the next manifest change
  smaller and harder to get wrong.

## [1.3.31] - 2026-09-09

### Fixed

- Your wish list no longer quietly claims it has forgotten what your items are.
  When the item database could not be read, every saved entry rendered as
  "Unknown Item" and the page looked successful — indistinguishable from the
  items genuinely having been removed from the game. That case now says the
  item database is still downloading, the same as everywhere else in the app,
  and an item that really has left the game still keeps its own row with your
  priority and notes intact.
- Saving an item can no longer report a failure for a save that actually
  worked. Adding an item looked it up a second time after storing it, so a
  badly timed item-database update could return an error for an entry that was
  already on your list — and trying again would then say it was a duplicate.
- Editing the priority or notes on an item that has left the game now works
  even while the item database is being updated, instead of depending on when
  the edit happened to land.

### Changed

- Armor on your wish list now shows what it is — "Helmet", "Gauntlets", and so
  on — instead of just "Armor". The wish list previously worked out item
  details on its own; it now reads the same source the collection grid and item
  detail already use, so an item reads the same everywhere. Weapons and
  everything else are unchanged.
- Opening your wish list looks items up once for the whole list rather than
  once per item, so a long list loads with a single lookup instead of dozens.

## [1.3.30] - 2026-09-09

### Fixed

- The documented local Staticcheck command now pins the Go toolchain, so it
  actually checks the code. Staticcheck 2026.1 is compiled by whatever Go is
  active, and on a newer toolchain it cannot decode the standard library's
  export data — it fails while loading and analyzes nothing. Every error names
  a standard-library package rather than project code, and the empty output
  afterwards is indistinguishable from a clean run, so an unpinned run reads as
  "no findings". `go.mod`'s `toolchain` directive is a floor rather than a pin
  and does not prevent it. `AGENTS.md` and `SETUP.md` now say why the prefix is
  there, so it is not mistaken for noise. CI was never affected: it installs the
  pinned Go before running.

### Added

- A repository policy test for that pin, so it cannot drift. Rather than naming
  the Go release a fifth time, it derives the expected value from the workflows'
  `GO_VERSION` and fails the build if the documented commands, the two
  workflows, or `go.mod`'s toolchain stop naming the same release. It also
  records the one deliberate exemption — CI's own invocation stays unpinned
  because the job already installed that Go, and a second inline pin would be a
  duplicate that could drift on its own.

## [1.3.29] - 2026-09-09

### Fixed

- Refreshing your data now actually means the next thing you load is fresh.
  Refreshing cleared the stored copies of your collection, characters,
  catalysts, crafting patterns, and seals, but a request that had already
  started fetching in the background could finish a moment later and quietly
  put its older copy back. The refresh said it worked and the next page you
  opened could still show you the data from before it. Anything already
  loading when you refresh now still finishes for the page that asked for it,
  but is no longer kept for the next one. Refreshing your own data has never
  affected anyone else's, and still does not.

## [1.3.28] - 2026-09-09

### Fixed

- Requests for your collection, characters, catalysts, crafting patterns, or
  seals now have to name the platform you actually signed in on, not just your
  membership number. Naming a different platform is refused before the app
  contacts Bungie or touches any cached data. This never exposed anyone else's
  data — the membership number was always checked against your own sign-in, and
  a mismatched platform names an account that does not exist rather than
  someone else's — but it did spend a Bungie request and leave stray cache
  entries behind. Refused requests now stop cleanly instead of continuing.

### Changed

- The collections summary and the full collection are now two separate reads
  rather than one read trimmed down. The Dashboard's summary does no item work
  and makes no vendor request at all, so it stays cheap; the full grid gets one
  complete answer per item — what the item is, whether you own it, and where it
  is on sale right now — instead of three separate lists that could disagree.
  What the app sends to the browser is unchanged.
- Vendor availability is now joined only for the full collection, only after
  your ownership data has loaded, and only for items your collection actually
  tracks. It stays best effort: if vendor data is unavailable, the collection
  still loads with nothing marked on sale rather than failing, and one request's
  vendor rotation is no longer able to linger into later reads.
- Refreshing a membership is now one operation owned by the collections service
  rather than a list the web layer kept, so the refresh still reaches
  collections, characters, and records without the route having to know that.

## [1.3.27] - 2026-09-09

### Changed

- The wish list now owns what a saved entry means. Priorities, note limits,
  bulk-selection rules, and the difference between "already saved", "not
  found", and "no database" are decided in one place instead of being spread
  across the HTTP handler and PostgreSQL error codes. Every wish list status
  code, error code, and message stays exactly as it was.
- Saving an item while the item database is unreadable now fails with the same
  "still downloading" response the rest of the app uses, rather than silently
  saving an item nothing could confirm exists. An item the database has read
  and does not contain is still refused as an unknown item.
- This Week reads saved items through a single narrow question instead of
  reaching into wish list storage. It still drops personalization rather than
  failing when the wish list cannot be read, and now records that it did.

### Fixed

- Wish list notes are measured in characters rather than bytes, so notes
  containing emoji or accented characters are no longer rejected below the
  500-character limit the database itself enforces.

## [1.3.26] - 2026-09-09

### Changed

- Updated `golang.org/x/time` from 0.15.0 to 0.16.0 in the API service.

## [1.3.25] - 2026-09-09

### Changed

- Collections now reads every item's name, icon, type, rarity, acquisition
  sources, and farm-only status from the shared item-facts owner instead of
  deriving its own projection from the manifest. The collection grid, item
  drawer, and wish list can no longer describe the same item differently. The
  REST shape and the values it carries are unchanged.
- A manifest swap now refreshes a cached collection analysis under a generation
  fence: the manifest-derived half is rebuilt from one coherent manifest, the
  rate-limited Bungie ownership fetch is kept, and analysis that began before
  the swap can still answer its own request but is never left behind for the
  next one. A swap also costs one shared catalog read rather than a manifest
  query per active membership.
- The API notifies the item-facts owner of a new manifest before Collections,
  so a collection analysis can never pair a new presentation tree with facts
  from the manifest it replaced.

## [1.3.24] - 2026-09-08

### Fixed

- `CHANGELOG.md`'s reference-link footer now defines every released version.
  Eight releases (1.3.16 through 1.3.23) had a version heading but no link
  definition, so those headings rendered as plain text, and `[Unreleased]`
  compared against a tag eight releases stale.

### Added

- A repository policy test for that footer, so it cannot fall behind again. It
  requires a definition per released section, rejects definitions for sections
  this file no longer holds, checks each one compares from the section directly
  below it, and holds `[Unreleased]` to the newest release. The `/ship` skill now
  writes the footer alongside the version section, including for the bot releases
  it backfills.

## [1.3.23] - 2026-09-08

### Added

- A repository policy test that keeps the jest-dom matcher type shim tied to the
  upstream gap it covers. It fails `Format Check` once
  `@testing-library/jest-dom` stops shipping the Vitest 4 assertion shape, so the
  shim is deleted rather than left in place indefinitely — a redundant type
  augmentation keeps working and would redden no other check. It also fails if
  the shim and its scoped lint exemption are ever separated.

## [1.3.22] - 2026-09-08

### Changed

- Updated Vitest and `@vitest/coverage-v8` from 4.1.11 to 5.0.0.

### Fixed

- Restored jest-dom matcher types under Vitest 5, which inlined `@vitest/expect`
  and dropped the `jest.Matchers` bridge that carried them into `expect(...)`.
  jest-dom's own entry point augments the Vitest 4 assertion shape, which cannot
  merge with the current one, so every matcher silently lost its type while
  continuing to work at runtime. The types are restored through Vitest's
  documented extension point for third-party Jest matcher libraries, not by
  suppressing type checking.

## [1.3.21] - 2026-09-08

### Fixed

- Corrected the frontend visual-baseline documentation: repository policy tests
  detect a stale Playwright image pin but do not update it. A `@playwright/test`
  bump arrives through Dependabot's npm ecosystem, which does not edit
  Dockerfiles, so `ARG PLAYWRIGHT_IMAGE` in `frontend/Dockerfile.playwright` must
  be bumped by hand — tag and digest. The note now names the checks that fail
  until it is, and records that a Playwright bump does not by itself invalidate
  the committed baselines.

## [1.3.20] - 2026-09-08

### Changed

- Updated Playwright from 1.62.1 to 1.63.0, including the pinned
  `mcr.microsoft.com/playwright` image used to regenerate visual baselines.

## [1.3.19] - 2026-09-08

### Changed

- Updated `github.com/mattn/go-sqlite3` from 1.14.50 to 1.14.52.

## [1.3.18] - 2026-09-07

### Changed

- Updated `@testing-library/user-event` from 14.6.6 to 14.6.7, `@types/node`
  from 26.4.0 to 26.4.1, `@types/react-dom` from 19.2.5 to 19.2.7, Oxlint from
  1.80.0 to 1.81.0, and PostCSS from 8.5.26 to 8.5.28.

## [1.3.17] - 2026-09-07

### Changed

- Updated the frontend runtime image digest for nginx-unprivileged
  1.31.5-alpine3.24.

## [1.3.16] - 2026-09-07

### Security

- Capture private-workspace bootstrap diagnostics and report value-free failures,
  cleaning temporary Git configuration and reference files on handled failures.
- Validate tracked paths, committed ignore rules, and symlinks before cloning;
  install through an ignored staging directory and preserve existing private
  clones and their uncommitted files.
- Remove tracing and injected Git routing controls from bootstrap child
  environments. Clarify the limits of repository-URL privacy in process arguments.

### Changed

- Require agent-completed work to record remaining human actions as private
  follow-up issues on the project board, with linked step-by-step private-wiki
  procedures and completion evidence. Apply the handoff contract during
  documentation updates, shipping, session closeout, and wizard delivery.

### Fixed

- Accept equivalent filesystem paths when validating bootstrap repository roots
  and use portable file URLs for the Windows regression harness.

## [1.3.15] - 2026-09-06

### Security

- Record successful session refreshes in the audit trail using verified
  membership/session identifiers and existing request metadata, without storing
  tokens or cookie values. Audit-write failures preserve the refresh response.
- Display successful refresh events as “Session refreshed” in the admin Audit
  Log, including its existing Sessions filter.

## [1.3.14] - 2026-09-06

### Security

- Refuse self-service role changes when the current database role is admin,
  including requests that began before a concurrent promotion. Commit permitted
  changes and their audit events together; audit failures roll back the change.
- Clear the local role cache after successful bootstrap administrator promotion.
  Self-service changes retain existing sessions and record the authoritative
  previous role in the audit trail.

## [1.3.13] - 2026-09-06

### Security

- Require PostgreSQL server identity verification in production guidance using
  `verify-full`, the intended hostname, and a trusted CA configuration. Document
  runtime certificate mounting and staging verification while retaining local
  development defaults.
- Add a real pgx TLS regression proving trusted matching certificates succeed
  and untrusted or wrong-host certificates fail before authentication traffic.

## [1.3.12] - 2026-09-06

### Security

- Exclude local environment variants, keys, certificates, browser credentials,
  and generated artifacts from backend and frontend Docker build contexts while
  retaining required sources and value-free root environment templates.
- Verify build-context isolation before CI image builds using real Docker
  probes with synthetic files, including checks that required sources remain
  available and untested Dockerfile-specific policies cannot bypass validation.

## [1.3.11] - 2026-09-06

### Security

- Bind OAuth login and reconnect completion to a short-lived HttpOnly browser
  transaction cookie. Completion requires the matching signed state and cookie;
  the latest sign-in attempt supersedes earlier pending attempts across tabs.
- Coordinate authorization start and reconnect through the shared browser
  lifecycle lock so concurrent responses cannot overwrite a newer transaction.
  Both operations now require Web Locks support.

## [1.3.10] - 2026-09-06

### Fixed

- Clear and replace account-bound query caches and reset provider state on logout
  or membership changes. Late query responses, optimistic rollback, and delayed
  mutations cannot restore the departing account's data or act for its replacement.
- Reset local preferences, reconnect intent, and weekly checklist marks at account
  boundaries while preserving collection filters and membership-specific character
  choices. Same-membership token refresh retains caches, checklist marks, and
  unsaved editor state.
- Handle application-shell sign-out failures consistently with Settings.

### Changed

- Completed ADR 0017's browser session projection with identity cleanup at
  application composition and regression coverage for cross-tab account changes,
  logout, delayed work, and application remounts.

## [1.3.9] - 2026-09-05

### Changed

- Routed login, authenticated requests, refresh, and logout through one browser
  session client. React now observes its atomic access-token/user projection;
  valid legacy storage migrates automatically. Login completion and refresh
  require Web Locks support for coordination across tabs.

### Fixed

- Coordinated concurrent tab refreshes so requests share one cookie rotation and
  replacement token. Kept request retries and reconnect navigation bound to the
  initiating session, and preserved local logout finality.
- Removed duplicate profile reads and auth storage/event handling from the
  frontend. Identity-bound query-cache cleanup remains a separate follow-up.

## [1.3.8] - 2026-09-04

### Changed

- Updated the API builder image from Go 1.27.0-alpine to 1.27.1-alpine and the
  frontend runtime from nginx-unprivileged 1.31.4-alpine3.24 to
  1.31.5-alpine3.24.

## [1.3.7] - 2026-09-04

### Fixed

- Made the efficiency engine's generation-fencing test deterministic. It failed
  intermittently under full-suite load and on CI, where it could block an
  unrelated merge. The manifest generation fence itself was correct and is
  unchanged: the test's fixture keyed its rows on a call counter, so a
  legitimate second rebuild of the same generation received a later
  generation's data and resembled a stale publish. The fixture now returns the
  rows belonging to the installed manifest version, and it stops accepting new
  builds before releasing the obsolete one, so the assertion reflects the
  fence's verdict rather than the order the Go scheduler happened to pick.

## [1.3.6] - 2026-09-04

### Added

- Recorded ADR 0022: task status lives on GitHub Issues and a linked project
  board, while interface contracts and agent operating context stay in the
  repository. The dividing line is whether the content changes when a pull
  request merges. It supersedes exactly one statement in ADR 0001 —
  implementation handoffs no longer belong under `private/` — and ADR 0001 now
  links back to it.
- Added a `Work Tracking` section to `AGENTS.md` covering the board, the
  public-repository-by-default rule for filing issues, the draft-versus-issue
  convention that keeps the open-issue list readable as live status, and the
  rules every architecture-deepening slice must satisfy. Those slice rules were
  previously reachable only from a private planning file.
- Documented `gh project` command forms in `docs/agents/issue-tracker.md`,
  including that a draft item cannot carry a native issue dependency and must
  record its order in the `Blocked By` field until it is converted.

### Changed

- The `end-session` skill now advances the project board rather than a private
  status document, so closing a session no longer recreates the file this
  release retires.

### Fixed

- Restored the missing `1.3.4` and `1.3.5` changelog comparison links and
  corrected the `Unreleased` comparison base, which still pointed at `v1.3.3`.

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

[Unreleased]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.36...HEAD
[1.3.36]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.35...v1.3.36
[1.3.35]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.34...v1.3.35
[1.3.34]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.33...v1.3.34
[1.3.33]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.32...v1.3.33
[1.3.32]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.31...v1.3.32
[1.3.31]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.30...v1.3.31
[1.3.30]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.29...v1.3.30
[1.3.29]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.28...v1.3.29
[1.3.28]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.27...v1.3.28
[1.3.27]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.26...v1.3.27
[1.3.26]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.25...v1.3.26
[1.3.25]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.24...v1.3.25
[1.3.24]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.23...v1.3.24
[1.3.23]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.22...v1.3.23
[1.3.22]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.21...v1.3.22
[1.3.21]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.20...v1.3.21
[1.3.20]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.19...v1.3.20
[1.3.19]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.18...v1.3.19
[1.3.18]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.17...v1.3.18
[1.3.17]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.16...v1.3.17
[1.3.16]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.15...v1.3.16
[1.3.15]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.14...v1.3.15
[1.3.14]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.13...v1.3.14
[1.3.13]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.12...v1.3.13
[1.3.12]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.11...v1.3.12
[1.3.11]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.10...v1.3.11
[1.3.10]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.9...v1.3.10
[1.3.9]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.8...v1.3.9
[1.3.8]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.7...v1.3.8
[1.3.7]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.6...v1.3.7
[1.3.6]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.5...v1.3.6
[1.3.5]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.4...v1.3.5
[1.3.4]: https://github.com/jwh3times/GuardianTracker/compare/v1.3.3...v1.3.4
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
