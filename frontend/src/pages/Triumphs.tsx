import React, { useMemo, useState } from "react";
import { Dropdown, PageHead, SealCard } from "../components/kit";
import { seals } from "../lib/mockData";

type Sort = "closest" | "name";

export function Triumphs() {
  const [sort, setSort] = useState<Sort>("closest");
  const [openId, setOpenId] = useState<string | null>(seals[0].id);

  const sorted = useMemo(() => {
    const l = seals.slice();
    if (sort === "closest") {
      l.sort((a, b) => (b.pct >= 100 ? -1 : b.pct) - (a.pct >= 100 ? -1 : a.pct));
    } else if (sort === "name") {
      l.sort((a, b) => a.name.localeCompare(b.name));
    }
    return l;
  }, [sort]);

  const gilded = seals.filter((s) => s.gilded > 0).length;

  return (
    <div className="gt-page">
      <PageHead
        title="Triumphs & Seals"
        sub={<span className="mono">{seals.length} seals · {gilded} gilded</span>}
        right={
          <Dropdown
            label="Sort: Closest to done"
            value={{ closest: "Sort: Closest to done", name: "Sort: Name" }[sort]}
            noClear
            options={[
              { v: "closest", l: "Sort: Closest to done" },
              { v: "name", l: "Sort: Name" },
            ]}
            onPick={(v) => v && setSort(v as Sort)}
          />
        }
      />

      <div className="gt-seal-grid">
        {sorted.map((s) => (
          <SealCard
            key={s.id}
            seal={s}
            expanded={openId === s.id}
            onToggle={() => setOpenId((id) => (id === s.id ? null : s.id))}
          />
        ))}
      </div>
    </div>
  );
}
