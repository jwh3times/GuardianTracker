// App-level types for items, wishlist, and future weekly-content features.
// Raw API wire types live in types/api.ts.

// DestinyItem with typed rarity/difficulty unions — used by lib/utils.ts helpers.
export interface DestinyItem {
  itemHash: string;
  name: string;
  description: string;
  icon: string;
  itemType: string;
  tierType: number;
  rarity: 'Common' | 'Uncommon' | 'Rare' | 'Legendary' | 'Exotic';
  difficulty: 'Easy' | 'Moderate' | 'Challenging';
  sources: string[];
  isExotic: boolean;
  isCollected?: boolean;
}

// --- Wishlist (DB persistence layer, not yet wired) ---

export interface WishListItem extends DestinyItem {
  id: string;
  priority: WishListPriority;
  notes?: string;
  dateAdded: string;
}

export type WishListPriority = 'LOW' | 'MEDIUM' | 'HIGH' | 'URGENT';

// --- Weekly content (backend not yet implemented) ---

export interface VendorItem {
  itemHash: string;
  name: string;
  description: string;
  icon: string;
  cost: number;
  currency: string;
  isLimited: boolean;
  endsAt?: string;
}

export interface Vendor {
  vendorHash: string;
  vendorName: string;
  location: string;
  items: VendorItem[];
}

export interface ActivityReward {
  itemHash: string;
  name: string;
  description: string;
  icon: string;
  dropChance?: number;
}

export interface Activity {
  activityHash: string;
  activityName: string;
  activityType: string;
  difficulty: string;
  rewards: ActivityReward[];
  isWeeklyFeatured: boolean;
  endsAt?: string;
}

export interface PursuitObjective {
  description: string;
  progress: number;
  completionValue: number;
  isCompleted: boolean;
}

export interface Pursuit {
  itemHash: string;
  name: string;
  description: string;
  icon: string;
  objectives: PursuitObjective[];
  rewards: DestinyItem[];
  expirationDate?: string;
}

export interface WeeklyRecommendations {
  vendors: Vendor[];
  activities: Activity[];
  pursuits: Pursuit[];
}

// --- Search / filtering (search backend not yet implemented) ---

export interface ItemFilters {
  itemType?: string[];
  rarity?: string[];
  difficulty?: string[];
  isExotic?: boolean;
  isCollected?: boolean;
  sources?: string[];
}
