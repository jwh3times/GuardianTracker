// Raw types that exactly mirror the JSON shapes returned by the Go api-service.
// Field names match the json struct tags in the Go source.

// --- Auth ---

export interface APIUser {
  id: string;
  displayName: string;
  membershipId: string;
  membershipType: number;
  platform?: string;
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
  sources: string[];
  isExotic: boolean;
}

/** CollectionSummary: totals + missing items for one category */
export interface APICollectionSummary {
  total: number;
  collected: number;
  missing: APIDestinyItem[];
  /** Present only when the request used ?include=all */
  collectedItems?: APIDestinyItem[];
}

/** GET /api/collections/:membershipType/:membershipId */
export interface APIUserCollections {
  weapons: APICollectionSummary;
  armor: APICollectionSummary;
  exotics: APICollectionSummary;
  cosmetics: APICollectionSummary;
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
  rarity: string;       // "Exotic", "Legendary", "Rare", "Uncommon", "Common"
  icon: string;         // bungie.net icon path, may be ""
  priority: string;     // "LOW", "MEDIUM", "HIGH", "URGENT"
  notes: string;
  sources: string[];
  availableNow: boolean;   // item is currently sold by Xûr (B6)
  availableFrom?: string;  // "Xûr" when availableNow
  dateAdded: string;    // RFC3339
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
  rarity: string;  // "Exotic", "Legendary", etc.
}

// --- Catalysts / Crafting / Seals ---
// Item shapes are identical to the design types; re-export them for use in
// fetch calls so callers can import from one place.

export type { Catalyst as APICatalyst, CraftPattern as APICraftPattern, Seal as APISeal } from "./design";

/** Envelope for GET /api/catalysts|crafting|seals — items + data freshness (B8) */
export interface APIRecordsEnvelope<T> {
  items: T[];
  fetchedAt: string; // RFC3339
}
