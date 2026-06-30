import type { Difficulty, Priority, Rarity } from "../types/design";

/** Base URL for Bungie CDN assets — item icon paths are relative to this. */
export const BUNGIE_CDN = "https://www.bungie.net";

export const RARITIES: Rarity[] = [
  "exotic",
  "legendary",
  "rare",
  "uncommon",
  "common",
];
export const DIFFS: Difficulty[] = [
  "easy",
  "moderate",
  "challenging",
  "unrated",
];

export const RARITY_LABEL: Record<Rarity, string> = {
  exotic: "Exotic",
  legendary: "Legendary",
  rare: "Rare",
  uncommon: "Uncommon",
  common: "Common",
};
export const DIFF_LABEL: Record<Difficulty, string> = {
  easy: "Easy",
  moderate: "Moderate",
  challenging: "Challenging",
  unrated: "Unrated",
};
export const PRIORITY_LABEL: Record<Priority, string> = {
  urgent: "Urgent",
  high: "High",
  medium: "Medium",
  low: "Low",
};
export const TYPE_GLYPH: Record<string, string> = {
  "Auto Rifle": "AR",
  "Hand Cannon": "HC",
  "Pulse Rifle": "PR",
  "Scout Rifle": "SC",
  "Fusion Rifle": "FR",
  "Sniper Rifle": "SN",
  Shotgun: "SG",
  Sidearm: "SD",
  "Submachine Gun": "SMG",
  Bow: "BOW",
  "Rocket Launcher": "RL",
  "Grenade Launcher": "GL",
  Sword: "SWD",
  Glaive: "GLV",
  "Machine Gun": "MG",
  "Linear Fusion": "LFR",
  "Trace Rifle": "TR",
  Helmet: "HEL",
  Gauntlets: "ARM",
  "Chest Armor": "CHS",
  "Leg Armor": "LEG",
  "Class Item": "CLS",
  "Ghost Shell": "GHO",
  Sparrow: "SPR",
  Ship: "SHP",
  Shader: "SHD",
  Emblem: "EMB",
  Emote: "EMT",
};
