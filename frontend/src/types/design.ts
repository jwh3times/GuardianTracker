// Design-system domain types (Guardian Tracker redesign).
// These describe the shape the new UI kit renders. Real GraphQL data is
// adapted into `GTItem` before being handed to the components.

export type Rarity = "exotic" | "legendary" | "rare" | "uncommon" | "common";
export type Difficulty = "easy" | "moderate" | "challenging";
export type Priority = "urgent" | "high" | "medium" | "low";
export type CatalystStatus = "missing" | "in-progress" | "complete";

export interface GTItem {
  id: string;
  name: string;
  type: string;
  slot: string;
  rarity: Rarity;
  diff: Difficulty;
  source: string;
  sourceDetail: string;
  obtainable: boolean;
  collected: boolean;
  desc: string;
  perks: string[];
  /** Real Bungie icon path (relative or absolute), when available. */
  icon?: string;
}

export interface Character {
  id: string;
  name: string;
  cls: string;
  race: string;
  power: number;
  emblem: number;
  /** Full URL to the character's emblem icon (real data only; mock omits it). */
  emblemUrl?: string;
}

export interface TreeNode {
  id: string;
  label: string;
  pct: number;
  count: [number, number];
  children?: TreeNode[];
}

export interface SummaryCategory {
  id: string;
  label: string;
  pct: number;
  count: [number, number];
}

export interface Summary {
  overall: number;
  categories: SummaryCategory[];
  totals: { collected: number; total: number; missing: number };
  updatedAgo: string;
}

export interface Duration {
  d?: number;
  h?: number;
  m?: number;
}

export interface XurItem {
  name: string;
  type: string;
  rarity: Rarity;
  missing: boolean;
  cost: string;
}

export interface Xur {
  present: boolean;
  leavesIn: Duration;
  location: string;
  items: XurItem[];
}

export interface Milestone {
  id: string;
  label: string;
  name: string;
  reward: string;
  /** Absent until the backend computes real per-milestone completion (B9). */
  missing?: number;
  note: string;
}

export interface VendorRotation {
  id: string;
  name: string;
  role: string;
  missing: number;
  items: string[];
}

export interface RecommendedAction {
  id: string;
  text: string;
  detail: string;
  badge: string;
  done: boolean;
  diff: Difficulty;
  time: string;
}

export interface DailyAction {
  id: string;
  category: "milestone" | "xur" | "vendor" | "activity";
  icon: string;
  text: string;
  detail: string;
  badge: string;
  resetsIn: Duration;
  done: boolean;
}

export interface Weekly {
  resetLabel: string;
  resetIn: Duration;
  dailyResetIn: Duration;
  /** Next weekly reset (RFC3339) — keys checkmark persistence. */
  resetAt: string;
  /** When the underlying Bungie data was fetched (RFC3339, B8). */
  fetchedAt: string;
  /** True when names/labels are placeholders because the manifest is still downloading. */
  degraded?: boolean;
  xur: Xur | null; // null when Xûr is not in town
  milestones: Milestone[];
  vendors: VendorRotation[];
  recommended: RecommendedAction[];
  dailyActions: DailyAction[];
}

export interface WishlistEntry {
  id: string;
  name: string;
  type: string;
  rarity: Rarity;
  /** Bungie icon path; absent when the manifest had no definition. */
  icon?: string;
  priority: Priority;
  avail: { now: boolean; where: string };
  notes: string;
  added: string;
}

export interface Catalyst {
  id: string;
  name: string;
  /** Weapon type from the manifest (e.g. "Hand Cannon"); "" when unresolvable. */
  type?: string;
  /** Record icon path on bungie.net; may be "". */
  icon?: string;
  status: CatalystStatus;
  obj: { label: string; cur: number; max: number } | null;
  source: string;
}

export interface CraftPattern {
  id: string;
  name: string;
  type: string;
  patterns: { cur: number; max: number };
  note: string;
  source: string;
}

export interface Triumph {
  label: string;
  done: boolean;
  cur: number;
  max: number;
}

export interface Seal {
  id: string;
  name: string;
  pct: number;
  gilded: number;
  left: string;
  triumphs: Triumph[];
}
