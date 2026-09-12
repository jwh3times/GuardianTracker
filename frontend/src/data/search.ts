import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "../lib/api";
import { toRarity } from "../lib/rarity";
import type { APISearchResult } from "../types/api";
import type { Rarity } from "../types/design";

/**
 * The Search data-access module (ADR 0020). It owns this resource's query
 * identity, endpoint path, minimum query length, and projection to
 * {@link SearchResult}.
 *
 * Read by the global search bar in the AppShell. Debouncing the input and
 * capping how many results are shown are presentation, and stay there.
 *
 * There is deliberately no invalidation entry point. Search reads the manifest
 * index, not membership data, so a membership refresh has nothing to re-fetch
 * here. Item perks and item views are a separate resource (`data/items.ts`).
 */

/** Terms shorter than this are not sent to the server. */
export const MIN_SEARCH_LENGTH = 2;

/** One item matching a search term. */
export interface SearchResult {
  /** The item hash, as the `?item=` deep link expects it. */
  id: string;
  name: string;
  icon: string;
  type: string;
  rarity: Rarity;
}

/** Private: the search text is a genuine query parameter, so it is part of the key. */
function searchKey(term: string) {
  return ["search", term] as const;
}

function toSearchResult(r: APISearchResult): SearchResult {
  return {
    id: String(r.hash),
    name: r.name,
    icon: r.icon,
    type: r.type,
    rarity: toRarity(r.rarity),
  };
}

/**
 * The query's `select`. Stable module-level reference so React Query's
 * per-observer `select` memoisation holds.
 */
function toSearchResults(rows: APISearchResult[]): SearchResult[] {
  return rows.map(toSearchResult);
}

/** Stable empty result, so `results` keeps referential identity. */
const NO_RESULTS: SearchResult[] = [];

/**
 * Items matching `term`, up to twenty. Nothing is requested until the term
 * reaches {@link MIN_SEARCH_LENGTH}. Results are reused for thirty seconds, so
 * retyping a recent term does not ask the server again.
 */
export function useItemSearch(term: string) {
  const { data, isLoading, isError } = useQuery({
    queryKey: searchKey(term),
    queryFn: () =>
      apiFetch<APISearchResult[]>(
        `/api/items/search?q=${encodeURIComponent(term)}&limit=20`,
      ),
    enabled: term.length >= MIN_SEARCH_LENGTH,
    staleTime: 30_000,
    select: toSearchResults,
  });

  return { results: data ?? NO_RESULTS, isLoading, isError };
}
