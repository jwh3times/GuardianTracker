# Guardian Tracker — Frontend

React + TypeScript SPA for Guardian Tracker, a Destiny 2 collection tracking app.

## Tech Stack

- **Framework**: React 18 + TypeScript
- **Styling**: Tailwind CSS + Radix UI (shadcn/ui primitives)
- **GraphQL**: Apollo Client 3
- **Routing**: React Router v6
- **Forms**: React Hook Form + Zod
- **Icons**: Lucide React
- **Build**: Create React App (react-scripts)

## Quick Start

```bash
npm install
cp .env.example .env.local   # fill in your values
npm start
```

Open http://localhost:3000.

The frontend proxies API calls to `http://localhost:4000` (GraphQL service) by default — see `"proxy"` in `package.json`.

## Environment Variables

```env
REACT_APP_GRAPHQL_ENDPOINT=http://localhost:4000/graphql
REACT_APP_AUTH_SERVICE_URL=http://localhost:8081
```

## Project Structure

```
src/
├── App.tsx                    # Router, lazy page loading, ProtectedRoute
├── contexts/
│   └── AuthContext.tsx         # Auth state, localStorage persistence, token refresh
├── pages/
│   ├── Login.tsx               # Bungie OAuth initiation
│   ├── OAuthCallback.tsx        # Handles /auth/callback redirect
│   ├── Dashboard.tsx           # Collection overview + weekly summary
│   ├── Collections.tsx         # Missing items browser with filters
│   └── WishList.tsx            # Wish list management
├── components/
│   ├── Navigation.tsx          # App nav bar
│   ├── ErrorBoundary.tsx       # React error boundary
│   └── ui/
│       ├── Button.tsx
│       ├── Card.tsx
│       ├── LoadingSpinner.tsx
│       └── Toast.tsx           # Toast notification system
├── graphql/
│   ├── queries.ts              # Apollo useQuery definitions
│   └── mutations.ts            # Apollo useMutation definitions
├── lib/
│   ├── apollo.ts               # Apollo Client setup with auth link
│   └── utils.ts                # Rarity colors, difficulty helpers, image validation
└── types/
    └── index.ts                # Shared TypeScript types
```

## Authentication Flow

1. User clicks "Login with Bungie" on `/login`
2. Frontend calls auth-service `GET /api/auth/bungie` → receives OAuth URL + CSRF state
3. User is redirected to Bungie.net and authorizes the app
4. Bungie redirects to `/auth/callback?code=...&state=...`
5. `OAuthCallback` page posts code + state to auth-service, receives JWT tokens
6. Tokens stored in `localStorage` (`guardian_token`, `guardian_refresh_token`)
7. Apollo Client's auth link injects `Authorization: Bearer <token>` on every request

Token refresh is handled automatically by `AuthContext.refreshAccessToken()`.

## Pages

### Dashboard
Overview of collection progress, recent activity, and weekly recommendations (placeholder while backend feature is built).

### Collections
- Browse missing items by category: Weapons / Armor / Exotics
- Filter by acquisition difficulty: Easy / Moderate / Challenging
- Each card shows: name, type, rarity, sources, difficulty
- "Add to Wish List" button on each missing item
- `DataSourceBanner` debug component — shows live vs. mock data status

### Wish List
- View, prioritize, and manage desired items
- Persisted via GraphQL mutations to auth-service

## Available Scripts

```bash
npm start           # Development server (port 3000)
npm test            # Jest tests
npm run build       # Production build to /build
npm run lint        # ESLint
npm run lint:fix    # ESLint with auto-fix
npm run type-check  # TypeScript compile check (no emit)
```

## Destiny 2 Theme

The app uses a custom Tailwind theme with Destiny 2 item rarity colors:

| Rarity | Color |
|---|---|
| Exotic | Gold / yellow |
| Legendary | Purple |
| Rare | Blue |
| Uncommon | Green |
| Common | White/gray |

CSS classes `destiny-card`, `destiny-card-exotic`, `destiny-card-legendary`, `destiny-card-rare` apply the themed card styles.

## Production Build

```bash
npm run build
```

The static output in `/build` is served by Nginx in Docker:

```dockerfile
FROM nginx:alpine
COPY build/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/nginx.conf
```
