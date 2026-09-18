import { useCallback } from "react";
import { useQuery, type QueryClient } from "@tanstack/react-query";
import { useAuth } from "../contexts/AuthContext";
import { apiFetch } from "../lib/api";
import type { APIAcquiredItem, APIDigest } from "../types/api";
import type { AcquiredItem, Digest } from "../types/design";

/**
 * The Digest data-access module (ADR 0023). It owns this resource's query
 * identity, endpoint path, projection to {@link Digest}, and its own
 * invalidation.
 *
 * The endpoint always answers 200 with a typed `status` — computing the
 * since-last-visit diff, freezing it for the visit's duration, and deciding
 * `ready` versus `first-visit` versus `unavailable` are all the backend's
 * (`services/digest`'s), not this module's. This module only reads that
 * result and projects it to the design shape.
 *
 * Read by the Dashboard.
 */

/**
 * Private, and deliberately not exported: no other module may name this key.
 * Membership-qualified because the endpoint path carries the membership.
 */
function digestKey(
  membershipType: number | undefined,
  membershipId: string | undefined,
) {
  return ["digest", membershipType, membershipId] as const;
}

/** Matches every membership's entry. Used only by the refresh seam. */
const DIGEST_ROOT_KEY = ["digest"] as const;

/**
 * This module's invalidation entry point, called by `membershipRefresh.ts` so
 * the refresh owner never names this key.
 *
 * Unlike `preferences.ts`'s exemption from the same fan-out — where a Bungie
 * refresh must never change a user setting — refreshing the digest mid-visit
 * is only ever redundant: the backend freezes and re-serves the same visit's
 * result regardless of how many times it is asked. Wiring it into the fan-out
 * costs nothing and keeps this resource covered like every other
 * membership-scoped one.
 */
export function invalidateDigest(client: QueryClient) {
  void client.invalidateQueries({ queryKey: DIGEST_ROOT_KEY });
}

function toAcquiredItem(a: APIAcquiredItem): AcquiredItem {
  return {
    itemHash: a.itemHash,
    name: a.name,
    icon: a.icon || undefined,
    type: a.itemType,
  };
}

/**
 * The query's `select`. Stable module-level reference so React Query's
 * per-observer `select` memoisation holds.
 */
function toDigest(api: APIDigest): Digest {
  const digest: Digest = {
    status: api.status,
    visitStartedAt: api.visitStartedAt,
    // `acquired` is always an array server-side, but guard a nil Go slice
    // anyway — it would serialize as null, matching this module's siblings.
    acquired: (api.acquired ?? []).map(toAcquiredItem),
  };
  if (api.previousVisitAt) digest.previousVisitAt = api.previousVisitAt;
  return digest;
}

/**
 * The since-last-visit digest for the signed-in membership.
 *
 * Takes no membership argument: it reads the current one from the session, so
 * a page cannot read another membership's digest.
 */
export function useDigest() {
  const { user } = useAuth();
  const membershipType = user?.membershipType;
  const membershipId = user?.membershipId;

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: digestKey(membershipType, membershipId),
    queryFn: () =>
      apiFetch<APIDigest>(`/api/digest/${membershipType}/${membershipId}`),
    enabled: membershipType != null && !!membershipId,
    select: toDigest,
  });

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return {
    digest: data,
    isLoading,
    isError,
    error,
    retry,
  };
}
