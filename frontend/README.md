# Guardian Tracker — Frontend

React + TypeScript SPA for Guardian Tracker, a Destiny 2 collection tracking app.

## Tech Stack

- **Framework**: React 19 + TypeScript
- **Build/Dev**: Vite
- **Data fetching**: TanStack React Query + `lib/api.ts` (`apiFetch` REST helper)
- **Routing**: React Router v7
- **Styling**: Custom "Guardian Tracker" design system — oklch design tokens + `gt-*` CSS classes (`src/styles/`). Tailwind is still installed for a few legacy primitives but new UI uses the design system.
- **Icons**: in-house single-path line set (`components/Icon.tsx`)
- **Forms/Validation**: React Hook Form + Zod (available)
- **Tests**: Vitest + React Testing Library

The design source (a self-contained prototype) lives in `frontend/design/`.

## Quick Start

```bash
npm install
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
│   ├── settings/pages/Settings.tsx       # Account + appearance preferences + sign out
│   └── admin/
│       ├── pages/Admin.tsx               # Admin console (roster, role mgmt, flag config; admin-gated)
│       └── components/kit/               # admin.tsx (RoleBadge, FlagCard, UserRow, LockedFeature…),
│                                         #   AuditTable.tsx — tests colocate per feature
├── lib/
│   ├── api.ts                   # apiFetch helper + ApiError (status/code) + QueryClient;
│   │                            #   API_URL exported; all REST calls go through apiFetch
│   ├── adapters.ts              # API response types → design GTItem/WishlistEntry; relTime
│   ├── constants.ts             # RARITIES, glyphs, BUNGIE_CDN base URL
│   ├── roles.ts                 # Role/tier constants, labels, colors
│   ├── utils.ts                 # cn() Tailwind class merger (LoadingSpinner + Toast)
│   ├── errorState.ts            # errorState(error) → ErrorStateCopy; branches on ApiError.code
│   └── queries.ts               # collectionsQuery() — shared React Query definition
├── types/
│   ├── api.ts                   # API response types (WishListItem + icon/availableNow,
│   │                            #   APIUserCollections + fetchedAt, APIRecordsEnvelope<T>, etc.)
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
| Collections          | Real data via `GET /api/collections?include=all`; supports ?item=<hash> deep-link     |
| Wishlist             | Real `GET/POST/PUT/DELETE /api/wishlist`; icon + Xûr availability cross-check         |
| This Week            | Real weekly data via `GET /api/weekly/recommendations` (Xûr, milestones, actions)     |
| Catalysts / Triumphs | Real data via `GET /api/catalysts`, `/api/crafting`, `/api/seals`                     |
| Settings             | Real account info from `useAuth`; appearance prefs via `PreferencesContext`           |

## User Preferences

`PreferencesContext` persists two settings to `localStorage` (`guardian_prefs`), surfaced in **Settings**:

- **Item card style** — `framed` (full cards) or `compact` (condensed rows), consumed by Collections
- **"For you" badges** — `on`/`off` for the personalized "Missing"/"Available now" badges

## Available Scripts

```bash
npm start           # Vite dev server (port 5273)
npm run build       # tsc + Vite production build → /dist
npm test            # Vitest
npm run lint        # ESLint (flat config)
npm run lint:fix    # ESLint with auto-fix
npm run type-check  # tsc --noEmit
```

## Destiny 2 Theme

Rarity and difficulty drive the visual language via design tokens in `styles/tokens.css`:

| Rarity    | Token                    |
| --------- | ------------------------ |
| Exotic    | `--c-exotic` (gold)      |
| Legendary | `--c-legendary` (purple) |
| Rare      | `--c-rare` (blue)        |
| Uncommon  | `--c-uncommon` (green)   |
| Common    | `--c-common` (gray)      |

Set `data-rarity` / `data-diff` on a wrapper element and children read the resolved
`--rarity` / `--diff` custom properties. Badges, item tiles, and the detail drawer all use this.

## Production Build

```bash
npm run build
```

The static output in `/dist` is served by Nginx in Docker:

```dockerfile
FROM nginxinc/nginx-unprivileged:1.25-alpine
COPY dist/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/conf.d/default.conf
```
