// Raw types that exactly mirror the JSON shapes returned by the Go api-service.
// Field names match the json struct tags in the Go source.

// --- Auth ---

export interface APIUser {
  id: string;
  displayName: string;
  membershipId: string;
  membershipType: number;
  platform?: string;
  /** Role tier as a display hint; authoritative role comes from GET /api/flags. */
  role?: "standard" | "beta" | "alpha" | "admin";
}

/** GET /api/auth/bungie */
export interface AuthURLResponse {
  authUrl: string;
  state: string;
}

/** POST /api/auth/bungie/callback  |  POST /api/auth/refresh */
export interface AuthTokenResponse {
  token: string;
  refreshToken: string;
  user: APIUser;
}

/** GET /api/auth/profile  |  GET /api/auth/validate */
export interface ProfileResponse {
  user: APIUser;
}

// --- Collections: mirrors services/collections/service.go ---

/** DestinyItem as serialised by the collections service */
export interface APIDestinyItem {
  itemHash: string;
  name: string;
  description: string;
  icon: string;
  itemType: string;
  tierType: number;
  rarity: string;
  difficulty: string;
  farmOnly?: boolean;
  sources: string[];
  isExotic: boolean;
}

/** GET /api/items/:itemHash — minimal manifest item view (mirrors manifest.ItemView). */
export interface APIItemView {
  itemHash: string;
  name: string;
  icon: string;
  itemType: string;
  tierType: number;
  rarity: string;
  description: string;
}

/** One perk column in a weapon's possible-roll pool (mirrors manifest.PerkColumn). */
export interface APIPerkColumn {
  role: string; // intrinsic | barrel | magazine | trait | origin
  label: string; // "Intrinsic", "Barrel", "Trait 1", …
  perks: string[];
}

/** One catalyst entry attached to a weapon (mirrors manifest catalyst records). */
export interface APIItemCatalyst {
  name: string;
  description: string; // may be empty — the manifest has at least one blank entry (Duality)
}

/** GET /api/items/:itemHash/perks */
export interface APIItemPerks {
  itemHash: string;
  perkColumns: APIPerkColumn[];
  /** Always present; empty array for non-exotics. Up to 4 entries for multi-catalyst exotics. */
  catalysts: APIItemCatalyst[];
}

/** One presentation node in the collections tree (mirrors CollectionNode). */
export interface APICollectionNode {
  hash: string;
  name: string;
  icon: string;
  collected: number;
  total: number;
  children?: APICollectionNode[];
  /** Item hashes of this node's direct leaf collectibles; present only on ?include=all */
  items?: string[];
}

/** Total/collected for one summary bucket. */
export interface APICategoryCount {
  total: number;
  collected: number;
}

/** Derived four-bucket rollup for the Dashboard hero / weekly. */
export interface APICategorySummary {
  weapons: APICategoryCount;
  armor: APICategoryCount;
  exotics: APICategoryCount;
  cosmetics: APICategoryCount;
}

/** GET /api/collections/:membershipType/:membershipId */
export interface APIUserCollections {
  tree: APICollectionNode[];
  /** hash→detail map; present only on ?include=all */
  items?: Record<string, APIDestinyItem>;
  /** item hashes the user owns; present only on ?include=all */
  collectedHashes?: string[];
  /** itemHash → vendor name for items obtainable right now; present only on ?include=all */
  availableNow?: Record<string, string>;
  summary: APICategorySummary;
  /** When this data was fetched from Bungie (RFC3339, B8) */
  fetchedAt: string;
}

// --- Characters: mirrors services/characters/service.go ---

/** GET /api/characters/:membershipType/:membershipId returns Character[] */
export interface APICharacter {
  characterId: string;
  classType: number;
  className: string;
  raceName: string;
  light: number;
  emblemPath: string;
  emblemBackgroundPath: string;
  dateLastPlayed: string;
}

// --- Misc ---

/** POST /api/collections/:type/:id/refresh */
export interface APICacheRefreshResponse {
  success: boolean;
  message: string;
}

// --- Wishlist ---

/** One item returned by GET/POST/PUT /api/wishlist */
export interface WishListItem {
  id: string;
  itemHash: number;
  name: string;
  itemType: string;
  rarity: string; // "Exotic", "Legendary", "Rare", "Uncommon", "Common"
  icon: string; // bungie.net icon path, may be ""
  priority: string; // "LOW", "MEDIUM", "HIGH", "URGENT"
  notes: string;
  sources: string[];
  availableNow: boolean; // item is currently sold by Xûr (B6)
  availableFrom?: string; // "Xûr" when availableNow
  dateAdded: string; // RFC3339
}

/** GET/PUT /api/preferences */
export interface APIPreferences {
  cardStyle: "framed" | "compact";
  personalize: boolean;
}

/** GET /api/items/search?q=<term>&limit=20 */
export interface APISearchResult {
  hash: number;
  name: string;
  icon: string;
  type: string;
  rarity: string; // "Exotic", "Legendary", etc.
}

// --- Catalysts / Crafting / Seals ---
// Item shapes are identical to the design types; re-export them for use in
// fetch calls so callers can import from one place.

export type {
  Catalyst as APICatalyst,
  CraftPattern as APICraftPattern,
  Seal as APISeal,
} from "./design";

/** Envelope for GET /api/catalysts|crafting|seals — items + data freshness (B8) */
export interface APIRecordsEnvelope<T> {
  items: T[];
  fetchedAt: string; // RFC3339
}

// --- Roles & feature flags (item 13) ---

import type { Role } from "../lib/roles";

/** One resolved flag from GET /api/flags — the design's flagState shape. */
export interface APIResolvedFlag {
  key: string;
  name: string;
  desc: string;
  category: string;
  minTier: string; // "standard" | "beta" | "alpha"
  enabled: boolean;
  accessible: boolean;
  locked: boolean;
}

/** GET /api/flags */
export interface APIFlagsResponse {
  role: Role;
  flags: APIResolvedFlag[];
}

/** PUT /api/account/role */
export interface APIRoleResponse {
  role: Role;
}

/** GET /api/admin/users */
export interface APIAdminUser {
  id: string;
  displayName: string;
  membershipId: string;
  platform: string;
  role: Role;
  lastActive: string; // RFC3339
}

/** GET/PUT /api/admin/flags */
export interface APIAdminFlag {
  key: string;
  name: string;
  description: string;
  category: string;
  minTier: string; // "standard" | "beta" | "alpha"
  enabled: boolean;
  updatedAt: string;
}

// --- Audit log (Task 10) ---

export interface APIAuditParty {
  membershipId: string;
  displayName: string;
}

export interface APIAuditEntry {
  id: string;
  eventType: string;
  outcome: "success" | "failure";
  actor: APIAuditParty;
  target?: APIAuditParty;
  ip?: string;
  userAgent?: string;
  details: Record<string, unknown>;
  createdAt: string;
}

export interface APIAuditPage {
  entries: APIAuditEntry[];
  nextCursor: string;
}
