# Guardian Tracker Frontend

The Guardian Tracker frontend is a React and TypeScript single-page application
served by Vite during development and nginx in its production container. It
calls the Go API through the shared `apiFetch` helper and uses TanStack Query for
server state.

## Tooling

- React 19, TypeScript 7, Vite, and React Router 8 (`react-router`).
- TanStack React Query for REST-backed state.
- The `gt-*` design system and oklch tokens under `src/styles/`.
- Vitest and React Testing Library for component tests.
- Playwright and axe for functional, accessibility, destructive-auth, and
  visual browser tests.
- Oxlint for linting and Prettier for formatting.

The exact supported Node patch is defined by the repository root `.nvmrc`.

## Quick Start

```bash
npm ci
cp .env.example .env.local
npm start
```

Vite runs at <http://localhost:5273> and proxies `/api` to
<http://localhost:8081>.

## Environment

Vite exposes only `VITE_`-prefixed values to browser code:

```env
VITE_API_URL=http://localhost:8081
VITE_AUTH_REDIRECT_URI=http://localhost:5273/auth/callback
```

`NGROK_HOST` is an optional Vite-server setting for an HTTPS development tunnel;
it is read by `vite.config.ts` but is not exposed to the browser. See
[SETUP.md](../SETUP.md) for the complete OAuth and environment procedure.

## Source Layout

```text
src/
├── App.tsx          Routes, authentication gate, lazy pages
├── components/      Shared app shell and design-system components
├── contexts/        Auth, preferences, feature flags, selected character
├── features/        Page-oriented feature slices and their tests
├── lib/             API client, query helpers, adapters, constants
├── styles/          Design tokens and production component/page CSS
├── test/            Shared Vitest/MSW setup and fixtures
└── types/           API and UI-domain types
```

The production feature slices are `admin`, `auth`, `collections`, `cosmetics`,
`dashboard`, `onboarding`, `settings`, `weekly`, and `wishlist`. Tests are
colocated with the source they cover.

`design/` is a frozen historical prototype, not a second implementation source.
See [its README](./design/README.md) before using any of its assets.

## Runtime Structure

`AppProviders` owns the authenticated-independent context tower: query client,
authentication, preferences, and toasts. `AuthedProviders` adds feature flags
and selected-character state below the authentication gate. Their order is
load-bearing and documented in `src/contexts/AppProviders.tsx`.

Authenticated routes render inside `AppShell`, which provides desktop and
mobile navigation, global search, settings access, and the selected-character
control. Page routes include Dashboard, Collections, Cosmetics, Wishlist, This
Week, Catalysts & Crafting, Triumphs & Seals, Settings, and the admin-only
console. Feature-flag and admin checks in the UI supplement server-side
authorization; they do not replace it.

## Authentication

1. The login page begins authorization through the shared browser session client.
2. Bungie returns the browser to `/auth/callback` with an authorization code and
   signed state; the callback delegates completion to that client.
3. The client stores the access JWT and user snapshot together in the versioned
   `guardian_browser_session` localStorage envelope. The API sets the rotating
   refresh JWT in a host-only HttpOnly cookie.
4. `apiFetch` delegates authenticated requests to the same client, then adapts
   response bodies and errors. The client coordinates callback, refresh, and
   logout across same-origin tabs using Web Locks.

`AuthProvider` subscribes to the client's public snapshot with
`useSyncExternalStore`; it neither decodes JWTs nor fetches a profile during
hydration. `useAuth()` exposes the user, authenticated state, and both logout
scopes, without exposing an access token. Existing valid access-token/user pairs
migrate once from the old separate storage keys; the client then removes those
keys. A logout persists an anonymous envelope before best-effort server cleanup.

Sign-in completion and refresh require Web Locks. When coordination is unavailable,
those operations fail without changing the shared refresh cookie; local logout
still persists. JavaScript never reads or writes the refresh credential. An
expired Bungie authorization redirects to `/reauthorize` without ending the
Guardian Tracker session.

On logout or a Destiny membership change, application composition cancels and
clears the old query cache, replaces its QueryClient, and remounts the provider
subtree. Preferences, onboarding, flags, character state, and page drafts reset
before the next identity uses them. Stored weekly checklist marks are also cleared
on logout or account switch. Same-membership refresh retains that state.
Authenticated mutations use `useIdentityMutation` so work delayed past an identity
change cannot start a request or apply its old completion callbacks.

## Preferences

`PreferencesContext` manages collection card density, personalization badges,
and onboarding state. For an authenticated user it loads and writes
`/api/preferences`; localStorage provides an immediate local value and a
fallback when the API is unavailable. Settings exposes the card and
personalization controls.

## Scripts

```bash
npm start           # Vite development server on :5273
npm run build       # TypeScript compile and production Vite build
npm run type-check  # TypeScript without emitting output
npm run lint        # Oxlint, including type-aware TypeScript checks
npm run format      # Write Prettier formatting
npm test            # Vitest
npm run test:coverage
npm run e2e         # functional, axe, and destructive-auth projects
npm run e2e:visual  # visual project
```

## Browser Tests

Playwright starts the fake Bungie service, real API, and Vite. The fake supplies
a generated manifest and deterministic profile/vendor data; browser tests never
contact Bungie.net.

From the repository root:

```powershell
docker compose stop frontend api-service
docker compose --profile e2e up -d --wait e2e-postgres
$env:E2E_FIXED_TIME="2026-07-18T18:00:00Z"
cd frontend
npm run e2e
```

Outside CI, Playwright reuses anything already listening on ports 5273 or 8081.
Stop the normal Compose frontend and API first or the suite may silently use the
wrong stack and fail during login.

### Visual baselines

Canonical baselines are Linux renderings. Never commit snapshots generated on
Windows or macOS; font rasterization differences will fail CI.

Regenerate baselines with `frontend/Dockerfile.playwright`, whose Playwright
image and Node runtime are kept aligned with the lockfile and root `.nvmrc` by
repository policy tests. Build the two Go test commands for Linux, then run the
snapshot script in that image:

```powershell
$repo = (Get-Location).Path -replace '\\','/'
$goImage = "golang:" + (Select-String -Path backend/api-service/go.mod `
  -Pattern '^toolchain go(.+)$').Matches.Groups[1].Value
docker build --pull -t guardian-tracker/playwright:local `
  -f frontend/Dockerfile.playwright frontend
docker compose --profile e2e up -d --wait e2e-postgres
docker run --rm -v "${repo}:/src" -v guardian-e2e-bin:/out -w /src/backend/api-service `
  -e CGO_ENABLED=1 $goImage `
  bash -c "go build -o /out/api-service . && go build -o /out/fake-bungie ./cmd/fake-bungie"
docker run --rm --init --ipc=host --network guardiantracker_default `
  -e CI=true -e E2E_FIXED_TIME=2026-07-18T18:00:00Z `
  -e E2E_FAKE_COMMAND=/out/fake-bungie -e E2E_API_COMMAND=/out/api-service `
  -e E2E_DATABASE_URL="postgres://guardian_app:guardian_dev_password@e2e-postgres:5432/guardian_tracker?sslmode=disable" `
  -v "${repo}:/repo" -v "/repo/frontend/node_modules" -v guardian-e2e-bin:/out -w /repo/frontend `
  guardian-tracker/playwright:local `
  bash -c "node --version && npm ci && npm run e2e:update-snapshots"
```

Run these commands from the repository root. The anonymous `node_modules`
volume prevents the container's Linux dependencies from overwriting the host
installation.

## Production Build

`npm run build` writes static assets to `dist/`. The production stages and
their reviewed image pins are defined only in [frontend/Dockerfile](./Dockerfile);
do not copy those version strings into documentation.
