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
CSS classes in `src/styles/{tokens,kit,app,admin}.css`, with reusable design-system
components in `components/` (`primitives.tsx`, `composite.tsx`, `Icon.tsx`). Build new UI
from these components and `gt-*` classes — not Tailwind utilities. The design source
prototype lives in `frontend/design/` (do not import from it at runtime).

The frontend is organized **feature-first**: each feature owns its pages, feature-only
components, lib helpers, and tests under `src/features/<feature>/`. Truly shared building
blocks live in `components/` (design system), `lib/` (cross-cutting helpers), `contexts/`,
`types/`, and `styles/`.

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
    adapters.ts                ← API response types → design GTItem/WishlistEntry; relTime (guards zero-time);
                                   toGTItemView(APIItemView) — maps a manifest-only item view to a view-only GTItem
    constants.ts               ← Label constants (RARITIES, glyphs) + BUNGIE_CDN base URL
    roles.ts                   ← Role/tier constants, labels, colors (used by admin kit + Settings)
    utils.ts                   ← cn() Tailwind class merger (used by LoadingSpinner + Toast)
    errorState.ts              ← errorState(error) → ErrorStateCopy; branches on ApiError.code
    queries.ts                 ← collectionsQuery() — shared React Query definition used by multiple pages;
                                   itemPerksQuery(itemHash) — lazy perk-pool fetch (staleTime: Infinity);
                                   itemByHashQuery(itemHash) — minimal item view for deep-link miss resolution
                                   (staleTime: Infinity, retry: false)
  components/                  ← Shared design-system components (flat — no kit/ or ui/ subfolders)
    AppShell.tsx               ← Sidebar + top bar + mobile nav; global search (navigates to
                                   /collections?item=<hash>); character switcher (reads CharacterContext);
                                   flag-gated nav + admin nav link
    Brand.tsx                  ← Logo mark
    ErrorBoundary.tsx          ← App-wide error boundary
    Icon.tsx                   ← single-path line-icon set
    primitives.tsx             ← Badge, Button, ProgressBar, RadialProgress, StatTile, EmptyState,
                                   Textarea, FilterChip, CountdownChip, DataFreshnessChip, Skeleton…
    composite.tsx              ← Panel, CategoryTree, ItemDetailDrawer (renders perk columns — label + chips —
                                   with loading state; props: perkColumns?, perksLoading?, catalysts?; renders a
                                   "Catalyst" section (name + description) when catalysts is non-empty; old flat
                                   item.perks block removed), SealCard, Dropdown…
    LoadingSpinner.tsx         ← (was components/ui/) shown while data loads
    Toast.tsx                  ← (was components/ui/) ToastProvider / useToast
                                 Import these directly (e.g. "../../../components/primitives") — the
                                 old components/kit/index.ts barrel has been removed.
  features/                    ← Feature slices; each owns its pages, feature-only components, helpers,
                                 and tests — all flat at the feature root (no pages/, components/kit/,
                                 or lib/ subfolders)
    auth/                      ← Login.tsx (OAuth initiation), OAuthCallback.tsx (code exchange)
    collections/               ← Collections.tsx (tree + grid/list + detail drawer; ?include=all for
                                   collected items; search deep-link ?item=<hash>; add/remove wishlist),
                                   Catalysts.tsx (cards render `effect` text when present; "complete" status
                                   shows a full progress bar instead of "Not yet acquired"), Triumphs.tsx
                                   ({ items, fetchedAt } envelopes),
                                   ItemCard.tsx (collection-only item card),
                                   collectionTree.ts (API node → TreeNode adapters, collection-specific),
                                   useCollectionsFilters.ts (URL-sourced filter/category state — see
                                   "Collections page features" below)
    cosmetics/                 ← browsable /cosmetics gallery (Road to v1 §1). Cosmetics.tsx (type tabs +
                                   owned/missing/all filter over ?include=all data), CosmeticTile.tsx
                                   (image-forward tile + owned/missing state), CosmeticsGrid.tsx
                                   (virtualized tile grid via @tanstack/react-virtual), CosmeticDetail.tsx
                                   (dedicated view-only drawer — shared ItemDetailDrawer left untouched),
                                   cosmeticItems.ts (classify by itemType + group), cosmeticBuckets.ts
                                   (COSMETIC_TYPES: Emblem/Shader/Ghost/Ship/Sparrow/Emote)
    wishlist/WishList.tsx      ← wishlist mgmt; real API w/ optimistic mutations; inline notes editor
    weekly/ThisWeek.tsx        ← weekly recommendations / Xûr / milestones (real API)
    dashboard/Dashboard.tsx    ← completion hero + "do this today"; real totals + cosmetics + weekly
    settings/Settings.tsx      ← account info, early-access tier opt-in, appearance prefs, sign out
    admin/                     ← Admin.tsx (admin console: user roster + role mgmt, flag config;
                                   admin-gated route), AdminKit.tsx (RoleBadge, Switch, RoleSelect,
                                   TierSegment, FlagCard, UserRow, LockedFeature), AuditTable.tsx
                               (Tests colocate per feature, e.g. features/dashboard/Dashboard.test.tsx,
                                features/collections/collectionTree.test.ts.)
  types/
    api.ts                     ← API response types (APIUser, AuthTokenResponse, WishListItem with
                                   icon/availableNow/availableFrom, APIUserCollections with fetchedAt,
                                   APICollectionSummary with collectedItems, APIRecordsEnvelope<T>,
                                   APIPerkColumn, APIItemCatalyst, APIItemPerks, APIItemView)
    design.ts                  ← Design-system domain types (GTItem — GTItem.perks field REMOVED, Seal,
                                   Weekly with resetAt/fetchedAt/degraded, Milestone.missing now optional,
                                   WishlistEntry with icon, PerkColumn, ItemCatalyst, Catalyst.effect)
  test/                        ← Shared test infra (referenced by vite.config setupFiles)
    setup.ts                   ← Vitest setup file
    testServer.ts              ← MSW server + shared fixtures
  vite-env.d.ts
```

Deleted files / paths (do not reference):
- `frontend/src/pages/` — page files now live directly at `frontend/src/features/<feature>/`
- `frontend/src/features/<feature>/pages/`, `.../components/kit/`, `.../lib/` — per-feature nesting flattened; page, component, and helper files now sit directly at the feature root (e.g. `features/admin/Admin.tsx`, `features/admin/AdminKit.tsx`, `features/collections/ItemCard.tsx`)
- `frontend/src/components/kit/` and `frontend/src/components/ui/` — flattened into `components/`
- `frontend/src/components/kit/index.ts` — barrel removed; import components directly
- `frontend/src/__tests__/` — tests colocated; shared infra moved to `frontend/src/test/`
- `frontend/src/components/ui/Button.tsx`, `frontend/src/components/ui/Card.tsx` (deleted earlier)
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

`lib/queries.ts` exports:
- `collectionsQuery(membershipType, membershipId, includeAll?)` — canonical React Query definition for the collections endpoint; Dashboard, Settings, and Collections all use it so they share a single cache entry per (membership, variant) instead of firing separate requests.
- `itemPerksQuery(itemHash)` — lazy query for weapon perk columns (`GET /api/items/:itemHash/perks`); `enabled` is controlled by the caller (typically `!!detail?.id`); `staleTime: Infinity` since manifest data doesn't change mid-session. Used by Collections when the item detail drawer opens (click or deep-link).
- `itemByHashQuery(itemHash)` — minimal item view (`GET /api/items/:itemHash`); resolves a deep-link miss — a `?item=<hash>` URL with no collectible entry — into a read-only drawer. `enabled: !!itemHash`; `staleTime: Infinity`; `retry: false` (a 404 means the hash is not in the manifest — no value in retrying). Used by Collections when a deep-link hash cannot be found in any collection bucket.

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
- Supports search deep-link `?item=<hash>`: finds the item in any bucket (missing or collected) and opens the detail drawer; if no collectible entry exists, falls back to `itemByHashQuery` → `toGTItemView` for a read-only view drawer
- Cosmetics is a full top-level category alongside Weapons, Armor, Exotics
- Add-to-wishlist and remove-from-wishlist mutations with pending-state guard (prevents double-click races)
- Filter/category state (rarity, difficulty, sort, view, missing-only, "available now", "hide farm-only", and the selected category) is owned by `useCollectionsFilters` (`features/collections/useCollectionsFilters.ts`), which makes the URL the source of truth: `parseFilters`/`serializeFilters` read and write `?node=&rarity=&diff=&sort=&view=&missing=&avail=&farm=`, keeping the URL shareable and reload-safe. Filter fields (not `node`) also mirror to a `gt.collections.filters` localStorage entry, which supplies defaults only when the URL carries no filter params (e.g. a fresh visit) — `node` is always URL-only, never persisted to storage. History behavior differs by field: `setNode` pushes a history entry so Back returns to the previous category; every filter setter (`setRarity`, `setDiff`, `setSort`, `setView`, `setMissing`, `setAvail`, `setFarm`) replaces, so toggling a filter doesn't spam history. Callers that must change multiple fields atomically (e.g. restoring node + missing together from a deep link) go through `setFilters`, not sequential single-field setters — `useSearchParams`'s functional updater snapshots `prev` per call, so two `write` calls in the same tick would clobber each other.

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
- `Milestone.missing`: now `number | undefined` — populated for raid/dungeon milestones; still omitted for non-raid/dungeon (no manifest reward→collectible signal)
- `APIPerkColumn`: `{ role: string; label: string; perks: string[] }` — one column of the weapon perk pool
- `APIItemCatalyst`: `{ name: string; description: string }` — one exotic catalyst entry; `description` may be empty (the manifest has at least one blank entry — Duality)
- `APIItemPerks`: `{ itemHash: number; perkColumns: APIPerkColumn[]; catalysts: APIItemCatalyst[] }` — response envelope for `/api/items/:itemHash/perks`; `catalysts` is always present (empty array for non-exotics), up to 4 entries for multi-catalyst exotics
- `GTItem.perks`: REMOVED — the old dead flat perks field no longer exists on the design type; use `PerkColumn[]` from `itemPerksQuery` instead
- `GTItem.farmOnly?: boolean` — set when the collectible source indicates "cannot be reacquired"; renders a "Farm only" chip in the item card and drawer
- `APIItemView`: `{ itemHash, name, icon, itemType, tierType, rarity, description }` — response type for `GET /api/items/:itemHash`; used by `itemByHashQuery` + `toGTItemView` for the view-only deep-link drawer
- `PerkColumn` (design.ts): `{ role: string; label: string; perks: string[] }` — design-layer parallel to `APIPerkColumn`
- `ItemCatalyst` (design.ts): `{ name: string; description: string }` — design-layer parallel to `APIItemCatalyst`; passed to `ItemDetailDrawer`'s `catalysts` prop
- `Catalyst.effect?: string` (design.ts) — catalyst perk/effect text on the `/api/catalysts/...` response; rendered on `Catalysts.tsx` cards when present

## Styling

The app uses the **Guardian Tracker design system**, not Tailwind utilities:

- Design tokens (oklch colors, fluid type, spacing, radii) live in `src/styles/tokens.css`; component and shell styles in `kit.css` / `app.css`; admin console styles in `admin.css`. These are imported once in `index.tsx`.
- Style with the `gt-*` class vocabulary (e.g. `gt-card`, `gt-panel`, `gt-item`, `gt-badge`). Reach for the shared components in `components/` (`primitives`, `composite`, `Icon`) before writing markup by hand.
- Rarity/difficulty theming: set `data-rarity` (`exotic|legendary|rare|uncommon|common`) or `data-diff` (`easy|moderate|challenging`) on a wrapper; children read the resolved `--rarity` / `--diff` custom properties.
- Dynamic numeric values are passed as inline CSS custom properties (e.g. `style={{ "--val": pct }}`), cast as needed. Avoid other inline styles where a `gt-*` class exists.
- Layout is fixed: sidebar nav, comfortable density, cyan "signal" accent. Tailwind is still installed for the `LoadingSpinner` / `Toast` primitives and `ErrorBoundary` only — do not add new Tailwind-styled UI.
- User-adjustable prefs (card style, "for you" badges) come from `usePreferences()` — do not hardcode them.

## Testing

- Framework: Vitest + React Testing Library
- Setup file: `src/test/setup.ts` (MSW server + fixtures in `src/test/testServer.ts`)
- Tests are colocated with the code they cover: lib tests in `lib/`, component tests in `components/`, and each feature's page tests inside `features/<feature>/`
- Run: `npm test` (from `frontend/`)
- Test behavior, not implementation: prefer `getByRole`, `getByText`, `findBy*` over snapshot tests
- Mock `AuthContext` when testing pages that call `useAuth()`
- Do not use `MockedProvider` from Apollo — there is no Apollo Client in this project

## Known limitations / TODOs

- `logout()` ends only the current session; other devices remain signed in. `logoutAll()` ends all sessions and evicts the Bungie token. The old access token stays valid server-side until expiry (up to 24h) — revocation cache window is 60s, not instant
- Search index is built in-memory on the server — lost on restart; pages show a "warming up" error state while it rebuilds (~30s)
- Xûr location is always "Unknown" — the public Bungie API does not expose vendor location
- Raid and dungeon milestones carry a real missing count; non-raid/dungeon milestones still omit the field (no manifest reward→collectible signal)
