// Design-system domain types (Guardian Tracker redesign).
// These describe the shape the new UI kit renders. Real GraphQL data is
// adapted into `GTItem` before being handed to the components.

export type Rarity = "exotic" | "legendary" | "rare" | "uncommon" | "common";
export type Difficulty = "easy" | "moderate" | "challenging" | "unrated";
export type Priority = "urgent" | "high" | "medium" | "low";
export type CatalystStatus = "missing" | "in-progress" | "complete";

/** A weapon perk column for the item drawer (same shape as APIPerkColumn). */
export interface PerkColumn {
  role: string;
  label: string;
  perks: string[];
}

/** A weapon catalyst entry for the item drawer (same shape as APIItemCatalyst). */
export interface ItemCatalyst {
  name: string;
  description: string;
}

/** One collectible-derived provenance attribution for an item. */
export interface AcquisitionSource {
  text: string;
  difficulty: Difficulty;
  raidDungeon: boolean;
}

export interface GTItem {
  id: string;
  name: string;
  type: string;
  slot: string;
  rarity: Rarity;
  /** Random-roll item that can't be pulled from Collections — must be farmed. */
  farmOnly?: boolean;
  /** Non-collectible item opened read-only via deep-link (no collection/wishlist state). */
  viewOnly?: boolean;
  acquisitionSources: AcquisitionSource[];
  availableNow: boolean;
  /** Vendor currently selling this item ("Banshee-44", "Xûr", …), when available now. */
  availFrom?: string;
  collected: boolean;
  desc: string;
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
  /** Full URL to the character's wide emblem background. */
  emblemBackgroundUrl?: string;
}

export type EquipmentGroup = "Weapons" | "Armor" | "Equipment";

export interface EquippedItem {
  id: string;
  slot: string;
  group: EquipmentGroup;
  name: string;
  type: string;
  rarity?: Rarity;
  icon?: string;
  power?: number;
}

export interface GuardianEquipment {
  characterId: string;
  state: "ready" | "unavailable";
  items: EquippedItem[];
  fetchedAt: string;
}

export interface GuardianActivity {
  activityHash: string;
  name: string;
  occurredAt?: string;
  duration?: string;
  privateMatch: boolean;
  resolved: boolean;
}

export interface GuardianActivityHistory {
  characterId: string;
  state: "ready" | "unavailable";
  activities: GuardianActivity[];
  fetchedAt: string;
}

/**
 * A Guardian's best-effort current activity. `idle` is only Bungie's zero
 * activity; data Bungie did not return is `unavailable`, and an activity the
 * Manifest cannot name is `unknown`.
 */
export interface GuardianCurrentActivity {
  state: "ready" | "idle" | "unavailable" | "unknown";
  activityName?: string;
  modeName?: string;
  playlistName?: string;
  fetchedAt: string;
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
  hash: string;
  name: string;
  type: string;
  /** Bungie CDN icon path; empty when the manifest has no icon. */
  icon: string;
  rarity: Rarity;
  missing: boolean;
  cost: string;
  /** Present only for class-specific armor verified from the manifest. */
  className?: string;
}

export interface Xur {
  present: boolean;
  leavesIn: Duration;
  /** Omitted when Bungie does not provide a resolvable live vendor location. */
  location?: string;
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

export interface RecommendedAction {
  id: string;
  text: string;
  detail: string;
  badge: string;
  done: boolean;
  diff: Difficulty;
  time: string;
}

export interface TodayAction {
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
  recommended: RecommendedAction[];
  /** Wire name retained for API compatibility; entries may expire on non-daily cadences. */
  dailyActions: TodayAction[];
}

export interface WishlistEntry {
  /** The wish list row id, not the item's. Addresses this row's REST endpoint. */
  id: string;
  /**
   * The wished item's hash, matching {@link GTItem.id}. Lets Collections match
   * its tiles against the wish list without reading the wire shape.
   */
  itemId: string;
  name: string;
  type: string;
  rarity: Rarity;
  /** Bungie icon path; absent when the manifest had no definition. */
  icon?: string;
  priority: Priority;
  avail: { now: boolean; where: string };
  acquisitionSources: AcquisitionSource[];
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
  /** Catalyst perk/effect description from the manifest; may be empty. */
  effect?: string;
}

export interface CraftPattern {
  id: string;
  name: string;
  type: string;
  /** Weapon icon path on bungie.net (the pattern record's own icon); may be "". */
  icon?: string;
  patterns: { cur: number; max: number };
  note: string;
  source: string;
}

export interface TriumphObjective {
  label: string;
  done: boolean;
  cur: number;
  max: number;
}

export interface Triumph {
  label: string;
  done: boolean;
  cur: number;
  max: number;
  /** Per-objective drill-down for the triumph's progress bar; absent (not an
   * empty array) when the triumph has no objective data. */
  objectives?: TriumphObjective[];
}

export interface Seal {
  id: string;
  name: string;
  pct: number;
  gilded: number;
  left: string;
  triumphs: Triumph[];
}

export type DigestStatus = "ready" | "first-visit" | "unavailable";

/** One collectible acquired since the membership's previous visit (ADR 0023). */
export interface AcquiredItem {
  itemHash: number;
  name: string;
  /** Bungie icon path; absent when the manifest had no definition. */
  icon?: string;
  type: string;
}

/**
 * The since-last-visit digest (ADR 0023). `status` is a real branch, not
 * inferred from `acquired`'s length: `first-visit` means there is no prior
 * snapshot to diff against, `unavailable` means no digest could be computed
 * this visit (no persistence, a failed read, or a private profile), and only
 * `ready` may legitimately report a genuinely empty `acquired` list.
 */
export interface Digest {
  status: DigestStatus;
  /** When the current visit began (RFC3339). */
  visitStartedAt: string;
  /** When the previous visit ended; absent on a first visit. */
  previousVisitAt?: string;
  acquired: AcquiredItem[];
}

/**
 * A player's own saved perk combination wanted on a specific weapon (or, when
 * `anyWeapon`, on any weapon that can roll it) — CONTEXT.md's "Roll target".
 * Distinct from a {@link WishlistEntry}, which wants an item hash with no perk
 * dimension.
 */
export interface RollTarget {
  id: string;
  /** The design-system item id (a hash string); null for an any-weapon target. */
  itemHash: string | null;
  anyWeapon: boolean;
  /** False for a roll the player has flagged as one to avoid, not to chase. */
  wanted: boolean;
  perks: string[];
  notes: string;
  /** RFC3339, left raw — formatting it here would bake a timestamp into a memo. */
  dateAdded: string;
}

/** Why one DIM import line's perk hash did not resolve into a saved perk name. */
export interface UnresolvedPerk {
  hash: number;
  /** Absent when the hash's own display name could not be resolved either. */
  name?: string;
  reason: string;
}

/** One DIM-format import line's fate, in file order. */
export interface RollTargetImportLine {
  line: number;
  outcome: string;
  detail?: string;
  unresolved?: UnresolvedPerk;
  /** The design-system item id; absent for an any-weapon line. */
  itemHash?: string;
  wanted: boolean;
  perks?: string[];
}

/** The result of importing one DIM-format file — CONTEXT.md's "DIM import report". */
export interface RollTargetImportReport {
  title?: string;
  description?: string;
  imported: number;
  counts: Record<string, number>;
  lines: RollTargetImportLine[];
}

/** One owned weapon satisfying (or matching, if unwanted) one saved roll target. */
export interface RollTargetMatch {
  targetId: string;
  /** The design-system item id of the owned copy that matched. */
  itemHash: string;
  instanceId: string;
  /** The owned copy's full resolved perks. */
  perks: string[];
  /** The target's own saved perks, for highlighting which of `perks` matched. */
  targetPerks: string[];
  notes?: string;
}

/** The highest-scoring partial owned copy for a still-chasing target. */
export interface RollTargetNearMiss {
  itemHash: string;
  instanceId: string;
  /** The owned copy's full resolved perks. */
  perks: string[];
  /** Target perks present on the copy; selected by the backend scoring rule. */
  matchedPerks: string[];
}

/** A saved target nothing fully satisfies, with its best copy when one exists. */
export interface UnmatchedRollTarget extends RollTarget {
  bestCopy?: RollTargetNearMiss;
}

/**
 * The result of matching saved roll targets against owned inventory —
 * CONTEXT.md's "Match report". `unmatchedTargets` is CONTEXT.md's "Unmatched
 * target": a saved roll nothing owned currently satisfies, still worth
 * chasing, never rendered as absence.
 */
export interface RollTargetMatchReport {
  wanted: RollTargetMatch[];
  unwanted: RollTargetMatch[];
  unmatchedTargets: UnmatchedRollTarget[];
}
