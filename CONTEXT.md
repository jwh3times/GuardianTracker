# CONTEXT

The project's shared vocabulary. When code, a test name, an issue title, a
commit message, or an agent's output names one of these concepts, it uses the
term as defined here rather than a synonym.

This is a glossary, not an architecture document — it says what a word means and
which module owns it. For how the pieces fit together see
[docs/architecture.md](./docs/architecture.md); for why a shape was chosen see
[docs/adr/](./docs/adr/).

**Adding a term.** A name earns a place here once it is used in more than one
module, or once two names for it have appeared in the codebase. Record the
canonical owner alongside the definition so there is somewhere to look when the
definition is not enough.

---

## Destiny concepts

Bungie's vocabulary. These words mean what they mean in the game and in the
Bungie API; we do not redefine them.

**Manifest** — Bungie's content database: every item, activity, vendor, and
collectible definition, published as a versioned SQLite file. Downloaded and
served by `services/manifest`, refreshed by the swap described in
[ADR 0010](./docs/adr/0010-manifest-swap-participants-and-observers.md).

**Collectible** — a manifest entry that links an inventory item to Bungie's
acquisition state. More than one collectible can refer to the same item, so a
collectible is a tracking record rather than the unit Guardian Tracker counts.

**Item** — a player-facing acquisition target linked from one or more
collectibles. The unit Guardian Tracker displays and counts; an item is owned
when any of its linked collectibles is acquired.

**Presentation node** — the manifest's tree structure that organises
collectibles into categories and subcategories. The Collections page renders it.
Our own name for the assembled result is the _collection tree_.

**Source string** — the manifest's human-readable text saying where an item
comes from ("Vault of Glass raid", "Complete Nightfall strikes"). It is prose,
not a structured field, which is why classifying it needs a keyword vocabulary.
Owned by `services/sources`.

**Milestone** — a Bungie-defined weekly or repeatable objective (a featured
raid, a Nightfall). Surfaced on This Week.

**Vendor** — an in-game merchant with a rotating inventory. **Xûr** is the
weekend-only exotic vendor and is the one vendor with bespoke handling, because
his location and stock change weekly and are only fully readable through an
authenticated character-scoped call.

**Record** — a manifest-tracked achievement. A **Triumph** is a record shown to
the player; a **Seal** is a named collection of triumphs awarding a title.
Served by `services/records`, rendered by `Triumphs.tsx`.

**Catalyst** — an upgrade for an exotic weapon, tracked like a collectible.
**Craft pattern** — the progress required to unlock crafting a weapon. Both
served by `services/records`, rendered by `Catalysts.tsx`.

**Bungie account** — the identity authorized through Bungie. It can expose more
than one Destiny membership and is not itself a Guardian Tracker user or a
character.

**Destiny membership** — one platform-specific Destiny identity belonging to a
Bungie account. Guardian Tracker tracks one membership for each user, preferring
Bungie's cross-save primary when one exists, and uses its membership ID and
membership type when requesting membership-wide game data.

**Profile** — Bungie's gameplay-state data for a Destiny membership. A profile
is data about a membership, not a synonym for the player or their identity.

**Character** — one playable avatar belonging to a Destiny membership, up to
three per membership. **Guardian** is the user-facing name for a character; do
not use it for the Bungie account, Destiny membership, or Guardian Tracker user.
Collections are **membership-wide**; vendor inventory, time-sensitive actions,
and Xûr's location can be **character-scoped**.

---

## Guardian Tracker concepts

Our vocabulary — words that describe what this app does with Bungie's data.

**Guardian Tracker user** — the app identity associated with one Destiny
membership. Its user-data record owns app-authored state such as sessions,
preferences, role tier, and wish list when persistence is available; do not use
_account_, _profile_, _player_, or _Guardian_ as synonyms.

**Missing item** — an item for which the player has acquired none of its linked
collectibles. The app's central noun: the whole product is "what are you
missing, and what should you do about it".

**Difficulty tier** — how hard a missing item is to acquire, derived from its
source string: `Challenging`, `Moderate`, `Easy`, or `Unrated`. `Unrated` means
no keyword matched, and is deliberately honest rather than a default guess.
Owned by `services/sources`; do not call this a "rating" or a "score".

**Raid/dungeon loot** — a source facet, not a tier. It marks loot dropping from
a named raid or dungeon, and gates the per-milestone missing count. Kept
separate from the tier because "Grandmaster Nightfall" run inside a dungeon is
`Challenging` on its own keyword yet still dungeon loot. `sources.IsRaidOrDungeon`.

**Availability / available now** — an item is _available now_ when a live vendor
is currently selling it. Distinct from difficulty: an item can be `Challenging`
and available this weekend. Carried as `availableNow` (item hash → vendor name)
on the collections payload.

**Farm-only item** — a random-perk item that cannot be reacquired from the
in-game Collections archive; another copy must be earned from its source. This
is an acquisition facet, independent of whether the player already owns one.

**Efficiency ranking** — scoring which activity buckets would close the most
missing items, so recommendations are ordered by payoff rather than by
Bungie's ordering. `services/efficiency`: `Rank` and `MissingForMilestone`.

**Acquisition recommendation** — a personalized action on This Week, ranked by
its payoff against missing items, the wish list, live availability, and featured
sources. Distinct from a _today action_, whose defining property is urgency.

**Today action** — a time-sensitive entry in Do This Today. It can expire at a
daily reset, weekly reset, or Xûr's departure; do not call it a _daily action_.

**Wish list** — the player's own saved set of wanted items. User-scoped and
persisted; it is the one collection the player authors rather than earns.
Two words, lowercase, in prose; `wishlist` as one word in code and routes.

**This Week** — the page combining weekly milestones, Xûr, and personalized
acquisition recommendations. **Do This Today** — the Dashboard's short list of
today actions. Both are served by `services/weekly`.

**Cosmetics** — shaders, ornaments, emblems and the like, browsed as a gallery
rather than as a tree. Same collectibles, different presentation.

---

## Architectural names

Names for seams, introduced deliberately and each with exactly one owner. These
are the ones most likely to drift, because a second name for a seam looks
harmless until the two halves disagree.

**Swap participant** (`bungie.SwapParticipant`) — a module holding an OS handle
on the manifest file, which must close it before the rename. **Manifest
observer** (`bungie.ManifestObserver`) — a module holding manifest-derived state,
which is notified only when a version genuinely changed. Two interfaces, not
one; observers do not fire on the rollback path. See
[ADR 0010](./docs/adr/0010-manifest-swap-participants-and-observers.md).

**Provider** (`manifest.Provider`) — the lazily-opened, swap-aware handle on the
manifest database. Returns `manifest.ErrNotReady` while the file is downloading
or mid-swap; that is a 503, not an error.

**Missing item reader** (`weekly.MissingItemReader`) — the one method
`services/weekly` uses from `services/collections`: the user's missing-item
hashes. Consumer-side, so `weekly` does not import `collections` at all, and
difficulty classification goes to `services/sources` directly rather than
through `collections.ClassifyDifficulty`. Required, never nil — a reader that
degraded to an empty set would report a complete collection.

**CollectionsView / CollectionsSummaryView** (`lib/collectionsView.ts`) — the
adapted collections payload the frontend reads. Two types because the endpoint
has two shapes: `CollectionsSummaryView` carries tree counts and the summary,
`CollectionsView` extends it with items. Asking the summary view for items is a
compile error, deliberately. No feature module handles the raw wire shape.

**AppProviders / AuthedProviders** (`contexts/AppProviders.tsx`) — the context
tower, split at the auth gate. `AppProviders` is what every page sits inside;
`AuthedProviders` adds Flags and Character below the gate. The split is
load-bearing, which is why it is two components rather than one with a flag.

**Composition root** — `main.go`, and only `main.go`: it loads configuration,
constructs services and handlers, declares swap participation, and serves. It
holds no logic of its own. The route table it hands to `api.NewRouter` lives in
`api/router.go`. **Authenticated group** (`authed`) — the one place the JWT gate
is applied; every authenticated route registers on it rather than naming the
middleware itself. See
[ADR 0011](./docs/adr/0011-route-table-as-a-testable-composition-root.md).

**Load-through** (`cache.Load` / `cache.LoadIf`) — the way a cached value is
read: hand it a key, a TTL and a loader, and get the value. An error is never
cached, a wrong-typed entry is a logged miss rather than a silent one, and
`LoadIf`'s predicate is where "do not cache an empty result" lives. Reach for
`cache.Cache` directly only to evict, or when the TTL depends on the value you
just loaded. `services/items` has its own bounded-map equivalent because it is
keyed by item hash with a size cap and no TTL.

**Degraded mode** — running without a database. Not a `nil`: `db.NewStores(nil)`
returns real implementations whose every method reports `db.ErrUnavailable`, and
`handlers.HandleStoreError` maps that to one 503. Ask `Stores.Available()` when
you genuinely need to know whether persistence exists. A degraded read never
returns empty success. Consumers that must not import `db` see the same
condition as their own sentinel — `auth.ErrUnavailable` — translated at
`db/adapters`. "There is no database" is not "the write failed": conflating them
is what made login 500 without Postgres.

**Role tier** — `standard < beta < alpha < admin`, ordered integers persisted on
the user row. Say _tier_, not "level" or "permission". **Feature flag** — a
rollout control, gated per role, and explicitly _not_ a security boundary; an
absent database must not hide pages. See
[ADR 0006](./docs/adr/0006-roles-feature-flags-and-admin-authorization.md).

**Session issuer** (`auth.SessionIssuer`) — the owner of the browser session
lifecycle: `AuthorizeURL`, `Login`, `Refresh`, `EndSession`, `EndAllSessions`.
It decides what a session is; the Gin handlers decide only what that looks like
over HTTP. Failures cross the seam as `auth.SessionError`, whose `Reason` is
also the audit reason string, so the two cannot drift. See
[ADR 0012](./docs/adr/0012-session-issuance-owns-the-session-lifecycle.md).

**Access token / refresh token / session** — the access JWT is short-lived and
lives in localStorage; the refresh JWT rotates per device and lives only in a
host-only HttpOnly cookie; the _session_ is the server-side row those rotate
against, with revocation and reuse detection. Bungie's own OAuth tokens are
authorized by the Bungie account and stored by Guardian Tracker against the
tracked Destiny membership. They are called _Bungie tokens_, never just
"tokens". See
[ADR 0002](./docs/adr/0002-bungie-oauth-and-token-storage.md) and
[ADR 0008](./docs/adr/0008-browser-refresh-cookie.md).

---

## Naming rules

- **One name per seam.** If an interface exists for a concept, every consumer
  declaring its own version uses the same name. Two names for one method set is
  how the halves drift apart.
- **Silent-empty is this codebase's recurring failure mode.** "There is
  genuinely nothing" and "you asked the wrong thing, or something broke" must be
  distinguishable. Prefer a shape that fails loudly or fails to compile over one
  that returns an empty value; it is worth a type or an error sentinel to get it.
- **Don't default a classification.** `Unrated`, `ErrNotReady` and
  `ErrUnavailable` all exist so an unknown stays visibly unknown instead of
  being rounded to a plausible-looking answer.
