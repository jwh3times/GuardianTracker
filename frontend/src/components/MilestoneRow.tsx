import { Badge } from "./primitives";
import type { Milestone } from "../types/design";

/**
 * One weekly milestone row. Shared because two features render it: the This
 * Week milestone module, and the Dashboard's two-item preview — which is why
 * this is not part of either. Only the row is common; the panel around it
 * differs (title, item cap, loading skeleton), so the modules keep their own.
 *
 * The caller supplies the React `key`, as it owns the list.
 */
export function MilestoneRow({ milestone }: { milestone: Milestone }) {
  return (
    <li className="gt-milestone">
      <div className="gt-milestone-l">
        <div className="gt-action-meta mono">{milestone.label}</div>
        <div className="gt-item-name">{milestone.name}</div>
        <div className="gt-item-type">Reward: {milestone.reward}</div>
      </div>
      {milestone.missing != null && milestone.missing > 0 && (
        <Badge kind="missing" dot>
          {milestone.missing} missing
        </Badge>
      )}
    </li>
  );
}
