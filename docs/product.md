# Product Direction

## Purpose

Guardian Tracker helps Destiny 2 players understand what is missing from their
collection and choose a useful next action. It turns account, manifest, and
time-limited game data into a focused plan rather than another inventory dump.

The product north star is:

> Open the app and quickly understand the best useful thing to do next.

This document owns durable product intent. It does not describe implementation
status or promise future features. See [README.md](../README.md) for the current
feature summary, [architecture.md](./architecture.md) for implemented behavior,
and [ROADMAP.md](../ROADMAP.md) for work that has not shipped.

## People We Serve

- **Completionists** need exact gaps, progress, and trustworthy acquisition
  guidance across a very large catalogue.
- **Returning players** need unfamiliar systems translated into clear choices
  without assuming recent game knowledge.
- **Time-limited players** need weekly and expiring opportunities prioritized
  so a short play session is useful.
- **Build-focused players** need collection and wishlist information organized
  around the item or outcome they are pursuing.

Guardian Tracker is primarily a desktop second-screen experience and should
remain effective on mobile for planning away from the game.

## Product Principles

### Recommend, do not merely catalogue

Raw ownership totals are context. The useful outcome is a short, explainable
set of actions based on missing items, wishlist intent, source, and verified
availability.

### Be precise about confidence

Difficulty is source-specific. Live availability represents verified signals,
not a universal claim that every item is obtainable. Best-effort or incomplete
data must be labelled or omitted rather than presented as certain.

### Preserve provenance

An item may have several collectibles and several acquisition sources. Product
surfaces should retain that complete source information and make the reason for
a recommendation understandable.

### Respect scope

Membership-wide collection data and character-specific live data are different
concepts. The interface should identify the selected Guardian where it affects
results and must not imply that account-wide ownership is character inventory.

### Make freshness visible

Bungie and manifest data are cached and time-sensitive. Loading, warming,
staleness, reset, unavailable-data, and reconnect states are normal product
states, not afterthoughts.

### Keep actions close to context

Players should be able to inspect an item, understand its sources, add it to a
wishlist, and follow a relevant recommendation without losing their place.

### Design for trust and access

Use clear language, keyboard-accessible controls, visible focus, sufficient
contrast, reduced-motion support, and labels that do not rely on color alone.
Never expose private account data or imply access the application does not have.

## Experience Priorities

1. Orient the player: whose data, how fresh, and what scope is being shown.
2. Show the highest-value next actions and why they matter.
3. Let the player explore collection gaps with strong filtering and search.
4. Preserve intent through wishlist notes, priorities, and preferences.
5. Provide graceful recovery when Bungie data, the manifest, or authorization
   is temporarily unavailable.

## Product Boundaries

- Guardian Tracker supplements Destiny 2; it does not automate gameplay.
- Bungie remains authoritative for account state and live game data.
- Recommendations must be derived from verified project data and explainable
  rules, not invented certainty.
- New external data sources require explicit licensing, freshness, privacy, and
  operational decisions before product design treats them as available.
