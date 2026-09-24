import type {
  RollTarget,
  RollTargetMatch,
  UnmatchedRollTarget,
} from "../../types/design";

/**
 * The pure grouping/filtering pieces behind the Roll targets page (mirrors
 * `features/collections/useCollectionsBrowser.ts`'s split of pure, directly
 * tested logic from the hook that wires it to live data). None of this
 * imports React Query or `types/api` — it operates only on the domain types
 * this feature is allowed to see.
 */

/** How the page resolves a weapon's display facts and acquired/wishlisted
 * status for one item hash. Supplied by the page so this module stays a pure
 * function of its inputs — resolving it here would require reaching into
 * `data/collections.ts` and `data/wishlist.ts` from a feature module. */
export interface WeaponLookup {
  name(itemHash: string): string;
  icon(itemHash: string): string | undefined;
  type(itemHash: string): string | undefined;
  /** True when the collectible is acquired, or the item is on the wish list. */
  acquiredOrWishlisted(itemHash: string): boolean;
}

/** Fallback name for a hash the lookup could not resolve — still a usable
 * label, never a blank row. */
export function fallbackWeaponName(itemHash: string): string {
  return `Item ${itemHash}`;
}

export interface ChasingGroup {
  /** The item hash, or "any" for the any-weapon bucket. */
  key: string;
  anyWeapon: boolean;
  name: string;
  icon?: string;
  type?: string;
  targets: UnmatchedRollTarget[];
}

/** Group targets by the weapon they name; any-weapon targets share one
 * "Any weapon" bucket regardless of their individual perks. */
export function groupChasingTargets(
  targets: UnmatchedRollTarget[],
  lookup: WeaponLookup,
): ChasingGroup[] {
  const groups = new Map<string, ChasingGroup>();
  for (const t of targets) {
    const key = t.anyWeapon || !t.itemHash ? "any" : t.itemHash;
    let group = groups.get(key);
    if (!group) {
      group =
        key === "any"
          ? { key, anyWeapon: true, name: "Any weapon", targets: [] }
          : {
              key,
              anyWeapon: false,
              name: lookup.name(key) || fallbackWeaponName(key),
              icon: lookup.icon(key),
              type: lookup.type(key),
              targets: [],
            };
      groups.set(key, group);
    }
    group.targets.push(t);
  }
  return Array.from(groups.values()).sort((a, b) => {
    if (a.anyWeapon !== b.anyWeapon) return a.anyWeapon ? 1 : -1;
    return a.name.localeCompare(b.name);
  });
}

/**
 * The default filter: only weapons whose collectible is acquired or that are
 * on the wish list; any-weapon targets always pass. `hiddenCount` counts
 * individual targets, not groups, so the toggle's copy can say exactly how
 * many rolls the filter is hiding.
 */
export function applyDefaultFilter(
  groups: ChasingGroup[],
  opts: { showAll: boolean; lookup: WeaponLookup },
): { groups: ChasingGroup[]; hiddenCount: number } {
  if (opts.showAll) return { groups, hiddenCount: 0 };
  let hiddenCount = 0;
  const out: ChasingGroup[] = [];
  for (const group of groups) {
    if (group.anyWeapon || opts.lookup.acquiredOrWishlisted(group.key)) {
      out.push(group);
    } else {
      hiddenCount += group.targets.length;
    }
  }
  return { groups: out, hiddenCount };
}

/** The search box: matches the weapon/group name, or any target's perk name. */
export function applySearch(
  groups: ChasingGroup[],
  term: string,
): ChasingGroup[] {
  const q = term.trim().toLowerCase();
  if (!q) return groups;
  const out: ChasingGroup[] = [];
  for (const group of groups) {
    const nameMatches = group.name.toLowerCase().includes(q);
    const targets = nameMatches
      ? group.targets
      : group.targets.filter((t) =>
          t.perks.some((p) => p.toLowerCase().includes(q)),
        );
    if (targets.length > 0) out.push({ ...group, targets });
  }
  return out;
}

export interface MatchGroup {
  /** The resolved target this group of copies satisfies (or would). */
  target: RollTarget;
  /** "Any weapon with X + Y" for an any-weapon target, else the weapon name. */
  heading: string;
  icon?: string;
  type?: string;
  copies: RollTargetMatch[];
}

/**
 * Build one card per distinct target's copies. `targetsById` resolves the
 * target's own saved perks/notes for the notes editor and delete action;
 * `wantedDefault` only matters for the rare defensive fallback below.
 */
export function groupMatchesByTarget(
  matches: RollTargetMatch[],
  targetsById: Map<string, RollTarget>,
  lookup: WeaponLookup,
  wantedDefault: boolean,
): MatchGroup[] {
  const byTarget = new Map<string, RollTargetMatch[]>();
  const order: string[] = [];
  for (const m of matches) {
    if (!byTarget.has(m.targetId)) {
      byTarget.set(m.targetId, []);
      order.push(m.targetId);
    }
    byTarget.get(m.targetId)!.push(m);
  }

  return order.map((targetId) => {
    const copies = byTarget.get(targetId)!;
    const target =
      targetsById.get(targetId) ??
      fallbackTarget(targetId, copies[0], wantedDefault);
    const heading = target.anyWeapon
      ? `Any weapon with ${target.perks.join(" + ")}`
      : lookup.name(target.itemHash ?? copies[0].itemHash) ||
        fallbackWeaponName(target.itemHash ?? copies[0].itemHash);
    const iconHash = target.anyWeapon ? copies[0].itemHash : target.itemHash;
    return {
      target,
      heading,
      icon: iconHash ? lookup.icon(iconHash) : undefined,
      type: iconHash ? lookup.type(iconHash) : undefined,
      copies,
    };
  });
}

/** Defensive fallback for a match whose target id is not (yet) in the target
 * list cache — e.g. a race right after a refresh. Keeps the group rendering
 * something reasonable rather than throwing. */
function fallbackTarget(
  targetId: string,
  sample: RollTargetMatch,
  wanted: boolean,
): RollTarget {
  return {
    id: targetId,
    itemHash: sample.itemHash,
    anyWeapon: false,
    wanted,
    perks: sample.targetPerks,
    notes: sample.notes ?? "",
    dateAdded: "",
  };
}

/** True when `perkName` is one of the target's own saved perks — used to
 * highlight which of an owned copy's perks satisfy the target. */
export function isTargetPerk(perkName: string, targetPerks: string[]): boolean {
  return targetPerks.some((p) => p.toLowerCase() === perkName.toLowerCase());
}
