import React, { useMemo, useState } from "react";
import { Dropdown, PageHead } from "../../components/composite";
import { SealCard } from "./SealCard";
import { QueryErrorPanel } from "../../components/QueryErrorPanel";
import { LoadingSpinner } from "../../components/LoadingSpinner";
import { useSeals } from "../../data/seals";

type Sort = "closest" | "name";

export function Triumphs() {
  const { seals, isLoading, isError, error, retry } = useSeals();

  const [sort, setSort] = useState<Sort>("closest");
  // undefined = never interacted (auto-open first); null = user explicitly closed
  const [openId, setOpenId] = useState<string | null | undefined>(undefined);
  const effectiveOpenId =
    openId === undefined ? (seals[0]?.id ?? null) : openId;

  const sorted = useMemo(() => {
    const l = seals.slice();
    if (sort === "closest") {
      l.sort(
        (a, b) => (b.pct >= 100 ? -1 : b.pct) - (a.pct >= 100 ? -1 : a.pct),
      );
    } else if (sort === "name") {
      l.sort((a, b) => a.name.localeCompare(b.name));
    }
    return l;
  }, [seals, sort]);

  const gilded = seals.filter((s) => s.gilded > 0).length;

  if (isLoading) {
    return (
      <div className="gt-page">
        <div className="gt-page-loading">
          <LoadingSpinner size="lg" />
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="gt-page">
        <PageHead
          title="Triumphs & Seals"
          sub={<span className="mono">Seal completion</span>}
        />
        <QueryErrorPanel error={error} onRetry={retry} />
      </div>
    );
  }

  return (
    <div className="gt-page">
      <PageHead
        title="Triumphs & Seals"
        sub={
          <span className="mono">
            {seals.length} seals · {gilded} gilded
          </span>
        }
        right={
          <Dropdown
            label="Sort: Closest to done"
            value={
              { closest: "Sort: Closest to done", name: "Sort: Name" }[sort]
            }
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
            expanded={effectiveOpenId === s.id}
            onToggle={() => setOpenId(effectiveOpenId === s.id ? null : s.id)}
          />
        ))}
      </div>
    </div>
  );
}
