import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ActionList,
  CountdownChip,
  MilestoneModule,
  PageHead,
  Panel,
  VendorModule,
  XurModule,
} from "../components/kit";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { useAuth } from "../contexts/AuthContext";
import { apiFetch } from "../lib/api";
import type { Weekly } from "../types/design";

export function ThisWeek() {
  const { user } = useAuth();

  const { data: w, isLoading } = useQuery({
    queryKey: ["weekly"],
    queryFn: () => apiFetch<Weekly>("/api/weekly/recommendations"),
    enabled: !!user,
  });

  const [doneIds, setDoneIds] = useState<Set<string>>(new Set());
  const [lastW, setLastW] = useState(w);
  if (w !== lastW) {
    setLastW(w);
    setDoneIds(new Set());
  }

  const acts = (w?.recommended ?? []).map((x) => ({
    ...x,
    done: doneIds.has(x.id),
  }));

  const toggle = (id: string) =>
    setDoneIds((prev) => {
      const n = new Set(prev);
      if (n.has(id)) { n.delete(id); } else { n.add(id); }
      return n;
    });

  const doneCount = acts.filter((a) => a.done).length;

  if (isLoading) {
    return (
      <div className="gt-page" style={{ display: "flex", justifyContent: "center", padding: "var(--s-8)" }}>
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  return (
    <div className="gt-page">
      <PageHead
        title="This Week"
        sub={w ? `Resets ${w.resetLabel}` : "Weekly activities"}
        right={
          <CountdownChip
            prefix="Resets in"
            time={w?.resetIn ?? { d: 0, h: 0, m: 0 }}
            icon="clock"
          />
        }
      />

      <Panel
        title="Recommended for you"
        icon="sparkle"
        accent="var(--c-signal)"
        right={
          <span className="gt-action-meta mono">
            {doneCount}/{acts.length} done
          </span>
        }
      >
        <ActionList items={acts} onToggle={toggle} />
      </Panel>

      <div className="gt-week-cols">
        <XurModule xur={w?.xur ?? null} />
        <MilestoneModule milestones={w?.milestones ?? []} />
      </div>

      <VendorModule vendors={w?.vendors ?? []} />
    </div>
  );
}
