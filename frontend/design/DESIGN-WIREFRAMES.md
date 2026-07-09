# Guardian Tracker — Wireframes & Component Inventory

> **Companion to `PRD.md`.** This document translates the PRD into concrete low-fidelity **screen wireframes** and a **component inventory** (existing vs. new, with variants and states). It is intended for a UX/UI design agent to use as the structural blueprint before producing high-fidelity visuals.
>
> Wireframes are intentionally **lo-fi ASCII** — they communicate layout, hierarchy, and content, not final styling. Apply the design system in `PRD.md` §8 (Destiny rarity language, dark theme, badges) when rendering high fidelity.
>
> **Legend:** `[Button]` action · `( )`/`(•)` radio · `[ ]`/`[x]` checkbox/toggle · `▸/▾` expand/collapse · `▮▮▮▯▯` progress · `«badge»` status pill · `▒▒` image/icon · `…` truncation.

---

## 0. Responsive Framing

Three breakpoints referenced throughout:

- **Desktop ≥1280px (primary):** persistent left sidebar nav + top utility bar; multi-column content grids. Often used on a second monitor while playing.
- **Tablet 768–1279px:** collapsible sidebar (icon rail), 2–3 column grids.
- **Mobile <768px:** bottom tab bar or hamburger; single-column; filters move into a bottom sheet; item detail is a full-screen sheet.

---

## 1. App Shell / Global Navigation

### Desktop

```text
┌──────────────┬──────────────────────────────────────────────────────────────┐
│  GUARDIAN     │  [🔍 Search items…]        «Updated 4m ago ↻»   [▒ Maya ▾]    │ ← top utility bar
│  TRACKER      ├──────────────────────────────────────────────────────────────┤
│              │                                                                 │
│  ▸ Dashboard  │                                                                 │
│  ▸ This Week  │                  ┌─ Character switcher (in [▒ Maya ▾]) ─┐        │
│  ▸ Collections│                  │  (•) Maya — Warlock  1810           │        │
│  ▸ Catalysts  │                  │  ( ) Titan  1808                    │        │
│    & Crafting │                  │  ( ) Hunter 1805                    │        │
│  ▸ Triumphs   │                  └─────────────────────────────────────┘        │
│  ▸ Wishlist   │                       (page content region)                     │
│              │                                                                 │
│  ──────────  │                                                                 │
│  ⚙ Settings   │                                                                 │
│  ⏻ Sign out   │                                                                 │
└──────────────┴──────────────────────────────────────────────────────────────┘
```

- **Search** (global, §Component `SearchBar`) and **data-freshness chip** (`«Updated 4m ago ↻»`) live in the top bar.
- **Character switcher** is a dropdown off the account menu.
- Active nav item gets a rarity-gold accent rail.

### Mobile

```text
┌─────────────────────────────┐
│ ☰  Guardian Tracker   [▒]   │ ← header: hamburger + account
│ «Updated 4m ago ↻»          │
├─────────────────────────────┤
│        (page content)        │
│                             │
├─────────────────────────────┤
│ ⌂      ▦       ★      ☰      │ ← bottom tabs: Dash / Collections / Wishlist / More
└─────────────────────────────┘
```

---

## 2. Login

```text
┌──────────────────────────────────────────────┐
│                                                │
│              ✶  GUARDIAN TRACKER               │
│        See what you're missing. Chase          │
│            what matters this week.             │
│                                                │
│   ┌──────────────────────────────────────┐    │
│   │     [  Sign in with Bungie  ]        │    │ ← primary CTA
│   └──────────────────────────────────────┘    │
│                                                │
│   We read your Destiny 2 collection            │
│   (read-only). We never modify your account.   │
│                                                │
│   • What's missing across all categories       │
│   • The best things to do each week            │
│   • Track catalysts, patterns & seals          │
└──────────────────────────────────────────────┘
```

**States:** default · `signing-in` (OAuth redirect spinner) · `callback-processing` ("Completing sign-in…") · `error` (auth failed → retry).

---

## 3. Dashboard

**Goal:** within 5 seconds, "how complete am I?" + "what should I do today?"

```text
┌────────────────────────────────────────────────────────────────────┐
│ Welcome back, Maya                         Weekly reset in  2d 14h   │
├────────────────────────────────────────────────────────────────────┤
│ ┌─ OVERALL COMPLETION ───────────────────────────────────────────┐ │
│ │   ◜◝   74%      Weapons ▮▮▮▮▮▮▮▯ 81%   Exotics ▮▮▮▮▮▯▯ 68%      │ │ ← hero summary (radial + bars)
│ │   ◟◞            Armor   ▮▮▮▮▮▮▯▯ 72%   Cosmetics ▮▮▮▯▯ 55%      │ │
│ └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│ ┌─ DO THIS TODAY ────────────────────────────────────────────────┐ │
│ │ ⚑ Xûr is selling «Missing» Hawkmoon — Tower, leaves in 1d 6h    │ │ ← "best next action"
│ │ ⚑ Featured: Vault of Glass — 2 weapons you're missing «New»     │ │
│ └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│ ┌─ THIS WEEK (preview) ──────────┐ ┌─ WISHLIST AVAILABLE ────────┐ │
│ │ Nightfall: «The Corrupted»     │ │ 3 of your wishlisted items  │ │
│ │ Reward: Loaded Question «Missing» │ │ are obtainable now →        │ │
│ │ [ See full week → ]            │ │ ▒ Fatebringer  «Vendor»     │ │
│ └────────────────────────────────┘ │ ▒ Eyasluna     «Activity»   │ │
│                                     └─────────────────────────────┘ │
│ [ View Collections ]  [ Manage Wishlist ]  [ ↻ Refresh data ]       │
└────────────────────────────────────────────────────────────────────┘
```

Every time-limited element carries a countdown + `«Missing»`/`«New»` personalization badge.
**States:** loading (skeleton tiles) · partial (weekly data unavailable → hide module, keep completion) · error per-module (degrade gracefully, never blank the whole page).

---

## 4. Collections

The most-used screen. Category-tree nav + filter bar + item grid + detail drawer.

```text
┌────────────────────────────────────────────────────────────────────────┐
│ Collections                                   74% complete · 1,204/1,628 │
├───────────────┬────────────────────────────────────────────────────────┤
│ CATEGORIES    │ [Missing only ✓] [Rarity ▾][Type ▾][Source ▾][Sort ▾]   │ ← filter/sort bar
│               │ Difficulty: «Easy»  «Moderate»  «Challenging»  (estimate) │
│ ▾ Weapons 81% │────────────────────────────────────────────────────────│
│   ▸ Raids 60% │ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐    │
│   ▸ Dungeons  │ │▒ exotic   │ │▒ legend.  │ │▒ legend.  │ │▒ legend.  │   │
│   ▸ Trials    │ │ Vex Myth. │ │Fatebringer│ │ Eyasluna  │ │ Found Verd│   │ ← item cards
│   ▸ Nightfall │ │ Auto Rifle│ │Hand Cannon│ │Hand Cannon│ │ Pulse     │   │
│   ▸ World     │ │«Challeng.»│ │«Moderate» │ │«Moderate» │ │«Easy»     │   │
│   ▸ Vendor    │ │ VoG raid  │ │ VoG raid  │ │ Trials    │ │ Gambit    │   │
│ ▸ Armor   72% │ │«Avail now»│ │           │ │«Avail now»│ │           │   │
│ ▸ Exotics 68% │ │ [+ Wish] ⓘ│ │ [+ Wish] ⓘ│ │ [+ Wish] ⓘ│ │ [+ Wish] ⓘ│   │
│ ▸ Cosmetics   │ └──────────┘ └──────────┘ └──────────┘ └──────────┘    │
│               │ … (responsive grid: 4 ▸ 3 ▸ 2 ▸ 1 cols)                 │
└───────────────┴────────────────────────────────────────────────────────┘
```

- **Category tree** mirrors Bungie presentation nodes, each row showing a completion %. Collapsible. (PRD open question: mirror exactly vs. cleaner taxonomy.)
- **Item card** (`ItemCard`): rarity frame, icon, name, type, difficulty badge, source, optional `«Avail now»` badge, `[+ Wish]` and detail `ⓘ`.
- **Filter bar** state persists per category.
- `(estimate)` annotation tempers the difficulty rating's credibility (PRD §13.5).

**States:** loading (card skeleton grid) · empty/all-collected (🎉 "All caught up!") · empty-after-filter ("No items match these filters · Clear") · error variants (auth-expired / Bungie-down / **private profile** / **manifest warming**) — each a distinct `EmptyState` treatment, not a generic spinner.

### 4b. Item Detail (drawer / sheet)

```text
┌─────────────────────────────────────────┐
│ ✕                                          │
│ ▒▒▒▒  Vex Mythoclast            «Exotic»  │
│ ▒▒▒▒  Fusion Rifle · Power                │
│                                            │
│ "A weapon born of the Vault…"              │
│                                            │
│ Acquisition «Challenging»  (why? ▾)        │
│   Drops from Vault of Glass (raid)         │
│   • Atheon, Time's Conflux encounter       │
│                                            │
│ Possible perks / rolls   (if available)    │
│   ▒ Overflow   ▒ Rangefinder   …           │
│                                            │
│ [ + Add to Wishlist ]   [ Where to farm ↗ ]│
└─────────────────────────────────────────┘
```

Desktop = right-side drawer; mobile = full-screen sheet. "why?" expands the difficulty rationale.

---

## 5. This Week (new — flagship)

```text
┌────────────────────────────────────────────────────────────────────┐
│ This Week                                  Resets Tue · 2d 14h left  │
├────────────────────────────────────────────────────────────────────┤
│ ┌─ RECOMMENDED FOR YOU ──────────────────────────────────────────┐ │
│ │ [x] Run Vault of Glass — 2 missing weapons «Completes set»      │ │ ← checkable action list
│ │ [ ] Buy Hawkmoon from Xûr — «Missing exotic» · 1d 6h            │ │
│ │ [ ] Nightfall on Legend — Loaded Question «Missing»             │ │
│ └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│ ┌─ XÛR (weekend) ─────────────┐ ┌─ MILESTONES & ACTIVITIES ───────┐ │
│ │ ▒ Hawkmoon   «Missing»      │ │ Featured Raid: Vault of Glass   │ │
│ │ ▒ Gemini Jester  (owned)    │ │ Nightfall: The Corrupted        │ │
│ │ ▒ Ophidian Aspect (owned)   │ │   ↳ Loaded Question «Missing»   │ │
│ │ Leaves in 1d 6h             │ │ Weekly: Vanguard / Crucible…    │ │
│ └─────────────────────────────┘ └─────────────────────────────────┘ │
│                                                                      │
│ ┌─ VENDOR ROTATIONS ─────────────────────────────────────────────┐ │
│ │ Banshee-44 ▒ ▒ ▒ «1 missing»   Ada-1 ▒ ▒ «shaders»             │ │
│ └────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
```

- Every module joins live weekly data against the player's collection to stamp `«Missing»` / `«Completes set»`.
- **Xûr module is conditional** (Fri–Tue): when absent → "Xûr returns Friday · 3d 2h."
- **States:** pre/post-reset, vendor data unavailable (skeleton → "Vendor data refreshing").

---

## 6. Wishlist

```text
┌────────────────────────────────────────────────────────────────────┐
│ Wishlist                                              12 items        │
│ [All 12] [Urgent 2] [High 4] [Medium 5] [Low 1]   Sort: [Availability ▾]│
├────────────────────────────────────────────────────────────────────┤
│ ┌────────────────────────────────────────────────────────────────┐ │
│ │ ▒ Fatebringer   Hand Cannon   «Legendary»     «URGENT»          │ │
│ │   «Available now — Vault of Glass»            Added 3d ago       │ │ ← availability surfacing
│ │   Notes: want Explosive + Firefly roll                          │ │
│ │   Priority [Urgent ▾]   [Edit notes]   [Remove]                 │ │
│ └────────────────────────────────────────────────────────────────┘ │
│ ┌────────────────────────────────────────────────────────────────┐ │
│ │ ▒ Vex Mythoclast  Fusion Rifle  «Exotic»      «HIGH»            │ │
│ │   Source: Vault of Glass (raid)               Added 1w ago      │ │
│ │   Priority [High ▾]   [Edit notes]   [Remove]                   │ │
│ └────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
```

- Adds **availability badges** and an **availability sort** on top of the existing priority model.
- **States:** loading · empty ("Build your wishlist from Collections" → CTA) · empty-after-filter.

---

## 7. Catalysts & Crafting (new)

```text
┌────────────────────────────────────────────────────────────────────┐
│ Catalysts & Crafting        [ Catalysts ]  [ Crafting Patterns ]    │ ← sub-tabs
├────────────────────────────────────────────────────────────────────┤
│ Filter: «Missing» «In progress» «Complete»          47/82 complete   │
│ ┌──────────────────────────────┐ ┌──────────────────────────────┐  │
│ │ ▒ Sunshot Catalyst           │ │ ▒ Whisper Catalyst           │  │
│ │ «In progress»                │ │ «Missing»                    │  │
│ │ Kills ▮▮▮▮▮▮▯▯ 320/500       │ │ Not yet acquired             │  │ ← objective progress bars
│ │ Source: random / activities  │ │ Source: The Whisper mission  │  │
│ └──────────────────────────────┘ └──────────────────────────────┘  │
│                                                                      │
│  (Crafting Patterns tab)                                             │
│ │ ▒ The Enigma   Patterns ▮▮▮▯▯ 3/5  «2 red-borders to go»         │ │
│ │   Farm: seasonal activity / Banshee                              │ │
└────────────────────────────────────────────────────────────────────┘
```

Progress-forward: objective/pattern bars are the primary visual. Shared `ProgressBar` + `ItemCard` (progress variant).

---

## 8. Triumphs & Seals (new)

```text
┌────────────────────────────────────────────────────────────────────┐
│ Triumphs & Seals                            Sort: [Closest to done ▾]│
├────────────────────────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐    │
│ │   ◜◝ 92%    │ │   ◜◝ 78%    │ │  ✓ Gilded   │ │   ◜◝ 40%    │    │
│ │  Conqueror  │ │  Flawless   │ │  Rivensbane │ │  Chronicler │    │ ← seal cards (radial %)
│ │  «3 left»   │ │  «titles»   │ │  x2         │ │             │    │
│ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘    │
│ ▾ Conqueror — 3 triumphs remaining                                  │
│    [x] Complete 5 GMs        ▮▮▮▮▮ done                              │
│    [ ] Solo a Legend NF      ▮▮▯▯▯ 2/5                               │ ← expandable objectives
│    [ ] Flawless Master NF    ▯▯▯▯▯ 0/1                               │
└────────────────────────────────────────────────────────────────────┘
```

Gilded seals show a gild count badge (`x2`). Expand a seal to see its triumph objectives.

---

## 9. Cross-Cutting States Catalog

Design these once as reusable templates; every screen references them:

| State                        | Trigger                | Treatment                                                                                            |
| ---------------------------- | ---------------------- | ---------------------------------------------------------------------------------------------------- |
| **Skeleton loading**         | Initial/refetch        | Layout-matched gray placeholders (cards, lists, radial) — preferred over spinners except first auth. |
| **Empty — success**          | 100% / nothing missing | Celebratory (🎉), affirming copy.                                                                    |
| **Empty — filtered**         | Filters exclude all    | Neutral + "Clear filters" CTA.                                                                       |
| **Empty — wishlist**         | No items               | Motivational + route to Collections.                                                                 |
| **Error — session expired**  | Token invalid          | Re-auth CTA; preserve location.                                                                      |
| **Error — Bungie down**      | API 5xx/maintenance    | Reassuring "try again shortly" + retry.                                                              |
| **Error — private profile**  | Privacy-restricted     | Explain how to make collections public + help link.                                                  |
| **State — manifest warming** | Cold start 503         | "Getting the latest game data…" + auto-retry.                                                        |
| **Stale data**               | Cache age              | `«Updated Xm ago ↻»` chip; manual refresh.                                                           |

---

## 10. Component Inventory

### 10.1 Existing primitives (keep, extend)

| Component                                  | Today                                                              | Needed changes                                                 |
| ------------------------------------------ | ------------------------------------------------------------------ | -------------------------------------------------------------- |
| `Button`                                   | variants: default/outline/ghost/secondary/destructive; sizes sm/lg | Add `icon` slot, `loading` state already ad-hoc → formalize.   |
| `Card` (+Header/Content/Title/Description) | rarity classes (`destiny-card-exotic/legendary/rare`), glow        | Generalize into `ItemCard` (below); keep base Card for layout. |
| `LoadingSpinner`                           | sm/lg                                                              | Keep for first-auth only; prefer skeletons elsewhere.          |
| `Toast` (`useToast`)                       | success/error                                                      | Add `info`/`warning`; use for wishlist + availability nudges.  |
| `Navigation`                               | top nav, 3 links                                                   | Rebuild as scalable sidebar + mobile bottom-tabs (§1).         |
| `ErrorBoundary`                            | app-wide                                                           | Keep; pair with per-module error states.                       |

### 10.2 New components to design

| Component                              | Purpose                                    | Variants / Props                                                                                                                   | States                                                       |
| -------------------------------------- | ------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| **`AppShell`**                         | Sidebar + top bar + content region         | desktop / tablet-rail / mobile-tabs                                                                                                | —                                                            |
| **`SearchBar`**                        | Global item search                         | inline / overlay (mobile)                                                                                                          | idle / typing / results / no-results                         |
| **`DataFreshnessChip`**                | Data freshness and manual refresh affordance | —                                                                                                                                  | fresh / stale / refreshing / error                           |
| **`CharacterSwitcher`**                | Pick active Guardian                       | dropdown                                                                                                                           | loading / multi / single                                     |
| **`ItemCard`**                         | The atomic collectible tile                | density: grid/list/compact; rarity frame; `progress` variant (catalysts/crafting)                                                  | default / missing / available-now / owned / loading-skeleton |
| **`ItemDetailDrawer`**                 | Full item info + actions                   | drawer (desktop) / sheet (mobile)                                                                                                  | loading / loaded / error                                     |
| **`RarityFrame`**                      | Rarity border/glow wrapper                 | exotic/legendary/rare/uncommon/common                                                                                              | reduced-motion (no glow)                                     |
| **`Badge`**                            | Status vocabulary                          | rarity / difficulty / availability (avail-now, expiring, new, missing, completes-set) / completion (complete, in-progress, gilded) | —                                                            |
| **`ProgressBar`**                      | Linear objective/completion                | sizes; labeled/unlabeled                                                                                                           | 0 / partial / complete                                       |
| **`RadialProgress`**                   | Completion % (dashboard, seals)            | sizes                                                                                                                              | —                                                            |
| **`CategoryTree`**                     | Nested presentation-node nav               | expandable rows w/ %                                                                                                               | loading / expanded / collapsed                               |
| **`FilterBar`**                        | Filter + sort controls                     | inline (desktop) / bottom-sheet (mobile)                                                                                           | active-filters / cleared                                     |
| **`FilterChip`**                       | Toggle a filter value                      | selected / unselected                                                                                                              | —                                                            |
| **`CountdownChip`**                    | Time-to-reset / expiry                     | reset / vendor-leaves / catalyst                                                                                                   | live ticking                                                 |
| **`StatTile`**                         | Headline numbers (Total/Collected/Missing) | —                                                                                                                                  | loading                                                      |
| **`ActionList`**                       | Checkable "recommended this week"          | —                                                                                                                                  | empty / items / completed                                    |
| **`SealCard`**                         | Triumph/seal w/ radial %                   | gilded variant                                                                                                                     | locked / in-progress / complete / gilded                     |
| **`EmptyState`**                       | Reusable empty/error template              | success / filtered / wishlist / error-\* / warming                                                                                 | (per §9)                                                     |
| **`Skeleton`**                         | Layout-matched loaders                     | card / list-row / radial / detail                                                                                                  | shimmer / reduced-motion                                     |
| **`VendorModule` / `MilestoneModule`** | This-Week sections                         | Xûr (conditional) / vendor / milestone                                                                                             | present / absent / unavailable                               |

### 10.3 Component dependency map

```text
AppShell
├─ SearchBar · DataFreshnessChip · CharacterSwitcher
└─ <page>
   ├─ Dashboard      → RadialProgress, ProgressBar, StatTile, ActionList,
   │                    VendorModule, Badge, CountdownChip
   ├─ Collections    → CategoryTree, FilterBar(FilterChip), ItemCard(RarityFrame,
   │                    Badge), ItemDetailDrawer, StatTile, EmptyState, Skeleton
   ├─ This Week      → ActionList, VendorModule, MilestoneModule, CountdownChip, Badge
   ├─ Wishlist       → ItemCard(list), Badge(availability/priority), FilterChip, EmptyState
   ├─ Catalysts/Craft→ ItemCard(progress), ProgressBar, Badge, FilterChip
   └─ Triumphs/Seals → SealCard(RadialProgress), ProgressBar, Badge, EmptyState
```

---

## 11. Handoff Notes for High-Fidelity

1. **Start with the kit, not the screens** — nail `ItemCard`, `Badge`, `ProgressBar`/`RadialProgress`, and the `EmptyState`/`Skeleton` templates first; every screen composes from them.
2. **Badge vocabulary is the brand glue** — define the full set (rarity × difficulty × availability × completion) with color _and_ label/icon (never color alone — PRD §10).
3. **Personalization badges (`«Missing»`, `«Completes set»`, `«Avail now»`) are the product's differentiator** — make them prominent and consistent across Dashboard, This Week, Collections, and Wishlist.
4. **Replace placeholder emoji** (⚔️🛡️✨🎉📝) with a coherent icon set.
5. **Design the freshness/refresh model visibly** — it sets the expectation that Bungie data is cache-backed, not live.
6. Validate rarity colors for **WCAG AA on dark** and **color-blind** safety before locking the palette.
