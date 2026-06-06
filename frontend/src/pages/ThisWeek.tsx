import React, { useState } from "react";
import {
  ActionList,
  CountdownChip,
  MilestoneModule,
  PageHead,
  Panel,
  VendorModule,
  XurModule,
} from "../components/kit";
import { weekly } from "../lib/mockData";

export function ThisWeek() {
  const w = weekly;
  const [acts, setActs] = useState(w.recommended);
  const toggle = (id: string) =>
    setActs((a) => a.map((x) => (x.id === id ? { ...x, done: !x.done } : x)));
  const doneCount = acts.filter((a) => a.done).length;

  return (
    <div className="gt-page">
      <PageHead
        title="This Week"
        sub={`Resets ${w.resetLabel}`}
        right={<CountdownChip prefix="Resets in" time={w.resetIn} icon="clock" />}
      />

      <Panel
        title="Recommended for you"
        icon="sparkle"
        accent="var(--c-signal)"
        right={<span className="gt-action-meta mono">{doneCount}/{acts.length} done</span>}
      >
        <ActionList items={acts} onToggle={toggle} />
      </Panel>

      <div className="gt-week-cols">
        <XurModule xur={w.xur} />
        <MilestoneModule milestones={w.milestones} />
      </div>

      <VendorModule vendors={w.vendors} />
    </div>
  );
}
