import { apiFetch } from "./api";
import { toWeekly } from "./weeklyView";
import type { APIItemPerks, APIItemView, APIWeekly } from "../types/api";

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

/**
 * This week's reset timing, milestones, Xûr inventory and ranked actions.
 * Shared by the Dashboard and the This Week page, which therefore sit on one
 * cache entry per selected character instead of firing two requests.
 *
 * `select` adapts the payload at the seam, so neither page sees the wire shape
 * or the wire difficulty vocabulary. Like the collections helpers it must stay
 * a stable module-level reference: React Query memoises `select` per observer
 * on function identity, and an inline arrow would re-run the adapter on every
 * render.
 *
 * `enabled` is deliberately left to the caller — the two consumers gate on
 * different identity facts, and one shared guard would be wrong for both.
 */
export function weeklyQuery(characterId: string | undefined | null) {
  return {
    queryKey: ["weekly", characterId ?? null] as const,
    queryFn: () =>
      apiFetch<APIWeekly>(
        `/api/weekly/recommendations${characterId ? `?characterId=${encodeURIComponent(characterId)}` : ""}`,
      ),
    select: toWeekly,
  };
}
