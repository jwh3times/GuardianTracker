import type { WishListItem } from "../types";
import type {
  APICharacter,
  APIDestinyItem,
} from "../types/api";
import type {
  Character,
  Difficulty,
  GTItem,
  Priority,
  Rarity,
  WishlistEntry,
} from "../types/design";

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

/** Adapt a REST API character into the design system's Character shape. */
export function toCharacter(c: APICharacter): Character {
  return {
    id: c.characterId,
    name: c.className,
    cls: c.className,
    race: c.raceName,
    power: c.light,
    emblem: 0,
    emblemUrl: c.emblemPath || undefined,
  };
}

/** Adapt a REST API DestinyItem into the design system's GTItem shape. */
export function toGTItem(d: APIDestinyItem): GTItem {
  const sources = d.sources ?? [];
  return {
    id: d.itemHash,
    name: d.name,
    type: d.itemType,
    slot: "",
    rarity: RARITY_MAP[d.rarity] ?? "legendary",
    diff: DIFF_MAP[d.difficulty] ?? "moderate",
    source: sources[0] ?? "Unknown source",
    sourceDetail: sources.slice(1).join(" · ") || sources[0] || "",
    obtainable: false,
    collected: false,
    desc: d.description ?? "",
    perks: [],
    icon: d.icon,
  };
}

/** Adapt a WishListItem into the design system's WishlistEntry shape. */
export function toWishlistEntry(w: WishListItem): WishlistEntry {
  const sources = w.sources ?? [];
  return {
    id: w.id,
    name: w.name,
    type: w.itemType,
    rarity: RARITY_MAP[w.rarity] ?? "legendary",
    priority: PRIORITY_MAP[w.priority] ?? "medium",
    avail: { now: false, where: sources[0] ?? "Unknown source" },
    notes: w.notes ?? "",
    added: w.dateAdded,
  };
}
