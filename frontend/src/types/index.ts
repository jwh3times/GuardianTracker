// Core Destiny 2 Types
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

export interface User {
  id: string;
  bungieId: string;
  displayName: string;
  membershipType: number;
  membershipId: string;
  profilePicture: string;
  lastLoginAt: string;
  createdAt: string;
  wishList?: WishListItem[];
}

export interface WishListItem extends DestinyItem {
  id: string;
  priority: WishListPriority;
  notes?: string;
  dateAdded: string;
}

export type WishListPriority = 'LOW' | 'MEDIUM' | 'HIGH' | 'URGENT';

export interface CollectionSummary {
  total: number;
  collected: number;
  missing: DestinyItem[];
}

export interface UserCollections {
  weapons: CollectionSummary;
  armor: CollectionSummary;
  exotics: CollectionSummary;
}

// Weekly Reset Content
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

// Search and Filtering
export interface ItemFilters {
  itemType?: string[];
  rarity?: string[];
  difficulty?: string[];
  isExotic?: boolean;
  isCollected?: boolean;
  sources?: string[];
}

// API Response Types
export interface ApiResponse<T> {
  success: boolean;
  message: string;
  data?: T;
}

export interface RefreshDataResponse {
  success: boolean;
  message: string;
  lastRefresh: string;
}

// UI Component Props Types
export interface ButtonProps {
  variant?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link';
  size?: 'default' | 'sm' | 'lg' | 'icon';
  className?: string;
  children: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  type?: 'button' | 'submit' | 'reset';
}

export interface CardProps {
  className?: string;
  children: React.ReactNode;
}

export interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

// Form Types
export interface WishListFormData {
  priority: WishListPriority;
  notes?: string;
}

export interface SearchFormData {
  query: string;
  filters: ItemFilters;
}

// Error Types
export interface ErrorBoundaryState {
  hasError: boolean;
  error?: Error;
}

// Local Storage Types
export interface StoredAuthData {
  token: string;
  expiresAt: string;
  user: User;
}
