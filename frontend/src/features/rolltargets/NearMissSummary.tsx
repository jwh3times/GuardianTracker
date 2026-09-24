import type { RollTargetNearMiss } from "../../types/design";
import { isTargetPerk } from "./rollTargetsView";

/**
 * Presents the backend-selected best partial copy. It deliberately performs no
 * scoring: matchedPerks is already the match report's decision, and this
 * component only renders its count and evidence.
 */
export function NearMissSummary({
  bestCopy,
  targetPerkCount,
}: {
  bestCopy?: RollTargetNearMiss;
  targetPerkCount: number;
}) {
  if (!bestCopy || bestCopy.matchedPerks.length === 0) return null;

  return (
    <div className="gt-rt-near-miss" role="note">
      <div className="gt-action-meta">
        Your best copy has {bestCopy.matchedPerks.length} of {targetPerkCount}{" "}
        target perks.
      </div>
      <div className="gt-rt-perks">
        {bestCopy.perks.map((perk) => (
          <span
            key={perk}
            className="gt-chip"
            data-highlight={isTargetPerk(perk, bestCopy.matchedPerks)}
          >
            {perk}
          </span>
        ))}
      </div>
    </div>
  );
}
