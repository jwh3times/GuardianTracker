import { apiFetch } from "./api";
import { toCollections, toCollectionsSummary } from "./collectionsView";
import type {
  APIItemPerks,
  APIItemView,
  APIUserCollections,
} from "../types/api";

function collectionsBase(
  membershipType: number | undefined,
  membershipId: string | undefined,
  includeAll: boolean,
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
 * The counts-only collections query — the tree, the four-category summary and
 * the fetch timestamp. Used by the Dashboard, Settings and the onboarding tour.
 *
 * Routing every reader through these helpers keeps the React Query keys
 * consistent, so the two variants share one cache entry each instead of every
 * page firing its own request. `select` adapts the payload at the seam, so no
 * feature module ever sees the wire shape.
 *
 * `select` must stay a stable module-level reference: React Query memoises it
 * per observer keyed on function identity, and an inline arrow would re-run the
 * adapter on every render.
 */
export function collectionsQuery(
  membershipType: number | undefined,
  membershipId: string | undefined,
) {
  return {
    ...collectionsBase(membershipType, membershipId, false),
    select: toCollectionsSummary,
  };
}

/**
 * The full collections query, carrying every item. Used by the Collections page
 * and the Cosmetics gallery.
 *
 * Separate from {@link collectionsQuery} rather than a boolean parameter so the
 * returned view type differs: only this variant's payload has items, so only
 * this one exposes an item surface. Asking the counts-only view for items is a
 * compile error rather than an empty array.
 */
export function collectionsFullQuery(
  membershipType: number | undefined,
  membershipId: string | undefined,
) {
  return {
    ...collectionsBase(membershipType, membershipId, true),
    select: toCollections,
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
