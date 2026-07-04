# Guardian Tracker — Roadmap Implementation Plan

- **Date:** 2026-07-03
- **Branch:** `claude/roadmap-implementation-plans-o9740c`
- **Sources used:**
  - `PRD.md` §12 "Roadmap / Prioritization Summary" (+ §§1–11 feature specs, §13 open design questions)
  - `CLAUDE.md` "Known Limitations"
  - `.claude/agents/react-frontend.md` "Known limitations / TODOs"
  - Code `TODO` comments (`backend/api-service/db/users.go:94`, `backend/api-service/auth/revocation.go:30`, `backend/api-service/api/handlers/account.go:112`, and the "TODO item 13" epic header in `backend/api-service/db/migrations/0002_roles_flags.sql:1`)
  - Direct code inspection of `frontend/src/` and `backend/api-service/` (all statuses below verified against the working tree at commit `459d3e2`)
- **Important context:** the PRD's "Current State" (§3) predates a large amount of shipped work. Most of Phase 1 and much of Phase 2 is **already implemented** (see `CHANGELOG.md` `[Unreleased]`). This plan therefore (a) records verified status per item, and (b) plans in full detail only the work that is genuinely outstanding — residual P0 gaps, the five Known Limitations, and the code-TODO items — then covers remaining Phase 2 at medium detail and Phase 3 as scoped outlines.

---

## 1. Status verification summary

### 1.1 PRD §12 roadmap items

| # | Item (PRD) | Verified state | Evidence |
|---|---|---|---|
| P0-1 | Remove debug `DataSourceBanner` | **Done** | No `DataSourceBanner` / "Mock data" strings anywhere in `frontend/src` (grep: zero matches). Replaced by `DataFreshnessChip` (`frontend/src/components/primitives.tsx:310-350`) used on Collections (`features/collections/Collections.tsx:331-342`), Dashboard (`features/dashboard/Dashboard.tsx:512`), Settings (`features/settings/Settings.tsx:289-297`). |
| P0-2 | Redesign Collections (category tree, filters, item detail, states) | **Done, two residual gaps** | Presentation-node tree served by `backend/api-service/services/collections/tree.go` (from manifest `DestinyPresentationNodeDefinition`, `services/manifest/repository.go:461`) and rendered via `CategoryTree` (`Collections.tsx:223-226`). Filters: missing-only (default on), rarity, difficulty ("estimate"-labelled), sort incl. availability, grid/list (`Collections.tsx:364-424`). `ItemDetailDrawer` in `components/composite.tsx:422-590` (perks, availability, farm-only, difficulty explainer). Coded error states: `PRIVACY_RESTRICTION`, `MANIFEST_NOT_READY`/503, `BUNGIE_ERROR`/502 (`frontend/src/lib/errorState.ts:19-40`); skeletons + "All caught up!" empty state (`Collections.tsx:444-504`). **Gaps:** filter state not persisted (plain `useState`, `Collections.tsx:66-74`); no obtainable-now / unobtainable filter toggle (availability exists only as a sort + badge). → Plan §3.2 |
| P0-3 | Real "This Week" planner with personalized missing-highlighting | **Done, gaps tracked as Known Limitations** | `GET /api/weekly/recommendations` (`main.go:321`) assembled by `services/weekly/service.go` from live Bungie data: public milestones (`fetchMilestones`, service.go:690), Xûr inventory with per-item `Missing` badges (service.go:312-318), reset/daily countdowns, efficiency-ranked "Recommended" actions (`rankRecommended`, service.go:526 → `services/efficiency/scoring.go:77`), daily actions. Frontend `features/weekly/ThisWeek.tsx` renders all of it with localStorage done-state keyed to `resetAt` (`ThisWeek.tsx:17-45`), degraded-manifest banner (`:120-127`). Nothing is mock. **Gaps:** milestone missing counts only for raid/dungeon (→ §4.5); Xûr location hardcoded `"Unknown"` (`weekly/service.go:325`, → §4.4); no general vendor section in the UI beyond Xûr + daily mods (→ §3.3). |
| P0-4 | Make Wishlist real with availability surfacing | **Done, two residual gaps** | Postgres-backed: `wishlist_items` table (`db/migrations/0001_init.sql:11`), store `db/wishlist.go` (Add/List/Update/Delete, ownership-scoped), full CRUD routes (`main.go:290-293`). Frontend: priority editing + notes with optimistic updates, availability badges, priority/availability sort, routed empty state (`features/wishlist/WishList.tsx:40-326`). **Gaps:** availability cross-check is **Xûr-only** (`api/handlers/wishlist.go:344-351,400-405` uses `weekly.Service.XurItemHashes`) even though the backend already computes a broader live-vendor set for Collections (`services/weekly/availability.go` `LiveVendorItemHashes`: Xûr, Banshee-44, Ada-1, Shaxx, Zavala, Drifter); no bulk actions (per-row remove only, `WishList.tsx:321-326`). → Plan §3.4 |
| P0-5 | Redesigned Dashboard | **Done** | No hardcoded mock vendor/activity blocks remain. Real queries: `/api/auth/profile`, shared collections query, `/api/weekly/recommendations`, `/api/wishlist` (`Dashboard.tsx:52-77`). Shows overall completion radial (`:184-191`), per-category bars incl. cosmetics (`:213-239`), character hero, "Do this today" best action + daily list (`:245-372`), this-week preview (`:375-419`), "Wishlist available now" strip (`:421-494`), freshness chip (`:512`), skeletons throughout. |
| P1-1 | Catalyst & Crafting tracker | **Done** | `GET /api/catalysts/…`, `/api/crafting/…` (`main.go:329-330`) via `services/records/service.go` (component 900 record walk, `GetCatalysts:306`, `GetCrafting:409`); page `features/collections/Catalysts.tsx` (flag-gated `catalysts-crafting`, `App.tsx:158-165`). |
| P1-2 | Triumphs / Seals | **Done** | `GET /api/seals/…` (`main.go:331`), `records/service.go GetSeals:509` (active + legacy seals, gilding via `CompletionRecordHash` interval objectives `:635-644`); page `features/collections/Triumphs.tsx` with "closest to done" sort (`Triumphs.tsx:12,43-47`); flag-gated `triumphs-seals`. |
| P1-3 | Cosmetics collections | **Done, ornaments/finishers excluded** | `/cosmetics` page with Emblem/Shader/Ghost/Ship/Sparrow/Emote tabs, owned/missing/all filter, detail modal (`features/cosmetics/*`; buckets in `cosmeticBuckets.ts:19-26`). Backend buckets cosmetics incl. shaders (`services/manifest/repository.go:246-254`). Ornaments deliberately out of v1 (`cosmeticBuckets.ts:12`); finishers not covered. → Plan §5.3 |
| P1-4 | Global search | **Done, index is in-memory** | `GET /api/items/search` (`main.go:324`) backed by `services/search/service.go` (manifest SQLite scan, itemTypes {2,3,14,21,22,23,24}, `:45`; 503 `SEARCH_NOT_READY` while warming, `api/handlers/search.go:42-48`). Header `SearchBar` with debounce + deep-link to `/collections?item=<hash>` (`components/AppShell.tsx:173-251`). Persistence gap → §4.3 |
| P1-5 | Multi-character | **Partial (display-only)** | `CharacterSwitcher` in header (`AppShell.tsx:79-170`, mounted `:346`) fed by `GET /api/characters/…` (component 200, `services/characters/service.go:39`); no backend endpoint accepts a `characterId` to scope data — collections/records/weekly are account-wide. → Plan §4.1 / §5.4 |
| P1-6 | Onboarding | **Not started** | No onboarding/tour/first-run UI anywhere in `frontend/src` (grep). Login is a single CTA screen (`features/auth/Login.tsx:14-36`). → Plan §5.5 |
| P2-1 | God-roll / owned-roll insights | **Not started (groundwork exists)** | Possible-perk pools already served (`GET /api/items/:itemHash/perks`, `services/items/service.go:41`, `services/manifest/perks.go`) and shown in the item drawer (`composite.tsx:552-574`). No item-instance (components 300–305) fetching, no owned-roll or god-roll matching. → Outline §6.1 |
| P2-2 | Character & loadout overview | **Not started (component 200 only)** | `bungie.GetCharacters` requests component 200 only (`services/bungie/client.go:163-164`); no equipment (205) / progressions (202) anywhere. → Outline §6.2 |
| P2-3 | Notifications / digests | **Not started** | No notification code, jobs, or email/push infrastructure. → Outline §6.3 |
| P2-4 | Sharing | **Not started** | No public/share routes; every data route requires JWT (`main.go:276-332`). → Outline §6.4 |

### 1.2 Known Limitations (CLAUDE.md) & agent/code TODOs

| # | Item | Verified state | Evidence |
|---|---|---|---|
| KL-1 | Character switcher drives display only | **Outstanding (by design, P2)** | See P1-5 above; backend never accepts/uses characterId for data scoping (`services/collections/service.go:178` fetches component 800 account-wide). → §4.1 |
| KL-2 | Redis in docker-compose but unused | **Outstanding (cleanup decision)** | `grep -ri redis backend/**/*.go` → zero matches; Redis exists only in `docker-compose.yml`, `.env.example`, `README.md`, `CLAUDE.md`, `.github/dependabot.yml`. → §4.2 |
| KL-3 | Search index in-memory, lost on restart | **Outstanding** | `entries []Entry` process memory only (`services/search/service.go:31`); async rebuild at startup + on manifest swap (`main.go:223,198`). → §4.3 |
| KL-4 | Xûr location always "Unknown" | **Outstanding (API limitation)** | Hardcoded at `services/weekly/service.go:325`. → §4.4 |
| KL-5 | Non-raid/dungeon milestones omit missing count | **Outstanding** | `Milestone.Missing *int` set only when `efficiency.MissingForMilestone` matches a raid/dungeon source bucket (`weekly/service.go:381-404`, `efficiency/milestone.go:9-15,33-58`). → §4.5 |
| TD-1 | `TODO 13.2` — role/revocation propagation window | **Implemented as designed; residual hardening** | 60s revocation+role cache (`auth/revocation.go:31-39`); single-session `logout` leaves the access token valid until JWT expiry (`expiryHours`, `auth/jwt.go:55`) — the agent-file limitation (`react-frontend.md:264`). → §4.6 |
| TD-2 | `TODO 13.4` — server-side tier gating for flagged features | **Partially realized** | `RequireTier` exists (`auth/roles.go:57-63`) but is wired only as `RequireAdmin` for `/api/admin` (`main.go:304`). `/api/catalysts`, `/api/crafting`, `/api/seals` are flag-gated in the UI only (`AppShell.tsx:53-56`) — the API enforces JWT but not flag/tier. → §4.7 |
| TD-3 | `TODO 13.1/13.3` | **13.1 done; 13.3 never referenced** | 13.1 = seed shipped flags enabled (`db/migrations/0002_roles_flags.sql:40`, done). No `13.3` reference exists anywhere; the "TODO item 13" epic breakdown is historical (migration header `0002_roles_flags.sql:1`), not an in-repo doc. No action. |
| — | react-frontend.md "warming up" search state | **Done** (UI state exists) | 503 `SEARCH_NOT_READY` handled; `errorState.ts:27-33` renders warming copy. Root cause (restart loss) is KL-3. |

---

## 2. Sequencing, priorities & dependencies

Ordering principle: finish the P0 residuals first (small, high-value, no new external facts needed), then the Known-Limitation fixes that harden what's shipped, then net-new P1 surface, then P2. Items marked **[verify-first]** depend on Bungie manifest facts that must be confirmed against a real manifest download (public CDN; needs only `BUNGIE_API_KEY` — no OAuth) *before* implementation, per the repo Ground Rules.

```text
Wave 1 — P0 residuals (independent of each other, do in any order)
  A. Wishlist availability across all live vendors (§3.4a)   [S]
  B. Collections filter persistence (§3.2a)                  [S]
  C. Collections availability/obtainability filters (§3.2b)  [S]
  D. Wishlist bulk actions (§3.4b)                           [M]

Wave 2 — hardening the shipped core
  E. TD-2: server-side flag/tier enforcement (§4.7)          [S]  ← do before promoting flags to paid/tiered use
  F. TD-1: shorten access-token TTL / logout window (§4.6)   [S]
  G. KL-2: remove Redis (§4.2)                               [S]
  H. KL-3: search index persistence (§4.3)                   [M]

Wave 3 — data-dependent improvements
  I. KL-5: milestone missing counts beyond raids (§4.5)      [M]  [verify-first]
  J. KL-4: Xûr location (§4.4)                               [S]  [verify-first]
  K. §3.3 This Week vendor section (uses A's plumbing)       [M]  (depends on A)

Wave 4 — remaining P1
  L. Cosmetics: ornaments + finishers (§5.3)                 [M]  [verify-first]
  M. Onboarding (§5.5)                                       [M]
  N. Multi-character groundwork (§4.1 / §5.4)                [L]  ← prerequisite for P2 character/loadout work

Wave 5 — P2 outlines (§6): god-rolls → loadouts → notifications → sharing
  (God-roll insights depend on N only loosely; loadout overview depends on N.)
```

Hard dependencies: **K depends on A** (shared vendor-availability plumbing); **§6.2 loadout overview depends on N** (character scoping); **§6.3 notifications depend on A/K** (availability signals are the notification triggers). Everything else is parallelizable.

---

## 3. Phase 1 (P0) — detailed plans for residual work

> P0-1 (debug banner) and P0-5 (dashboard) are complete — no work remains; evidence in §1.1. P0-2/3/4 are complete in the large; the sections below plan **only** the verified residual gaps, at full detail.

### 3.1 P0-1 Debug banner & P0-5 Dashboard — no action

Both verified done (§1.1). The only follow-up is documentation: `PRD.md` §3 "Current State" is badly stale (still describes the banner, mock dashboard, mock wishlist, 3-tab collections) and should be refreshed or annotated when convenient — see §7.

### 3.2 P0-2 residuals — Collections filter persistence & obtainability filters

**Objective & user value.** PRD §5.1: "Persist filter state per tab" and "filtering by … obtainable-vs-sunset". Players drilling through the category tree lose rarity/difficulty/missing-only/sort/view choices on every navigation or reload; and there is no way to filter to "things I can actually get right now."

**Current state.** All filter state is transient `useState` (`frontend/src/features/collections/Collections.tsx:66-74`: `active`, `missingOnly`, `rarity`, `diff`, `sort`, `view`). The only URL interaction is the `?item=<hash>` deep-link seed (`Collections.tsx:78,178-203`). Availability data already reaches the client: the full collections response (`?include=all`) carries an `avail` overlay stamped from live vendors (`backend/api-service/api/handlers/collections.go:77-80`, `availableNowOverlay:168`), and items carry a `farmOnly` flag ("cannot be reacquired", `services/collections/service.go:294`) rendered as a "Farm only" chip. But neither is offered as a *filter* — availability is only a sort option (`Collections.tsx:386-404`).

**Design / approach.**

- *Persistence mechanism:* encode filter state in **URL search params** (`rarity`, `diff`, `sort`, `view`, `missing`, `avail`, plus selected node hash `node`), with `localStorage` (`gt.collections.filters`) as the fallback default when params are absent.
  - **Decision:** URL-first. Rationale: shareable/bookmarkable views, back-button correctness, and it composes with the existing `?item=` deep-link.
  - **Rejected:** localStorage-only (not shareable; surprising cross-device behavior); React context (doesn't survive reload); per-tree-node filter memory (PRD said "per tab"; the tree replaced tabs — per-page state + node in the URL is the faithful equivalent and far simpler).
- *New filters:* two additive `FilterChip` toggles next to "Missing only":
  - **"Available now"** — keep items where `item.avail?.now` (client-side; data already present on the `include=all` response).
  - **"Hide farm-only"** — exclude `farmOnly` items (the closest honest proxy the manifest gives us for "unobtainable"; the source string is the only signal, already parsed server-side).
  - **Open question [verify-first]:** a true "sunset/unobtainable" signal. Candidates in the manifest are `DestinyCollectibleDefinition.acquisitionInfo` / collectible `stateOverrides`, or curated community lists. Whether the manifest actually distinguishes retired items must be confirmed against a real manifest download before promising a "sunset" filter — do **not** guess; ship `farmOnly` now and file the sunset filter as a follow-up spike.
- *State management:* a small `useCollectionsFilters()` hook wrapping `useSearchParams` + localStorage hydration; pure param-(de)serialization helpers unit-tested separately.

**Implementation tasks.**

1. Create `frontend/src/features/collections/useCollectionsFilters.ts` — hook + pure `parseFilters(searchParams)` / `serializeFilters(state)` helpers; hydrate defaults from localStorage; write-through on change (replaceState, not push, except node selection which pushes).
2. Modify `frontend/src/features/collections/Collections.tsx` — replace the six `useState` calls (`:66-74`) with the hook; add the two new `FilterChip`s to the filter bar (`:364-424`); extend the filter predicate (`:274-286`) with `availNow` and `hideFarmOnly`; keep `?item=` handling intact.
3. Add `frontend/src/features/collections/useCollectionsFilters.test.ts` (param round-trip, localStorage fallback, precedence) and extend `Collections.test.tsx` (filters restored from URL; chips filter the grid; clearing filters resets URL).
4. No backend changes.

**Testing plan.** Vitest + MSW using the existing collections fixtures (`src/test/testServer.ts`); assert grid contents by role/text per repo convention. Keep coverage ≥70% lines / ≥65% branches (`frontend/vite.config.ts` thresholds) — the new pure helpers make branch coverage cheap. Run `npm run format` (Prettier gate) before committing. No Go changes → no backend test impact.

**Docs updates.** `.claude/agents/react-frontend.md` (filter-state pattern, new hook); CHANGELOG `[Unreleased] Added`. CLAUDE.md unaffected.

**Risks & open questions.** URL param bloat is cosmetic only; the sunset-signal open question above. Deep-link `?item=` + `?node=` interaction needs an explicit test (drawer opens while tree selection honors `node`).

**Size:** S (persistence) + S (filters).

### 3.3 P0-3 residual — "This Week" vendor section (beyond Xûr)

**Objective & user value.** PRD §5.2/§7.4: the This Week hub should include "time-limited vendor offerings" as a section. Today the page renders Xûr, milestones, recommended and daily actions (`ThisWeek.tsx:112-146`) — daily vendor **mods** feed the dashboard "Do This Today" strip, but there is no browsable "Vendors this week/today" module showing what Banshee-44 / Ada-1 (and the ritual vendors) are selling with missing-item badges.

**Current state.** The backend already fetches everything needed: `getDailyVendorItems` (authed component-402 fetch, mods only, `services/weekly/service.go:829,940`) and `extractVendorItems`/`LiveVendorItemHashes` (all sale items from the allowlisted rotating vendors: Xûr, Banshee-44, Ada-1, Shaxx, Zavala, Drifter — `services/weekly/availability.go:14-26`), both cached against the daily/weekly reset. Nothing exposes a structured per-vendor sale list to the frontend.

**Design / approach.**

- Extend the existing `GET /api/weekly/recommendations` payload rather than adding a route (the page already consumes it; one fetch, one cache, one degraded story).
  - New field on `Weekly` (`services/weekly/service.go:19`):
    ```jsonc
    "vendors": [
      {
        "hash": "672118013",
        "name": "Banshee-44",
        "refreshIn": {"h": 5, "m": 12},          // daily reset for daily vendors
        "items": [
          {"hash": "...", "name": "...", "type": "...", "rarity": "Legendary",
           "missing": true, "wishlisted": false}
        ]
      }, ...
    ]
    ```
  - Reuse `extractVendorItems` output; enrich names/types/rarity from the manifest exactly as `fetchXurInventory` does (`service.go:633-651`); intersect with `GetMissingItemHashes` (weapons/armor/exotics — `services/collections/service.go:363`) and the user's wishlist hashes (already loaded per-user, `service.go:293-304`) for the badges. Cap items per vendor (e.g. sale items that resolve to tracked collectibles only — the same intersection `LiveVendorItemHashes` already relies on) to keep payloads sane.
  - **Decision:** collectible-relevant items only (drop mods/consumables from this module; mods stay in "Do This Today"). Rationale: the module's job is collection gap-filling, and the collectible intersection is already the established pattern. **Rejected:** a raw full-inventory vendor browser (payload bloat, duplicates Bungie's own vendor UI, no personalization value); a separate `/api/vendors` route (second Bungie fetch path + second cache + second degraded mode for the same data).
- Frontend: a `VendorsModule` alongside `XurModule` in `components/composite.tsx`, one collapsible card per vendor, `ItemTile` rows with `missing`/`wishlisted` badges, deep-link to `/collections?item=<hash>`, and a "vendor data unavailable" state (the fetch is best-effort authed 402 — it can fail while the rest of the page succeeds, mirroring `LiveVendorItemHashes`' best-effort contract).

**Implementation tasks.**

1. `backend/api-service/services/weekly/service.go` — add `Vendor`/`VendorItem` structs; build `Weekly.Vendors` in `GetWeekly` reusing the cached 402 response + manifest enrichment + missing/wishlist intersection.
2. `backend/api-service/services/weekly/availability.go` — factor the vendor-name allowlist so service + availability share one table.
3. Tests: `services/weekly/getweekly_test.go` and a new `vendors_test.go` (fixture 402 response → expected vendor blocks; degraded/missing-token path yields `vendors: []` not an error).
4. `frontend/src/types/api.ts` — extend the weekly type; `frontend/src/components/composite.tsx` — `VendorsModule`; `frontend/src/features/weekly/ThisWeek.tsx` — render between Xûr and Milestones; tests in `composite.test.tsx` + `ThisWeek.test.tsx` (present, empty, unavailable states).

**Testing plan.** Go: table-driven unit tests with fixture JSON (no live Bungie); run under `go test -race`; keep ≥60% statement coverage — weekly is pure-Go (no cgo needed), but run `./test-local.ps1` for the CI-equivalent sweep. Frontend: MSW fixture for the extended weekly payload; assert badges by text. Format gates: `gofmt -w .`, `npm run format`.

**Docs updates.** CLAUDE.md key-directories blurb unchanged; `.claude/agents/go-services.md` weekly section; CHANGELOG `Added`; PRD §7.4 note.

**Risks & open questions.** Component-402 responses vary by character/account state — the existing fixtures in `services/weekly` are the shape reference; if a vendor's sale shape deviates (e.g. Eververse categories), verify against a captured response before widening the allowlist (Ground Rules). Tess/Eververse deliberately excluded (real-money adjacency, low collection value) — revisit only with a PRD change.

**Size:** M.

### 3.4 P0-4 residuals — Wishlist availability breadth & bulk actions

#### (a) Availability across all live vendors

**Objective & user value.** PRD §5.3: "when a wishlisted item is in a vendor's inventory or this week's activity rewards, highlight it." Today a wishlisted Banshee-44 or Ada-1 sale item shows as unavailable while the *Collections* page shows it "Available now" — an inconsistency users will notice.

**Current state.** `api/handlers/wishlist.go` narrows availability to Xûr: the `xurInventoryIface` (`wishlist.go:39-43`), `enrichItems` → `h.xurHashes` → `weekly.Service.XurItemHashes` (`:344-351,371-376`), and `buildResponse` hardcodes `AvailableFrom:"Xûr"` (`:400-405`). Meanwhile `weekly.Service.LiveVendorItemHashes` (`services/weekly/availability.go`) already returns `map[uint32]string` item→vendor-name across Xûr + Banshee-44 + Ada-1 + Shaxx + Zavala + Drifter, cached, best-effort — and Collections consumes it (`api/handlers/collections.go:77-80`).

**Design / approach.** Swap the wishlist handler onto the same source Collections uses.

- Replace `xurInventoryIface` with `liveVendorIface { LiveVendorItemHashes(ctx, userID, membershipType, membershipID) (map[uint32]string, error) }` (match the real method signature in `availability.go`); `buildResponse` sets `AvailableNow=true, AvailableFrom=<vendor name>` from the map. Response shape (`avail: {now, from}`) is already what the frontend renders (`WishList.tsx:260-269`) — **no frontend change** beyond fixtures.
- Weekly *activity* rewards (featured raid/dungeon drops) as a second availability source: the efficiency engine already knows featured buckets (`services/efficiency/scoring.go` availability/featured multipliers). Surface as `avail.from = "Featured: <activity>"` in a follow-up step once (a) lands — keep the first PR one-source-swap small.
- **Decision:** reuse `LiveVendorItemHashes` verbatim. **Rejected:** duplicating vendor fetch logic in the wishlist handler (second cache, drift risk); pushing availability stamping into the DB layer (availability is per-request live data, not persistent state).

**Implementation tasks.**

1. `backend/api-service/api/handlers/wishlist.go` — replace the interface (`:39-43`), `enrichItems`/`xurHashes` (`:344-376`), and `buildResponse` vendor label (`:400-405`).
2. `backend/api-service/main.go` — pass the weekly service as the live-vendor provider (it already is the Xûr provider).
3. `backend/api-service/api/handlers/wishlist_test.go` — update the fake to the new interface; add cases: multi-vendor map, empty map (no availability), provider error (best-effort → no availability, no 5xx).
4. Frontend: update MSW fixtures only; `WishList.test.tsx` assertion for a non-Xûr vendor label.

**Testing plan.** Go handler tests are pure (no cgo/DB); `go test -race ./api/handlers`; coverage unaffected or improved. Frontend fixture change only.

**Docs updates.** CHANGELOG `Changed` ("wishlist availability now covers all rotating vendors"); `.claude/agents/go-services.md` wishlist note; CLAUDE.md project-overview sentence already generic — no change.

**Risks.** None significant — consumes an existing, tested code path. **Size: S.**

#### (b) Bulk actions

**Objective & user value.** PRD §5.3 "Bulk management": clear several completed/stale wishes or re-prioritize a set at once instead of N single-row round-trips.

**Current state.** Per-row remove and per-row priority/notes edits only (`WishList.tsx:71-131,321-326`); backend has per-id `PUT/DELETE /api/wishlist/:id` (`main.go:292-293`).

**Design / approach.**

- New endpoint `POST /api/wishlist/bulk` (JWT):
  ```jsonc
  // request
  {"action": "delete" | "set_priority", "ids": [1,2,3], "priority": "HIGH"}  // priority only for set_priority
  // response 200
  {"updated": 3, "skipped": 0}
  ```
  Single SQL statement per action, ownership-scoped exactly like the store's existing pattern (`WHERE id = ANY($1) AND user_id = $2` — mirrors `db/wishlist.go:77,97`). Cap `ids` at 100. Ids not owned by the caller are silently skipped (counted in `skipped`) — consistent with the per-id endpoints' 404-on-foreign-id semantics but batch-friendly.
  - **Decision:** one `bulk` verb-style endpoint. **Rejected:** client-side loop over `DELETE /:id` (N requests, partial-failure UX, rate pressure); RESTful `PATCH /api/wishlist` with a filter DSL (overkill).
- Frontend: selection mode on the wishlist page — checkbox per row + "Select all in group", sticky action bar (Delete, Set priority ▾), optimistic update with rollback mirroring the existing single-item mutations (`WishList.tsx:50-93`).

**Implementation tasks.**

1. `backend/api-service/db/wishlist.go` — `BulkDelete(ctx, userID, ids)` and `BulkSetPriority(ctx, userID, ids, priority)` returning affected counts.
2. `backend/api-service/api/handlers/wishlist.go` — `BulkUpdate` handler (validate action/priority label via the existing label↔int16 mapping `wishlist.go:21-22`; enforce cap).
3. `backend/api-service/main.go` — register route beside the other wishlist routes (`:290-293`).
4. Tests: `db/wishlist_test.go` (Postgres-backed via `TEST_DATABASE_URL`: ownership scoping, mixed-ownership skip counts) + `api/handlers/wishlist_test.go` (validation, cap, response shape).
5. Frontend: `features/wishlist/WishList.tsx` selection state + action bar; `frontend/src/lib` no change (uses `apiFetch`); `WishList.test.tsx` — select→bulk delete happy path, rollback on 500, priority bulk-set.

**Testing plan.** Go db tests need `TEST_DATABASE_URL` (self-skip otherwise) — run `./test-local.ps1` for the CI-equivalent (Postgres on :5533, cgo on, `-race`) per CLAUDE.md. Handler tests pure. Frontend Vitest + MSW; watch branch coverage on the selection-state reducer (extract as a pure function if needed to keep ≥65% branches).

**Docs updates.** CHANGELOG `Added`; `.claude/agents/go-services.md` route table; `.claude/agents/postgres-specialist.md` if it catalogs store methods.

**Risks & open questions.** None external. UX question (PRD §13.6 density): keep selection mode explicit (a "Select" toggle) so the default list stays clean. **Size: M.**

---

## 4. Known-Limitation fixes & code-TODO items (full plans)

### 4.1 KL-1 — Character switcher is display-only

**Objective & user value.** Make the header switcher (`AppShell.tsx:79-170`) mean something: per-character context where the data is genuinely per-character, and honest UI where it isn't. Collections/records are account-wide by Bungie's model — the note "Collections are account-wide" (`AppShell.tsx:163-165`) is correct and should stay.

**Current state.** `CharacterContext` holds the roster + active character from `GET /api/characters/…` (component 200 only, `services/characters/service.go:39`; `services/bungie/client.go:163-164`); the only backend use of a character is weekly's `resolvePrimaryCharacter` picking the most-recently-played Guardian for the vendor fetch (`services/weekly/service.go:860`). No endpoint accepts `characterId`.

**Design / approach.** Two honest increments, deferring true per-character *surfaces* to §6.2 (they're P2 by both CLAUDE.md and the PRD):

1. **Make the switcher drive vendor context.** Vendor inventories (component 402) are per-character (class-specific armor at Xûr/rituals). Accept optional `?characterId=` on `GET /api/weekly/recommendations`; `GetWeekly` passes it through instead of always using `resolvePrimaryCharacter`; frontend appends the active character id from `CharacterContext`. Cache keys already vary per user — extend the daily-vendor cache key with the character id (`liveVendorItemsCacheKey` and the daily-mods key in `service.go:829`).
2. **Label the boundary in the UI.** Switcher subtitle: "Vendors shown for this Guardian · collections are account-wide" (extends the existing note).

- **Decision:** validate the characterId against the user's roster server-side (fetch is already cached) and fall back to most-recent on mismatch — fail-soft, no 4xx. **Rejected:** plumbing characterId through collections/records (the underlying Bungie data is account-scoped; a parameter that does nothing is worse than none); a per-character collections view (misrepresents the game's data model).

**Implementation tasks.**

1. `backend/api-service/api/handlers/weekly.go` — read optional `characterId` query param.
2. `backend/api-service/services/weekly/service.go` — thread `characterID` through `GetWeekly` → `getDailyVendorItems` / 402 fetch; roster validation via the characters service; character-scoped cache keys.
3. Tests: `services/weekly/getweekly_test.go` (explicit vs fallback character; cache-key isolation), handler param test.
4. Frontend: `features/weekly/ThisWeek.tsx` + `features/dashboard/Dashboard.tsx` weekly queries include `characterId` (and it joins the React Query key so switching refetches); switcher copy tweak in `AppShell.tsx`; tests for query-key change.
5. Update CLAUDE.md Known Limitations (switcher now scopes vendor data; collections remain account-wide).

**Testing plan.** Pure Go unit tests (fixtures); `-race`; frontend Vitest asserting refetch-on-switch via MSW request spy. **Docs:** CLAUDE.md limitation rewrite, agent files, CHANGELOG.

**Risks & open questions.** Whether Xûr's *public* vendor fetch (400/402 public) differs materially per character — the current Xûr path is public/character-independent (`fetchXurInventory`, `service.go:633`) and should stay as-is; only the authed 402 path becomes character-scoped. **Size: M** (S backend + S frontend).

### 4.2 KL-2 — Remove unused Redis

**Objective.** Kill the standing "why is Redis here?" confusion; shrink the dev stack.

**Current state.** Zero Go references (grep-verified). Present in `docker-compose.yml` (service + `:6379` port + volume), `.env.example`, `README.md` (ports table + mentions), `CLAUDE.md` (Option A + Known Limitations), `.github/dependabot.yml` (docker ecosystem may reference the image), `.claude/agents/postgres-specialist.md`.

**Design / approach.** **Decision: remove**, don't adopt. Revocation (60s in-process cache over Postgres, `auth/revocation.go`), API caching (`go-cache`, `cache/cache.go`), and sessions are all deliberately Postgres/in-process; the deployment is single-instance; adopting Redis would add an operational dependency for zero current need. **Rejected:** wiring Redis into `cache.Cache` (premature — the interface already exists if multi-instance ever demands it; note that as the re-entry path).

**Implementation tasks.**

1. `docker-compose.yml` — delete the redis service, port, volume; check `depends_on`.
2. `.env.example` — remove any REDIS_* vars.
3. `README.md` — ports table row + prose; `CLAUDE.md` — Option A bullet, "down -v" wording, delete the Known Limitation; `.claude/agents/postgres-specialist.md` mention.
4. `k8s/` — verify no redis manifest exists (none expected); `.github/dependabot.yml` — drop the entry if the redis image is tracked.
5. Sanity: `docker compose up --build` clean; `docker compose config` valid.

**Testing plan.** No code paths touched → no unit-test change. CI `build-docker-images` validates compose-referenced Dockerfiles still build. **Docs:** included above + CHANGELOG `Removed`. **Risks:** none — grep-proven dead. **Size: S.**

### 4.3 KL-3 — Persist the search index across restarts

**Objective & user value.** Eliminate the ~30s post-restart window where `GET /api/items/search` returns 503 `SEARCH_NOT_READY` (`api/handlers/search.go:42-48`) despite the manifest SQLite already being on disk.

**Current state.** Index = `entries []Entry` in process memory (`services/search/service.go:31`), built async at startup (`main.go:223`) and on manifest swap (`main.go:198`); build scans `DestinyInventoryItemDefinition` via cgo sqlite (`service.go:133`), filtered to itemTypes {2,3,14,21,22,23,24} (`:45`); `builtVersion` tracks the manifest version (`:33`). The manifest itself persists in the `manifest-data` volume with a `_version.txt` sidecar (`services/bungie/manifest.go:35,139`).

**Design / approach.** **Snapshot the built index to disk, keyed by manifest version.**

- After a successful `BuildIndex`, write `<manifestDBPath dir>/search_index_<sanitized-version>.json.gz` (gob or gzip-JSON of `[]Entry` + version header; delete stale versions). On startup, if the snapshot version equals the manifest `_version.txt`, load it synchronously (fast — tens of ms for ~20–30k entries) and mark ready immediately; kick the async rebuild only when versions mismatch. Manifest-swap path unchanged (rebuild → overwrite snapshot).
- **Decision:** snapshot file beside the manifest in the existing persistent volume — no schema, no new infra, atomic rename like the manifest extract (`manifest.go:147-192`). **Rejected:** (i) SQLite FTS5 table built on swap — heavier build, cgo query per keystroke, and ranking semantics (`prefix beats substring`, `service.go:74-104`) would need porting; (ii) Postgres table — search is manifest-derived, not user data, and k8s dev mode runs DB-less; (iii) synchronous startup build — delays `/ready` for everything else.

**Implementation tasks.**

1. `backend/api-service/services/search/service.go` — `saveSnapshot()` / `loadSnapshot(expectedVersion string) bool`; call load in `NewService` or a new `Init()`; call save at the end of `BuildIndex`; single-flight guard already exists (`building` flag).
2. `backend/api-service/main.go` — pass a snapshot dir (derive from `cfg.ManifestDBPath`); attempt load before the async `go searchService.BuildIndex()` (`:223`).
3. Tests: `services/search/service_test.go` + `buildindex_test.go` — round-trip snapshot; stale-version snapshot ignored; corrupt file ignored (falls back to rebuild); these are pure-Go (no cgo) if the snapshot codec is tested against in-memory entries.
4. Docker/k8s: confirm the snapshot lands in the already-mounted manifest volume (no manifest changes needed).

**Testing plan.** `go test -race ./services/search` (snapshot tests cgo-free; index-build tests remain cgo-gated as today per CLAUDE.md); `./test-local.ps1` for full parity. **Docs:** CLAUDE.md Known Limitation rewrite ("index snapshot persists across restarts; cold start with a new manifest still rebuilds ~30s"), go-services agent, CHANGELOG. **Risks:** snapshot/manifest version skew — mitigated by strict version equality + fallback rebuild. **Size: M.**

### 4.4 KL-4 — Xûr location

**Objective.** Stop showing a permanently useless `Location: "Unknown"` (`services/weekly/service.go:325`; rendered by `XurModule`, `components/composite.tsx:239-307`).

**Current state.** The public Bungie API genuinely does not expose Xûr's location — documented in CLAUDE.md:174 and `react-frontend.md:266`.

**Design / approach — verification-first, then one of:**

1. **[verify-first] Game-state check:** in current Destiny 2, Xûr may have a *fixed* location (Bazaar/Tower) rather than rotating. This is a game fact that must be confirmed (not guessable from the repo). If fixed → hardcode the string as a named constant with a dated comment, done.
2. If still rotating: **drop the field** from the payload and UI — replace the location line with Xûr's schedule copy ("Arrives Friday · leaves at weekly reset", data we do have: `LeavesIn`, `NextXurArrival` `service.go:210-220`). Honest > wrong.
3. **Rejected (for now):** scraping a community source (e.g. third-party Xûr trackers) — adds an unvetted external dependency, ToS/reliability questions, and violates the "don't build on unverified third-party behavior" ground rule without a deliberate decision. Record as a possible P3 enhancement behind a feature flag.

**Implementation tasks (path 2, the default if verification is inconclusive).** Remove `Location` from `Xur` struct (`service.go:43`) and `:325`; adjust `XurModule` copy + tests (`composite.test.tsx`); update both docs' Known Limitations (limitation resolved by removal). **Testing:** trivial unit-test updates both stacks. **Size: S.**

### 4.5 KL-5 — Missing counts for non-raid/dungeon milestones

**Objective & user value.** "This Week" milestones currently show "N missing items" only for raids/dungeons; Nightfalls, seasonal challenges, etc. omit the badge (`Milestone.Missing *int`, set only when `efficiency.MissingForMilestone` keyword-matches a raid/dungeon source bucket — `services/weekly/service.go:381-404`, `services/efficiency/milestone.go:9-15,33-58`). Extending it makes the flagship personalization complete.

**Current state.** The gap is a *data* problem: the efficiency index buckets missing collectibles by **source string** (`services/efficiency/index.go`), and only raid/dungeon keywords reliably map milestone names → source buckets. There is no verified manifest signal linking an arbitrary milestone to reward collectibles (this is exactly why the limitation exists — CLAUDE.md:176).

**Design / approach — a mandatory verification spike, then implementation.**

- **Spike [verify-first]:** download a real manifest (public CDN, `BUNGIE_API_KEY` only) and inspect, for the currently-active milestones (`/Destiny2/Milestones/` is public):
  - `DestinyMilestoneDefinition.rewards[].rewardEntries[].items[]` — are reward item hashes populated for Nightfall/seasonal milestones?
  - `DestinyMilestoneDefinition.activities[] → DestinyActivityDefinition.rewards[].rewardItems[]` — per-activity reward lists;
  - whether reward items map to `DestinyCollectibleDefinition` rows (the collections module's currency, `services/manifest/repository.go`).
  Output: a short findings note (which milestone classes have a usable signal) checked into the PR description — **do not** design past what the data supports (repo Ground Rules).
- **Implementation (contingent):** for milestone classes with a verified signal, add `manifest/repository.go` readers for the reward paths; extend `efficiency.MissingForMilestone` with a hash-based path (milestone hash → reward item hashes → collectible missing check) that takes precedence over the keyword heuristic; keep the keyword path as fallback for raids (their reward lists are famously *not* in milestone rewards — the loot table lives in source strings, which is why the bucket heuristic exists). Nightfall's weapon rotation may only be exposed via activity modifiers/vendor focus — if the spike finds no signal, the limitation stays, documented with the spike's evidence.
- **Rejected:** hardcoded community loot tables (maintenance treadmill, wrong-data risk); guessing reward paths without the spike (explicitly banned by Ground Rules).

**Implementation tasks.** (1) Spike script/queries + findings; (2) `services/manifest/repository.go` reward readers + tests (cgo-gated, fixture DB like existing repo tests); (3) `services/efficiency/milestone.go` hash-based resolution + tests; (4) `services/weekly` wiring (no payload change — `Missing` just populates more often); (5) frontend: none (badge already renders when present — the `omitempty` contract, `weekly/service.go:60-65`); (6) docs: CLAUDE.md + react-frontend.md limitation update, CHANGELOG.

**Testing plan.** Manifest-reader tests follow the existing cgo/sqlite fixture pattern (self-skip without cgo — run `./test-local.ps1`); efficiency tests pure Go. **Risks:** the spike may conclude "no signal" for some milestone classes — that is a valid outcome; the plan's success criterion is *verified* coverage, not universal coverage. **Size: M** (spike S + impl S–M).

### 4.6 TD-1 — Logout/revocation window (access-token TTL)

**Objective.** Close the documented gap: "the old access token stays valid server-side until expiry (up to 24h) — revocation cache window is 60s, not instant" (`react-frontend.md:264`). Single-session logout revokes the *refresh* session but the outstanding access JWT keeps working until `exp`.

**Current state.** Access tokens expire after `expiryHours` hours (`auth/jwt.go:55`); refresh tokens `refreshExpiryDays` days (`:82`). Revocation is `token_version` (bumped on logout-all) checked through a 60s cache (`auth/revocation.go:31-39` — the deliberate TODO-13.2 design so role changes propagate without token churn). Nothing invalidates a *single* session's access token before `exp`.

**Design / approach.** **Shorten the access-token TTL to 15–60 minutes** (config default change) and lean on the already-shipped silent refresh (single-flight 401 refresh in `frontend/src/lib/api.ts:29-104`).

- With a 15m access TTL, single-session logout's residual exposure drops from "up to 24h" to ≤15m + the 60s cache — proportionate for this app's threat model (read-only Destiny data + wishlist).
- **Decision:** TTL reduction + docs. **Rejected:** per-session `jti` denylist consulted on every request (defeats the 60s cache's purpose, adds a Postgres read per request or another cache layer, and `logout/all` already exists for the "stolen device" case); moving revocation to Redis (see §4.2 — removed).

**Implementation tasks.** (1) `backend/api-service/config` — change the default `JWT_EXPIRY_HOURS` (or introduce minutes granularity if the config is hour-typed — check `config/config.go` and prefer `JWT_ACCESS_TTL` duration string with back-compat); (2) `.env.example`, `k8s/` configmaps, compose env; (3) verify frontend refresh flow under short TTL (existing `AuthContext` tests + one MSW test simulating expiry mid-session); (4) docs: SECURITY.md token-lifetime section, react-frontend.md limitation rewrite (window now ≤TTL+60s), CLAUDE.md if it states TTLs.

**Testing plan.** Go: jwt/middleware tests already cover expiry paths; add a config-parse test. Frontend: no behavior change (401→refresh path already tested). **Risks:** users with clock-skewed clients — keep standard leeway in verification. **Size: S.**

### 4.7 TD-2 — Server-side tier/flag enforcement for gated features

**Objective.** Realize `TODO 13.4` (`api/handlers/account.go:112`): feature access "enforced server-side with RequireTier, not by this UI hint." Today `/api/catalysts`, `/api/crafting`, `/api/seals` (`main.go:329-331`) are JWT-only; the `catalysts-crafting` / `triumphs-seals` flags gate nav visibility only (`AppShell.tsx:53-56`, `App.tsx:158-173`). Any authenticated standard-tier user can call the APIs directly regardless of flag state/min-tier.

**Current state.** `RequireTier` exists and is production-tested for admin (`auth/roles.go:57-63`, `main.go:304`). Flags live in `feature_flags` (key, `min_tier`, `enabled` — `db/migrations/0002_roles_flags.sql`), read via the flags store with a short cache in `GetFlags` (`account.go`).

**Design / approach.** New middleware `authz.RequireFlag(flags FlagStore, key string)`:

- Resolves the flag (through the same cached lookup `GetFlags` uses — extract that cache into the store or a small `FlagResolver` so handler + middleware share it): if `enabled=false` → `404 FEATURE_DISABLED` (hide existence, matching "hidden when disabled" UI semantics); if caller role < `min_tier` → `403 TIER_REQUIRED` with the tier name (frontend already renders an upsell for locked features).
- **Failure semantics decision:** fail-**open** on store error/absence, mirroring `GetFlags`' degraded mode (`account.go:114-125` — k8s dev runs DB-less and treats all shipped features accessible). Rationale: these flags are rollout/upsell controls, not security boundaries; admin remains strictly gated by `RequireAdmin`. State this explicitly in the middleware comment.
- Apply to the three routes; keep the mapping table (`route → flag key`) beside `NAV_FLAG`'s backend twin in one place.
- **Rejected:** per-handler checks (repeats logic, easy to forget on new routes); making flags fail-closed (would brick the DB-less dev mode and turn a flag-table hiccup into an outage).

**Implementation tasks.** (1) `backend/api-service/auth/` (or a new `authz` file colocated with `roles.go`) — `RequireFlag`; (2) shared cached flag resolver refactor (`api/handlers/account.go` + new middleware consume it); (3) `main.go:329-331` — wrap the three routes; (4) tests: `auth/roles_test.go`-style unit tests (enabled/disabled/under-tier/store-error), handler integration tests asserting 404/403/200; (5) frontend: map `FEATURE_DISABLED`/`TIER_REQUIRED` in `lib/errorState.ts` to the existing ComingSoon/upsell surfaces (defensive — nav already hides them); (6) docs: `account.go:112` comment update (13.4 done), SECURITY.md, go-services agent, CHANGELOG.

**Testing plan.** Pure Go (no cgo/DB — fake flag store); `-race`; helps the ≥60% gate. **Risks:** flag-key drift between `NAV_FLAG` (frontend) and the backend map — add a comment cross-reference both sides; consider asserting the keys in a test against the 0002 seed list. **Size: S.**

---

## 5. Phase 2 (P1) — medium-detail plans (outstanding items)

> P1-1 Catalysts & Crafting, P1-2 Triumphs/Seals, and the cores of P1-3 Cosmetics and P1-4 Global search are **done** (§1.1). Below: the verified remainders.

### 5.1 Triumphs/Seals polish — no planned work

"Closest to done" sort and gilding already shipped (`Triumphs.tsx:12-51`, `records/service.go:635-644`). Optional future: expandable per-seal triumph objective drill-down (PRD §7.7) — the records walk already collects objectives; would need a `GET /api/seals/:hash/records` detail endpoint + drawer. Backlog, not planned.

### 5.2 Global search follow-ups

- **Persistence** → planned in full at §4.3 (KL-3).
- **Search within Collections** (PRD §5.1): today search results deep-link to the item drawer (`AppShell.tsx:229-233`). Add an optional `nodeHash` scope param to `/api/items/search`? **No** — simpler: client-side filter box over the currently-selected tree node's items (all item names are already in the loaded collections payload). Tasks: filter input in the Collections toolbar; extend the §3.2 filter hook; tests. Files: `Collections.tsx`, `useCollectionsFilters.ts`. Risk: none. **Size: S.**

### 5.3 Cosmetics — ornaments & finishers

- **Approach:** extend the backend cosmetic bucketing (`services/manifest/repository.go:246-254` `CollectibleCategory`) with ornaments (armor/weapon ornaments are itemType 19 mods with ornament plug categories) and finishers, then add the buckets to `cosmeticBuckets.ts` (currently excludes ornaments explicitly, `:12`).
- **[verify-first]:** the exact manifest signal for "ornament" vs other mods (plugCategoryIdentifier prefixes vs itemSubType) and for finishers must be confirmed against a real manifest before coding — the current shader special-case (`itemType 19 + subType shader`) shows the pattern but ornaments are known to be messier. Spike first.
- **Main tasks/files:** repository classifier + cgo-fixture tests; `cosmeticBuckets.ts` + `cosmeticItems.ts` + `Cosmetics.tsx` tab additions; summary counts flow automatically (four-bucket summary already includes cosmetics, `services/collections/service.go:80`).
- **Risks:** ornament volume (thousands) — virtualized tiles already exist (`CosmeticsGrid`); misclassification noise if the signal is fuzzy — gate on the spike. **Size: M.**

### 5.4 Multi-character (beyond §4.1)

§4.1 delivers character-scoped vendor context. Remaining P1 scope: surface per-character *facts* the app already fetches (class/light/last-played roster is in `CharacterContext`) — e.g. class-specific "this exotic armor is for your Warlock" annotations on Xûr items (armor class lives in the manifest item definition). Tasks: manifest classType read; annotate `XurItem`/vendor items; badge in `XurModule`. Full per-character surfaces (progressions, loadouts) remain P2 (§6.2). **Size: S–M.**

### 5.5 Onboarding

- **Approach:** a first-run experience keyed on `user_preferences` (add `onboarded_at TIMESTAMPTZ NULL` via migration `0006`): (1) post-OAuth welcome interstitial explaining data freshness + read-only access; (2) a 3-step spotlight tour (Dashboard → This Week → Collections) using a lightweight in-house overlay (no new dep); (3) returning-player framing ("X missing items across Y categories" from the existing summary).
- **Main tasks/files:** migration + `db/prefs.go` + `GET/PUT /api/preferences` extension; `frontend/src/contexts/PreferencesContext.tsx` (`:60-98` sync pattern); new `frontend/src/features/onboarding/` components; `Dashboard.tsx` entry hook; Vitest for tour state machine (pure reducer).
- **Risks:** scope creep — cap at the 3-step tour; PRD §13.3/§13.6 design questions (personalization prominence, density) should be answered in the PR's screenshots rather than blocking. **Size: M.**

---

## 6. Phase 3 (P2) — scoped outlines

### 6.1 God-roll / owned-roll insights

- **Scope:** per-weapon "your copies" list (rolled perks per instance) + match against a curated wish/god-roll set + "better copy already owned" hint. Groundwork shipped: possible-perk pools (`GET /api/items/:itemHash/perks`, `services/manifest/perks.go`) rendered in the drawer.
- **Shape:** new Bungie fetch with components 102/201/205/300/302/304/305 (instances + sockets) — client method in `services/bungie/client.go` beside `GetProfile:142`; new `services/instances` mapping instance sockets → perk hashes; endpoint `GET /api/items/:itemHash/owned` returning owned rolls; frontend drawer tab "Your rolls."
- **Open questions [verify-first]:** socket/plug response shapes (capture a real 300-series response first); god-roll source (community wishlist file format e.g. voltron.txt — licensing/format verification needed vs. hand-curated starter set).
- **Depends on:** nothing hard; benefits from §4.1 roster plumbing. **Size: L.**

### 6.2 Character & loadout overview

- **Scope:** per-Guardian page — equipped loadout (component 205), subclass, artifact, progressions/ranks (202). Extends `services/characters` beyond component 200 (`client.go:163-164`); new `GET /api/characters/:mt/:mid/:characterId` detail endpoint; frontend `features/characters/` page reachable from the switcher.
- **Depends on:** §4.1 (switcher becomes navigational). **[verify-first]:** 202/205 response shapes. **Size: L.**

### 6.3 Notifications / digests

- **Scope:** opt-in weekly digest ("Xûr has 2 exotics you're missing; 1 wishlisted item at Banshee") — email-first (no infra for push). Requires: a scheduler (k8s CronJob or in-process ticker), a digest composer reusing §3.4a/§3.3 availability plumbing, an email provider (new secret + `services/notify`), per-user opt-in in preferences, and unsubscribe tokens.
- **Depends on:** §3.4a + §3.3. Deliberate open questions: provider choice, sending domain, GDPR-ish retention — decide before build. **Size: L.**

### 6.4 Shareable collection progress

- **Scope:** read-only public share links: `POST /api/share` mints a token → `GET /share/:token` serves a snapshot summary (category completion + seal highlights, no wishlist). New table (`share_links`), unauthenticated route group with strict rate limiting, OG-image-friendly public page (new unauthenticated frontend route).
- **Security note:** first unauthenticated data surface — needs `penetration-tester` agent review per CLAUDE.md delegation table. **Size: M–L.**

---

## 7. Cross-cutting documentation debt (all waves)

- **PRD.md §3 "Current State" is stale** (describes the pre-rewrite app: debug banner, mock weekly/wishlist/dashboard, 3 tabs, emoji icons). Refresh §3.1–3.3 after Wave 1 so future planning doesn't re-litigate shipped work.
- Every wave item updates: `CHANGELOG.md` `[Unreleased]` (correct category), CLAUDE.md Known Limitations (KL fixes), the relevant `.claude/agents/*.md` (route tables, patterns, limitations), and README ports/env if infra changes (§4.2, §4.6). The repo's Stop hook auto-detects drift; `docs-updater` fixes it.
- CI gates every PR must pass (from `.github/workflows/ci-cd.yml` / CLAUDE.md): Prettier + `gofmt`; frontend type-check, lint, Vitest ≥70% lines / ≥65% branches, build; `go vet`, `govulncheck`, `go test -race` with Postgres container, ≥60% statement coverage (use `./test-local.ps1` locally — cgo + `TEST_DATABASE_URL` on :5533 — since plain `go test ./...` under-reports); Docker build validation; CodeQL scanning gate.

---

*Plan authored 2026-07-03 on branch `claude/roadmap-implementation-plans-o9740c`. All "Current state" claims were verified by direct code inspection at commit `459d3e2`; items marked **[verify-first]** must be confirmed against a real Bungie manifest (public CDN download, `BUNGIE_API_KEY` only) before implementation, per the repository's Ground Rules.*
