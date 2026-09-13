import { useCallback } from "react";
import { useQuery, type QueryClient } from "@tanstack/react-query";
import { useAuth } from "../contexts/AuthContext";
import { apiFetch } from "../lib/api";
import type { APIRecordsEnvelope, APISeal } from "../types/api";
import type { Seal, Triumph, TriumphObjective } from "../types/design";

/**
 * The Seals data-access module (ADR 0020). It owns this resource's query
 * identity, endpoint path, projection to {@link Seal}, and its own
 * invalidation.
 *
 * Read by the Triumphs & Seals page. Seal progress is membership-scoped, so a
 * membership refresh re-fetches it through {@link invalidateSeals}.
 */

/**
 * Private, and deliberately not exported: no other module may name this key.
 * Membership-qualified because the endpoint path carries the membership.
 */
function sealsKey(
  membershipType: number | undefined,
  membershipId: string | undefined,
) {
  return ["seals", membershipType, membershipId] as const;
}

/** Matches every membership's entry. Used only by the refresh seam. */
const SEALS_ROOT_KEY = ["seals"] as const;

/**
 * This module's invalidation entry point, called by `membershipRefresh.ts` so
 * the refresh owner never names this key.
 */
export function invalidateSeals(client: QueryClient) {
  void client.invalidateQueries({ queryKey: SEALS_ROOT_KEY });
}

function toObjective(o: TriumphObjective): TriumphObjective {
  return { label: o.label, done: o.done, cur: o.cur, max: o.max };
}

/**
 * `objectives` stays absent when the wire omits it, rather than becoming `[]`:
 * the design type documents absence as "no objective data", and the seal card
 * offers a drill-down only when the list is present.
 */
function toTriumph(t: APISeal["triumphs"][number]): Triumph {
  const triumph: Triumph = {
    label: t.label,
    done: t.done,
    cur: t.cur,
    max: t.max,
  };
  if (t.objectives) triumph.objectives = t.objectives.map(toObjective);
  return triumph;
}

/**
 * Field-by-field today: the wire seal is the design type re-exported. The copy
 * is the seam ADR 0020 asks for, so a future wire change cannot silently
 * become the domain shape.
 */
function toSeal(s: APISeal): Seal {
  return {
    id: s.id,
    name: s.name,
    pct: s.pct,
    gilded: s.gilded,
    left: s.left,
    triumphs: s.triumphs.map(toTriumph),
  };
}

/**
 * The query's `select`. Stable module-level reference so React Query's
 * per-observer `select` memoisation holds. The fallback only guards a nil Go
 * slice, which would serialize as null.
 */
function toSeals(envelope: APIRecordsEnvelope<APISeal>): Seal[] {
  return (envelope.items ?? []).map(toSeal);
}

/** Stable empty list, so `seals` keeps referential identity. */
const NO_SEALS: Seal[] = [];

/**
 * The signed-in membership's seal progress.
 *
 * Takes no membership argument: it reads the current one from the session, so
 * a page cannot show another membership's seals.
 */
export function useSeals() {
  const { user } = useAuth();
  const membershipType = user?.membershipType;
  const membershipId = user?.membershipId;

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: sealsKey(membershipType, membershipId),
    queryFn: () =>
      apiFetch<APIRecordsEnvelope<APISeal>>(
        `/api/seals/${membershipType}/${membershipId}`,
      ),
    enabled: membershipType != null && !!membershipId,
    select: toSeals,
  });

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return { seals: data ?? NO_SEALS, isLoading, isError, error, retry };
}
