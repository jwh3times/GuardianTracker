# Guardian Tracker — Product Requirements Document (PRD)

> **Audience:** UX/UI design agent tasked with designing frontend improvements.
> **Purpose:** Give a complete picture of what Guardian Tracker does today, what it is planned to do, what the Bungie API makes possible, and the experience we want to build — so you can design screens, flows, components, and a coherent design system.
> **Scope note:** This document is product- and experience-focused. It is not the implementation source of truth. For current architecture, setup, shipped work, and remaining work, see `docs/architecture.md`, `SETUP.md`, `CHANGELOG.md`, and `ROADMAP.md`.

---

## 1. Product Vision

**Guardian Tracker is a Destiny 2 companion web app that helps players see what they're missing and decide what to chase next.**

Destiny 2 has thousands of collectible weapons, armor pieces, exotics, catalysts, cosmetics, and triumphs spread across raids, dungeons, vendors, seasonal activities, and limited-time events. The in-game Collections screen tells you _what you have_ but is poor at answering the questions players actually ask:

- "What am I still missing, and is it even obtainable right now?"
- "What's the most efficient thing to do this week to fill gaps in my collection?"
- "Which exotic catalysts / crafting patterns am I partway through?"
- "Is the god roll I want available from a vendor or activity this week?"

Guardian Tracker exists to answer those questions with **clarity, prioritization, and a plan of action** — turning a sprawling, opaque collection into a focused, motivating checklist.

**Design north star:** _"Open the app, and within five seconds know the single best thing to do today."_

---

## 2. Target Users & Personas

| Persona                            | Description                                                                            | Primary Goals                                                                                                           | What Frustrates Them                                                                    |
| ---------------------------------- | -------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| **The Completionist** ("Maya")     | Veteran player chasing 100% collections, seals, and titles. Thousands of hours played. | See exact gaps, track long-grind items (catalysts, patterns, raid exotics), measure % completion across every category. | No single view of "what's left." Has to cross-reference wikis.                          |
| **The Returning Player** ("Devon") | Played years ago, just came back. Overwhelmed by new systems.                          | Understand what they own, what changed, what's worth chasing now vs. what's sunset/unavailable.                         | Doesn't know what's still obtainable or where to start.                                 |
| **The Weekly Optimizer** ("Sam")   | Limited playtime (a few hours/week). Wants maximum reward per session.                 | A weekly to-do list: best vendor rolls, pinnacle sources, time-limited items expiring soon.                             | Wasting limited time on low-value activities; missing time-limited content.             |
| **The Min-Maxer** ("Riley")        | Cares about perfect rolls and meta loadouts.                                           | Track god rolls on owned weapons, find where to farm specific rolls, manage a curated wishlist.                         | Can't tell which of their copies has the best perks, or where the roll they want drops. |

**Platform context:** Players are on PC, Xbox, or PlayStation (Bungie cross-save). They'll use Guardian Tracker on **desktop (primary, often second-monitor while playing)** and **mobile (secondary, planning on the go)**. Design must be responsive and second-screen friendly.

---

## 3. Current Product State

### 3.1 Implemented & Working

- **Bungie OAuth login** — Players authorize via Bungie.net; the API stores Bungie tokens encrypted server-side and issues Guardian Tracker access/refresh tokens.
- **Dashboard** — Real account, collections, weekly, and wishlist queries power overall completion, per-category progress, character context, weekly preview, and wishlist availability.
- **Collections** — A Bungie presentation-node category tree shows weapons, armor, exotics, and cosmetics with collected/missing counts, item cards, filters, sorting, data freshness, and item detail drawers.
- **Wishlist** — User-scoped persisted wishlist CRUD with priority, notes, sorting, and availability badges.
- **This Week** — Real weekly milestones, Xur inventory, daily actions, reset countdowns, missing/wishlist flags, and ranked recommendations.
- **Catalysts, crafting, triumphs, and seals** — Real progress surfaces backed by Bungie records data.
- **Cosmetics gallery** — Dedicated visual browsing for emblems, shaders, ghost shells, ships, sparrows, and emotes.
- **Global search** — Manifest-backed search with deep links into item drawers.
- **Settings, roles, flags, admin, and audit log** — User preferences, early-access role opt-in, admin management, feature flags, and audit trail are implemented.
- **Design language** — Dark Destiny-flavored UI with rarity styling, `gt-*` classes, skeleton/error/empty states, toasts, and lazy-loaded routes.

### 3.2 Known Gaps and Product Debt

- **Difficulty and availability are qualified signals** — Difficulty is inferred from source text and can be `Unrated`; "available now" reflects reliable live vendor signals, not a universal obtainable/unobtainable truth.
- **Collection filters are not fully persistent yet** — Persisted filter state and more explicit obtainability filters remain roadmap work.
- **Wishlist availability can become broader** — Wishlist availability should reuse the broader reliable vendor availability map where appropriate.
- **Search index is in memory** — It rebuilds on startup and manifest swap; persistence is roadmap work.
- **Character switcher is mostly display context** — Collection data is account-wide; deeper character-scoped surfaces remain future work.
- **Some weekly facts are intentionally omitted** — Xur location and non-raid/dungeon missing counts are not shown unless a reliable Bungie data signal exists.
- **No E2E/a11y/visual regression suite yet** — Unit/integration coverage exists; browser-level regression coverage is roadmap work.

### 3.3 Current Navigation

Authenticated navigation includes **Dashboard**, **This Week**, **Collections**, **Cosmetics**, **Catalysts & Crafting**, **Triumphs & Seals**, **Wishlist**, **Settings**, and admin-only surfaces where applicable. Product/design work should preserve this broader information architecture unless deliberately proposing a replacement.

---

## 4. What the Bungie API Makes Possible

This is the menu of data the app can surface or already surfaces. Design with the full menu in mind and call out which data source each screen uses.

### 4.1 Already wired up

- **Collectibles (component 800)** — per-item collected/not-collected state, including "obscured" (hidden until found) and visibility flags. This powers the Collections page.
- **Manifest** — the full static game database: every item's name, description, icon (and high-res icon / animated icon sequences), rarity tier, item type/subtype, equipment slot, and the **source string** ("Drops from the Vault of Glass raid"). Updated automatically when Bungie patches.

### 4.2 Available but not yet used (high-value design opportunities)

- **Records / Triumphs (component 900)** — progress on every triumph, including **Seals/Titles** (with gilding), **exotic catalyst** completion, and challenge tracking. Each record has objectives with progress (e.g., "47/100 kills").
- **Presentation Nodes (700)** — Bungie's own **category tree** for Collections and Triumphs (the nested folders you see in-game: Weapons › Raids › Vault of Glass). Lets us mirror the game's organization and show per-node completion counts.
- **Vendors (400–402)** — live vendor inventories: **Xûr** (weekend exotics), **Banshee-44** (gunsmith weapons + mods), **Ada-1** (shaders/armor mods), **Saint-14**, **Tess** (Eververse), faction vendors, and their **current sale items, costs, and rotation**. This is what makes "what's worth buying this week" possible.
- **Public Milestones & Activities (204)** — the **weekly rotators**: featured raid/dungeon, Nightfall weapon, weekly challenges, and the rewards each offers. Powers "what to do this week."
- **Character data (200, 202, 205)** — characters (class, race, light/power level), subclasses, equipped loadouts, seasonal artifact, progression/ranks (Valor, Glory, Trials, vendor reputation).
- **Inventory & Item Instances (102, 201, 300–305)** — actual owned item instances with their **rolled perks, stats, masterwork, and sockets** — i.e., _which_ roll of a gun you have. Enables god-roll tracking and "you already have a better copy" insights.
- **Craftables (1300) & Deepsight** — weapon **crafting pattern progress** (red-border / Deepsight resonance: "3 of 5 patterns extracted"). Huge for completionists.
- **Metrics (1100)** — tracked lifetime stats.
- **Profile-level cosmetics** — the manifest + collectibles cover **shaders, emblems, ghost shells, ships, sparrows, finishers, emotes, transmog ornaments** — all collectible categories we don't yet surface.
- **Activity History / PGCR** — post-game reports; could support "you ran this raid but didn't get X."
- **Clan / Groups** — clan membership and roster.

### 4.3 Constraints the design must respect

- **Rate limits** — Bungie throttles requests; the app caches per-user collection data. Design should not imply real-time/live updates. A **"last refreshed" timestamp + manual refresh** model fits better than auto-polling.
- **Manifest load** — on a cold start the backend may be briefly unavailable (manifest download). Collections can return "not ready yet" (503). Design a graceful warming/empty state.
- **Privacy** — some players set collections/profile to private; the API may return restricted data. Design for a "this Guardian's data is private" state.
- **Auth/session expiry** — Bungie tokens expire; design clear re-auth prompts.
- **Time-limited data** — vendor and weekly data is only valid until the next reset; always pair it with an expiry/countdown.

---

## 5. Feature Requirements

Features are grouped by horizon. Each lists the **user value**, **data source** (from §4), and **design needs**. Priority tags: **P0** = core/now, **P1** = next, **P2** = later/aspirational.

### 5.1 Collections (Enhance) — P0

**Value:** The core loop — see what's missing and where to get it.
**Data:** Collectibles (800), Manifest, Presentation Nodes (700).
**Design needs:**

- Maintain a clean **collection summary header**: overall completion %, counts, and a subtle **"last updated · Refresh"** control.
- Expand beyond 3 tabs. Mirror Bungie's **presentation-node tree** (e.g., Weapons by source: Raids, Dungeons, Trials, Nightfall, World, Vendors, Seasonal; Armor by class; plus Exotics). Support nested category browsing with per-category completion counts.
- **Item card** redesign: rarity-framed, icon, name, type, source, difficulty badge, "obtainable now?" indicator, and quick actions (add to wishlist, view details).
- **Item detail** view (drawer or modal): full description, all sources, difficulty rationale, related activity, and — where available — the perks/rolls possible on the item.
- **Powerful filtering & sorting:** by rarity, item type, difficulty, source/activity, obtainable-vs-sunset, and **"missing only" vs "all."** Persist filter state per tab.
- **Search** within collections (P1 — backend search to be implemented).
- States: loading (skeletons), empty ("all caught up" celebration), error (auth vs. Bungie outage vs. private profile), manifest-warming.

### 5.2 Weekly Planner / "This Week" — P0

**Value:** The single most-requested optimizer feature — "what should I do this week?"
**Data:** Public Milestones, Vendors (400–402), Activities (204), cross-referenced with the player's missing items.
**Design needs:**

- A **"This Week" hub**: featured raid/dungeon, Nightfall weapon, weekly challenges, and time-limited vendor offerings — each with a **countdown to reset**.
- **Personalized highlighting:** flag vendor items / activity rewards that the player is **still missing** ("New for you" / "Completes a set"). This is the killer differentiator — join weekly data against the collection.
- **Xûr weekend module** (appears Fri–Tue): his exotic inventory with "you're missing this" badges.
- A prioritized, checkable **"recommended this week"** action list.
- States: pre-reset vs. post-reset, "Xûr not in town," vendor data unavailable.

### 5.3 Wishlist (Make Real) — P0

**Value:** Curate and prioritize chase items; get nudged when they're obtainable.
**Data:** persisted wishlist + cross-reference with vendor/weekly data.
**Design needs:**

- Persistence for items, **priority (Low/Med/High/Urgent)**, and **freeform notes**.
- **"Available now" surfacing:** when a wishlisted item is in a vendor's inventory or this week's activity rewards, highlight it on the wishlist and dashboard.
- Add-to-wishlist from anywhere (collections, item detail, search).
- Bulk management, sort by priority/date/availability, empty state that routes to Collections.

### 5.4 Catalyst & Crafting Tracker — P1

**Value:** Completionists' biggest long-grind blind spot.
**Data:** Records (900) for catalysts, Craftables (1300) for patterns/Deepsight.
**Design needs:**

- **Exotic catalyst progress**: which catalysts are missing, dropped-but-incomplete (with objective progress bars, e.g., "320/500 kills"), or complete.
- **Crafting pattern progress**: red-border weapons, "patterns extracted X/Y," which weapons are craftable, where to farm Deepsight copies.
- Progress-forward visual language (objective bars everywhere).

### 5.5 Triumphs, Seals & Titles — P1

**Value:** Track prestige goals and title completion.
**Data:** Records (900), Presentation Nodes (700).
**Design needs:**

- Seal/title gallery with **completion % and gilding state**; expandable triumph breakdowns with objective progress; "closest to complete" surfacing.

### 5.6 Cosmetics Collections — P1

**Value:** Completionists & fashion players want shaders, emblems, ghosts, ships, sparrows, emotes, ornaments tracked too.
**Data:** Collectibles (800) + Manifest (categories already exist).
**Design needs:** Visual-forward galleries (cosmetics are about looks) with collected/missing and source.

### 5.7 Character & Loadout Overview — P2

**Value:** See your Guardians at a glance; context for what you own.
**Data:** Characters (200), Equipment (205), Progressions (202).
**Design needs:** Character switcher, power level, equipped loadout, seasonal artifact, reputation/rank progress.

### 5.8 God-Roll / Owned-Roll Insights — P2

**Value:** Min-maxers want to know which copy is best and where to farm the roll they want.
**Data:** Item Instances + Sockets/Perks (300–305), community wishlists.
**Design needs:** Per-weapon owned-rolls list with perks, "god roll" matching, and farm-source guidance.

### 5.9 Account-wide niceties — P1/P2

- **Multi-character support** (collections are mostly account-wide, but some data is per-character).
- **Onboarding** for returning/new players ("here's what changed, here's where to start").
- **Notifications/digests** (P2): "Xûr has an exotic you're missing," "wishlisted roll available this week."
- **Shareable** collection progress (P2).

---

## 6. Proposed Information Architecture

A scalable nav to replace the current 3-page structure. Propose/refine in your designs:

```text
Guardian Tracker
├── Dashboard            "At a glance": completion %, this-week highlights,
│                         wishlist availability, character summary
├── This Week            Weekly planner — milestones, vendors, Xûr, recommended actions
├── Collections          Browsable category tree (weapons/armor/exotics/cosmetics)
│     └── Item Detail    Drawer/modal: sources, difficulty, rolls, actions
├── Catalysts & Crafting Catalyst progress + crafting patterns / Deepsight
├── Triumphs & Seals     Titles, seals, triumph progress
├── Wishlist             Prioritized chase list + availability alerts
└── (Profile/Settings)   Account, characters, refresh, logout, data freshness
```

Secondary patterns to design:

- **Global search** (items across all collections).
- **Global filters** that read consistently across Collections / Wishlist / This Week.
- A persistent **"data freshness + refresh"** affordance.
- A **character switcher** in the header.

---

## 7. Screen-by-Screen UX Requirements

For each screen, design **all states**: loading (prefer skeletons over spinners), empty, error (auth-expired / Bungie-down / private-profile / manifest-warming), and populated.

### 7.1 Login

- Single clear **"Sign in with Bungie"** CTA; explain what the app does and what it accesses (read-only collection data). Trust-building, low-friction. Handle OAuth callback with a graceful "completing sign-in…" state and error recovery.

### 7.2 Dashboard

- **Hero summary:** overall collection completion %, plus per-category mini-stats. Make the headline answer "how complete am I?" instantly.
- **"Best thing to do today"** callout (driven by This Week + wishlist availability).
- **This-week preview:** featured activity, Xûr-if-present, expiring items, with reset countdown.
- **Wishlist availability strip:** "3 wishlisted items available now."
- **Character summary** (P2). Quick actions. Keep weekly/vendor previews tied to real data and graceful empty states.

### 7.3 Collections

- Category tree navigation (replace flat 3-tab where it doesn't scale; keep fast top-level switching).
- Summary header (completion %, counts, last-updated + refresh).
- Robust filter/sort bar; **missing-only toggle**; difficulty filter (keep, but clarify it's an estimate).
- Responsive item-card grid; rarity framing; obtainable-now indicator; add-to-wishlist; open detail.
- **Item detail** drawer/modal.
- Celebration empty state ("All caught up!") and per-filter empty state.

### 7.4 This Week (new)

- Sectioned: **Milestones/Activities**, **Vendors**, **Xûr** (conditional), **Recommended for you**.
- Every time-limited element shows a **countdown/expiry**.
- Personalized "missing"/"completes a set" badges throughout.

### 7.5 Wishlist

- Priority-grouped list with quick priority edit and notes.
- **Availability badges** (in a vendor / dropping this week).
- Sort by priority/date/availability; bulk actions; routed empty state.

### 7.6 Catalysts & Crafting (new)

- Two sub-views (Catalysts, Crafting Patterns). Progress bars front-and-center; filter by complete/in-progress/missing; source guidance.

### 7.7 Triumphs & Seals (new)

- Seal/title gallery with completion %, gilding; drill-down to triumph objectives; "closest to done" sort.

### 7.8 Settings/Profile

- Account info, character list, **data freshness & manual refresh**, sign out, privacy notes, theme (if offered).

---

## 8. Design System & Visual Language

Guardian Tracker should feel like a **native extension of Destiny 2** — sci-fi, premium, dark, with the rarity-color language players already know — while being cleaner and more information-dense than the in-game UI.

### 8.1 Foundations

- **Dark-first theme** (already in place). Consider an optional light mode but prioritize dark.
- **Rarity color system** (core to the brand — use consistently for borders, accents, glows):
  - Exotic — gold/yellow (with subtle glow for the most prestigious items)
  - Legendary — purple
  - Rare — blue
  - Uncommon — green
  - Common — white/gray
- **Difficulty system** (acquisition effort): Easy / Moderate / Challenging — give each a distinct, accessible color + label (don't rely on color alone).
- **Status/availability accents:** "available now," "expiring soon," "new for you," "complete" — define a small, consistent badge vocabulary.

### 8.2 Iconography & imagery

- Lean on Bungie's rich **item icons / high-res icons / animated icon sequences** (exotics have animated frames) and item screenshots where available. Cards are image-forward.
- Replace placeholder emoji (⚔️🛡️✨🎉📝 currently used as tab/section icons) with a coherent icon set.

### 8.3 Components to define

A reusable kit (some primitives exist: Button, Card, LoadingSpinner, Toast):

- Item card (multiple densities: grid, list, compact), rarity-framed.
- Item detail drawer/modal.
- **Progress bar / radial progress** (used pervasively: completion %, catalyst/crafting/triumph objectives).
- Category tree / nested nav.
- Filter & sort bar; filter chips.
- Countdown/timer chip.
- Badge set (availability, difficulty, rarity, completion).
- Stat tiles.
- Character switcher.
- **Skeleton loaders** for cards, lists, and detail.
- Data-freshness indicator + refresh control.
- Empty, error, and "warming up" state illustrations/messages.

### 8.4 Data density & hierarchy

- Players scan **lots** of items. Favor scannable grids, strong visual grouping, and progressive disclosure (summary → detail). Avoid overwhelming first paint; lead with the headline number and the "do this next" action.

---

## 9. Interaction, States & Content Guidelines

- **Freshness model:** show "last updated X ago" + manual **Refresh**; never imply live data. Cache-backed; refresh re-fetches.
- **Loading:** skeletons that match final layout; avoid full-page spinners except on first auth.
- **Empty states:** make them motivating and actionable (celebration for 100%, routing CTAs otherwise).
- **Error states (design distinct treatments):**
  - _Session expired_ → clear re-auth CTA.
  - _Bungie API down/maintenance_ → reassuring "try again shortly."
  - _Private profile_ → explain the player must make collections public, with a help link.
  - _Manifest warming_ (cold start) → "Getting the latest game data…" with retry.
- **Time-limited content** always paired with a countdown and a clear reset reference (weekly reset = Tuesday 17:00 UTC; a countdown helper already exists).
- **Voice & tone:** knowledgeable Destiny companion — uses correct in-game terminology (Guardian, Triumphs, Catalysts, Pinnacle, Deepsight, Seals) but stays approachable for returning players.

---

## 10. Accessibility & Responsive Requirements

- **Don't rely on color alone** for rarity/difficulty/status — pair with labels, icons, or patterns (critical given the heavy color-coding and color-blind players in the community).
- WCAG AA contrast on the dark theme (rarity colors on dark backgrounds need checking).
- Full keyboard navigation; visible focus states; semantic structure; screen-reader labels for icon-only controls.
- Respect reduced-motion (the exotic glow/animated icons should degrade gracefully).
- **Responsive breakpoints:** mobile (planning on the go), tablet, desktop (primary, second-monitor). Grids reflow; nav collapses sensibly; touch targets sized for mobile.
- Lazy-loaded images (large icon sets) and code-split routes (already in place) — keep performance budgets in mind for image-heavy galleries.

---

## 11. Success Metrics (what good design should move)

- **Time-to-value:** seconds from login to "I know what to do" (target: < 5s to a clear next action).
- **Weekly engagement:** return visits around weekly reset and Xûr's arrival.
- **Wishlist usage:** items added; wishlist-availability alerts acted on.
- **Collection completion lift:** players close gaps over time (the app demonstrably helps).
- **Task completion:** can a returning player understand their account state within one session?

---

## 12. Roadmap / Prioritization Summary

Completed product work is tracked in `CHANGELOG.md`; current remaining work is
tracked in `ROADMAP.md`. At a product level, future work clusters into:

| Horizon | Theme | Examples |
| --- | --- | --- |
| Near-term polish | Make shipped surfaces more durable and ergonomic | Persisted collection filters, broader wishlist availability, search-index persistence |
| Quality gates | Improve confidence before broader release | E2E tests, accessibility checks, visual regression |
| Account depth | Use more verified Bungie data | Character-scoped surfaces, owned-roll/god-roll insights |
| Sharing and re-engagement | Expand beyond single-session use | Notifications/digests, shareable collection progress |

Each item should be treated as gated until its data assumptions, security model,
and test plan are explicit.

---

## 13. Open Questions for Design

1. **Navigation scale:** top-nav vs. sidebar as feature count grows (7+ sections)? Recommend a pattern that scales to Phase 2/3.
2. **Collections organization:** mirror Bungie's exact presentation-node tree, or impose a cleaner Guardian Tracker taxonomy (by source/activity)? Trade-off: familiarity vs. clarity.
3. **Personalization prominence:** how aggressively should "missing/for-you" badges drive the UI vs. neutral browsing?
4. **Mobile depth:** full feature parity on mobile, or a focused "planning" subset?
5. **Difficulty rating credibility:** it's an estimate from source text — how should the UI communicate confidence without undermining trust?
6. **Density preference:** how much should default to summary-first with progressive disclosure vs. dense dashboards for power users?

---

### Appendix A — Glossary (for terminology accuracy in copy)

- **Guardian** — the player's character/account persona.
- **Collectible / Collection** — a record of having acquired an item; the in-game Collections screen.
- **Triumph / Record** — an achievement with objectives; some grant **Seals/Titles**.
- **Seal / Title** — prestige completion of a triumph set; can be **gilded** (re-earned).
- **Catalyst** — an upgrade for an exotic weapon, often with a kill/objective grind.
- **Deepsight / Crafting Pattern** — red-border weapons grant **patterns**; collect enough to **craft** a weapon with chosen perks.
- **Pinnacle / Powerful** — high-tier reward sources that raise power level.
- **Xûr** — weekend-only vendor (Fri–Tue) selling exotics.
- **Weekly Reset** — Tuesday 17:00 UTC; rotates activities, vendors, challenges.
- **Manifest** — Bungie's static database of all item/activity definitions.
- **Presentation Node** — a category/folder in the Collections/Triumphs tree.
