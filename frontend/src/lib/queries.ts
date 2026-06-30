import { apiFetch } from "./api";
import type {
  APIItemPerks,
  APIItemView,
  APIUserCollections,
} from "../types/api";

/**
 * Shared collections query definition. Dashboard, Settings, and the Collections
 * page all read the same endpoint; routing them through one helper keeps their
 * React Query keys consistent so they share a single cache entry per
 * (membership, includeAll) variant instead of each firing its own request.
 *
 * `includeAll` controls both the cache-key suffix and the `?include=all` param,
 * so the "missing" and "all" variants never collide.
 */
export function collectionsQuery(
  membershipType: number | undefined,
  membershipId: string | undefined,
  includeAll = false,
) {
  return {
    queryKey: [
      "collections",
      membershipType,
      membershipId,
      includeAll ? "all" : "missing",
    ] as const,
    queryFn: () =>
      apiFetch<APIUserCollections>(
        `/api/collections/${membershipType}/${membershipId}${includeAll ? "?include=all" : ""}`,
      ),
    enabled: membershipType != null && !!membershipId,
  };
}

/**
 * Per-item perk pool, fetched lazily when the item drawer opens. Perk pools are
 * static for a manifest version, so this never goes stale within a session.
 * `enabled` guards against a null/closed drawer (undefined hash).
 */
export function itemPerksQuery(itemHash: string | undefined) {
  return {
    queryKey: ["item-perks", itemHash] as const,
    queryFn: () => apiFetch<APIItemPerks>(`/api/items/${itemHash}/perks`),
    enabled: !!itemHash,
    staleTime: Infinity,
  };
}

/**
 * Minimal item view by hash, used to resolve a deep-link (`?item=<hash>`) that has no
 * collectible entry into a read-only drawer instead of dead-ending. Static per manifest
 * version. `enabled` guards the no-deep-link-miss case (undefined hash).
 */
export function itemByHashQuery(itemHash: string | undefined | null) {
  return {
    queryKey: ["item-view", itemHash] as const,
    queryFn: () => apiFetch<APIItemView>(`/api/items/${itemHash}`),
    enabled: !!itemHash,
    staleTime: Infinity,
    retry: false,
  };
}
