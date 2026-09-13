import { useCallback } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "../lib/api";
import { toTier, type Role, type Tier } from "../lib/roles";
import type { APIFlagsResponse, APIResolvedFlag } from "../types/api";

/**
 * The Flags data-access module (ADR 0020). It owns this resource's query
 * identity, endpoint path, projection to {@link Flag}, and its own refresh.
 *
 * `FlagsContext` survives as a thin wrapper over {@link useResolvedFlags}, as
 * `CharacterContext` does over Characters: its throw-outside-the-provider guard
 * is what stops a page hoisted above the auth gate from fetching flags
 * anonymously and failing open to "everything accessible".
 *
 * Before this module, Admin and Settings both invalidated this resource's key
 * directly — across the ownership line ADR 0020 draws — and Settings also
 * called the context's refetch, so every self opt-in asked the server twice.
 *
 * Flags follow the signed-in user's role rather than their Destiny data, so a
 * membership refresh does not re-fetch them; the identity cleanup in ADR 0017
 * clears them when the user changes.
 */

/** Private, and deliberately not exported: no other module may name this key. */
const FLAGS_KEY = ["flags"] as const;

/** One server-resolved feature flag, as the design's flagState shape. */
export interface Flag {
  key: string;
  name: string;
  desc: string;
  category: string;
  /** The lowest tier the flag is gated to. */
  minTier: Tier;
  enabled: boolean;
  accessible: boolean;
  locked: boolean;
}

interface ResolvedFlags {
  role: Role;
  flags: Flag[];
}

function toFlag(f: APIResolvedFlag): Flag {
  return {
    key: f.key,
    name: f.name,
    desc: f.desc,
    category: f.category,
    minTier: toTier(f.minTier),
    enabled: f.enabled,
    accessible: f.accessible,
    locked: f.locked,
  };
}

/**
 * The query's `select`. Stable module-level reference so React Query's
 * per-observer `select` memoisation holds.
 */
function toResolvedFlags(r: APIFlagsResponse): ResolvedFlags {
  return { role: r.role, flags: (r.flags ?? []).map(toFlag) };
}

/** Stable empty list, so `flags` keeps referential identity while loading. */
const NO_FLAGS: Flag[] = [];

/**
 * The signed-in user's role and every resolved flag. Until the response
 * arrives the role is `standard` and there are no flags — an unknown flag fails
 * open in `FlagsContext`, so nothing shipped is hidden while this loads.
 *
 * `refresh` re-fetches after something that changes the user's access: a self
 * opt-in, or an admin editing a role or a flag.
 */
export function useResolvedFlags() {
  const client = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: FLAGS_KEY,
    queryFn: () => apiFetch<APIFlagsResponse>("/api/flags"),
    staleTime: 60_000,
    select: toResolvedFlags,
  });

  const refresh = useCallback(() => {
    void client.invalidateQueries({ queryKey: FLAGS_KEY });
  }, [client]);

  return {
    role: data?.role ?? "standard",
    flags: data?.flags ?? NO_FLAGS,
    isLoading,
    refresh,
  };
}
