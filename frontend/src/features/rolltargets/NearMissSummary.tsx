import type { RollTargetNearMiss } from "../../types/design";
import { isTargetPerk } from "./rollTargetsView";

/**
 * Presents the backend-selected best partial copy. It deliberately performs no
 * scoring: matchedColumns/targetColumns are already the match report's
 * decision, and matchedPerks only identifies the chips to highlight.
 */
export function NearMissSummary({
  bestCopy,
}: {
  bestCopy?: RollTargetNearMiss;
}) {
  if (!bestCopy || bestCopy.matchedColumns === 0) return null;

  return (
    <div className="gt-rt-near-miss" role="note">
      <div className="gt-action-meta">
        Your best copy has {bestCopy.matchedColumns} of {bestCopy.targetColumns}{" "}
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
