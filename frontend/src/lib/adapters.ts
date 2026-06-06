import type { DestinyItem, WishListItem } from "../types";
import type { Difficulty, GTItem, Priority, Rarity, WishlistEntry } from "../types/design";

const RARITY_MAP: Record<string, Rarity> = {
  Exotic: "exotic",
  Legendary: "legendary",
  Rare: "rare",
  Uncommon: "uncommon",
  Common: "common",
};
const DIFF_MAP: Record<string, Difficulty> = {
  Easy: "easy",
  Moderate: "moderate",
  Challenging: "challenging",
};
const PRIORITY_MAP: Record<string, Priority> = {
  URGENT: "urgent",
  HIGH: "high",
  MEDIUM: "medium",
  LOW: "low",
};

/** Adapt a GraphQL DestinyItem into the design system's GTItem shape. */
export function toGTItem(d: DestinyItem): GTItem {
  const sources = d.sources ?? [];
  return {
    id: d.itemHash,
    name: d.name,
    type: d.itemType,
    slot: "", // damage/equip slot not exposed by the collections query
    rarity: RARITY_MAP[d.rarity] ?? "legendary",
    diff: DIFF_MAP[d.difficulty] ?? "moderate",
    source: sources[0] ?? "Unknown source",
    sourceDetail: sources.slice(1).join(" · ") || sources[0] || "",
    obtainable: false,
    collected: d.isCollected ?? false,
    desc: d.description ?? "",
    perks: [],
    icon: d.icon,
  };
}

/** Adapt a GraphQL WishListItem into the design system's WishlistEntry shape. */
export function toWishlistEntry(w: WishListItem): WishlistEntry {
  const sources = w.sources ?? [];
  return {
    id: w.id,
    name: w.name,
    type: w.itemType,
    rarity: RARITY_MAP[w.rarity] ?? "legendary",
    priority: PRIORITY_MAP[w.priority] ?? "medium",
    // The collections API does not yet surface live availability; show the
    // item's source so the list still reads correctly.
    avail: { now: false, where: sources[0] ?? "Unknown source" },
    notes: w.notes ?? "",
    added: w.dateAdded,
  };
}
