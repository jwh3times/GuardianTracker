import { relTime } from "./format";
import type { APICharacter, APIItemView, WishListItem } from "../types/api";
import type {
  Character,
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

/** Adapt a minimal item view (deep-linked non-collectible) into a view-only GTItem. */
export function toGTItemView(v: APIItemView): GTItem {
  return {
    id: v.itemHash,
    name: v.name,
    type: v.itemType,
    slot: "",
    rarity: RARITY_MAP[v.rarity] ?? "legendary",
    diff: "unrated",
    farmOnly: false,
    source: "",
    sourceDetail: "",
    obtainable: false,
    collected: false,
    desc: v.description ?? "",
    icon: v.icon,
    viewOnly: true,
  };
}

/** Adapt a REST API WishListItem into the design system's WishlistEntry shape. */
export function toWishlistEntry(w: WishListItem): WishlistEntry {
  const sources = w.sources ?? [];
  return {
    id: w.id,
    name: w.name,
    type: w.itemType,
    rarity: RARITY_MAP[w.rarity] ?? "legendary",
    icon: w.icon || undefined,
    priority: PRIORITY_MAP[w.priority] ?? "medium",
    avail: {
      now: w.availableNow ?? false,
      where: w.availableNow
        ? (w.availableFrom ?? "Xûr")
        : (sources[0] ?? "Unknown source"),
    },
    notes: w.notes ?? "",
    added: relTime(w.dateAdded),
  };
}
