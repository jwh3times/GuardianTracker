import { relTime } from "./format";
import { toAcquisitionSource } from "./acquisitionSources";
import type { APICharacter, APIItemView, WishListItem } from "../types/api";
import type {
  Character,
  GTItem,
  Priority,
  Rarity,
  WishlistEntry,
} from "../types/design";

/**
 * Shared rarity vocabulary. Exported because the Wish list data-access module
 * owns its own projection now (ADR 0020) but still maps the same five wire
 * names. The Characters and Items slices (E7, E8) retire the last local
 * callers, at which point this belongs with them rather than here.
 */
export const RARITY_MAP: Record<string, Rarity> = {
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
    farmOnly: false,
    acquisitionSources: [],
    availableNow: false,
    collected: false,
    desc: v.description ?? "",
    icon: v.icon,
    viewOnly: true,
  };
}

