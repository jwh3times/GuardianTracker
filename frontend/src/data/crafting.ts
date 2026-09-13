import { useCallback } from "react";
import { useQuery, type QueryClient } from "@tanstack/react-query";
import { useAuth } from "../contexts/AuthContext";
import { apiFetch } from "../lib/api";
import type { APICraftPattern, APIRecordsEnvelope } from "../types/api";
import type { CraftPattern } from "../types/design";

/**
 * The Crafting data-access module (ADR 0020). It owns this resource's query
 * identity, endpoint path, projection to {@link CraftPattern}, and its own
 * invalidation.
 *
 * Read by the Catalysts & Crafting page, beside `data/catalysts.ts`; the two
 * share a page, not a resource.
 *
 * Pattern progress is membership-scoped, so a membership refresh re-fetches it
 * through {@link invalidateCrafting}.
 */

/**
 * Private, and deliberately not exported: no other module may name this key.
 * Membership-qualified because the endpoint path carries the membership.
 */
function craftingKey(
  membershipType: number | undefined,
  membershipId: string | undefined,
) {
  return ["crafting", membershipType, membershipId] as const;
}

/** Matches every membership's entry. Used only by the refresh seam. */
const CRAFTING_ROOT_KEY = ["crafting"] as const;

/**
 * This module's invalidation entry point, called by `membershipRefresh.ts` so
 * the refresh owner never names this key.
 */
export function invalidateCrafting(client: QueryClient) {
  void client.invalidateQueries({ queryKey: CRAFTING_ROOT_KEY });
}

/**
 * Field-by-field today: the wire item is the design type re-exported. The copy
 * is the seam ADR 0020 asks for, so a future wire change cannot silently
 * become the domain shape.
 */
function toCraftPattern(c: APICraftPattern): CraftPattern {
  return {
    id: c.id,
    name: c.name,
    type: c.type,
    icon: c.icon,
    patterns: { cur: c.patterns.cur, max: c.patterns.max },
    note: c.note,
    source: c.source,
  };
}

/**
 * The query's `select`. Stable module-level reference so React Query's
 * per-observer `select` memoisation holds. The fallback only guards a nil Go
 * slice, which would serialize as null.
 */
function toCraftPatterns(
  envelope: APIRecordsEnvelope<APICraftPattern>,
): CraftPattern[] {
  return (envelope.items ?? []).map(toCraftPattern);
}

/** Stable empty list, so `patterns` keeps referential identity. */
const NO_PATTERNS: CraftPattern[] = [];

/**
 * The signed-in membership's weapon crafting pattern progress.
 *
 * Takes no membership argument: it reads the current one from the session, so
 * a page cannot show another membership's patterns.
 */
export function useCraftingPatterns() {
  const { user } = useAuth();
  const membershipType = user?.membershipType;
  const membershipId = user?.membershipId;

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: craftingKey(membershipType, membershipId),
    queryFn: () =>
      apiFetch<APIRecordsEnvelope<APICraftPattern>>(
        `/api/crafting/${membershipType}/${membershipId}`,
      ),
    enabled: membershipType != null && !!membershipId,
    select: toCraftPatterns,
  });

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return {
    patterns: data ?? NO_PATTERNS,
    isLoading,
    isError,
    error,
    retry,
  };
}
