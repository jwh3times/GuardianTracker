# Guardian Tracker — Frontend

React + TypeScript SPA for Guardian Tracker, a Destiny 2 collection tracking app.

## Tech Stack

- **Framework**: React 19 + TypeScript
- **Build/Dev**: Vite
- **Data fetching**: TanStack React Query + `lib/api.ts` (`apiFetch` REST helper)
- **Routing**: React Router v8 — import from `react-router`; the `react-router-dom` package no longer exists
- **Styling**: Custom "Guardian Tracker" design system — oklch design tokens + `gt-*` CSS classes (`src/styles/`). Tailwind is still installed for a few legacy primitives but new UI uses the design system.
- **Icons**: in-house single-path line set (`components/Icon.tsx`)
- **Tests**: Vitest + React Testing Library; Playwright + axe for hermetic full-browser checks

The design source (a self-contained prototype) lives in `frontend/design/`.

## Quick Start

```bash
npm ci
cp .env.example .env.local   # fill in your values
npm start                    # Vite dev server on http://localhost:5273
```

The dev server proxies `/api` to the API service (`http://localhost:8081`) — see `vite.config.ts`.

## Environment Variables

Vite exposes only `VITE_`-prefixed vars to the client:

```env
VITE_API_URL=http://localhost:8081
VITE_AUTH_REDIRECT_URI=http://localhost:5273/auth/callback
# Optional dev tunnel host for OAuth over HTTPS (ngrok, Cloudflare Tunnel)
NGROK_HOST=
```

## Project Structure

```text
src/
├── App.tsx                     # Router, lazy pages, ProtectedLayout (CharacterProvider + AppShell + auth gate)
├── index.tsx                   # Root; imports styles/{tokens,kit,app}.css
├── contexts/
│   ├── AuthContext.tsx          # Access/user localStorage state + cookie-backed token refresh
│   ├── PreferencesContext.tsx   # Card style + "for you" badge prefs (localStorage)
│   └── CharacterContext.tsx     # Characters query + persisted active-character pick (display-only)
├── styles/
│   ├── tokens.css               # Design tokens (colors, type, spacing, rarity maps)
│   ├── kit.css                  # Component styles
│   └── app.css                  # App shell + page styles
├── components/                 # Shared design-system components (flat; no kit/ or ui/ subfolders)
│   ├── AppShell.tsx             # Sidebar + top bar + mobile nav; global search (deep-links to
│   │                            #   /collections?item=<hash>); character switcher (CharacterContext)
│   ├── Brand.tsx                # Logo mark
│   ├── ErrorBoundary.tsx        # React error boundary
│   ├── Icon.tsx                 # Single-path line-icon set
│   ├── primitives.tsx           # Badge, Button, Textarea, ProgressBar, RadialProgress, StatTile,
│   │                            #   CountdownChip, DataFreshnessChip, FilterChip, EmptyState…
│   ├── composite.tsx            # Panel, CategoryTree, ItemDetailDrawer, SealCard, Dropdown…
│   ├── LoadingSpinner.tsx       # (was components/ui/) shown while data loads
│   └── Toast.tsx                # (was components/ui/) ToastProvider / useToast
│                                # Import these directly — the components/kit/index.ts barrel is gone
├── features/                   # Feature slices; each owns its pages, feature-only components, lib, tests
│   ├── auth/pages/             # Login.tsx (OAuth initiation), OAuthCallback.tsx (code exchange)
│   ├── collections/
│   │   ├── pages/              # Collections.tsx (tree + grid + drawer; ?item=<hash> deep-link),
│   │   │                       #   Catalysts.tsx, Triumphs.tsx
│   │   ├── components/kit/ItemCard.tsx   # Collection-only item card
│   │   └── lib/collectionTree.ts         # API node → TreeNode adapters
│   ├── wishlist/pages/WishList.tsx       # Wishlist mgmt; inline notes editor; Xûr availability
│   ├── weekly/pages/ThisWeek.tsx         # Weekly recs / Xûr / milestones (real API)
│   ├── dashboard/pages/Dashboard.tsx     # Completion hero + "do this today" (real totals, real weekly)
│   ├── settings/pages/Settings.tsx       # Destiny membership + appearance preferences + sign out
│   └── admin/
│       ├── pages/Admin.tsx               # Admin console (roster, role mgmt, flag config; admin-gated)
│       └── components/kit/               # admin.tsx (RoleBadge, FlagCard, UserRow, LockedFeature…),
│                                         #   AuditTable.tsx — tests colocate per feature
├── lib/
│   ├── api.ts                   # apiFetch helper + ApiError (status/code) + QueryClient;
│   │                            #   API_URL exported; all REST calls go through apiFetch
│   ├── acquisitionSources.ts    # Canonical API acquisition-source → design adapter
│   ├── adapters.ts              # API response types → design GTItem/WishlistEntry; relTime
│   ├── constants.ts             # RARITIES, glyphs, BUNGIE_CDN base URL
│   ├── roles.ts                 # Role/tier constants, labels, colors
│   ├── utils.ts                 # cn() Tailwind class merger (LoadingSpinner + Toast)
│   ├── errorState.ts            # errorState(error) → ErrorStateCopy; branches on ApiError.code
│   └── queries.ts               # collectionsQuery() — shared React Query definition
├── types/
│   ├── api.ts                   # API response types (WishListItem + icon/availableNow,
│   │                            #   APIMembershipCollections + fetchedAt, APIRecordsEnvelope<T>, etc.)
│   └── design.ts                # Design-system domain types (GTItem, Seal, Weekly + resetAt/fetchedAt…)
└── test/                        # Shared test infra (referenced by vite.config setupFiles)
    ├── setup.ts                 # Vitest setup file
    └── testServer.ts            # MSW server + shared fixtures
```

## Authentication Flow

1. User clicks "Sign in with Bungie" on `/login`
2. Frontend calls API service `GET /api/auth/bungie` → receives OAuth URL + CSRF state
3. User is redirected to Bungie.net and authorizes the app
4. Bungie redirects to `/auth/callback?code=...&state=...`
5. `OAuthCallback` posts code + state with credentials to the API service
6. The response stores `guardian_token` and `guardian_user` in localStorage; the API sets the rotating refresh JWT in the host-only HttpOnly `guardian_refresh_token` cookie
7. `apiFetch` sends credentials and injects `Authorization: Bearer <token>` on every request

Token refresh (on 401) is handled automatically by `apiFetch` using an empty JSON request and the cookie — a single shared refresh call prevents duplicate refresh requests from concurrent queries. JavaScript never reads or writes the refresh credential, and the legacy localStorage key is removed.

## App Shell & Navigation

Authenticated routes render inside `AppShell` (persistent left sidebar + top bar on desktop,
bottom tab bar + drawer on mobile). Layout is fixed to sidebar nav, comfortable density, and
the cyan "signal" accent. Sections: Dashboard, This Week, Collections, Catalysts & Crafting,
Triumphs & Seals, Wishlist, Settings.

## Pages & Data Sources

| Page                 | Data                                                                                  |
| -------------------- | ------------------------------------------------------------------------------------- |
| Dashboard            | Real collection totals (weapons/armor/exotics/cosmetics); real weekly recommendations |
| Collections          | Full item/source data via `GET /api/collections?include=all`; ?item=<hash> deep-link  |
| Wishlist             | Real CRUD; acquisition sources plus live-vendor availability                          |
| This Week            | Real weekly data via `GET /api/weekly/recommendations` (Xûr, milestones, actions)     |
| Catalysts / Triumphs | Real data via `GET /api/catalysts`, `/api/crafting`, `/api/seals`                     |
| Settings             | Destiny membership info from `useAuth`; appearance prefs via `PreferencesContext`     |

## User Preferences

`PreferencesContext` persists two settings to `localStorage` (`guardian_prefs`), surfaced in **Settings**:

- **Item card style** — `framed` (full cards) or `compact` (condensed rows), consumed by Collections
- **"For you" badges** — `on`/`off` for the personalized "Missing"/"Available now" badges

## Available Scripts

```bash
npm start           # Vite dev server (port 5273)
npm run build       # tsc + Vite production build → /dist
npm test            # Vitest
npm run lint        # Oxlint, including type-aware TypeScript checks
npm run lint:fix    # Apply safe Oxlint fixes
npm run type-check  # tsc --noEmit
npm run e2e         # functional + axe + destructive auth (one shared worker)
npm run e2e:visual  # visual project; canonical CI rendering uses pinned Linux image
```

Oxlint is the sole linter. Its type-aware TypeScript rules run through
`oxlint-tsgolint` for `src/`, `vite.config.ts`, and the TypeScript files under
`e2e/`; the current type-safety cleanup backlog is reported as warnings so it
does not block CI. `playwright.config.ts` receives Oxlint's native rules but is
outside the type-aware scope. The React compiler rule is also warning-only.
TypeScript 7 is the project compiler, Prettier remains the formatter, and
`npm run type-check` remains the main frontend project's compilation gate.

## Browser Tests

From the repository root, start the isolated database and set the deterministic
fixture clock before running Playwright:

```powershell
docker compose stop frontend api-service   # see "Port conflicts" below
docker compose --profile e2e up -d --wait e2e-postgres
$env:E2E_FIXED_TIME="2026-07-18T18:00:00Z"
cd frontend
npm run e2e
```

Playwright launches the test-only fake Bungie command, real API, and Vite. The
fake serves a runtime-generated manifest and deterministic profile/vendor data;
no browser test contacts Bungie.net. Functional, accessibility, visual, and
destructive-auth projects share one worker, with logout/logout-all sequenced
last so they cannot invalidate other journeys.

### Port conflicts

Playwright runs with `reuseExistingServer` outside CI, so anything already
listening on **5273** or **8081** is silently adopted instead of the hermetic
stack. If the Docker Compose app stack is up, the browser talks to your real API
— which points at Bungie.net and a different CORS origin — and the suite fails at
login with `Failed to start authentication: Failed to fetch`. Stop the `frontend`
and `api-service` containers first; leave `e2e-postgres` running.

### Visual baselines

Baselines are Linux renderings. Generating them on Windows or macOS produces
different font rasterization and will fail CI on every run, so `npm run
e2e:visual` locally is only useful for eyeballing the report — never commit
snapshots it writes.

Regenerate them in the same image CI uses. `Dockerfile.playwright` layers the
project’s pinned Node 26 runtime onto the official Playwright image. The
Playwright image tag must match the installed `@playwright/test` exactly — its
browsers sit at a version-stamped path, so a mismatched tag fails every test
with `Executable doesn't exist at /ms-playwright/...`. The Dockerfile pins that
tag and its multi-platform index digest; the repository policy test keeps it
aligned with the lockfile.

Build the Go servers for Linux, then run Playwright with them inside the
container (everything stays on loopback, which the fake Bungie service
requires):

```powershell
$repo = (Get-Location).Path -replace '\\','/'
docker build --pull -t guardian-tracker/playwright-node26:local `
  -f frontend/Dockerfile.playwright frontend
docker compose --profile e2e up -d --wait e2e-postgres
docker run --rm -v "${repo}:/src" -v guardian-e2e-bin:/out -w /src/backend/api-service `
  -e CGO_ENABLED=1 golang:1.26.5 `
  bash -c "go build -o /out/api-service . && go build -o /out/fake-bungie ./cmd/fake-bungie"
docker run --rm --init --ipc=host --network guardiantracker_default `
  -e CI=true -e E2E_FIXED_TIME=2026-07-18T18:00:00Z `
  -e E2E_FAKE_COMMAND=/out/fake-bungie -e E2E_API_COMMAND=/out/api-service `
  -e E2E_DATABASE_URL="postgres://guardian_app:guardian_dev_password@e2e-postgres:5432/guardian_tracker?sslmode=disable" `
  -v "${repo}:/repo" -v "/repo/frontend/node_modules" -v guardian-e2e-bin:/out -w /repo/frontend `
  guardian-tracker/playwright-node26:local `
  bash -c "node --version && npm ci && npm run e2e:update-snapshots"
```

Run it from the repo root so the lockfile path resolves. The anonymous
`node_modules` volume keeps the container's Linux install from overwriting your
host `node_modules`.

## Destiny 2 Theme

Rarity and source-specific difficulty use design tokens in `styles/tokens.css`:

| Rarity    | Token                    |
| --------- | ------------------------ |
| Exotic    | `--c-exotic` (gold)      |
| Legendary | `--c-legendary` (purple) |
| Rare      | `--c-rare` (blue)        |
| Uncommon  | `--c-uncommon` (green)   |
| Common    | `--c-common` (gray)      |

Set `data-rarity` on item surfaces and `data-diff` on source-specific surfaces;
children read the resolved `--rarity` / `--diff` custom properties. Collection
cards do not display an aggregate difficulty badge. The detail drawer lists each
acquisition source with its own tier. Difficulty remains a filter that matches any
source, but it is no longer a sort option; legacy saved or URL `sort=difficulty`
state migrates to rarity.

## Production Build

```bash
npm run build
```

The static output in `/dist` is served by Nginx in Docker:

```dockerfile
FROM nginxinc/nginx-unprivileged:1.31.3-alpine3.24@sha256:334d92979f15aaecd5dd50af5105e1230e2bb70765d45b1e2f964e7c5eda81c3
COPY dist/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/conf.d/default.conf
```
