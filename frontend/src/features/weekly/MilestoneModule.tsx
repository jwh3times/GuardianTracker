import { Badge, EmptyState } from "../../components/primitives";
import { Panel } from "../../components/composite";
import type { Milestone } from "../../types/design";

export function MilestoneModule({ milestones }: { milestones: Milestone[] }) {
  return (
    <Panel title="Milestones & Activities" icon="week">
      {milestones.length === 0 ? (
        <EmptyState
          icon="week"
          title="No milestones this week"
          body="Bungie is not reporting any active weekly milestones right now."
        />
      ) : (
        <ul className="gt-vendor-list">
          {milestones.map((m) => (
            <li key={m.id} className="gt-milestone">
              <div className="gt-milestone-l">
                <div className="gt-action-meta mono">{m.label}</div>
                <div className="gt-item-name">{m.name}</div>
                <div className="gt-item-type">Reward: {m.reward}</div>
              </div>
              {m.missing != null && m.missing > 0 && (
                <Badge kind="missing" dot>
                  {m.missing} missing
                </Badge>
              )}
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}
