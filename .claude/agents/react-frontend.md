---
name: react-frontend
description: Use for any work in frontend/src/ — React pages, components, the Guardian Tracker design system (gt-* classes + design tokens), TanStack React Query REST calls, AuthContext, FlagsContext, and Vitest tests.
tools: Read, Write, Edit, Bash, Glob, Grep
model: sonnet
---

You are working inside the Guardian Tracker React frontend (`frontend/src/`). Know these patterns and enforce them without deviation.

## Stack

React + TypeScript + Vite + **TanStack React Query** (REST, not GraphQL). All source in `frontend/src/`. There is no Apollo Client and no GraphQL layer.

The UI uses the custom **Guardian Tracker design system**: oklch design tokens + `gt-*`
CSS classes in `src/styles/{tokens,kit,app,admin}.css`, with a reusable component kit in
`components/kit/`. Build new UI from the kit and `gt-*` classes — not Tailwind utilities.
The design source prototype lives in `frontend/design/` (do not import from it at runtime).

Dev commands (run from `frontend/`):
```powershell
npm start          # Vite dev server on :5273
npm run build      # tsc + Vite production build
npm test           # Vitest
npm run lint       # ESLint
```

## File structure

```
frontend/src/
  App.tsx                      ← Router, lazy-loaded pages, ProtectedLayout (CharacterProvider + AppShell + auth gate)
  index.tsx                    ← Root: AuthProvider, PreferencesProvider; imports styles/*.css
  contexts/
    AuthContext.tsx             ← Auth state, localStorage persistence, token refresh;
                                   logout (this device) + logoutAll (everywhere)
    PreferencesContext.tsx      ← Card style + "for you" badge prefs (localStorage guardian_prefs)
    CharacterContext.tsx        ← Characters query + persisted active-character pick (display-only)
    FlagsContext.tsx            ← GET /api/flags query; useFlag(key) / useFlags() resolved gating
                                   state + role
  styles/
    tokens.css                 ← Design tokens (color, type, spacing, rarity/difficulty maps)
    kit.css                    ← Component styles
    app.css                    ← App shell + page styles
    admin.css                  ← Admin console styles
  lib/
    api.ts                     ← apiFetch helper + ApiError (status/code/retryAfter) + QueryClient;
                                   API_URL exported; all REST calls go through apiFetch
    adapters.ts                ← API response types → design GTItem/WishlistEntry; relTime (guards zero-time)
    constants.ts               ← Label constants (RARITIES, glyphs) + BUNGIE_CDN base URL
    roles.ts                   ← Role/tier constants, labels, colors (used by admin kit + Settings)
    utils.ts                   ← cn() Tailwind class merger (used only by legacy ui/ primitives)
    errorState.ts              ← errorState(error) → ErrorStateCopy; branches on ApiError.code
    queries.ts                 ← collectionsQuery() — shared React Query definition used by multiple pages
  pages/
    Login.tsx                  ← Bungie OAuth initiation
    OAuthCallback.tsx          ← /auth/callback — exchanges code, stores tokens
    Dashboard.tsx              ← Completion hero + "do this today"; real collection totals + cosmetics, real weekly
    Collections.tsx            ← Category tree + filterable item grid/list + detail drawer; ?include=all for
                                   collected items; search deep-link (?item=<hash>); add/remove wishlist
    WishList.tsx               ← Wishlist management; real API with optimistic mutations; inline notes editor
    ThisWeek.tsx               ← Weekly recommendations / Xûr / milestones (real API)
    Catalysts.tsx              ← Catalysts & crafting patterns (real API; { items, fetchedAt } envelope)
    Triumphs.tsx               ← Triumphs & seals (real API; { items, fetchedAt } envelope)
    Settings.tsx               ← Account info + early-access tier opt-in (PUT /api/account/role) +
                                   appearance preferences + sign out
    Admin.tsx                  ← Admin console: user roster + role management, feature-flag config
                                   (admin-gated route, reads FlagsContext for role)
  components/
    AppShell.tsx               ← Sidebar + top bar + mobile nav; global search (navigates to
                                   /collections?item=<hash>); character switcher (reads CharacterContext);
                                   flag-gated nav + admin nav link
    Brand.tsx                  ← Logo mark
    ErrorBoundary.tsx          ← App-wide error boundary
    kit/                       ← Design component kit (build new UI from here)
      Icon.tsx                 ← single-path line-icon set
      primitives.tsx           ← Badge, Button, ProgressBar, RadialProgress, StatTile, EmptyState,
                                   Textarea, FilterChip, CountdownChip, DataFreshnessChip, Skeleton…
      ItemCard.tsx
      composite.tsx            ← Panel, CategoryTree, ItemDetailDrawer, SealCard, Dropdown…
      admin/                   ← Admin-specific kit: RoleBadge, Switch, RoleSelect, TierSegment,
                                   FlagCard, UserRow, LockedFeature
      index.ts                 ← barrel export (import from "../components/kit")
    ui/                        ← Legacy primitives still used by the shell: LoadingSpinner, Toast
                                   (Button.tsx and Card.tsx have been deleted)
  types/
    api.ts                     ← API response types (APIUser, AuthTokenResponse, WishListItem with
                                   icon/availableNow/availableFrom, APIUserCollections with fetchedAt,
                                   APICollectionSummary with collectedItems, APIRecordsEnvelope<T>)
    design.ts                  ← Design-system domain types (GTItem, Seal, Weekly with resetAt/fetchedAt/degraded,
                                   Milestone.missing now optional, WishlistEntry with icon)
  __tests__/
    utils.test.ts
    setup.ts                   ← Vitest setup file
  vite-env.d.ts
```

Deleted files (do not reference):
- `frontend/src/components/ui/Button.tsx`
- `frontend/src/components/ui/Card.tsx`
- `frontend/src/types/index.ts`
- `frontend/src/lib/apollo.ts`
- `frontend/src/lib/mockData.ts`
- `frontend/src/graphql/` directory

## Data fetching

All data fetching uses **TanStack React Query** (`useQuery` / `useMutation`) with `apiFetch` from `lib/api.ts`. There is no Apollo Client.

```typescript
// apiFetch injects Authorization: Bearer <token> automatically and throws ApiError on non-2xx
import { apiFetch, ApiError } from "../lib/api";

const { data } = useQuery({
  queryKey: ["wishlist"],
  queryFn: () => apiFetch<WishListItem[]>("/api/wishlist"),
});
```

**Never call `fetch()` directly** for API operations (except the OAuth callback flow which has no token yet). The `apiFetch` helper handles token injection, 401 refresh, and error shaping.

`ApiError` carries `.status` (HTTP status) and `.code` (backend machine-readable code). Use `errorState(error)` from `lib/errorState.ts` to map errors to UI copy — it branches on `PRIVACY_RESTRICTION`, `MANIFEST_NOT_READY`, `BUNGIE_ERROR`.

## Shared query definitions

`lib/queries.ts` exports `collectionsQuery(membershipType, membershipId, includeAll?)` — the canonical React Query definition for the collections endpoint. Dashboard, Settings, and Collections all use it so they share a single cache entry per (membership, variant) instead of firing separate requests.

## Authentication

Auth state lives entirely in `AuthContext` (`contexts/AuthContext.tsx`):

- Access token stored in `localStorage` under `guardian_token` (JWT, 24h expiry)
- Refresh token stored in `localStorage` under `guardian_refresh_token` (30d)
- `useAuth()` returns `{ user, loading, login, logout, logoutAll, isAuthenticated }`
- On mount, `AuthContext` checks if the stored JWT is expired and silently refreshes using the stored refresh token
- `login()` stores both tokens and sets user state
- `logout()` calls `POST /api/auth/logout` (ends this device's session only), then clears both tokens from `localStorage`
- `logoutAll()` calls `POST /api/auth/logout/all` (bumps token_version, ends all sessions, evicts Bungie token), then clears both tokens

**Never read `guardian_token` or `guardian_refresh_token` directly from `localStorage` in pages or components.** Always go through `useAuth()` or `apiFetch`.

**Never decode the JWT outside of `AuthContext`.** JWT claim parsing (displayName, membershipId, etc.) belongs only in `AuthContext`.

**Never put token refresh logic in page components or custom hooks.** It belongs in `AuthContext`.

## Roles and feature flags

`FlagsContext` (`contexts/FlagsContext.tsx`) queries `GET /api/flags` and exposes:
- `useFlag(key)` — returns `{ enabled, accessible, locked }` for a single flag key
- `useFlags()` — returns the full resolved flag map + caller's role

Use `useFlag` to gate UI — never make server-side enforcement decisions in the frontend based on role alone; that's the backend's job. `AppShell` reads `FlagsContext` to show/hide nav items and the admin console link.

`lib/roles.ts` exports role/tier constants, labels, and colors used by the admin kit components and the Settings tier opt-in flow.

## Protected routes

`App.tsx` groups authenticated pages under a single layout route, `ProtectedLayout`, which
wraps `CharacterProvider`, `AppShell`, and `<Outlet/>`:

```tsx
<Route element={<ProtectedLayout />}>
  <Route path="/dashboard" element={<Dashboard />} />
  <Route path="/collections" element={<Collections />} />
  {/* …this-week, catalysts, triumphs, wishlist, settings, admin */}
</Route>
```

`ProtectedLayout` reads from `useAuth()` and redirects to `/login` if not authenticated. New
authenticated pages go inside this group — do not add inline auth checks or re-wrap pages in
`AppShell` individually.

## Character context

`CharacterContext` (`contexts/CharacterContext.tsx`) fetches and caches the character list, and persists the active character selection per account. `useCharacters()` returns `{ characters, activeCharacter, setActiveCharacter }`. Data is display-only — collections and other data are account-wide.

## OAuth callback flow

`OAuthCallback.tsx` handles the redirect from Bungie at `/auth/callback?code=...&state=...`:
1. Reads `code` and `state` from URL params
2. POSTs `{ code, state }` to `POST /api/auth/bungie/callback` via `apiFetch`
3. On success: stores tokens via `AuthContext.login()`, redirects to `/dashboard`
4. On failure: shows error, redirects back to `/login` with an error param

## Collections page features

- Loads full data with `?include=all` (collected + missing) and filters display client-side via `missingOnly` toggle — no re-fetch when toggling the filter
- Supports search deep-link `?item=<hash>`: finds the item in any bucket (missing or collected), opens the detail drawer, then clears the URL param
- Cosmetics is a full top-level category alongside Weapons, Armor, Exotics
- Add-to-wishlist and remove-from-wishlist mutations with pending-state guard (prevents double-click races)

## Wishlist page features

- `WishListItem` now includes `icon`, `availableNow` (Xûr cross-check), and `availableFrom`
- `ItemTile` renders the Bungie CDN icon when present
- Inline notes editor: click to edit, optimistic update with rollback on error, 500 char max

## API response type changes

- `WishListItem`: `+ icon`, `+ availableNow: boolean`, `+ availableFrom?: string`
- `APIUserCollections`: `+ fetchedAt: string`
- `APICollectionSummary`: `+ collectedItems?: APIDestinyItem[]`
- `APIRecordsEnvelope<T>`: `{ items: T[]; fetchedAt: string }` — envelope for catalysts/crafting/seals
- `Weekly`: `+ resetAt`, `+ fetchedAt`, `+ degraded?`
- `Milestone.missing`: now `number | undefined` (omitted until computed by backend)

## Styling

The app uses the **Guardian Tracker design system**, not Tailwind utilities:

- Design tokens (oklch colors, fluid type, spacing, radii) live in `src/styles/tokens.css`; component and shell styles in `kit.css` / `app.css`; admin console styles in `admin.css`. These are imported once in `index.tsx`.
- Style with the `gt-*` class vocabulary (e.g. `gt-card`, `gt-panel`, `gt-item`, `gt-badge`). Reach for the kit components in `components/kit/` before writing markup by hand.
- Rarity/difficulty theming: set `data-rarity` (`exotic|legendary|rare|uncommon|common`) or `data-diff` (`easy|moderate|challenging`) on a wrapper; children read the resolved `--rarity` / `--diff` custom properties.
- Dynamic numeric values are passed as inline CSS custom properties (e.g. `style={{ "--val": pct }}`), cast as needed. Avoid other inline styles where a `gt-*` class exists.
- Layout is fixed: sidebar nav, comfortable density, cyan "signal" accent. Tailwind is still installed for `components/ui` legacy primitives and `ErrorBoundary` only — do not add new Tailwind-styled UI.
- User-adjustable prefs (card style, "for you" badges) come from `usePreferences()` — do not hardcode them.

## Testing

- Framework: Vitest + React Testing Library
- Setup file: `src/__tests__/setup.ts`
- Tests live in `src/__tests__/`
- Run: `npm test` (from `frontend/`)
- Test behavior, not implementation: prefer `getByRole`, `getByText`, `findBy*` over snapshot tests
- Mock `AuthContext` when testing pages that call `useAuth()`
- Do not use `MockedProvider` from Apollo — there is no Apollo Client in this project

## Known limitations / TODOs

- `logout()` ends only the current session; other devices remain signed in. `logoutAll()` ends all sessions and evicts the Bungie token. The old access token stays valid server-side until expiry (up to 24h) — revocation cache window is 60s, not instant
- Search index is built in-memory on the server — lost on restart; pages show a "warming up" error state while it rebuilds (~30s)
- Xûr location is always "Unknown" — the public Bungie API does not expose vendor location
- Milestone `missing` counts are not yet computed — the field is absent from the weekly payload; the UI hides the badge rather than implying completion
