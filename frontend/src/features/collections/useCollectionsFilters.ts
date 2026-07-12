import { useCallback, useEffect, useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import type { Difficulty, Rarity } from "../../types/design";
import { DIFFS, RARITIES } from "../../lib/constants";

export type SortKey = "rarity" | "name" | "difficulty" | "avail";

export interface CollectionsFilters {
  node: string;
  rarity: Rarity | null;
  diff: Difficulty | null;
  sort: SortKey;
  view: "grid" | "list";
  missing: boolean;
  avail: boolean;
  farm: boolean;
}

const STORAGE_KEY = "gt.collections.filters";
// Keys that count as "filters" (node is tracked separately).
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
    rarity:
      rarity && RARITIES.includes(rarity as Rarity) ? (rarity as Rarity) : null,
    diff:
      diff && DIFFS.includes(diff as Difficulty) ? (diff as Difficulty) : null,
    sort:
      sort === "name" || sort === "difficulty" || sort === "avail"
        ? sort
        : "rarity",
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

// loadStoredFilters returns saved filter fields (never node), or null.
function loadStoredFilters(): Partial<CollectionsFilters> | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as Partial<CollectionsFilters>) : null;
  } catch {
    return null;
  }
}

export function useCollectionsFilters() {
  const [searchParams, setSearchParams] = useSearchParams();

  // Effective filters: from the URL when it carries filter params; otherwise
  // filter fields from localStorage (node always from the URL).
  const filters = useMemo<CollectionsFilters>(() => {
    const fromUrl = parseFilters(searchParams);
    if (urlHasFilterParams(searchParams)) return fromUrl;
    const stored = loadStoredFilters();
    return stored ? { ...fromUrl, ...stored, node: fromUrl.node } : fromUrl;
  }, [searchParams]);

  // Persist filter fields (not node) whenever they change.
  useEffect(() => {
    const { node: _node, ...rest } = filters;
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(rest));
    } catch {
      /* ignore quota/availability errors */
    }
  }, [filters]);

  // write applies a patch to the current URL params and preserves any
  // non-filter/non-node params already present (e.g. item=). Note: react-router's
  // functional updater snapshots `prev` from the current render, not a live ref —
  // so two `write` calls in the same synchronous tick each start from the same
  // stale snapshot and the second silently clobbers the first. Callers that need
  // to change multiple fields together (e.g. the deep-link effect setting node +
  // missing) MUST go through `setFilters`, which patches all fields in a single
  // `write` call, not sequential single-field setter calls.
  const write = useCallback(
    (patch: Partial<CollectionsFilters>, opts?: { replace?: boolean }) => {
      setSearchParams((prev) => {
        const base = urlHasFilterParams(prev)
          ? parseFilters(prev)
          : { ...parseFilters(prev), ...(loadStoredFilters() ?? {}) };
        const next = { ...base, ...patch };
        const out = serializeFilters(next);
        for (const [k, v] of prev) {
          if (k === "node" || (FILTER_KEYS as readonly string[]).includes(k))
            continue;
          out.set(k, v);
        }
        return out;
      }, opts);
    },
    [setSearchParams],
  );

  return {
    ...filters,
    // Atomic multi-field setter — routes through a single `write` call so all
    // patched fields land together (see note on `write` above). Defaults to a
    // replace-style navigation (e.g. deep-link restore) unless told otherwise.
    setFilters: (
      patch: Partial<CollectionsFilters>,
      opts: { replace?: boolean } = { replace: true },
    ) => write(patch, opts),
    setNode: (node: string) => write({ node }), // push (Back returns to prev category)
    setRarity: (rarity: Rarity | null) => write({ rarity }, { replace: true }),
    setDiff: (diff: Difficulty | null) => write({ diff }, { replace: true }),
    setSort: (sort: SortKey) => write({ sort }, { replace: true }),
    setView: (view: "grid" | "list") => write({ view }, { replace: true }),
    setMissing: (missing: boolean) => write({ missing }, { replace: true }),
    setAvail: (avail: boolean) => write({ avail }, { replace: true }),
    setFarm: (farm: boolean) => write({ farm }, { replace: true }),
    clearFilters: () =>
      write(
        { rarity: null, diff: null, avail: false, farm: false },
        { replace: true },
      ),
    hasFilters: !!(
      filters.rarity ||
      filters.diff ||
      filters.avail ||
      filters.farm
    ),
  };
}
