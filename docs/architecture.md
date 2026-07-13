# Architecture

Guardian Tracker is a two-service Destiny 2 companion app. Players authenticate
with Bungie OAuth, the API stores the tokens encrypted, and the frontend renders
collection, wishlist, weekly, and account surfaces through same-origin REST calls
to the API service.

## Runtime Shape

```text
React/Vite frontend (:5273)
    -> Go/Gin API service (:8081)
        -> Bungie API
        -> Postgres user data
        -> SQLite Destiny manifest
```

- **Frontend:** React 19, TypeScript, Vite, React Router, TanStack Query, and the
  custom `gt-*` design system.
- **API:** Go + Gin HTTP service with Bungie OAuth, JWT access/refresh tokens,
  manifest management, collection analysis, weekly recommendations, search,
  wishlist, preferences, roles, flags, and admin endpoints.
- **Postgres:** users, wishlist, preferences and onboarding completion, encrypted
  Bungie tokens, refresh sessions, roles, feature flags, and audit log.
- **SQLite:** local copy of the Bungie Destiny 2 manifest, downloaded and
  swapped by the API service.
- **Cache:** in-memory service caches for collection results, weekly data,
  manifest-derived indexes, and role/revocation checks; the search index also
  persists as a versioned snapshot beside the manifest.

## Authentication and Sessions

Bungie OAuth login uses a stateless HMAC-signed CSRF state. On callback, the API
stores Bungie OAuth tokens AES-256-GCM encrypted in Postgres and issues Guardian
Tracker JWTs.

Access tokens are short-lived bearer tokens. Refresh tokens are rotating,
per-device sessions backed by Postgres. Reused refresh tokens revoke the affected
session. Sign-out-everywhere bumps the user's token version and removes sessions.

Authorization reads the current role from the DB-backed revocation cache rather
than trusting the JWT role hint.

## Roles, Flags, and Admin

Roles are `standard`, `beta`, `alpha`, and `admin`. Admin users can manage user
roles and feature flags through the admin console. Last-admin demotion is blocked
transactionally. Role and flag mutations write audit events.

Feature flags control frontend visibility and rollout state. Server-side
authorization remains the boundary for protected API surfaces.

## Bungie Data Flow

The API talks to Bungie for:

- OAuth token exchange and refresh
- player profile and membership data
- collections, records, characters, and weekly public data
- manifest version and manifest database download

The app does not treat Bungie data as live. User-facing surfaces should show
freshness and reset timing rather than implying real-time state.

## Manifest Pipeline

On startup and periodic checks, the API downloads the current Destiny manifest,
extracts the SQLite database, opens it through the manifest provider, and notifies
dependent services to rebuild manifest-derived indexes.

Collections, cosmetics, catalysts, crafting, triumphs, search, and item detail
views all depend on the manifest. The search index restores a matching versioned
snapshot from beside the manifest on startup, then rebuilds asynchronously when
the snapshot is missing or the manifest changes. Other affected endpoints can
return warming responses during cold start or manifest swap.

## API Surface

Primary route groups:

- auth: Bungie login, callback, refresh, logout, logout-all
- account: profile, role opt-in, feature flags
- collections: collection tree, refresh, item views, item perk pools
- weekly: recommendations, Xur inventory and authenticated location, milestones,
  reset countdowns; authenticated vendor calls validate and follow the selected
  character because Bungie's vendor inventory can be class-specific
- wishlist: user-scoped CRUD
- preferences: user preferences plus irreversible first-run onboarding completion
- records: catalysts, crafting, seals
- characters: account characters
- admin: users, roles, flags, audit log
- health: `/health` liveness and `/ready` readiness

See `backend/api-service/main.go` for the authoritative route registration.

## Local Infrastructure

Docker Compose is the recommended full-stack development path. It starts the
frontend, API service, Postgres, pgAdmin, and a test Postgres profile.

Minikube manifests under `k8s/` validate container and Kubernetes wiring. That
environment runs in development mode and is not production parity.

## Security Posture

- Secrets are read from environment files or runtime environment variables, never
  committed.
- Bungie tokens are encrypted at rest with key rotation support.
- CORS allows only configured origins.
- API server timeouts and body limits are configured.
- Admin and role changes are audited.
- Audit rows include client IP and User-Agent, retained by configured policy.

See [SECURITY.md](../SECURITY.md) for the security guide and checklist.

## Tests

- Frontend: Vitest, Testing Library, MSW, type-check, lint, build.
- Backend: Go unit and integration tests, `go vet`, `govulncheck`, race detector
  in CI, Postgres-backed integration tests.
- Docker: CI builds production images for validation.

## Related Decisions

See [docs/adr](./adr/README.md) for accepted architecture and operating decisions.
