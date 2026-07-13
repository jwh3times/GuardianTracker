# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Guardian Tracker uses the SemVer target version in `VERSION`; merges to `main`
are stamped with annotated version tags and GitHub Releases such as `v0.2.0`,
`v0.3.10`, `v0.3.11`, and `v0.3.12`.

## [Unreleased]

### Added

- Structured application and access logs with server-owned request UUIDs,
  route-template records, privacy-safe identifier pseudonyms, and
  `X-Request-ID` response/CORS propagation.
- Staticcheck 2026.1 to the required Go CI job, alongside `go vet`,
  `govulncheck`, race-enabled tests, and coverage enforcement.
- Collectible ornaments and finishers to the Cosmetics gallery after verifying
  their manifest item type/subtype classification.
- A first-run guided tour of Dashboard, This Week, and Collections. Completion
  is stored with the user's server-side preferences so it follows the Bungie
  account across browsers and devices.

### Changed

- Authenticated weekly vendor inventory, daily actions, availability ranking,
  and Xûr location now follow the selected Guardian. Xûr armor also identifies
  its manifest-defined class and highlights armor for the active class.
- Xûr's location now resolves best-effort from authenticated vendor and manifest
  data and is displayed as "The Tower"; the location row is omitted when Bungie
  cannot resolve it.
- Non-raid/dungeon milestone missing counts remain intentionally omitted after
  current Bungie reward definitions were verified to contain no collectible-linked
  items.

### Fixed

- Milestone definitions with Bungie's mapping-shaped `rewards` data no longer fail
  to parse and disappear from the weekly response.

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

[Unreleased]: https://github.com/jwh3times/GuardianTracker/compare/v0.3.12...HEAD
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
