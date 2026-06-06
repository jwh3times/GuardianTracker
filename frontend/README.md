# Guardian Tracker — Frontend

React + TypeScript SPA for Guardian Tracker, a Destiny 2 collection tracking app.

## Tech Stack

- **Framework**: React 18 + TypeScript
- **Build/Dev**: Vite
- **GraphQL**: Apollo Client 3
- **Routing**: React Router v6
- **Styling**: Custom "Guardian Tracker" design system — oklch design tokens + `gt-*` CSS classes (`src/styles/`). Tailwind is still installed for a few legacy primitives but new UI uses the design system.
- **Icons**: in-house single-path line set (`components/kit/Icon.tsx`)
- **Forms/Validation**: React Hook Form + Zod (available)
- **Tests**: Vitest + React Testing Library

The design source (a self-contained prototype) lives in `frontend/design/`.

## Quick Start

```bash
npm install
cp .env.example .env.local   # fill in your values
npm start                    # Vite dev server on http://localhost:3000
```

The dev server proxies `/api` to the auth service (`http://localhost:8081`) — see `vite.config.ts`.

## Environment Variables

Vite exposes only `VITE_`-prefixed vars to the client:

```env
VITE_GRAPHQL_URL=http://localhost:4000/graphql
VITE_AUTH_SERVICE_URL=http://localhost:8081
# Optional dev tunnel host for OAuth over HTTPS (ngrok, Cloudflare Tunnel)
NGROK_HOST=
```

## Project Structure

```
src/
├── App.tsx                     # Router, lazy pages, ProtectedLayout (AppShell + auth gate)
├── index.tsx                   # Root; imports styles/{tokens,kit,app}.css
├── contexts/
│   ├── AuthContext.tsx          # Auth state, localStorage persistence, token refresh
│   └── PreferencesContext.tsx   # Card style + "for you" badge prefs (localStorage)
├── styles/
│   ├── tokens.css               # Design tokens (colors, type, spacing, rarity maps)
│   ├── kit.css                  # Component styles
│   └── app.css                  # App shell + page styles
├── components/
│   ├── AppShell.tsx             # Sidebar + top bar + mobile nav, search, character switcher
│   ├── Brand.tsx                # Logo mark
│   ├── ErrorBoundary.tsx        # React error boundary
│   ├── kit/                     # Design component kit
│   │   ├── Icon.tsx
│   │   ├── primitives.tsx       # Badge, Button, ProgressBar, RadialProgress, StatTile,
│   │   │                        #   CountdownChip, DataFreshnessChip, FilterChip, EmptyState…
│   │   ├── ItemCard.tsx
│   │   ├── composite.tsx        # Panel, CategoryTree, ItemDetailDrawer, SealCard, Dropdown…
│   │   └── index.ts             # Barrel export
│   └── ui/                      # Legacy primitives still in use (LoadingSpinner, Toast)
├── pages/
│   ├── Login.tsx                # Bungie OAuth initiation
│   ├── OAuthCallback.tsx        # Handles /auth/callback redirect
│   ├── Dashboard.tsx            # Completion hero + "do this today" (real totals, mock weekly)
│   ├── Collections.tsx          # Category tree + filterable grid/list + detail drawer
│   ├── WishList.tsx             # Wishlist management
│   ├── ThisWeek.tsx             # Weekly recs / Xûr / milestones (mock)
│   ├── Catalysts.tsx            # Catalysts & crafting patterns (mock)
│   ├── Triumphs.tsx             # Triumphs & seals (mock)
│   └── Settings.tsx             # Account + appearance preferences + sign out
├── graphql/
│   ├── queries.ts               # Apollo useQuery definitions
│   └── mutations.ts             # Apollo useMutation definitions
├── lib/
│   ├── apollo.ts                # Apollo Client setup with auth link + token refresh
│   ├── mockData.ts              # Mock data for backend-less screens + fallbacks
│   ├── adapters.ts              # GraphQL item → design GTItem/WishlistEntry adapters
│   └── utils.ts                 # Date/number/rarity helpers, image validation
└── types/
    ├── index.ts                 # Shared GraphQL/domain types
    └── design.ts                # Design-system domain types (GTItem, Seal, Weekly…)
```

## Authentication Flow

1. User clicks "Sign in with Bungie" on `/login`
2. Frontend calls auth-service `GET /api/auth/bungie` → receives OAuth URL + CSRF state
3. User is redirected to Bungie.net and authorizes the app
4. Bungie redirects to `/auth/callback?code=...&state=...`
5. `OAuthCallback` posts code + state to auth-service, receives JWT tokens
6. Tokens stored in `localStorage` (`guardian_token`, `guardian_refresh_token`)
7. Apollo Client's auth link injects `Authorization: Bearer <token>` on every request

Token refresh is handled automatically by `AuthContext.refreshAccessToken()` and the Apollo error link.

## App Shell & Navigation

Authenticated routes render inside `AppShell` (persistent left sidebar + top bar on desktop,
bottom tab bar + drawer on mobile). Layout is fixed to sidebar nav, comfortable density, and
the cyan "signal" accent. Sections: Dashboard, This Week, Collections, Catalysts & Crafting,
Triumphs & Seals, Wishlist, Settings.

## Pages & Data Sources

| Page | Data |
|---|---|
| Dashboard | Real collection totals (weapons/armor/exotics); weekly/wishlist modules use mock data |
| Collections | Real `userCollections` missing items mapped to `GTItem`; mock fallback when unavailable |
| Wishlist | Real `wishList` query + remove/update mutations; mock fallback |
| This Week / Catalysts / Triumphs | Mock data (`lib/mockData.ts`) — no backend yet |
| Settings | Real account info from `useAuth`; appearance prefs via `PreferencesContext` |

## User Preferences

`PreferencesContext` persists two settings to `localStorage` (`guardian_prefs`), surfaced in **Settings**:

- **Item card style** — `framed` (full cards) or `compact` (condensed rows), consumed by Collections
- **"For you" badges** — `on`/`off` for the personalized "Missing"/"Available now" badges

## Available Scripts

```bash
npm start           # Vite dev server (port 3000)
npm run build       # tsc + Vite production build → /dist
npm test            # Vitest
npm run lint        # ESLint (flat config)
npm run lint:fix    # ESLint with auto-fix
npm run type-check  # tsc --noEmit
```

## Destiny 2 Theme

Rarity and difficulty drive the visual language via design tokens in `styles/tokens.css`:

| Rarity | Token |
|---|---|
| Exotic | `--c-exotic` (gold) |
| Legendary | `--c-legendary` (purple) |
| Rare | `--c-rare` (blue) |
| Uncommon | `--c-uncommon` (green) |
| Common | `--c-common` (gray) |

Set `data-rarity` / `data-diff` on a wrapper element and children read the resolved
`--rarity` / `--diff` custom properties. Badges, item tiles, and the detail drawer all use this.

## Production Build

```bash
npm run build
```

The static output in `/dist` is served by Nginx in Docker:

```dockerfile
FROM nginx:alpine
COPY dist/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/nginx.conf
```
