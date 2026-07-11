# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Guardian Tracker uses the SemVer target version in `VERSION`; merges to `main`
are stamped with annotated version tags and GitHub Releases such as `v0.2.0`,
`v0.2.1`, and `v0.2.2`.

## [Unreleased]

### Added

- **Exotic weapon catalysts** in the item detail drawer: a new "Catalyst"
  section lists each catalyst's name and effect, sourced from the manifest's
  catalyst-socket text (multi-catalyst exotics show all of theirs). New
  `catalysts` array on `GET /api/items/:itemHash/perks`.
- **Catalyst effect text** on the Catalysts page: each card now shows what the
  catalyst changes about the weapon. New `effect` field on the catalysts API
  response, linked to its weapon via record objective-hash overlap with an
  exact-name fallback instead of fuzzy name matching.

### Changed

- Versioning now creates a GitHub Release for each auto-incremented
  `v<major>.<minor>.<build>` tag on `main`; fresh major/minor lines may start at
  build `0`.

### Fixed

- Weapon detail perk pools no longer drop perk columns for roughly half the
  arsenal. Scopes, launcher barrels, grenade launcher magazines, batteries,
  stocks, sword blades and guards, bow arrows and bowstrings, glaive hafts,
  grips, rails, and bolts now render alongside the previously supported
  barrels, magazines, traits, and origin traits.
- Collections no longer count owned, re-issued items as missing. Ownership is now
  derived per item instead of per manifest collectible entry, fixing inflated
  tree and summary counts, the "Missing" stat, and weekly recommendations that
  suggested already-owned items.
- Completed catalysts on the Catalysts page now show a full "Catalyst complete"
  progress bar instead of "Not yet acquired".
- Cosmetics gallery tiles now display item icons instead of collapsing to zero
  size.
- Global header search results now display item icons.

### Security

- Feature flags are now enforced server-side (`RequireFlag` middleware) on the
  weekly, search, catalysts, crafting, and seals routes — previously JWT-only, so
  the UI gating was cosmetic. Disabled → 404, under-tier → 403; fails open in
  degraded mode.

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

[Unreleased]: https://github.com/jwh3times/GuardianTracker/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/jwh3times/GuardianTracker/compare/v0.1.0...v0.2.0
