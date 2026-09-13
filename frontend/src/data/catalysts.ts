import { useCallback } from "react";
import { useQuery, type QueryClient } from "@tanstack/react-query";
import { useAuth } from "../contexts/AuthContext";
import { apiFetch } from "../lib/api";
import type { APICatalyst, APIRecordsEnvelope } from "../types/api";
import type { Catalyst } from "../types/design";

/**
 * The Catalysts data-access module (ADR 0020). It owns this resource's query
 * identity, endpoint path, projection to {@link Catalyst}, and its own
 * invalidation.
 *
 * Read by the Catalysts & Crafting page. Crafting patterns are a separate
 * resource even though they share that page.
 *
 * Catalyst progress is membership-scoped, so a membership refresh re-fetches
 * it through {@link invalidateCatalysts}.
 */

/**
 * Private, and deliberately not exported: no other module may name this key.
 * Membership-qualified because the endpoint path carries the membership.
 */
function catalystsKey(
  membershipType: number | undefined,
  membershipId: string | undefined,
) {
  return ["catalysts", membershipType, membershipId] as const;
}

/** Matches every membership's entry. Used only by the refresh seam. */
const CATALYSTS_ROOT_KEY = ["catalysts"] as const;

/**
 * This module's invalidation entry point, called by `membershipRefresh.ts` so
 * the refresh owner never names this key.
 */
export function invalidateCatalysts(client: QueryClient) {
  void client.invalidateQueries({ queryKey: CATALYSTS_ROOT_KEY });
}

/**
 * Field-by-field today: the wire item is the design type re-exported. The copy
 * is the seam ADR 0020 asks for, so a future wire change cannot silently
 * become the domain shape.
 */
function toCatalyst(c: APICatalyst): Catalyst {
  return {
    id: c.id,
    name: c.name,
    type: c.type,
    icon: c.icon,
    status: c.status,
    obj: c.obj ? { label: c.obj.label, cur: c.obj.cur, max: c.obj.max } : null,
    source: c.source,
    effect: c.effect,
  };
}

/**
 * The query's `select`. Stable module-level reference so React Query's
 * per-observer `select` memoisation holds. A nil Go slice serializes as null,
 * so a missing list is an empty one.
 */
function toCatalysts(envelope: APIRecordsEnvelope<APICatalyst>): Catalyst[] {
  return (envelope.items ?? []).map(toCatalyst);
}

/** Stable empty list, so `catalysts` keeps referential identity. */
const NO_CATALYSTS: Catalyst[] = [];

/**
 * The signed-in membership's exotic catalyst progress.
 *
 * Takes no membership argument: it reads the current one from the session, so
 * a page cannot show another membership's catalysts.
 */
export function useCatalysts() {
  const { user } = useAuth();
  const membershipType = user?.membershipType;
  const membershipId = user?.membershipId;

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: catalystsKey(membershipType, membershipId),
    queryFn: () =>
      apiFetch<APIRecordsEnvelope<APICatalyst>>(
        `/api/catalysts/${membershipType}/${membershipId}`,
      ),
    enabled: membershipType != null && !!membershipId,
    select: toCatalysts,
  });

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return {
    catalysts: data ?? NO_CATALYSTS,
    isLoading,
    isError,
    error,
    retry,
  };
}
