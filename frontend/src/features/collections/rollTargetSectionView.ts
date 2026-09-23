import type { RollTarget, RollTargetMatch } from "../../types/design";

/**
 * The pure pieces behind `RollTargetSection.tsx`, the item detail drawer's
 * per-weapon roll-target block (roll targets slice 6). Kept separate from the
 * component so the any-weapon exclusion rule and the by-copy grouping are
 * table-tested directly, mirroring `features/rolltargets/rollTargetsView.ts`'s
 * split of pure logic from the hook that wires it to live data. None of this
 * imports React Query or `types/api`.
 *
 * This module deliberately does not reuse `rollTargetsView.ts`'s
 * `groupMatchesByTarget`: that groups by *target* (one card per target,
 * copies listed underneath), which is the right shape for the Roll targets
 * page's multi-weapon view. Scoped to one weapon, a copy can satisfy more
 * than one saved target (e.g. a specific-weapon target and an any-weapon
 * target both matching the same owned roll), so grouping by *copy* — one row
 * per owned instance, badged with every target it satisfies — is the shape
 * that avoids re-numbering the same physical copy under two different
 * "Copy 1" labels.
 */

/**
 * This weapon's own weapon-bound targets: `itemHash === itemHash`. An
 * any-weapon target's `itemHash` is `null`, so it is excluded by this
 * comparison alone — no separate `anyWeapon` check is needed. Used both for
 * "Still chasing" (filtering `unmatchedTargets`) and for the neutral
 * match-failure fallback (filtering the saved-target list), since both are
 * `RollTarget[]`.
 */
export function weaponBoundTargetsFor(
  targets: RollTarget[],
  itemHash: string,
): RollTarget[] {
  return targets.filter((t) => t.itemHash === itemHash);
}

/**
 * Match-report entries for this weapon. A match's `itemHash` is always the
 * *owned copy's* hash, so this single equality check is also how an
 * any-weapon target that matched one of this weapon's copies is included —
 * no separate any-weapon branch is needed here either.
 */
export function matchesForItem(
  matches: RollTargetMatch[],
  itemHash: string,
): RollTargetMatch[] {
  return matches.filter((m) => m.itemHash === itemHash);
}

/** One target this owned copy satisfies (or, in the unwanted list, matches). */
export interface SatisfiedTarget {
  targetId: string;
  /** Short identifying label for the badge — the any-weapon phrasing for an
   * any-weapon target, else its own saved perks. */
  label: string;
}

export interface CopyMatchGroup {
  instanceId: string;
  /** "Copy 1", "Copy 2", … in order of first appearance. */
  label: string;
  /** The owned copy's full resolved perks. */
  perks: string[];
  /** Union of every satisfied target's own perks, for highlighting `perks`. */
  targetPerks: string[];
  satisfiedTargets: SatisfiedTarget[];
}

function targetLabel(
  target: RollTarget | undefined,
  fallback: string[],
): string {
  if (target?.anyWeapon) {
    return `Any weapon: ${target.perks.join(" + ")}`;
  }
  return (target?.perks ?? fallback).join(" + ");
}

/**
 * Group this weapon's matches by owned copy (`instanceId`), not by target.
 * `targetsById` resolves each match's target for its badge label and
 * any-weapon phrasing; a match whose target id is not in the map (a race
 * right after a refresh) falls back to the match's own `targetPerks`.
 */
export function groupMatchesByCopy(
  matches: RollTargetMatch[],
  targetsById: Map<string, RollTarget>,
): CopyMatchGroup[] {
  const byInstance = new Map<string, RollTargetMatch[]>();
  const order: string[] = [];
  for (const m of matches) {
    if (!byInstance.has(m.instanceId)) {
      byInstance.set(m.instanceId, []);
      order.push(m.instanceId);
    }
    byInstance.get(m.instanceId)!.push(m);
  }

  return order.map((instanceId, i) => {
    const ms = byInstance.get(instanceId)!;
    const targetPerks = Array.from(new Set(ms.flatMap((m) => m.targetPerks)));
    const satisfiedTargets = ms.map((m) => ({
      targetId: m.targetId,
      label: targetLabel(targetsById.get(m.targetId), m.targetPerks),
    }));
    return {
      instanceId,
      label: `Copy ${i + 1}`,
      perks: ms[0].perks,
      targetPerks,
      satisfiedTargets,
    };
  });
}
