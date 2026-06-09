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
}

/** GET /api/collections/:membershipType/:membershipId */
export interface APIUserCollections {
  weapons: APICollectionSummary;
  armor: APICollectionSummary;
  exotics: APICollectionSummary;
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
