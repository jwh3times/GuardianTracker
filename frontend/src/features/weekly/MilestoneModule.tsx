import { EmptyState } from "../../components/primitives";
import { Panel } from "../../components/composite";
import { MilestoneRow } from "../../components/MilestoneRow";
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
            <MilestoneRow key={m.id} milestone={m} />
          ))}
        </ul>
      )}
    </Panel>
  );
}
