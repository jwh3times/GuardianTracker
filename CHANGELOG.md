# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Guardian Tracker uses the SemVer target version in `VERSION`; every merge to
`main` is stamped with an annotated version tag and GitHub Release such as
`v0.3.20`, `v0.3.21`, and `v0.3.22`. One merge to `main` produces exactly one
version, so each released section below corresponds to a single merged pull
request.

## [Unreleased]

No unreleased changes.

## [1.1.0] - 2026-08-24

### Added

- `GET /api/preferences` now reports whether its result is authoritative through
  an additive `persisted` field. Successful stored and fresh-account reads return
  `true`; degraded defaults returned while persistence is unavailable return
  `false` and retain the existing `200` response.

### Changed

- Preference defaults, validation, atomic partial updates, and irreversible
  onboarding completion now belong to a dedicated Preferences service behind a
  membership-keyed persistence adapter and a thin HTTP handler. Internal
  Guardian Tracker user IDs and PostgreSQL-shaped values no longer cross into
  preference policy or Gin.

### Fixed

- Independent preference updates are now applied as one field-presence-aware SQL
  statement, preventing concurrent partial requests from restoring stale values.
  Degraded writes retain the existing `503 DB_UNAVAILABLE` response.

## [1.0.16] - 2026-08-20

### Added

- Added the `end-session` agent skill, which closes out a work session by
  recording what it produced before clearing what holds the evidence. It
  inventories the session first, routes each discovery to the single place that
  owns it — agent memory, a `private/` working document, or a GitHub issue — and
  only then cleans the local workspace: uncommitted work resolved with the
  developer, the regenerable test and coverage artifacts removed by name, and the
  throwaway test and end-to-end database containers stopped. It leaves the
  Bungie manifest volume, environment files, and private documents in place, and
  it neither pushes nor merges; shipping remains `/ship`'s job.

## [1.0.15] - 2026-08-20

### Added

- Item detail, wish list, and collection views can now read one canonical
  description of an item — its name, icon, slot-specific type, rarity,
  collection category, every linked collectible, the combined list of ways it
  can be acquired, and whether it can still be reacquired. Previously each of
  those views joined the manifest itself and had already drifted apart. Item
  detail reads through the new description now; collections and the wish list
  follow as they are reworked.

### Fixed

- The item-detail endpoint reported a generic "Armor" for every armour piece
  while collections reported the equipment slot, so the same item was described
  two different ways depending on where it was viewed. Item detail now reports
  the slot — "Helmet", "Chest Armor", "Class Item" — matching collections.
  Weapons, mods, emblems, and ships are unaffected.
- An item's acquisition sources and its reacquirable status no longer depend on
  the order the manifest happens to return rows in. Where an item has more than
  one collectible, the one that decides reacquirable status is now chosen
  deterministically. Verified against a current real manifest: this changes the
  chosen collectible for 17 items and changes the answer for none of them.

### Changed

- Item lookups now read item definitions and their collectibles together under a
  single lock, so a manifest update landing mid-request can no longer pair an
  item from one manifest version with acquisition sources from another. Work
  that began before a manifest update still answers the request that started it
  but can no longer leave a stale result behind for later requests.

## [1.0.14] - 2026-08-20

### Added

- Added the manifest generation fence that will stop a slow request from
  republishing data derived from a manifest version that has since been
  replaced. Work now has a way to capture the generation it started under and
  install its result only while that generation is still current; a request that
  loses the race still returns its own coherent result but leaves nothing behind
  for later requests to inherit. Advancing the generation and clearing the
  owner's state happen as one transition, so no reader can observe a moved
  generation over data that has not yet been cleared.

  This is the seam only. No module uses it yet, so manifest-derived data behaves
  exactly as before; each module adopts the fence as part of its own later
  change.

## [1.0.13] - 2026-08-20

### Fixed

- Removed a race from the search-handler test suite that could fail any pull
  request, including ones touching no Go code. The cold-start test asserted that
  the first search against an unbuilt index returns 503, but the handler starts
  the rebuild asynchronously before checking readiness, so a fast build could
  legitimately return 200 instead. Application behavior is unchanged; the test
  now accepts either response and still proves that a not-ready search triggers
  the rebuild.

## [1.0.12] - 2026-08-20

### Added

- Documented the architecture-decision status vocabulary in the ADR index.
  A record now states plainly whether its decision describes how the code
  works, is sequenced but not yet built, or has been implemented in a named
  version — so a reader can tell a plan from a description without opening the
  record. The index also records the narrow, named, bidirectional supersession
  convention.

### Changed

- Marked the eight architecture decisions ADR 0014 through ADR 0021 as
  sequenced for implementation rather than pending a handoff, now that their
  dependency order, slice boundaries, and verification gates are settled. These
  remain accepted plans; the code does not work this way yet, and each record
  will name the version that implements it as its work lands.

## [1.0.11] - 2026-08-20

### Added

- Documented the accepted Preferences synchronization contract: one
  framework-neutral preferences client owns the browser preference projection
  through a membership-keyed single-slot cache, one read per resolved
  membership, single-flight coalescing writes that roll back and surface a
  typed error instead of failing silently, same-membership cross-tab adoption
  ordered by a monotonic revision, and an onboarding gate that stays closed in
  every unresolved, failed, and degraded state. It also accepts an additive
  `persisted` field on `GET /api/preferences` so a genuinely new account is
  distinguishable from unavailable persistence. Implementation remains deferred
  to the final Wayfinder handoff.

### Changed

- Recorded the repository's first architecture-decision supersession, together
  with the convention it establishes: supersede a named statement rather than
  reissuing a record, and link the two decisions in both directions.

## [1.0.10] - 2026-08-20

### Changed

- Re-pinned the frontend runtime image to the current
  `nginxinc/nginx-unprivileged:1.31.3-alpine3.24` publication digest. The image
  tag is unchanged; only the digest advanced.

## [1.0.9] - 2026-08-20

### Changed

- Advanced the pinned `docker/setup-buildx-action` release commit from `v4.2.0`
  to `v4.3.0` in the CI/CD workflow.

## [1.0.8] - 2026-08-20

### Added

- Documented the accepted frontend data-access ownership contract: one
  data-access module per domain resource owning its query identity, endpoint
  paths, projection to domain types, mutations with their optimistic
  coordination, and its own invalidation. Modules export hooks, membership-scoped
  reads take the current membership from the browser session client rather than
  as an argument, a membership-refresh module invalidates through each module
  instead of naming its keys, and the import boundary is enforced in CI rather
  than documented. Implementation remains deferred to the final Wayfinder
  handoff.

## [1.0.7] - 2026-08-19

### Added

- The repository policy tests now enforce PostgreSQL container-image alignment
  in `format-check`. A Compose PostgreSQL bump that leaves the CI integration
  database, another Compose service, or a documented image reference on the old
  version now fails the build instead of drifting unnoticed, and both refs must
  stay tag-qualified and digest-pinned.

## [1.0.6] - 2026-08-17

### Changed

- Updated the PostgreSQL container images from 18.4 to 18.6, covering the three
  Compose services, the `Test Go Services` integration database, and the
  documented image references in `SETUP.md` and the agent guides.

## [1.0.5] - 2026-08-17

### Changed

- Updated frontend development dependencies: `@axe-core/playwright` 4.12.1 to
  4.13.0 and `@testing-library/user-event` 14.6.3 to 14.6.4.

## [1.0.4] - 2026-08-17

### Added

- Documented the accepted acquisition-recommendation ownership contract,
  including its complete outcome interface, separation from Efficiency ranking,
  preserved fallback policy, and canonical backend/frontend difficulty-tier
  projection. Implementation remains deferred to the final Wayfinder handoff.
- Documented the accepted browser-session projection contract, including its
  atomic user snapshot, origin-wide refresh coordination, identity-boundary
  cleanup, and preservation of server-side session ownership. Implementation
  remains deferred to the final Wayfinder handoff.
- Documented the accepted complete Collections contract, including its typed
  summary/full outcomes, best-effort availability completion, membership-refresh
  publication fences, and Item/Collections observer ordering. Implementation
  remains deferred to the final Wayfinder handoff.
- Documented the accepted Wish list and Preferences ownership contracts,
  including the two-stage Wish list construction, complete entry outcomes,
  explicit unknown-Item tombstones, atomic preference patches, and verified
  Unicode note-length fix. Implementation remains deferred to the final
  Wayfinder handoff.

## [1.0.3] - 2026-08-17

### Added

- Documented the accepted Item acquisition-facts ownership contract, including
  its canonical Items interface, Manifest-generation coherence rules, module
  boundaries, and verification gate before implementation.

## [1.0.2] - 2026-08-16

### Added

- Documented the accepted Manifest-derived publication contract, including its
  owner-local generation fence, module-specific invalidation boundaries, and
  surviving verification requirements.

## [1.0.1] - 2026-08-14

### Security

- The API service now builds and runs on Go 1.26.6, picking up standard-library
  fixes for six advisories that were reachable from application code: quadratic
  path resolution in `net/url` (GO-2026-6218), unbounded post-handshake messages
  in `crypto/tls` (GO-2026-6090), a missing `ReadHeaderTimeout` on the
  unencrypted HTTP/2 check in `net/http` (GO-2026-6089), missing recursion-depth
  guards in `encoding/xml` (GO-2026-6088) and `encoding/asn1` (GO-2026-5972), and
  acceptance of ASCII-only Punycode labels in `x/net/idna` (GO-2026-5026).

### Changed

- The Go toolchain pin moved from 1.26.5 to 1.26.6 everywhere it is declared —
  the module toolchain directive, the api-service builder image, both GitHub
  Actions workflows, and the browser-test build snippet — so local, CI, and
  container builds stay on one patch release. The supported language version is
  unchanged, so no consumer or operator action is required.

## [1.0.0] - 2026-08-13

### Changed

- Collection and wishlist items now expose every collectible-derived acquisition
  source with its own difficulty and raid/dungeon facets. The item-level
  `difficulty` and `sources` REST fields were replaced by `acquisitionSources`,
  requiring API consumers to migrate to the new source-specific contract.
- Collection cards summarize multiple acquisition paths without inventing a
  primary source, item drawers list every source and its tier, and difficulty
  filters match items having any source in the selected tier. The ambiguous
  item-level difficulty sort was removed; saved legacy sort values fall back to
  rarity.
- Wishlist provenance is now shown independently from live vendor availability,
  so an item's origin remains visible while it is currently being sold.

### Fixed

- Acquisition recommendations and milestone missing counts now count an item
  only once within each applicable source union, even when multiple collectible
  definitions reference it.

## [0.3.82] - 2026-08-12

### Changed

- Documentation and internal code now consistently distinguish Bungie accounts,
  tracked Destiny memberships, Guardian Tracker users, selected characters,
  items, collectibles, and currently available acquisition recommendations.
  Existing REST routes, JSON fields, and JWT claims remain compatible.

### Fixed

- Membership collection payloads now return a deterministic set of unique owned
  item hashes when multiple acquired collectible definitions reference the same
  item.

## [0.3.81] - 2026-08-12

### Security

- Pinned the frontend production-builder and development Node images to Node
  26.7.0 on Alpine 3.24 at a portable multi-platform OCI index digest. The
  development image now installs the locked dependency graph with `npm ci`.

### Changed

- Added a root `.nvmrc` as the exact Node patch source for local tooling and both
  GitHub Actions workflows. Frontend package metadata accepts only Node 26, and
  a repository policy test rejects drift across local, CI, container, engine,
  and Node type declarations.
- The visual-regression image now layers that pinned Node 26 runtime onto the
  lockfile-matched Playwright image, keeping its browsers reproducible without
  falling back to Playwright's older bundled Node runtime.

## [0.3.80] - 2026-08-12

### Security

- Pinned every third-party GitHub Action to a reviewed release commit while
  retaining readable version comments. A repository policy test now rejects
  moving action tags or incomplete pins, and Dependabot continues to advance
  both the commit and comment for new releases.
- Declared `govulncheck` v1.6.0 as a Go tool and changed CI to invoke that exact,
  Go-module-managed version instead of installing `@latest`. Staticcheck remains
  pinned at 2026.1.

## [0.3.79] - 2026-08-12

### Security

- Refreshed the nginx runtime to 1.31.3 on Alpine 3.24, PostgreSQL to 18.4,
  and the API runtime to Alpine 3.24.1. Security-targeted images now use
  patch-qualified tags pinned to portable OCI index digests.

### Changed

- Minikube validation now pulls base-image metadata, builds both application
  images with the build cache disabled, and ensures exactly one rollout consumes
  each rebuilt local tag. Container refresh and digest-drift steps are documented
  for Docker Compose and Minikube. The documented no-Postgres development mode
  now skips the database readiness probe while continuing to require the
  manifest.

## [0.3.78] - 2026-08-12

### Changed

- Replaced the legacy `patrickmn/go-cache` dependency with Guardian Tracker's
  own in-memory cache behind the existing interface. Cache keys, synchronous
  visibility, absolute TTL behavior, invalidation, and service configuration
  are unchanged; the cleanup worker now has an explicit application-owned
  shutdown lifecycle.

## [0.3.77] - 2026-08-12

### Security

- Updated the backend's indirect `golang.org/x/net` dependency past the version
  range affected by GO-2026-5942. Guardian Tracker does not call the affected
  DNS message API; this is preventative dependency hygiene.

## [0.3.76] - 2026-08-12

### Changed

- Frontend compiler and React type packages are now classified as development
  dependencies, the test DOM matchers were updated, and clean npm installs now
  explicitly skip MSW's unused browser-worker setup script.

## [0.3.75] - 2026-08-12

### Changed

- Removed 21 unused frontend runtime dependencies, including the Radix UI,
  React Hook Form, Lucide, and Zod packages. Guardian Tracker already uses its
  own components and icon system, so application behavior is unchanged while
  the install and supply-chain surface is smaller.

## [0.3.74] - 2026-08-11

### Changed

- Bumped the frontend `react-hook-form` dependency from 7.84.0 to 7.85.0.

## [0.3.73] - 2026-08-11

### Fixed

- A search-index test could fail during its own cleanup, turning the Go test job
  red on unrelated pull requests. The index is published before its snapshot is
  saved to disk, so the test finished while a background build was still writing
  into the temporary directory it was about to delete. Tests now wait for any
  in-flight build before that directory is removed. No application code changed.

## [0.3.72] - 2026-08-11

### Changed

- Cached values are now read through one helper instead of a dozen hand-written
  copies of the same lookup. Three rules that had been comments at some call
  sites and nothing at all at others are now guaranteed for every caller: a
  failed load is never cached, so the next request retries instead of being
  served the failure until the cache entry expires; a cache entry left over from
  an older build with a different value type is now reported and replaced rather
  than silently ignored, which previously meant that cache never hit again; and
  the rule that an empty result must not be cached — a transient empty vendor
  response would otherwise have shown every player an empty day until the next
  daily reset — is now a required argument rather than a comment. Cached data,
  keys, and lifetimes are unchanged.
- Item detail, weapon perks, and catalyst pools now share one bounded cache
  implementation rather than three copies of it. Behavior is unchanged, including
  the deliberate difference that an unrecognized item hash is not cached while a
  weapon with no perks or no catalyst is.

## [0.3.71] - 2026-08-11

### Fixed

- Signing in on a deployment with no database configured returned "Failed to
  create session" (HTTP 500) instead of logging the user in. Those deployments
  never had a session row to write and nothing that reads one, so login now
  succeeds without one. Where a database _is_ configured, a failed session write
  still fails the login, because the access token is checked against that row on
  every request.
- `POST /api/auth/refresh` and `GET /api/auth/validate` omitted the `role` field
  that the login callback and profile endpoints returned. All four now return the
  same user object, and refresh reports the account's current role rather than
  the one recorded when the token was issued. Role remains a display hint;
  `GET /api/flags` stays authoritative for what a tier unlocks.

### Changed

- The browser session lifecycle — starting the Bungie OAuth flow, turning a
  callback or a refresh into a session, and ending one session or all of them —
  moved out of the HTTP handlers into a single `auth.SessionIssuer` module. The
  endpoints, their status codes, their audit events, and the refresh cookie are
  unchanged; the login and refresh flows behave as before. Recorded in
  [ADR 0012](docs/adr/0012-session-issuance-owns-the-session-lifecycle.md) and
  `CONTEXT.md`.
- The two calls to Bungie's OAuth token endpoint — exchanging an authorization
  code at login and exchanging a refresh token later — now share one
  implementation, so the 90-day fallback used when Bungie omits a refresh-token
  lifetime is defined once instead of twice.

## [0.3.70] - 2026-08-10

### Changed

- The weekly recommendations service now depends on a one-method
  `MissingItemReader` interface rather than the concrete collections service, and
  no longer imports the collections package at all — difficulty classification
  goes to `services/sources` directly. The assembled weekly payload is unchanged;
  the seam exists so the assembly logic can be tested, which it previously could
  not be. `CONTEXT.md` records the new seam name.
- The weekly service's remaining cache keys are built by named constructors
  instead of being formatted at each use site. The keys themselves are unchanged,
  so no cached data is invalidated.

### Fixed

- Corrected a stale claim in the `go-services` agent notes that the collections
  service's difficulty classifier is called from the weekly service.

## [0.3.69] - 2026-08-10

### Changed

- Bumped the frontend `npm-minor-and-patch` dependency group: `lucide-react`
  1.29.0 to 1.30.0 and `@types/node` 26.1.2 to 26.2.0. No product behavior or API
  surface changes.

## [0.3.68] - 2026-08-10

### Removed

- The retired `private/known-bugs.md` tracker no longer appears in the private-doc
  lists in `AGENTS.md` and the `docs-updater` agent definition, which pointed
  contributors and agents at a file that no longer exists. Its contents were folded
  into the private work archive, and defect reports already belong in GitHub Issues —
  the tracker of record documented in `docs/agents/issue-tracker.md`. No public
  documentation, product behavior, or API surface changes.

## [0.3.67] - 2026-08-10

### Changed

- Upgraded the frontend compiler from TypeScript 6.0.3 to 7.0.2. The existing
  type-check, Oxlint type-aware analysis, Vite build, tests, and production
  Alpine image now run on the TypeScript 7 toolchain; application behavior and
  public interfaces are unchanged.

## [0.3.66] - 2026-08-10

### Changed

- Frontend linting now runs entirely on Oxlint, including type-aware checks for
  application, test, Vite, and Playwright end-to-end TypeScript sources. The new
  typed findings are reported as warnings so the existing cleanup backlog is
  visible without making the migration itself block CI, and the TypeScript 6
  compiler check remains in place ahead of a separate TypeScript 7 upgrade.
- React Hooks and React Compiler diagnostics now use Oxlint's native React rules.
  The compiler diagnostics are reported through Oxlint's consolidated
  warning-level rule rather than ESLint's individually configurable checks.

### Removed

- Removed ESLint, typescript-eslint, their React plugins, compatibility config,
  and frontend ESLint configuration. Prettier remains the formatting owner.

## [0.3.65] - 2026-08-10

### Fixed

- Item search now recovers on its own after a failed search-index build. The
  index is built asynchronously at startup; if that first build failed, the
  search endpoint reported "index not ready" and returned 503 without ever
  reaching the code that retries the build, so header search stayed unavailable
  until the next hourly manifest update happened to rebuild it. A search request
  that finds the index unready now starts the rebuild itself. The rebuild still
  runs in the background — a full item-table scan takes seconds and must not
  block a request — so the request that triggers it still gets 503 and a later
  one returns results. Repeat attempts are limited to one every 30 seconds, so a
  manifest that cannot be indexed is retried steadily rather than once per
  keystroke. The 503 response body and its `SEARCH_NOT_READY` code are unchanged.

## [0.3.64] - 2026-08-10

### Fixed

- A stale Go coverage profile committed at `backend/api-service/coverage` is no
  longer tracked. Every local test run with coverage rewrote it, so it appeared
  as a large unrelated modification in otherwise unrelated pull requests. The
  extensionless name escaped the existing `*.out` ignore rule; the file is now
  ignored by name. No build, test, or CI path read it.

## [0.3.63] - 2026-08-10

### Added

- `CONTEXT.md`, a project glossary defining the domain vocabulary and the
  architectural seam names, with one canonical owner per term.
- Route-table tests asserting that every `/api` endpoint sits behind
  authentication, that each feature-flag-gated endpoint enforces the flag key it
  claims, and that admin endpoints refuse non-admins and refuse to serve a
  build running without a database.

### Changed

- The API route table moved out of the service entry point into its own package,
  and authentication is now applied once to the authenticated route group rather
  than repeated on each of the 24 protected routes. The set of routes and their
  behavior are unchanged; adding a new endpoint without authentication now fails
  the test suite instead of shipping silently. Recorded as
  [ADR 0011](docs/adr/0011-route-table-as-a-testable-composition-root.md).
- The database-to-consumer adapters and the session and audit retention pruners
  moved into testable packages. Their behavior is unchanged.

## [0.3.62] - 2026-08-10

### Changed

- Updated frontend dependencies: `lucide-react` 1.28 to 1.29, `postcss` 8.5.25
  to 8.5.26, and `vite` 8.2.0 to 8.2.1.

## [0.3.61] - 2026-08-09

### Changed

- Shipping a branch now evaluates its highest-impact change before choosing the
  release line. Incompatible changes start a new major version,
  backward-compatible capabilities start a new minor version, and maintenance
  changes retain the standard auto-incrementing build number. The shipping
  report includes the classification and its rationale.

## [0.3.60] - 2026-08-09

### Fixed

- Fixed the admin console crashing the request instead of reporting a problem
  when the server runs without its account database. Three admin endpoints
  assumed the database was present; they were shielded only because the
  admin-access check happened to reject the request first, so anything that
  changed that order would have produced a crash rather than an explanation.
  They now report the same unavailable message as everywhere else.

### Changed

- Wishlist and preference endpoints now report an unavailable account database
  with the same machine-readable code as the rest of the app. They previously
  returned an error the frontend could not identify, so the app showed generic
  "couldn't load" copy instead of explaining that the feature needs a database.
  All such messages now read identically wherever they come from.
- Running without an account database is now handled in one place rather than
  re-decided at each endpoint. Behaviour is unchanged: role-gated features still
  return unavailable, feature flags still stay visible rather than hiding pages,
  and preferences still fall back to defaults so the app renders.

## [0.3.59] - 2026-08-09

### Changed

- The collections data the app fetches — the category tree, every item's detail,
  what the player owns, and what a vendor is selling right now — is now assembled
  in one place before any page sees it. Five pages previously read it raw and
  three of them combined those pieces themselves; the copies drifted, which is
  what left the Cosmetics gallery unable to show "Available now" until it was
  fixed by hand last release. Combining them once means a page cannot get a
  half-complete item, and the pages that only need progress totals can no longer
  ask for item data that their request never included. No user-visible change.

## [0.3.58] - 2026-08-09

### Changed

- The Destiny source vocabulary — the keywords that decide an item's difficulty
  estimate, whether its source is a raid or dungeon, whether it is something you
  can go and do, and which category a weekly milestone belongs to — now lives in
  one place instead of being transcribed into four separate lists across three
  parts of the backend. The lists overlapped almost entirely but nothing kept
  them in step, so adding a new dungeon meant editing three files and missing one
  made the raid milestone's missing-item count quietly stop appearing, with
  nothing reported. Behaviour is unchanged; the arrangement now makes that class
  of mistake fail a test instead of shipping.

## [0.3.57] - 2026-08-09

### Fixed

- Fixed a page rendered outside the character context silently showing an empty
  Guardian list instead of reporting the mistake. Three of the app's four shared
  data contexts already failed loudly when used from the wrong place; the
  character one returned an empty result that was indistinguishable from an
  account with no Guardians. It now behaves like the other three. No page in the
  app was affected — this closes a way for a future one to break quietly.

### Changed

- The set of shared data contexts every page runs inside is now defined in one
  place rather than assembled by hand at each use. The arrangement is not
  interchangeable — some contexts depend on others — and the tests had drifted
  into ten different versions of it, none matching the app, so adding a feature
  flag or Guardian-aware element to a page could break unrelated tests or, worse,
  pass while shipping something broken. Tests now render pages through the same
  arrangement the app uses. No user-visible change.

## [0.3.56] - 2026-08-09

### Changed

- Reworked how services take part in the hourly Destiny manifest update. Taking
  part used to mean registering a pair of anonymous callbacks, a shape that could
  not express what any given service needed — which is how the item-search index
  came to be left out of it entirely. Services now declare their role: those
  holding an open handle on the manifest file are closed before it is replaced and
  reopened afterwards, and those holding data derived from the manifest are told
  when a new version has actually been installed. Each service decides for itself
  what to discard or rebuild, so adding one no longer means editing startup code
  that has to know its internals.
- A manifest update that fails partway through no longer discards good data.
  Previously a failed file replacement still cleared every manifest-derived cache
  and restarted both background index builds, even though the manifest had not
  changed. Services are now reconnected to the existing manifest and left
  otherwise untouched.

### Fixed

- Fixed the Collections page serving a stale category tree after a manifest update
  until a user's cached data expired. The tree and item definitions are now
  refreshed from the new manifest on the next request, while the player's
  collection progress — which is fetched from Bungie and unaffected by a manifest
  change — is kept, so the refresh costs no additional Bungie requests.
- Fixed "Do This Today" vendor rows keeping item names and types from the previous
  manifest until the next daily reset.
- Reduced log noise: an index rebuild that briefly overlaps a manifest update is an
  expected, self-correcting condition and is no longer recorded as an error.

## [0.3.55] - 2026-08-09

### Fixed

- Fixed the search index holding an open handle on the Destiny manifest database
  across the hourly manifest swap. The search service opens its own SQLite
  connection rather than going through the shared manifest provider, so the
  provider's before-swap hook never covered it and a full item-table scan could
  still be running when the new manifest was moved into place. On Windows the
  move failed and the server silently kept serving the previous manifest; on
  Linux it succeeded but the index was recorded against a manifest version whose
  contents it had never read, so item search stayed stale until the next manifest
  update. The search service now closes down before the swap and rebuilds after
  it, and an interrupted build is discarded rather than recorded.
- Fixed stale milestone names and reward labels on This Week after a manifest
  update; the shared weekly payload is now evicted when the manifest changes
  instead of persisting until the weekly reset. Per-character vendor caches are
  unaffected by this change and can still show manifest labels up to one daily
  reset old.
- Fixed Wishlist, Admin, This Week, and header search reporting a failed request
  as an empty result. A failed wishlist load told the user "Your wishlist is
  empty" and offered to start a new one; Admin reported "No members match" on a
  failed member load; This Week rendered an empty week; and header search
  reported no matching items. All four now show the same failure panel the
  Collections pages use, with an explanation, a Bungie privacy-settings link when
  privacy is the cause, and a retry — except header search, which shows a
  "Search unavailable" line to suit the dropdown.
- Fixed the Cosmetics gallery never showing the "Available now" marker. Cosmetics
  did not carry live vendor availability, and the gallery had nowhere to display
  it, so an emblem or shader a vendor was actively selling looked identical to
  one that was unobtainable — while the same item showed as available in
  Collections. Cosmetic tiles and the cosmetic detail panel now name the vendor
  selling an item the player does not yet own.

### Changed

- The failed-request panel shared by Collections, Catalysts, and Triumphs now
  lives in one place and is reused by every page that loads data, so failure
  wording, the privacy-settings link, and retry behave the same everywhere.

## [0.3.54] - 2026-08-08

### Added

- Added per-repo configuration for third-party "Matt Pocock" engineering skills
  (`to-tickets`, `triage`, `to-spec`, etc.) installed under `.agents/skills/`:
  `docs/agents/issue-tracker.md` (GitHub Issues via `gh`), `docs/agents/triage-labels.md`
  (default canonical label vocabulary), and `docs/agents/domain.md` (single-context
  domain-docs layout). Documented the new directory in AGENTS.md's `## Agent skills`
  section and in the README.md / docs/README.md documentation indexes.

## [0.3.53] - 2026-08-07

### Added

- Backfilled the changelog entries for v0.3.43 through v0.3.51 — nine consecutive
  Dependabot dependency-bump merges that the `Changelog Version` CI guard's bot
  exemption let through without a section, leaving the changelog silently missing
  nine released versions.

## [0.3.52] - 2026-08-07

### Changed

- Inverted the generated-skill sync direction: `.agents/skills/**` is now the
  authored source and `.claude/skills/**` is generated, the opposite of before.
  A third-party skill installer had been writing skill sources into
  `.agents/skills/<name>` and symlinking `.claude/skills/<name>` to them, which
  didn't work in this repo: `git config core.symlinks` is `false`, so committing
  the symlinks would have silently duplicated every file instead of recording a
  link, and the existing drift-checking generator couldn't see through a symlink
  to find the source at all. `npm run sync:agents` (new root `package.json`)
  regenerates both mirrors; `npm run sync:agents -- --check` is what CI runs.

## [0.3.51] - 2026-08-07

### Changed

- Bumped `@testing-library/user-event` from 14.6.1 to 14.6.3 in `frontend/`.

## [0.3.50] - 2026-08-05

### Changed

- Bumped `@hookform/resolvers` from 5.5.8 to 5.7.1 in `frontend/`.

## [0.3.49] - 2026-08-04

### Changed

- Bumped the npm minor-and-patch group in `frontend/` (2 updates): `@hookform/resolvers`
  5.5.7 to 5.5.8 and `react-hook-form` 7.83.0 to 7.84.0.

## [0.3.48] - 2026-08-03

### Changed

- Bumped the npm minor-and-patch group in `frontend/` (5 updates): `@types/react`
  19.2.17 to 19.2.18, `@types/react-dom` 19.2.3 to 19.2.4, `@playwright/test` 1.62.0
  to 1.62.1, `@vitejs/plugin-react` 6.0.4 to 6.0.5, and `vite` 8.1.5 to 8.2.0.

## [0.3.47] - 2026-08-03

### Changed

- Bumped the `dpage/pgadmin4` Compose image from 9.16 to 9.17.

## [0.3.46] - 2026-07-30

### Changed

- Bumped the npm minor-and-patch group in `frontend/` (2 updates): `lucide-react`
  1.27.0 to 1.28.0 and `postcss` 8.5.24 to 8.5.25.

## [0.3.45] - 2026-07-29

### Changed

- Bumped `jsdom` from 29.1.1 to 30.0.1 in `frontend/`.

## [0.3.44] - 2026-07-29

### Changed

- Bumped `github.com/mattn/go-sqlite3` from 1.14.48 to 1.14.49 in
  `backend/api-service/`.

## [0.3.43] - 2026-07-29

### Changed

- Bumped the npm minor-and-patch group in `frontend/` (5 updates):
  `@tanstack/react-virtual` 3.14.8 to 3.14.9, `@types/node` 26.1.1 to 26.1.2,
  `eslint` 10.7.0 to 10.8.0, `globals` 17.7.0 to 17.8.0, and `postcss` 8.5.23 to
  8.5.24.

## [0.3.42] - 2026-07-27

### Changed

- Upgraded React Router from 7.18.1 to 8.3.0. The `react-router-dom` package was
  removed in v8, so all 26 imports across the app and its tests now come from
  `react-router`. No routing behavior changed: the app routes declaratively and
  uses none of the APIs v8 altered — no data router, loaders, or actions — and
  the v8 baselines (Node 22.22+, React 19.2.7+, Vite 7+, ESM-only) were already
  met by React 19.2.8, Vite 8.1.5, and Node 26.

### Security

- Resolves Dependabot alert GHSA-qwww-vcr4-c8h2 (CSRF bypass in React Router's
  unstable RSC code paths), which flagged every 7.12.0-8.3.0 install. Guardian
  Tracker was never exposed: the advisory affects only the unstable RSC APIs, and
  this frontend is a static single-page bundle served by nginx with no server
  runtime, no RSC usage, and no router actions. The upgrade clears the alert
  rather than fixing a reachable vulnerability.

## [0.3.41] - 2026-07-27

### Fixed

- The visual-regression CI job now derives its Playwright container tag from
  `frontend/package-lock.json` instead of carrying a hardcoded one. The job runs
  the suite inside that container and installs dependencies there, so the image
  had to match `@playwright/test` exactly — its browsers live at a
  version-stamped path. Dependabot cannot bump an image literal written into a
  workflow step (its Docker updater reads only Dockerfiles, and the Compose
  updater only Compose files), so every Playwright minor bump silently broke the
  job until someone noticed. The tag and the package can no longer drift apart.
- The baseline-regeneration recipe in `frontend/README.md` derives the tag the
  same way, so following the documented steps can no longer produce snapshots
  from a mismatched browser.

## [0.3.40] - 2026-07-27

### Changed

- Bumped the npm minor-and-patch group in `frontend/` (20 packages), including
  `@playwright/test` 1.61.1 to 1.62.0, `@hookform/resolvers` 5.4.0 to 5.5.7,
  `react-hook-form` 7.82.0 to 7.83.0, `lucide-react` 1.26.0 to 1.27.0, `postcss`
  8.5.22 to 8.5.23, and thirteen `@radix-ui/*` primitives.
- Moved the visual-regression job's Playwright image to `v1.62.0-noble` to match
  the new `@playwright/test`. Without it the container's browser build no longer
  matched the installed package and every visual test failed to launch Chromium.

## [0.3.39] - 2026-07-23

### Changed

- Bumped the npm minor-and-patch group in `frontend/` (16 packages), including
  `lucide-react` 1.25.0 to 1.26.0 and thirteen `@radix-ui/*` primitives.

## [0.3.38] - 2026-07-22

### Added

- `.claude/agents/` and `.claude/skills/` are now the single source of truth for
  AI-agent configuration, with the Codex equivalents (`.codex/agents/*.toml` and
  `.agents/skills/`) generated from them by a new zero-dependency Node script,
  `scripts/sync-agent-configs.mjs`. Editing a source and re-running the script keeps
  both tools in sync; the generated mirrors are committed so a fresh clone works for
  Claude Code and Codex immediately. The `tools:` and `model:` frontmatter are dropped
  in the generated TOML because Codex has no per-agent tool allowlist or model
  selector, and agent bodies are copied verbatim.
- The `Format Check` CI job now runs the script's tests and
  `scripts/sync-agent-configs.mjs --check`, failing the build if a `.claude/` source
  was edited without regenerating the Codex mirrors.

### Fixed

- Corrected the generated `docs-updater` Codex mirror, which a prior hand-conversion
  had garbled — rewriting `.claude/` paths to `.Codex/` (wrong case and extension) and
  inverting a rule by swapping `CLAUDE.md` for `AGENTS.md`. Regenerating from source
  restores the correct text, and the CI drift gate prevents the class of error from
  recurring.

## [0.3.37] - 2026-07-22

### Changed

- Bumped the `npm-minor-and-patch` group in `frontend/` (8 updates): `react` and
  `react-dom` 19.2.7 to 19.2.8, `@tanstack/react-query` 5.101.3 to 5.101.4,
  `@tanstack/react-virtual` 3.14.7 to 3.14.8, `@vitejs/plugin-react` 6.0.3 to 6.0.4,
  `eslint` 10.6.0 to 10.7.0, `postcss` 8.5.21 to 8.5.22, and `typescript-eslint`
  8.63.0 to 8.65.0.

## [0.3.36] - 2026-07-22

### Security

- Bumped the indirect `golang.org/x/text` dependency from 0.38.0 to 0.40.0 in
  `backend/api-service/`, resolving GO-2026-5970 (infinite loop on invalid input).
  `govulncheck` found the vulnerable `norm.Form` symbols reachable from `db.Connect`
  through `pgxpool.NewWithConfig`, so once the advisory was published the
  `Test Go Services` CI job began failing on `main` and on every open pull request
  regardless of what that branch changed. `go mod tidy` carried `golang.org/x/sync`
  from 0.21.0 to 0.22.0 alongside it.

## [0.3.35] - 2026-07-21

### Changed

- Bumped the transitive `brace-expansion` dependency from 5.0.6 to 5.0.7 in
  `frontend/`. Lockfile-only update; no direct dependency changed.

## [0.3.34] - 2026-07-21

### Changed

- Bumped `@testing-library/jest-dom` from 6.9.1 to 7.0.0 in `frontend/` (testing-library
  update group). Test-only dependency; application code is unaffected.

## [0.3.33] - 2026-07-21

### Changed

- Bumped `@tanstack/react-query` from 5.101.2 to 5.101.3, `postcss` from 8.5.20 to
  8.5.21, and `prettier` from 3.9.5 to 3.9.6 in `frontend/` (npm minor-and-patch update
  group).

## [0.3.32] - 2026-07-20

### Changed

- Bumped 15 packages in `frontend/` (npm minor-and-patch update group): the Radix UI
  primitives `react-alert-dialog` 1.1.19 to 1.1.20, `react-avatar` 1.2.2 to 1.2.3,
  `react-dropdown-menu` 2.1.20 to 2.1.21, `react-form` 0.1.12 to 0.1.13,
  `react-navigation-menu` 1.2.18 to 1.2.19, `react-progress` 1.1.12 to 1.1.13,
  `react-select` 2.3.3 to 2.3.4, `react-separator` 1.1.11 to 1.1.12, `react-switch`
  1.3.3 to 1.3.4, `react-tabs` 1.1.17 to 1.1.18, `react-toast` 1.2.19 to 1.2.20, and
  `react-tooltip` 1.2.12 to 1.2.13, plus `@tanstack/react-virtual` 3.14.6 to 3.14.7,
  `react-hook-form` 7.81.0 to 7.82.0, and `postcss` 8.5.19 to 8.5.20.

## [0.3.31] - 2026-07-17

### Changed

- Bumped `lucide-react` from 1.24.0 to 1.25.0 and `@tailwindcss/postcss` from 4.3.2 to
  4.3.3 in `frontend/` (npm minor-and-patch update group).

## [0.3.30] - 2026-07-16

### Changed

- Bumped `actions/setup-go` from 6 to 7 in the CI and browser GitHub Actions workflows.

## [0.3.29] - 2026-07-16

### Changed

- Bumped `vite` from 8.1.4 to 8.1.5 in `frontend/` (npm minor-and-patch update group).

## [0.3.28] - 2026-07-15

### Changed

- Dependabot now opens dependency update PRs at 05:00 America/New_York instead of
  07:00. Each ecosystem also stamps its PRs with an ecosystem-specific label
  (`gomod`, `npm`, `github-actions`, `docker`, `docker-compose`) alongside the
  existing `dependencies` label, so update PRs can be filtered by ecosystem.

## [0.3.27] - 2026-07-14

### Changed

- Bumped `actions/setup-node` from 6 to 7 in the CI and browser GitHub Actions
  workflows.

## [0.3.26] - 2026-07-14

### Changed

- Bumped `@playwright/test` from 1.61.0 to 1.61.1 in `frontend/` (npm minor-and-patch
  update group).

## [0.3.25] - 2026-07-14

### Added

- The `Format Check` CI job now runs Prettier over the repository's markdown, not just
  `frontend/`. Markdown in `README.md`, `SETUP.md`, `docs/`, and `.claude/` was outside
  the reach of the frontend-scoped run and could land unformatted; it is now gated. Fix
  formatting from the repository root with
  `./frontend/node_modules/.bin/prettier --write "**/*.md"`.
- A root `.prettierignore`, so the repo-wide markdown check skips generated and vendored
  paths.

### Changed

- `AGENTS.md` is now the canonical, tool-neutral operating guide for AI coding agents,
  and it is self-contained: agents that read `AGENTS.md` natively — Codex and others —
  expand no imports, so the agent-facing subset of setup (ports, environment table, test
  commands) is duplicated inline on purpose rather than linked out. Contributors adding
  repo operating context should add it here.
- `CLAUDE.md` is reduced to an `@AGENTS.md` import plus the mechanics that are specific
  to Claude Code (subagents in `.claude/agents/`, skills in `.claude/skills/`). It no
  longer carries repo context of its own, so the two files can no longer drift apart.
- Documentation that pointed contributors and agents at `CLAUDE.md` as the operating
  guide now points at `AGENTS.md` — `README.md`, `SETUP.md`, `CONTRIBUTING.md`,
  `docs/README.md`, `SECURITY.md`, `SUPPORT.md`, `PRD.md`, the pull request template,
  and ADR 0009.
- The agent definitions in `.claude/agents/` state that `AGENTS.md` is canonical, so a
  dispatched agent updates the right file.

## [0.3.24] - 2026-07-13

### Added

- A `/ship` skill that takes a branch from "code done" to "PR open": it refreshes the
  docs against the branch diff, writes the `CHANGELOG.md` entry for the version the
  merge will mint, runs fast format/lint/typecheck gates, pushes, and opens or updates
  the pull request.
- `scripts/next-version.sh`, the single source of truth for the build number. The tag
  workflow, the new CI guard, and `/ship` all call it, so the three cannot disagree.
- A required `Changelog Version` CI check that fails a pull request whose changelog
  version does not match the tag its merge would create. Bot-authored pull requests are
  exempt, since they never touch the changelog; `/ship` backfills their entries.

### Removed

- The per-turn docs-freshness `Stop` hook. Documentation is now checked when a branch is
  shipped, which is when it matters, rather than after every response turn.

## [0.3.23] - 2026-07-13

### Changed

- Release-notes maintenance only: promoted the v0.3.22 changes out of `[Unreleased]`
  into a dated section. No functional, API, or schema changes.

## [0.3.22] - 2026-07-13

### Added

- An accessible "Search this category…" field to the Collections toolbar.
  The search term lives only in the URL (never persisted to localStorage,
  unlike the other Collections filters), matches item names case-insensitively,
  composes with every existing filter and sort, and is included in "Clear
  filters" with a search-specific empty state naming the term.
- A collapsed, keyboard-accessible objective disclosure on seal triumphs that
  carry per-objective progress data. Multi-objective triumphs summarize as
  completed-objective count/total; expanding shows each objective's own exact
  progress. `GET /api/seals/:membershipType/:membershipId` now includes an
  optional `objectives` array per triumph; triumphs without objective data are
  unchanged.

### Fixed

- Seal triumph objectives with no explicit `visible` field now decode as
  visible instead of hidden. `RecordObjective.Visible` was a plain `bool`, so
  an absent field (Bungie's default for a visible objective) decoded to
  `false`, the inverse of Bungie's semantics; it is now `*bool` (`nil` = visible).

### Security

- A regression test now pins the Postgres/pgAdmin/test-Postgres/e2e-Postgres
  loopback-only Compose bindings shipped in 0.3.18, closing a gap where that
  binding shipped without the assertion its own rollout gate called for.

## [0.3.21] - 2026-07-13

### Changed

- Bumped the frontend npm minor/patch group: `@tanstack/react-virtual` 3.14.5 to
  3.14.6 (pulling `@tanstack/virtual-core` 3.17.3 to 3.17.4) and `postcss` 8.5.17
  to 8.5.19.

## [0.3.20] - 2026-07-13

### Added

- A hermetic Playwright browser suite backed by a local fake Bungie service and
  isolated Postgres profile, covering functional journeys, WCAG 2.2 axe checks,
  keyboard/reduced-motion behavior, and deterministic visual regression.
- Separate advisory browser CI jobs with one functional retry and 14-day
  reports, logs, traces, screenshots, videos, and visual diffs. Visual tests run
  in the package-matched Playwright 1.61.0 Noble image.
- A test-only `fake-bungie` command and `e2efixture` package serving a synthetic
  manifest and deterministic profile/vendor data, plus an `e2e` Docker Compose
  profile with its own isolated Postgres. No browser test contacts Bungie.net.
- A development-only `E2E_FIXED_TIME` setting (RFC3339) that injects a fixed
  clock into the weekly service so Xûr's weekend window is deterministic.
  Startup fails closed if it is set in production.

### Changed

- `--c-text-3` lightness raised from L 0.62 to 0.68 so tertiary text clears WCAG
  AA on tinted surfaces. `--c-text-4` is now reserved for genuinely disabled
  controls, which WCAG exempts from contrast minimums.

### Fixed

- Manifest downloads now create the destination directory. Writing the manifest
  archive to a `MANIFEST_DB_PATH` whose directory did not exist yet failed every
  retry immediately, leaving `/ready` at 503 and the manifest missing until the
  next hourly update check.
- Tertiary text, admin table headers and timestamps, the collections estimate
  note, and the active cosmetics filter button all fell below the WCAG AA 4.5:1
  contrast minimum (as low as 1.78:1 for white on the aqua signal color).
- Xûr's vendor rows carried `role="button"` on their `<li>`, which replaced the
  implicit `listitem` role and left the surrounding list with no list items.
  The row is now a real listitem containing a real button.
- The Xûr and milestone panels rendered an empty `<ul>` when Bungie reported no
  vendor sales or milestones; they now render an empty state.
- The browser workflow was not valid YAML, so it never actually ran; it now
  parses and executes.
- The fake OAuth redirect is built from configuration rather than the incoming
  request.

## [0.3.19] - 2026-07-13

### Added

- Structured application and access logs with server-owned request UUIDs,
  route-template records, privacy-safe identifier pseudonyms, and
  `X-Request-ID` response/CORS propagation.
- `LOG_LEVEL` (`debug`, `info`, `warn`, `error`; defaults to `info`) and
  `LOG_FORMAT` (`text` or `json`; text in development, JSON in production)
  settings, validated at startup.
- Staticcheck 2026.1 to the required Go CI job, alongside `go vet`,
  `govulncheck`, race-enabled tests, and coverage enforcement.

### Changed

- The API service migrated from the standard `log` package and `gin.Default()`
  to `log/slog`, `gin.New()`, and a new `observability` package providing the
  logger, HTTP middleware, and custom panic recovery.

## [0.3.18] - 2026-07-13

### Security

- Refresh JWTs now rotate in a host-only HttpOnly `guardian_refresh_token`
  cookie instead of localStorage. Callback and refresh require an exact
  allowlisted origin, definitive session failures clear the cookie, and token
  responses expose only the access token and user snapshot. Existing browser
  sessions sign in once again after this migration.
- Token-encryption keys now carry exact positive versions so an A/v1 to B/v2
  rotation can retain A/v1 for old rows while all new ciphertext uses B/v2;
  unknown versions are rejected.
- API responses now receive no-sniff and no-referrer headers, auth responses are
  non-cacheable, manifest status no longer exposes its filesystem path, and the
  frontend CSP no longer permits inline scripts. Inline styles remain a
  documented CSP residual risk.
- `GO_ENV` must be explicitly `development` or `production`, with one prominent
  degraded-development warning and production fail-closed requirements.
- Compose binds Postgres, pgAdmin, and the disposable test database only to
  `127.0.0.1`; frontend and API bindings are unchanged.
- Revocation now distinguishes a deleted user from a database outage. The user
  lookup reports whether the row was found, so an unknown or deleted user is
  rejected with 401 while a genuine database failure retains the documented
  fail-open behavior; previously the two were indistinguishable.

## [0.3.17] - 2026-07-13

### Changed

- Bumped `github.com/mattn/go-sqlite3` from 1.14.47 to 1.14.48.

## [0.3.16] - 2026-07-13

### Changed

- Bumped `postcss` from 8.5.16 to 8.5.17.

## [0.3.15] - 2026-07-13

### Added

- Collectible ornaments and finishers to the Cosmetics gallery after verifying
  their manifest item type/subtype classification.
- A first-run guided tour of Dashboard, This Week, and Collections. Completion
  is stored with the user's server-side preferences so it follows the Bungie
  account across browsers and devices. `GET`/`PUT /api/preferences` carry
  `onboardedAt`, and `PUT` accepts a write-once `onboardingComplete` flag;
  attempting to set it back to `false` is rejected.
- A `characterId` query parameter on `GET /api/weekly/recommendations`.

### Changed

- Authenticated weekly vendor inventory, daily actions, availability ranking,
  and Xûr location now follow the selected Guardian. Xûr armor also identifies
  its manifest-defined class and highlights armor for the active class.
- The Dashboard's weekly query is keyed by the active character and defers until
  the character list has loaded.

### Removed

- The Dashboard "Get Started" panel and its localStorage first-run flag,
  superseded by the server-backed guided tour. The Dashboard header no longer
  alternates between "Welcome" and "Welcome back".

## [0.3.14] - 2026-07-12

### Added

- A manifest repository lookup that resolves a vendor and location index to its
  destination definition, with zero-value fallbacks for best-effort callers.
- An authenticated character-vendor fetch with a five-minute membership-scoped
  cache, plus a daily-cached live vendor sale-item lookup behind a vendor
  allowlist, enriching item availability.
- A `location` field on the Xûr response, omitted when it cannot be resolved.

### Changed

- Xûr's location now resolves best-effort from authenticated vendor and manifest
  data and is displayed as "The Tower"; the location row is omitted when Bungie
  cannot resolve it.
- Non-raid/dungeon milestone missing counts remain intentionally omitted after
  current Bungie reward definitions were verified to contain no collectible-linked
  items.

### Fixed

- Milestone definitions with Bungie's mapping-shaped `rewards` data no longer fail
  to parse and disappear from the weekly response.

## [0.3.13] - 2026-07-12

### Changed

- Release-notes maintenance only: backfilled the `[0.3.0]` through `[0.3.12]`
  sections and refreshed the version examples in this file's header. No
  functional, API, or schema changes.

## [0.3.12] - 2026-07-12

### Changed

- Manifest-derived search now restores a versioned gzip snapshot from the
  persistent manifest volume on restart; missing or new-version snapshots fall
  back to the existing asynchronous rebuild.

### Removed

- Unused Redis service, host port, volume, and environment variables from the
  local Docker Compose stack; caching and authentication remain Postgres- and
  process-backed.

### Security

- Access tokens now default to a 30-minute lifetime via the duration-based
  `JWT_ACCESS_TTL` setting; existing deployments may continue using the legacy
  `JWT_EXPIRY_HOURS` setting until they migrate.

## [0.3.11] - 2026-07-12

### Added

- Icon URLs to Xûr item responses so inventory cards can render item imagery.

## [0.3.10] - 2026-07-12

### Fixed

- Collections now re-assert the selected node from the URL and fall back to the
  first root when a persisted node is unknown.

## [0.3.9] - 2026-07-12

### Added

- Collections filters for rarity, difficulty, sort, view, missing-only,
  "Available now," and "Hide farm-only." The selected category and filters now
  persist in the URL, with a localStorage default for filters.

## [0.3.8] - 2026-07-11

### Fixed

- Bulk wishlist actions now validate the selection before the user lookup,
  preserve selection mode after a failed action, and disable set-priority when
  nothing is selected.
- Corrected the wishlist table name in the PostgreSQL specialist guide.

## [0.3.7] - 2026-07-11

### Added

- Bulk wishlist actions to delete or re-prioritize multiple selected items in
  one request (`POST /api/wishlist/bulk`).

### Changed

- Wishlist availability now covers all rotating vendors (Xûr, Banshee-44,
  Ada-1, and ritual vendors), matching Collections — previously Xûr-only.

## [0.3.6] - 2026-07-11

### Changed

- Public and agent documentation now describes server-side feature-flag
  enforcement consistently.

## [0.3.5] - 2026-07-10

### Security

- Feature flags are now enforced server-side (`RequireFlag` middleware) on the
  weekly, search, catalysts, crafting, and seals routes — previously JWT-only,
  so UI gating was cosmetic. Disabled → 404, under-tier → 403; fails open in
  degraded mode.

## [0.3.4] - 2026-07-10

### Changed

- Updated frontend minor and patch dependencies.

## [0.3.3] - 2026-07-10

### Changed

- Feature flags now gate the weekly planner, global search, and cosmetics UI and
  routes, including tier-locked states.
- Retired the vestigial `wishlist-alerts` and `ui-tweaks` flags.

## [0.3.2] - 2026-07-10

### Added

- **Exotic weapon catalysts** in the item detail drawer: a new "Catalyst"
  section lists each catalyst's name and effect, sourced from the manifest's
  catalyst-socket text (multi-catalyst exotics show all of theirs). New
  `catalysts` array on `GET /api/items/:itemHash/perks`.
- **Catalyst effect text** on the Catalysts page: each card now shows what the
  catalyst changes about the weapon. New `effect` field on the catalysts API
  response, linked to its weapon via record objective-hash overlap with an
  exact-name fallback instead of fuzzy name matching.

### Fixed

- Weapon detail perk pools no longer drop perk columns for roughly half the
  arsenal. Scopes, launcher barrels, grenade launcher magazines, batteries,
  stocks, sword blades and guards, bow arrows and bowstrings, glaive hafts,
  grips, rails, and bolts now render alongside the previously supported
  barrels, magazines, traits, and origin traits.

## [0.3.1] - 2026-07-10

### Fixed

- Collections no longer count owned, re-issued items as missing. Ownership is
  now derived per item instead of per manifest collectible entry, fixing
  inflated tree and summary counts, the "Missing" stat, and weekly
  recommendations that suggested already-owned items.
- Completed catalysts on the Catalysts page now show a full "Catalyst complete"
  progress bar instead of "Not yet acquired".
- Cosmetics gallery tiles now display item icons instead of collapsing to zero
  size.
- Global header search results now display item icons.

## [0.3.0] - 2026-07-09

### Changed

- Versioning now creates a GitHub Release for each auto-incremented
  `v<major>.<minor>.<build>` tag on `main`; fresh major/minor lines may start at
  build `0`.

## [0.2.1] - 2026-07-09

### Changed

- Updated frontend minor and patch dependencies.

## [0.2.0] - 2026-07-09

### Added

- **Bungie OAuth login** with stateless, HMAC-signed CSRF protection.
- **JWT authentication** with access tokens, rotating per-device refresh sessions,
  revocation checks, and refresh-token reuse detection.
- **Collection analysis** for missing weapons, armor, exotics, and cosmetics.
- **Nested Collections tree** from Bungie presentation nodes, with item cards,
  details, filters, sorting, data freshness, and deep links.
- **Browsable cosmetics gallery** for emblems, shaders, ghosts, ships, sparrows,
  and emotes.
- **Wish list management** with persisted priority, notes, sorting, and
  availability surfacing.
- **Destiny 2 manifest pipeline** with automatic download, version tracking,
  hourly update checks, SQLite extraction, and manifest swap handling.
- **This Week** with milestones, Xur inventory, daily actions, reset countdowns,
  and ranked recommendations.
- **Per-raid and per-dungeon milestone missing counts** where a reliable source
  bucket can be matched.
- **Catalysts, crafting, triumphs, and seals** from Bungie records data.
- **Global item search** over a manifest-derived in-memory index.
- **Read-only item views for deep-linked hashes** through `GET /api/items/:itemHash`.
- **Weapon perk pools** in the item drawer from manifest socket and plug-set data.
- **Dashboard** with real account, collection, weekly, wishlist, character, and
  cosmetics data.
- **Settings and preferences** with persisted user preferences and account context.
- **Roles, feature flags, and admin console** for early-access rollout and user
  management.
- **Unified audit log** for auth, session, role, and flag events, including admin
  UI access.
- **Docker Compose, Minikube manifests, and GitHub Actions CI** for local
  development and validation.
- **Public documentation system** with setup guide, architecture overview,
  public/private docs boundary, public roadmap, and ADRs.

### Changed

- **Difficulty ratings** use positive-match classification; unmatched sources are
  shown as `Unrated` instead of a misleading default.
- **Farm-only items** are surfaced explicitly when source text indicates the item
  cannot be reacquired.
- **Refresh behavior** treats transient refresh failures as reconnectable instead
  of immediately logging the user out.
- **Production hardening** added rate limits, body caps, readiness checks,
  streamed manifest downloads, config-driven Bungie URLs, stricter CORS, and Go
  version pinning.
- **Dashboard and Cosmetics states** now use explicit warming, privacy, error,
  empty, and retry states rather than silently zeroed data.
- **Documentation structure** now separates public committed docs from private
  gitignored runbooks, reviews, research, and implementation handoffs.

### Removed

- Legacy mock-data paths and placeholder collection/dashboard data.
- Public deep implementation handoff docs from `docs/`; detailed plans now belong
  under `private/`.

### Fixed

- OAuth callback double-submit handling in React StrictMode.
- Bungie token upsert race handling.
- Manifest swap lifecycle on Windows by closing and reopening SQLite handles.
- Cross-save login membership selection.
- CORS wildcard plus credentials rejection in production.
- Several accessibility issues in icon buttons, tabs, search inputs, and reduced
  motion handling.

### Security

- Bungie OAuth tokens are encrypted at rest with AES-256-GCM and key-rotation
  support.
- Refresh-token reuse revokes the affected session.
- Authorization reads current roles from the DB-backed revocation cache rather
  than trusting JWT role hints.
- Admin role and feature-flag changes are audited.
- Last-admin demotion is blocked transactionally.
- Audit rows capture client IP and User-Agent with configurable retention.

[Unreleased]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.82...HEAD
[0.3.82]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.81...v0.3.82
[0.3.12]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.11...v0.3.12
[0.3.11]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.10...v0.3.11
[0.3.10]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.9...v0.3.10
[0.3.9]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.8...v0.3.9
[0.3.8]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.7...v0.3.8
[0.3.7]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.6...v0.3.7
[0.3.6]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.5...v0.3.6
[0.3.5]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.4...v0.3.5
[0.3.4]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.3...v0.3.4
[0.3.3]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/jwh3times/GuardianTracker/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/jwh3times/GuardianTracker/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/jwh3times/GuardianTracker/compare/v0.1.0...v0.2.0
