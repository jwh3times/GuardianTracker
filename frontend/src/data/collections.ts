import { useCallback } from "react";
import { useQuery, type QueryClient } from "@tanstack/react-query";
import { useAuth } from "../contexts/AuthContext";
import { apiFetch } from "../lib/api";
import { toCollections, toCollectionsSummary } from "../lib/collectionsView";
import type {
  CollectionsSummaryView,
  CollectionsView,
} from "../lib/collectionsView";
import type { APIMembershipCollections } from "../types/api";

/**
 * The Collections data-access module (ADR 0020). It owns this resource's query
 * identity, endpoint path, projection to the two view types, and its own
 * invalidation.
 *
 * Five features read it — Collections, Cosmetics, Dashboard, Settings and the
 * onboarding tour — through two variants: a counts-only summary and the full
 * item set. Each variant is one cache entry shared by its consumers.
 *
 * Unlike the Wish list module, the projections are NOT moved in here.
 * `toCollections` and `toCollectionsSummary` stay in `lib/collectionsView.ts`
 * with the ADR 0018 survivor tests that exercise them directly, which this
 * slice is required to retain unchanged.
 *
 * The cache-refresh endpoint is not here either: it invalidates six resources,
 * only one of which is this one, so it belongs to `membershipRefresh.ts`.
 */

/**
 * Private, and deliberately not exported: no other module may name this key.
 * The trailing segment separates the two variants, so the counts-only readers
 * and the full-item readers do not share a cache entry with each other while
 * each still shares one among themselves.
 */
function collectionsKey(
  membershipType: number | undefined,
  membershipId: string | undefined,
  includeAll: boolean,
) {
  return [
    "collections",
    membershipType,
    membershipId,
    includeAll ? "all" : "missing",
  ] as const;
}

/** Matches every variant for every membership. Used only by the refresh seam. */
const COLLECTIONS_ROOT_KEY = ["collections"] as const;

/**
 * This module's invalidation entry point. Exported for `membershipRefresh.ts`
 * so the refresh owner can invalidate this resource without naming its key —
 * before this existed, Collections and Settings each named `["collections"]`
 * directly, which is the cross-ownership reach ADR 0020 removes.
 */
export function invalidateCollections(client: QueryClient) {
  void client.invalidateQueries({ queryKey: COLLECTIONS_ROOT_KEY });
}

interface CollectionsQueryOptions {
  /**
   * Extra gate ANDed with this module's own membership gate. Only the
   * onboarding tour needs it — it waits on saved preferences — and it takes a
   * plain boolean so React Query's option shape stays inside this module.
   */
  enabled?: boolean;
}

function useCollectionsMembership() {
  const { user } = useAuth();
  return {
    membershipType: user?.membershipType,
    membershipId: user?.membershipId,
  };
}

/**
 * The counts-only view: the node tree, the four-category summary and the fetch
 * timestamp. Read by the Dashboard, Settings and the onboarding tour.
 *
 * `select` must stay a stable module-level reference — React Query memoises it
 * per observer on function identity, so an inline arrow would re-run the
 * adapter on every render of all three consumers.
 */
export function useCollectionsSummary({
  enabled = true,
}: CollectionsQueryOptions = {}) {
  const { membershipType, membershipId } = useCollectionsMembership();
  const query = useQuery({
    queryKey: collectionsKey(membershipType, membershipId, false),
    queryFn: () =>
      apiFetch<APIMembershipCollections>(
        `/api/collections/${membershipType}/${membershipId}`,
      ),
    enabled: enabled && membershipType != null && !!membershipId,
    select: toCollectionsSummary,
  });

  return useCollectionsResult<CollectionsSummaryView>(query);
}

/**
 * The full view, carrying every item. Read by the Collections browser and the
 * Cosmetics gallery.
 *
 * A separate hook rather than a boolean parameter so the returned view type
 * differs: only this payload has items, so only this hook exposes an item
 * surface, and asking the counts-only view for items is a compile error rather
 * than an empty array.
 *
 * The browser always loads collected + missing and filters client-side, which
 * is why toggling `missingOnly` or following a deep link to a collected item
 * does not refetch.
 */
export function useCollections({
  enabled = true,
}: CollectionsQueryOptions = {}) {
  const { membershipType, membershipId } = useCollectionsMembership();
  const query = useQuery({
    queryKey: collectionsKey(membershipType, membershipId, true),
    queryFn: () =>
      apiFetch<APIMembershipCollections>(
        `/api/collections/${membershipType}/${membershipId}?include=all`,
      ),
    enabled: enabled && membershipType != null && !!membershipId,
    select: toCollections,
  });

  return useCollectionsResult<CollectionsView>(query);
}

/** What both hooks hand back. No React Query types cross this boundary. */
export interface CollectionsResult<T> {
  view: T | undefined;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  /** Wire to a retry affordance; the promise is swallowed here. */
  retry: () => void;
}

/**
 * Shared tail of both query hooks. A hook itself, not a plain helper, because
 * it holds a `useCallback` — naming it as one keeps the rules-of-hooks check
 * meaningful here rather than suppressed.
 */
function useCollectionsResult<T>(query: {
  data: T | undefined;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  refetch: () => Promise<unknown>;
}): CollectionsResult<T> {
  const { data, isLoading, isError, error, refetch } = query;
  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);
  return { view: data, isLoading, isError, error, retry };
}
