# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to adhere to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **No official releases yet.** Guardian Tracker is in active pre-release
> development — no versioned tags have been published. Everything below is
> unreleased and may change before the first tagged release (`0.1.0`). When that
> release is cut, the items under **Unreleased** will move into a dated, versioned
> section.

## [Unreleased]

### Added

- **Bungie OAuth login** with stateless, HMAC-signed CSRF protection.
- **JWT authentication** — access/refresh tokens with revocation, per-device
  refresh sessions, and refresh-token reuse detection.
- **Collection analysis** — surfaces missing weapons, armor, and exotics,
  classified by acquisition difficulty (Easy / Moderate / Challenging), plus
  cosmetics.
- **Wish list management** — add, prioritize, and annotate desired items, with an
  "available now" cross-check against the current Xûr inventory.
- **Destiny 2 manifest pipeline** — automatic download, version tracking, hourly
  update checks, and SQLite extraction.
- **This Week** — weekly milestones, Xûr inventory, daily recommended actions, and
  reset countdowns.
- **Catalysts, crafting & seals** — exotic catalyst progress, crafting pattern
  unlocks, and triumph/seal completion from the Bungie records API.
- **Item search** — in-memory manifest search index with async rebuild on manifest
  updates.
- **Roles & feature flags** — tiered access (standard / beta / alpha / admin),
  self-service tier opt-in, and an admin console for user role and feature-flag
  management.
- **User preferences** — card style and personalization options.
- **Infrastructure** — Docker Compose stack, Kubernetes (Minikube) manifests, and a
  GitHub Actions CI pipeline (format, test, Docker build).
- **Per-raid-milestone missing counts** — raid and dungeon milestones in This Week now
  show how many of the player's missing items drop there, via
  `efficiency.MissingForMilestone`. Non-raid/dungeon milestones still omit the count (no
  manifest reward→collectible signal).
- **Read-only item view for deep-linked hashes** — a `GET /api/items/:itemHash`
  (manifest-only) endpoint and a matching `itemByHashQuery` + `toGTItemView` on the
  frontend resolve a deep-link miss (`?item=<hash>` with no collectible entry) into a
  read-only item drawer instead of dead-ending.

### Changed

- **Difficulty ratings** — `ClassifyDifficulty` is now a positive-match table; unmatched
  sources return `"Unrated"` instead of a misleading "Easy". Items whose source string
  indicates "cannot be reacquired" are additionally flagged as `farmOnly` and shown with
  a "Farm only" chip in the item card and drawer.

### Deprecated

- _Nothing yet._

### Removed

- _Nothing yet._

### Fixed

- _Nothing yet._

### Security

- Bungie OAuth tokens stored AES-256-GCM encrypted at rest, with key-rotation
  support.

[Unreleased]: https://github.com/jwh3times/GuardianTracker/commits/main
