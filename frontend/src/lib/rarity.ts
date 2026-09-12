import type { Rarity } from "../types/design";

/**
 * Wire rarity name → design vocabulary. The manifest's tier names, as the API
 * serializes them for item views and wish list rows.
 */
const WIRE_RARITIES: Record<string, Rarity> = {
  Exotic: "exotic",
  Legendary: "legendary",
  Rare: "rare",
  Uncommon: "uncommon",
  Common: "common",
};

/**
 * Looked up through a Map rather than by indexing {@link WIRE_RARITIES}
 * directly: the input is arbitrary server-supplied text, and an object index
 * would answer for inherited keys like `toString` with something that is not a
 * rarity at all. Mirrors `lib/difficulty.ts`.
 */
const RARITY_BY_WIRE = new Map<string, Rarity>(Object.entries(WIRE_RARITIES));

/**
 * The shared adapter from a wire rarity name to the design vocabulary, used by
 * the Items and Wish list data-access modules. An unrecognised value becomes
 * `legendary`, the fallback both call sites already used.
 *
 * `lib/collectionsView.ts` keeps its own private copy of the map; it is an
 * ADR 0018 survivor and is deliberately left unchanged.
 */
export function toRarity(raw: string | undefined | null): Rarity {
  if (!raw) return "legendary";
  return RARITY_BY_WIRE.get(raw) ?? "legendary";
}
