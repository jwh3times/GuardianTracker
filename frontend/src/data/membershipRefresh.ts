import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useAuth } from "../contexts/AuthContext";
import { useIdentityMutation } from "../contexts/IdentityMutation";
import { apiFetch, type ApiError } from "../lib/api";
import { invalidateCollections } from "./collections";
import type { APICacheRefreshResponse } from "../types/api";

/**
 * The membership-refresh data-access module (ADR 0020).
 *
 * It owns the cache-refresh endpoint and the fan-out that follows it. Before
 * this module, the mutation was duplicated verbatim in the Collections page and
 * Settings — same endpoint, same six invalidations, differing only in how each
 * sourced the membership — and both named query keys belonging to six other
 * resources.
 *
 * The refresh lives here rather than in `collections.ts` even though the
 * endpoint sits under `/api/collections/`: what it invalidates is every
 * membership-scoped resource, and a module that owns one resource has no claim
 * on the other five.
 *
 * ADR 0018 requires this fan-out to keep covering both Collections variants
 * plus Characters, Weekly, Catalysts, Crafting and Seals.
 */

/**
 * The membership-scoped resources a refresh invalidates.
 *
 * Each entry becomes a call to that resource's own invalidation entry point as
 * its ADR 0020 slice lands — Collections already has (E5). The rest are still
 * raw keys, which is the cross-ownership reach this decision removes, so they
 * are held here, in one greppable list, rather than in the feature modules
 * where they used to live. E16 adds the test that fails when a membership-
 * scoped module is added without being wired in.
 */
const UNMIGRATED_KEYS = [
  "characters", // E7
  "weekly", // E6
  "catalysts", // E12
  "crafting", // E13
  "seals", // E14
] as const;

function fanOut(client: QueryClient) {
  invalidateCollections(client);
  for (const key of UNMIGRATED_KEYS) {
    void client.invalidateQueries({ queryKey: [key] });
  }
}

/**
 * Re-fetch this membership's data from Bungie and refresh every view of it.
 *
 * Takes no membership argument: it reads the current one from the session, so
 * a page cannot refresh a membership other than the signed-in one. The two
 * callers previously derived it through different expressions.
 */
export function useMembershipRefresh(callbacks?: {
  onError?: (error: ApiError) => void;
}) {
  const client = useQueryClient();
  const { user } = useAuth();
  const mutation = useIdentityMutation<APICacheRefreshResponse, ApiError, void>(
    {
      mutationFn: () =>
        apiFetch<APICacheRefreshResponse>(
          `/api/collections/${user?.membershipType}/${user?.membershipId}/refresh`,
          { method: "POST" },
        ),
      onSuccess: () => fanOut(client),
      onError: (error) => callbacks?.onError?.(error),
    },
  );

  return {
    refresh: () => mutation.mutate(),
    isRefreshing: mutation.isPending,
  };
}
