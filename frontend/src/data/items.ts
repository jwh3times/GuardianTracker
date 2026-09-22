import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "../lib/api";
import { toRarity } from "../lib/rarity";
import type { APIItemPerks, APIItemView } from "../types/api";
import type { GTItem, ItemCatalyst, PerkColumn } from "../types/design";

/**
 * The Items data-access module (ADR 0020). It owns this resource's query
 * identity, endpoint paths, and projection to domain types.
 *
 * Two manifest lookups by item hash, both read by the Collections item drawer:
 * the perk pool and catalysts for an item, and the minimal item view that turns
 * a deep link to a non-collectible into a read-only drawer.
 *
 * There is deliberately no invalidation entry point. Item data is static for a
 * manifest version and carries no membership, so a membership refresh has
 * nothing to re-fetch here and the fan-out in `membershipRefresh.ts` does not
 * include it. Search is a separate resource.
 */

/** Private: the hash is a genuine query parameter, so it is part of the key. */
function itemPerksKey(itemHash: string | undefined) {
  return ["item-perks", itemHash] as const;
}

/** Private: the hash is a genuine query parameter, so it is part of the key. */
function itemViewKey(itemHash: string | null | undefined) {
  return ["item-view", itemHash] as const;
}

/** An item's possible-roll perk columns and its catalysts. */
export interface ItemPerks {
  perkColumns: PerkColumn[];
  catalysts: ItemCatalyst[];
}

/**
 * The perks query's `select`. Stable module-level reference so React Query's
 * per-observer `select` memoisation holds.
 */
function toItemPerks(p: APIItemPerks): ItemPerks {
  return {
    // `plugs` is deliberately not projected: it carries the roll-target plug
    // hashes and nothing renders them yet. Adding it here would put an unused
    // field in the view model; the wire type keeps it so the contract stays honest.
    perkColumns: (p.perkColumns ?? []).map((c) => ({
      role: c.role,
      label: c.label,
      perks: c.perks,
    })),
    catalysts: (p.catalysts ?? []).map((c) => ({
      name: c.name,
      description: c.description,
    })),
  };
}

/** Adapt a minimal item view (deep-linked non-collectible) into a view-only GTItem. */
function toGTItemView(v: APIItemView): GTItem {
  return {
    id: v.itemHash,
    name: v.name,
    type: v.itemType,
    slot: "",
    rarity: toRarity(v.rarity),
    farmOnly: false,
    acquisitionSources: [],
    availableNow: false,
    collected: false,
    desc: v.description ?? "",
    icon: v.icon,
    viewOnly: true,
  };
}

/**
 * The perk pool and catalysts for one item, fetched lazily. Pass `undefined`
 * while no item is selected and nothing is requested.
 *
 * Perk pools are static for a manifest version, so this never goes stale
 * within a session.
 */
export function useItemPerks(itemHash: string | undefined) {
  const { data, isLoading } = useQuery({
    queryKey: itemPerksKey(itemHash),
    queryFn: () => apiFetch<APIItemPerks>(`/api/items/${itemHash}/perks`),
    enabled: !!itemHash,
    staleTime: Infinity,
    select: toItemPerks,
  });

  return {
    perkColumns: data?.perkColumns,
    catalysts: data?.catalysts,
    isLoading,
  };
}

/**
 * The minimal, view-only item for a hash, used to resolve a deep link that has
 * no collectible entry. Pass `null` when there is nothing to resolve.
 *
 * Not retried: a 404 means the hash is not in the manifest, and retrying cannot
 * change that.
 */
export function useItemView(itemHash: string | null) {
  const { data, isError } = useQuery({
    queryKey: itemViewKey(itemHash),
    queryFn: () => apiFetch<APIItemView>(`/api/items/${itemHash}`),
    enabled: !!itemHash,
    staleTime: Infinity,
    retry: false,
    select: toGTItemView,
  });

  return { item: data, isError };
}
