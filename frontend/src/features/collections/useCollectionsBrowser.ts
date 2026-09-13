import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useSearchParams } from "react-router";
import type {
  CollectionNodeView,
  CollectionsView,
} from "../../lib/collectionsView";
import { DIFFS, RARITIES, RARITY_RANK } from "../../lib/constants";
import type { Difficulty, GTItem, Rarity } from "../../types/design";
import { useItemView } from "../../data/items";

export type SortKey = "rarity" | "name" | "avail";

export interface CollectionsFilters {
  node: string;
  q: string;
  rarity: Rarity | null;
  diff: Difficulty | null;
  sort: SortKey;
  view: "grid" | "list";
  missing: boolean;
  avail: boolean;
  farm: boolean;
}

/** The fields a filter control may change. The category is `selectNode`'s. */
export type FilterPatch = Partial<Omit<CollectionsFilters, "node">>;

const STORAGE_KEY = "gt.collections.filters";
const MAX_Q_LENGTH = 100;

// Keys that live only in the URL: never persisted to localStorage, never
// read back from it, and re-asserted from the URL wherever stored/legacy
// state is merged in. `node` is the original member of this set; `q` (the
// in-page search term) joins it for the same reason — a lone `q` (or `node`)
// param must not be treated as "the user has filters set" and must not be
// overwritten by a hand-edited/legacy stored payload.
const URL_ONLY_KEYS = ["node", "q"] as const;
type UrlOnlyKey = (typeof URL_ONLY_KEYS)[number];

// Strips URL-only keys (see URL_ONLY_KEYS) from a full filter state — the
// single source of truth for "what gets persisted to localStorage". Adding a
// key to URL_ONLY_KEYS automatically excludes it here too.
function omitUrlOnly(
  f: CollectionsFilters,
): Omit<CollectionsFilters, UrlOnlyKey> {
  const rest: Partial<CollectionsFilters> = { ...f };
  for (const key of URL_ONLY_KEYS) {
    delete rest[key];
  }
  return rest as Omit<CollectionsFilters, UrlOnlyKey>;
}

// Overwrites `base`'s URL-only keys (see URL_ONLY_KEYS) with the values from
// `fromUrl` — the single source of truth for "stored/legacy filter state can
// never leak a URL-only field". Adding a key to URL_ONLY_KEYS automatically
// re-asserts it here too.
function reassertUrlOnly(
  base: CollectionsFilters,
  fromUrl: CollectionsFilters,
): CollectionsFilters {
  const out = { ...base };
  for (const key of URL_ONLY_KEYS) {
    out[key] = fromUrl[key];
  }
  return out;
}

// Keys that count as "filters" — presence of any of these in the URL means
// the URL is the source of truth and stored defaults are ignored. Keys in
// URL_ONLY_KEYS are deliberately excluded so a bare `?node=` or `?q=` deep
// link still applies the user's stored filter defaults.
const FILTER_KEYS = [
  "rarity",
  "diff",
  "sort",
  "view",
  "missing",
  "avail",
  "farm",
] as const;

// parseFilters reads a full filter state from URL params, applying defaults for
// absent keys. Pure — no storage/side effects.
export function parseFilters(p: URLSearchParams): CollectionsFilters {
  const rarity = p.get("rarity");
  const diff = p.get("diff");
  const sort = p.get("sort");
  return {
    node: p.get("node") ?? "",
    q: (p.get("q") ?? "").slice(0, MAX_Q_LENGTH),
    rarity:
      rarity && RARITIES.includes(rarity as Rarity) ? (rarity as Rarity) : null,
    diff:
      diff && DIFFS.includes(diff as Difficulty) ? (diff as Difficulty) : null,
    sort: sort === "name" || sort === "avail" ? sort : "rarity",
    view: p.get("view") === "list" ? "list" : "grid",
    missing: p.has("missing") ? p.get("missing") !== "0" : true,
    avail: p.get("avail") === "1",
    farm: p.get("farm") === "1",
  };
}

// serializeFilters writes only non-default values so URLs stay clean.
export function serializeFilters(f: CollectionsFilters): URLSearchParams {
  const p = new URLSearchParams();
  if (f.node) p.set("node", f.node);
  if (f.q) p.set("q", f.q.slice(0, MAX_Q_LENGTH));
  if (f.rarity) p.set("rarity", f.rarity);
  if (f.diff) p.set("diff", f.diff);
  if (f.sort !== "rarity") p.set("sort", f.sort);
  if (f.view !== "grid") p.set("view", f.view);
  if (!f.missing) p.set("missing", "0");
  if (f.avail) p.set("avail", "1");
  if (f.farm) p.set("farm", "1");
  return p;
}

function urlHasFilterParams(p: URLSearchParams): boolean {
  return FILTER_KEYS.some((k) => p.has(k));
}

/** Filter defaults the user persisted, already validated. */
export type StoredFilters = Partial<CollectionsFilters> | null;

// loadStoredFilters validates the persisted boundary instead of trusting a
// historical JSON shape. Removed values such as sort="difficulty" are omitted
// and therefore fall back to the current defaults; URL-only keys never escape
// this function.
function loadStoredFilters(): StoredFilters {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed))
      return null;
    const stored = parsed as Record<string, unknown>;
    const normalized: Partial<CollectionsFilters> = {};

    if (RARITIES.includes(stored.rarity as Rarity))
      normalized.rarity = stored.rarity as Rarity;
    else if (stored.rarity === null) normalized.rarity = null;
    if (DIFFS.includes(stored.diff as Difficulty))
      normalized.diff = stored.diff as Difficulty;
    else if (stored.diff === null) normalized.diff = null;
    if (
      stored.sort === "rarity" ||
      stored.sort === "name" ||
      stored.sort === "avail"
    )
      normalized.sort = stored.sort;
    if (stored.view === "grid" || stored.view === "list")
      normalized.view = stored.view;
    for (const key of ["missing", "avail", "farm"] as const) {
      if (typeof stored[key] === "boolean") normalized[key] = stored[key];
    }
    return normalized;
  } catch {
    return null;
  }
}

/**
 * The filters in force: the URL's when it carries filter params, otherwise the
 * stored defaults — with node/q always from the URL (see URL_ONLY_KEYS).
 */
export function effectiveFilters(
  params: URLSearchParams,
  stored: StoredFilters,
): CollectionsFilters {
  const fromUrl = parseFilters(params);
  if (urlHasFilterParams(params) || !stored) return fromUrl;
  return reassertUrlOnly({ ...fromUrl, ...stored }, fromUrl);
}

/** One navigation: the fields to change, how to navigate, and params to consume. */
export interface UrlWrite {
  patch: Partial<CollectionsFilters>;
  replace: boolean;
  drop?: readonly string[];
}

/**
 * The params after `write`. Params the browser does not own (e.g. `item=`)
 * are carried over unless the write consumes them.
 */
export function nextParams(
  prev: URLSearchParams,
  stored: StoredFilters,
  write: UrlWrite,
): URLSearchParams {
  const out = serializeFilters({
    ...effectiveFilters(prev, stored),
    ...write.patch,
  });
  for (const [k, v] of prev) {
    if (
      (URL_ONLY_KEYS as readonly string[]).includes(k) ||
      (FILTER_KEYS as readonly string[]).includes(k) ||
      write.drop?.includes(k)
    )
      continue;
    out.set(k, v);
  }
  return out;
}

/** What the user can ask the browser to do to the URL. */
export type Intent =
  | { type: "selectNode"; node: string }
  | { type: "setFilter"; patch: FilterPatch }
  | { type: "clearFilters" };

export function intentWrite(intent: Intent): UrlWrite {
  switch (intent.type) {
    // A push, so Back returns to the previous category.
    case "selectNode":
      return { patch: { node: intent.node }, replace: false };
    // Replaces: typing or toggling must not push a history entry per change.
    case "setFilter":
      return { patch: intent.patch, replace: true };
    case "clearFilters":
      return {
        patch: { rarity: null, diff: null, avail: false, farm: false, q: "" },
        replace: true,
      };
  }
}

/** What the URL needs once collections data is in hand. */
export type Reconciliation =
  /** `?item=` names a collectible: select its owning node and open it. */
  | { type: "openDeepLink"; item: GTItem; write: UrlWrite }
  /** `?item=` names something outside the tree: consume it and look it up. */
  | { type: "lookUpDeepLink"; hash: string; write: UrlWrite }
  /** No category, or one the tree no longer has: canonicalize to the first root. */
  | { type: "seedRoot"; write: UrlWrite };

/**
 * The single URL correction the data calls for, or null when the URL already
 * agrees with it. A pending deep link always wins over seeding, so the two can
 * never both write. Seeding is URL canonicalization, not a user navigation, so
 * it replaces rather than leaving a Back-button stop.
 */
export function reconcile(
  params: URLSearchParams,
  collections: CollectionsView | undefined,
): Reconciliation | null {
  if (!collections) return null;
  const itemParam = params.get("item");
  if (itemParam) {
    const item = collections.itemByHash(itemParam);
    const path = collections.pathToItem(itemParam);
    if (item && path) {
      const node = path[path.length - 1];
      return {
        type: "openDeepLink",
        item,
        write: {
          // A collected item is hidden under missing-only; reveal it.
          patch: item.collected ? { node, missing: false } : { node },
          replace: true,
          drop: ["item"],
        },
      };
    }
    return {
      type: "lookUpDeepLink",
      hash: itemParam,
      write: { patch: {}, replace: true, drop: ["item"] },
    };
  }
  const node = params.get("node") ?? "";
  if (collections.rootHashes.length > 0 && (!node || !collections.node(node))) {
    return {
      type: "seedRoot",
      write: { patch: { node: collections.rootHashes[0] }, replace: true },
    };
  }
  return null;
}

/** The selected category's items after every filter, in the chosen order. */
export function visibleItems(
  collections: CollectionsView | undefined,
  filters: CollectionsFilters,
): GTItem[] {
  if (!collections || !filters.node) return [];
  let list = collections.itemsUnder(filters.node);
  if (filters.missing) list = list.filter((i) => !i.collected);
  else list = list.slice();
  const { rarity, diff, avail, farm, sort } = filters;
  if (rarity) list = list.filter((i) => i.rarity === rarity);
  if (diff)
    list = list.filter((i) =>
      i.acquisitionSources.some((source) => source.difficulty === diff),
    );
  if (avail) list = list.filter((i) => i.availableNow);
  if (farm) list = list.filter((i) => !i.farmOnly);
  const term = filters.q.trim().toLowerCase();
  if (term) list = list.filter((i) => i.name.toLowerCase().includes(term));
  if (sort === "rarity")
    list.sort((a, b) => RARITY_RANK[a.rarity] - RARITY_RANK[b.rarity]);
  else if (sort === "name") list.sort((a, b) => a.name.localeCompare(b.name));
  else if (sort === "avail")
    list.sort((a, b) => (b.availableNow ? 1 : 0) - (a.availableNow ? 1 : 0));
  return list;
}

/**
 * The ancestors to open so the selected node is visible in the tree. Roots are
 * always visible, so a root selection reveals nothing.
 */
export function revealPath(
  collections: CollectionsView | undefined,
  node: string,
): string[] {
  const path = node ? collections?.pathToNode(node) : null;
  return path && path.length > 1 ? path.slice(0, -1) : NOTHING_TO_REVEAL;
}

const NOTHING_TO_REVEAL: string[] = [];

type Selection =
  { type: "item"; item: GTItem } | { type: "lookup"; hash: string } | null;

export interface CollectionsBrowserOptions {
  /** A deep-linked item turned out not to exist at all. */
  onItemUnavailable?: () => void;
}

/**
 * The Collections page's state: URL-held filters and category, the stored
 * filter defaults, `?item=` deep links, and the open detail drawer.
 *
 * It exports intents, not setters, and every URL change is computed from the
 * latest params rather than the render's snapshot — react-router's functional
 * updater would hand each same-tick write the same stale `prev`, so the second
 * silently undid the first. Here intents compose, however many run in a tick.
 */
export function useCollectionsBrowser(
  collections: CollectionsView | undefined,
  { onItemUnavailable }: CollectionsBrowserOptions = {},
) {
  const [searchParams, setSearchParams] = useSearchParams();

  const filters = useMemo(
    () => effectiveFilters(searchParams, loadStoredFilters()),
    [searchParams],
  );

  // Persist filter fields (not node/q — see URL_ONLY_KEYS) whenever they change.
  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(omitUrlOnly(filters)));
    } catch {
      /* ignore quota/availability errors */
    }
  }, [filters]);

  // The params every write builds on: the committed URL, advanced by any write
  // that has not rendered yet.
  const latest = useRef(searchParams);
  useLayoutEffect(() => {
    latest.current = searchParams;
  }, [searchParams]);

  const write = useCallback(
    (change: UrlWrite) => {
      const next = nextParams(latest.current, loadStoredFilters(), change);
      latest.current = next;
      setSearchParams(next, change.replace ? { replace: true } : undefined);
    },
    [setSearchParams],
  );

  const [selection, setSelection] = useState<Selection>(null);
  const lookup = useItemView(
    selection?.type === "lookup" ? selection.hash : null,
  );

  // The URL is the external system synchronized here; a deep link opens the
  // drawer in the same pass that consumes it.
  useEffect(() => {
    const step = reconcile(latest.current, collections);
    if (!step) return;
    write(step.write);
    if (step.type === "openDeepLink") {
      setSelection({ type: "item", item: step.item });
    } else if (step.type === "lookUpDeepLink") {
      setSelection({ type: "lookup", hash: step.hash });
    }
  }, [collections, searchParams, write]);

  // A failed lookup resolves to no detail by itself; report it once per deep
  // link, however often this re-runs.
  const reported = useRef<Selection>(null);
  useEffect(() => {
    if (selection?.type !== "lookup" || !lookup.isError) return;
    if (reported.current === selection) return;
    reported.current = selection;
    onItemUnavailable?.();
  }, [selection, lookup.isError, onItemUnavailable]);

  const items = useMemo(
    () => visibleItems(collections, filters),
    [collections, filters],
  );
  const expandPath = useMemo(
    () => revealPath(collections, filters.node),
    [collections, filters.node],
  );

  const activeNode: CollectionNodeView | null =
    (filters.node && collections?.node(filters.node)) || null;
  const detail =
    selection?.type === "item"
      ? selection.item
      : selection?.type === "lookup"
        ? (lookup.item ?? null)
        : null;
  // A whitespace-only search behaves as no search at all, so a stray space
  // doesn't hijack the "all caught up" empty state.
  const searching = filters.q.trim() !== "";

  return {
    filters,
    items,
    activeNode,
    expandPath,
    detail,
    searching,
    hasFilters: !!(
      filters.rarity ||
      filters.diff ||
      filters.avail ||
      filters.farm ||
      searching
    ),
    selectNode: (node: string) =>
      write(intentWrite({ type: "selectNode", node })),
    setFilter: (patch: FilterPatch) =>
      write(intentWrite({ type: "setFilter", patch })),
    clearFilters: () => write(intentWrite({ type: "clearFilters" })),
    openItem: (item: GTItem) => setSelection({ type: "item", item }),
    closeDetail: () => setSelection(null),
  };
}
