import { useCallback } from "react";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useAuth } from "../contexts/AuthContext";
import { useIdentityMutation } from "../contexts/IdentityMutation";
import { apiFetch, type ApiError } from "../lib/api";
import { invalidateCatalysts } from "./catalysts";
import { invalidateCharacters } from "./characters";
import { invalidateCollections } from "./collections";
import { invalidateCrafting } from "./crafting";
import { invalidateSeals } from "./seals";
import { invalidateWeekly } from "./weekly";
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
 * Every membership-scoped resource a refresh invalidates, each through its own
 * module's entry point — this module names no other module's key. The list is
 * explicit and greppable on purpose (ADR 0020 rejected an import-time
 * registry); E16 adds the test that fails when a membership-scoped module is
 * added without being wired in here.
 */
function fanOut(client: QueryClient) {
  invalidateCatalysts(client);
  invalidateCharacters(client);
  invalidateCollections(client);
  invalidateCrafting(client);
  invalidateSeals(client);
  invalidateWeekly(client);
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

/**
 * Re-fetch everything this session has cached, once a Bungie authorization is
 * restored. Deliberately wider than the membership fan-out: while authorization
 * was expired any Bungie-backed request may have failed, whichever resource it
 * belonged to. It names no query key, so it reaches into no module's identity.
 */
export function useReloadAfterReconnect(): () => Promise<void> {
  const client = useQueryClient();
  return useCallback(() => client.invalidateQueries(), [client]);
}
