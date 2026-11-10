import { useCallback, useEffect, useMemo } from "react";
import {
  useQuery,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";
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
 * How long a digest read stays fresh. It throttles visit-activity reads: no
 * new read is issued while the last one is younger than this. It must stay
 * well below the backend's two-hour visit gap (ADR 0023), or a player who keeps
 * navigating could still be seen as away.
 */
const DIGEST_FRESH_MS = 5 * 60 * 1000;

/**
 * The one query definition every reader of this resource shares, so the
 * Dashboard's display and the shell's visit-activity read are the same cache
 * entry and never cost a second request.
 */
function useDigestQueryOptions() {
  const { user } = useAuth();
  const membershipType = user?.membershipType;
  const membershipId = user?.membershipId;
  return useMemo(
    () => ({
      queryKey: digestKey(membershipType, membershipId),
      queryFn: () =>
        apiFetch<APIDigest>(`/api/digest/${membershipType}/${membershipId}`),
      enabled: membershipType != null && !!membershipId,
      staleTime: DIGEST_FRESH_MS,
    }),
    [membershipType, membershipId],
  );
}

/**
 * The since-last-visit digest for the signed-in membership.
 *
 * Takes no membership argument: it reads the current one from the session, so
 * a page cannot read another membership's digest.
 */
export function useDigest() {
  const options = useDigestQueryOptions();
  const { data, isLoading, isError, error, refetch } = useQuery({
    ...options,
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

/**
 * Records visit activity for the signed-in membership (ADR 0023's Visit
 * boundary, #470). Mounted once by the authenticated app shell so every
 * authenticated page — not only the Dashboard — issues the digest read: on
 * mount, whenever `page` changes, and when the window regains focus, each
 * only once the last read has gone stale.
 *
 * `page` is the current route path; this module stays router-free, so the
 * shell passes it in.
 *
 * The read itself is the activity signal: within a visit the backend only
 * advances `last_activity_at` and re-serves the frozen result; after a gap
 * longer than the visit gap it starts the new visit on whichever page loads
 * first. Never replace this with an unconditional touch that skips the
 * backend's gap check — that would erase a real gap and make the next
 * Dashboard read re-serve the previous visit's frozen digest.
 *
 * Returns nothing: the shell never displays the digest, so a failed read is
 * deliberately silent. Only the Dashboard panel shows digest outcomes.
 */
export function useDigestActivity(page: string): void {
  const options = useDigestQueryOptions();
  const client = useQueryClient();
  // The observer covers the first read and focus; the effect covers
  // navigation. prefetchQuery skips a fresh entry, joins an in-flight read
  // instead of starting a second one, and never throws.
  useQuery(options);
  useEffect(() => {
    if (!options.enabled) return;
    void client.prefetchQuery(options);
  }, [client, options, page]);
}
