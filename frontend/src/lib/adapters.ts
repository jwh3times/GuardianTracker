import type { APIItemView } from "../types/api";
import type { GTItem, Rarity } from "../types/design";

/**
 * Shared rarity vocabulary. Exported because the Wish list data-access module
 * owns its own projection now (ADR 0020) but still maps the same five wire
 * names. The Items slice (E8) retires the last local caller, at which point
 * this belongs with it rather than here. `toCharacter` moved to
 * `data/characters.ts` in E7.
 */
export const RARITY_MAP: Record<string, Rarity> = {
  Exotic: "exotic",
  Legendary: "legendary",
  Rare: "rare",
  Uncommon: "uncommon",
  Common: "common",
};

/** Adapt a minimal item view (deep-linked non-collectible) into a view-only GTItem. */
export function toGTItemView(v: APIItemView): GTItem {
  return {
    id: v.itemHash,
    name: v.name,
    type: v.itemType,
    slot: "",
    rarity: RARITY_MAP[v.rarity] ?? "legendary",
    farmOnly: false,
    acquisitionSources: [],
    availableNow: false,
    collected: false,
    desc: v.description ?? "",
    icon: v.icon,
    viewOnly: true,
  };
}
