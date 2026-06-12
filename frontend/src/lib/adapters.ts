import type {
  APICharacter,
  APIDestinyItem,
  WishListItem,
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

/** Convert a UTC RFC3339 timestamp to a human-readable relative string ("3d ago", "just now"). */
export function relTime(iso: string): string {
  const t = new Date(iso).getTime();
  // Guard against empty/invalid input (NaN) and Go's zero time
  // ("0001-01-01T00:00:00Z" → a large negative epoch) so we never render
  // "NaNmo ago" or "24168mo ago".
  if (!iso || Number.isNaN(t) || t < 1_000_000_000_000) return "unknown";
  const diff = Date.now() - t;
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
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
