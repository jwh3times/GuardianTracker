import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button, EmptyState, Skeleton } from "../../components/primitives";
import { useAuth } from "../../contexts/AuthContext";
import { errorState } from "../../lib/errorState";
import { collectionsQuery } from "../../lib/queries";
import { COSMETIC_TYPES } from "./cosmeticBuckets";
import { cosmeticItems, groupByType } from "./cosmeticItems";
import { CosmeticsGrid } from "./CosmeticsGrid";
import { CosmeticDetail } from "./CosmeticDetail";
import type { GTItem } from "../../types/design";

const RARITY_RANK: Record<string, number> = {
  exotic: 0,
  legendary: 1,
  rare: 2,
  uncommon: 3,
  common: 4,
};

type Filter = "all" | "owned" | "missing";
const FILTERS: Filter[] = ["all", "owned", "missing"];

export function Cosmetics() {
  const { user } = useAuth();
  const { data, isLoading, isError, error, refetch } = useQuery(
    collectionsQuery(user?.membershipType, user?.membershipId, true),
  );

  const items = useMemo(() => (data ? cosmeticItems(data) : []), [data]);
  const groups = useMemo(() => groupByType(items), [items]);
  const tabs = useMemo(
    () => COSMETIC_TYPES.filter((t) => groups.has(t)),
    [groups],
  );

  const [active, setActive] = useState<string | null>(null);
  const [filter, setFilter] = useState<Filter>("all");
  const [selected, setSelected] = useState<GTItem | null>(null);

  const activeTab = useMemo(
    () => (active && groups.has(active) ? active : (tabs[0] ?? null)),
    [active, groups, tabs],
  );
  const bucket = useMemo(
    () => (activeTab ? (groups.get(activeTab) ?? []) : []),
    [activeTab, groups],
  );

  const shown = useMemo(() => {
    const filtered = bucket.filter((it) =>
      filter === "all"
        ? true
        : filter === "owned"
          ? it.collected
          : !it.collected,
    );
    return [...filtered].sort(
      (a, b) =>
        (RARITY_RANK[a.rarity] ?? 9) - (RARITY_RANK[b.rarity] ?? 9) ||
        a.name.localeCompare(b.name),
    );
  }, [bucket, filter]);

  const owned = bucket.filter((it) => it.collected).length;

  if (isLoading) {
    return (
      <div className="gt-cosmetics">
        <h1>Cosmetics</h1>
        <div className="gt-tilegrid-scroll">
          <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--s-3)" }}>
            {Array.from({ length: 12 }, (_, i) => (
              <Skeleton key={i} w="88px" h="6.5rem" r="var(--r-md)" />
            ))}
          </div>
        </div>
      </div>
    );
  }
  if (isError) {
    const copy = errorState(error);
    return (
      <div className="gt-cosmetics">
        <h1>Cosmetics</h1>
        <EmptyState
          icon={copy.icon}
          title={copy.title}
          body={copy.body}
          action={<Button onClick={() => void refetch()}>Retry</Button>}
        />
      </div>
    );
  }
  if (tabs.length === 0) {
    return (
      <div className="gt-cosmetics">
        <h1>Cosmetics</h1>
        <EmptyState
          title="No cosmetics data"
          body="Refresh your profile or try again after the next manifest update."
        />
      </div>
    );
  }

  return (
    <div className="gt-cosmetics">
      <h1>Cosmetics</h1>
      <div className="gt-cosmetic-tabs" role="tablist">
        {tabs.map((t) => (
          <button
            key={t}
            role="tab"
            id={`cosmetics-tab-${t}`}
            aria-selected={t === activeTab}
            aria-controls="cosmetics-panel"
            className="gt-cosmetic-tab"
            data-active={t === activeTab}
            onClick={() => setActive(t)}
          >
            {t}
          </button>
        ))}
      </div>
      <div className="gt-cosmetic-bar">
        <div className="gt-cosmetic-filter" role="group" aria-label="Filter">
          {FILTERS.map((f) => (
            <button
              key={f}
              className="gt-cosmetic-filter-btn"
              data-active={filter === f}
              onClick={() => setFilter(f)}
            >
              {f[0].toUpperCase() + f.slice(1)}
            </button>
          ))}
        </div>
        <span className="mono gt-cosmetic-count">
          {owned}/{bucket.length} collected
        </span>
      </div>
      {shown.length === 0 ? (
        <p className="gt-cosmetic-empty">Nothing to show for this filter.</p>
      ) : (
        <CosmeticsGrid
          items={shown}
          onOpen={setSelected}
          id="cosmetics-panel"
          role="tabpanel"
          aria-labelledby={activeTab ? `cosmetics-tab-${activeTab}` : undefined}
        />
      )}
      <CosmeticDetail item={selected} onClose={() => setSelected(null)} />
    </div>
  );
}
