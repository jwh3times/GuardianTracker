import React, { useMemo, useState } from "react";
import { useMutation, useQuery } from "@apollo/client";
import {
  CategoryTree,
  DataFreshnessChip,
  Dropdown,
  EmptyState,
  FilterChip,
  Icon,
  ItemCard,
  ItemCardSkeleton,
  ItemDetailDrawer,
  PageHead,
  StatTile,
} from "../components/kit";
import { Button } from "../components/kit";
import {
  GET_CURRENT_USER,
  GET_USER_COLLECTIONS,
  GET_WISH_LIST,
} from "../graphql/queries";
import { ADD_TO_WISH_LIST } from "../graphql/mutations";
import { useToast } from "../components/ui/Toast";
import { usePreferences } from "../contexts/PreferencesContext";
import { toGTItem } from "../lib/adapters";
import {
  DIFFS,
  DIFF_LABEL,
  items as MOCK_ITEMS,
  RARITIES,
  RARITY_LABEL,
  tree,
} from "../lib/mockData";
import type {
  Difficulty,
  GTItem,
  Rarity,
  TreeNode,
} from "../types/design";
import type { DestinyItem } from "../types";

type TopCategory = "weapons" | "armor" | "exotics" | "cosmetics";
type SortKey = "rarity" | "name" | "difficulty" | "avail";

const RARITY_RANK: Record<Rarity, number> = {
  exotic: 0,
  legendary: 1,
  rare: 2,
  uncommon: 3,
  common: 4,
};
const DIFF_RANK: Record<Difficulty, number> = {
  challenging: 0,
  moderate: 1,
  easy: 2,
};

function topOf(id: string): TopCategory {
  if (id === "weapons" || id.startsWith("w-")) return "weapons";
  if (id === "armor" || id.startsWith("a-")) return "armor";
  if (id === "exotics" || id.startsWith("e-")) return "exotics";
  return "cosmetics";
}

function flattenTree(nodes: TreeNode[]): TreeNode[] {
  const flat: TreeNode[] = [];
  nodes.forEach((n) => {
    flat.push(n);
    (n.children || []).forEach((c) => flat.push(c));
  });
  return flat;
}

// Mock-fallback category filter (mirrors the design prototype).
function mockCategoryItems(active: string): GTItem[] {
  const top = topOf(active);
  if (top === "exotics") return MOCK_ITEMS.filter((i) => i.rarity === "exotic");
  if (top === "cosmetics") return MOCK_ITEMS.filter((i) => i.slot === "Cosmetic");
  if (top === "armor")
    return MOCK_ITEMS.filter(
      (i) =>
        ["Helmet", "Gauntlets", "Chest Armor", "Leg Armor", "Class Item"].includes(i.type) ||
        i.slot === "Energy"
    );
  return MOCK_ITEMS.filter((i) => i.slot !== "Cosmetic");
}

export function Collections() {
  const { showToast } = useToast();
  const { cardStyle, personalize } = usePreferences();

  const [active, setActive] = useState("weapons");
  const [missingOnly, setMissingOnly] = useState(true);
  const [rarity, setRarity] = useState<Rarity | null>(null);
  const [diff, setDiff] = useState<Difficulty | null>(null);
  const [sort, setSort] = useState<SortKey>("rarity");
  const [view, setView] = useState<"grid" | "list">("grid");
  const [wished, setWished] = useState<Set<string>>(new Set());
  const [detail, setDetail] = useState<GTItem | null>(null);

  const { data: userData } = useQuery(GET_CURRENT_USER);
  const { data: collectionsData, loading, refetch } = useQuery(GET_USER_COLLECTIONS, {
    variables: {
      membershipType: userData?.currentUser?.membershipType,
      membershipId: userData?.currentUser?.membershipId,
    },
    skip: !userData?.currentUser,
    fetchPolicy: "cache-and-network",
  });

  const [addToWishList] = useMutation(ADD_TO_WISH_LIST, {
    refetchQueries: [{ query: GET_WISH_LIST }],
    onError: (err) => showToast(`Failed to add item: ${err.message}`, "error"),
  });

  const top = topOf(active);
  const realBucket =
    top !== "cosmetics" ? collectionsData?.userCollections?.[top] : undefined;
  const hasReal = !!realBucket?.missing;

  const node = useMemo(
    () => flattenTree(tree).find((n) => n.id === active) || tree[0],
    [active]
  );

  const baseItems: GTItem[] = useMemo(() => {
    if (hasReal) {
      return (realBucket.missing as DestinyItem[]).map(toGTItem);
    }
    return mockCategoryItems(active);
  }, [hasReal, realBucket, active]);

  const items = useMemo(() => {
    let list = baseItems.slice();
    if (missingOnly) list = list.filter((i) => !i.collected);
    if (rarity) list = list.filter((i) => i.rarity === rarity);
    if (diff) list = list.filter((i) => i.diff === diff);
    if (sort === "rarity") list.sort((a, b) => RARITY_RANK[a.rarity] - RARITY_RANK[b.rarity]);
    else if (sort === "name") list.sort((a, b) => a.name.localeCompare(b.name));
    else if (sort === "difficulty") list.sort((a, b) => DIFF_RANK[a.diff] - DIFF_RANK[b.diff]);
    else if (sort === "avail")
      list.sort((a, b) => (b.obtainable ? 1 : 0) - (a.obtainable ? 1 : 0));
    return list;
  }, [baseItems, missingOnly, rarity, diff, sort]);

  const clearFilters = () => {
    setRarity(null);
    setDiff(null);
    setMissingOnly(false);
  };
  const hasFilters = !!(rarity || diff || missingOnly);

  const onWish = (item: GTItem) => {
    const adding = !wished.has(item.id);
    setWished((s) => {
      const n = new Set(s);
      if (n.has(item.id)) n.delete(item.id);
      else n.add(item.id);
      return n;
    });
    showToast(
      adding ? `${item.name} added to wishlist` : `Removed ${item.name}`,
      adding ? "success" : "info"
    );
    // Best-effort persist when we have a logged-in session and live data.
    if (adding && hasReal && userData?.currentUser) {
      addToWishList({ variables: { itemHash: item.id, priority: "MEDIUM" } }).catch(() => {});
    }
  };

  // Stats — prefer live totals, fall back to the tree node counts.
  const total = realBucket?.total ?? node.count[1];
  const collected = realBucket?.collected ?? node.count[0];
  const missing = total - collected;

  return (
    <div className="gt-page gt-collections">
      <PageHead
        title="Collections"
        sub={<span className="mono">Track what you're missing across every category</span>}
        right={<DataFreshnessChip ago="4m" onRefresh={() => refetch()} />}
      />

      <div className="gt-coll-layout">
        {/* CATEGORY TREE */}
        <aside className="gt-coll-aside">
          <div className="gt-section-title" style={{ marginBottom: "var(--s-3)" }}>
            Categories
          </div>
          <CategoryTree nodes={tree} activeId={active} onSelect={setActive} />
        </aside>

        {/* MAIN */}
        <div className="gt-coll-main">
          {/* FILTER BAR */}
          <div className="gt-coll-toolbar">
            <div className="gt-filterbar">
              <FilterChip on={missingOnly} onClick={() => setMissingOnly((v) => !v)}>
                <Icon name="check" size="0.85rem" /> Missing only
              </FilterChip>
              <Dropdown
                label="Rarity"
                value={rarity ? RARITY_LABEL[rarity] : null}
                options={RARITIES.map((r) => ({ v: r, l: RARITY_LABEL[r] }))}
                onPick={(v) => setRarity(v as Rarity | null)}
              />
              <Dropdown
                label="Difficulty"
                value={diff ? DIFF_LABEL[diff] : null}
                options={DIFFS.map((d) => ({ v: d, l: DIFF_LABEL[d] }))}
                onPick={(v) => setDiff(v as Difficulty | null)}
                note="estimate"
              />
              <Dropdown
                label="Sort"
                value={{ rarity: "Rarity", name: "Name", difficulty: "Difficulty", avail: "Availability" }[sort]}
                options={[
                  { v: "rarity", l: "Rarity" },
                  { v: "name", l: "Name" },
                  { v: "difficulty", l: "Difficulty" },
                  { v: "avail", l: "Availability" },
                ]}
                onPick={(v) => v && setSort(v as SortKey)}
                noClear
              />
            </div>
            <div className="gt-viewtoggle">
              <button
                className="gt-iconbtn"
                data-on={view === "grid"}
                onClick={() => setView("grid")}
                aria-label="Grid"
              >
                <Icon name="grid" size="1rem" />
              </button>
              <button
                className="gt-iconbtn"
                data-on={view === "list"}
                onClick={() => setView("list")}
                aria-label="List"
              >
                <Icon name="list" size="1rem" />
              </button>
            </div>
          </div>

          <div className="gt-coll-stats">
            <StatTile num={total.toLocaleString()} label="Total" mono />
            <StatTile num={collected.toLocaleString()} label="Collected" mono color="var(--c-complete)" />
            <StatTile num={missing.toLocaleString()} label="Missing" mono color="var(--c-signal)" />
            <div className="gt-coll-resultcount mono">{items.length} shown</div>
          </div>

          {/* GRID / LIST */}
          {loading && items.length === 0 ? (
            <div className="gt-itemgrid">
              {Array.from({ length: 8 }).map((_, i) => (
                <ItemCardSkeleton key={i} />
              ))}
            </div>
          ) : items.length === 0 ? (
            <div className="gt-card">
              <EmptyState
                icon={hasFilters ? "filter" : "check"}
                color={hasFilters ? "var(--c-text-3)" : "var(--c-complete)"}
                title={hasFilters ? "No items match these filters" : "All caught up!"}
                body={
                  hasFilters
                    ? "Try loosening a filter to see more of this category."
                    : "You've collected everything in this category. Nice work, Guardian."
                }
                action={
                  hasFilters ? (
                    <Button variant="outline" sm onClick={clearFilters}>
                      Clear filters
                    </Button>
                  ) : null
                }
              />
            </div>
          ) : view === "grid" ? (
            <div className="gt-itemgrid">
              {items.map((it) => (
                <ItemCard
                  key={it.id}
                  item={it}
                  density={cardStyle === "compact" ? "compact" : "grid"}
                  personalize={personalize}
                  showCollected={!missingOnly}
                  wished={wished.has(it.id)}
                  onWish={onWish}
                  onOpen={setDetail}
                />
              ))}
            </div>
          ) : (
            <div className="gt-itemlist">
              {items.map((it) => (
                <ItemCard
                  key={it.id}
                  item={it}
                  density="list"
                  personalize={personalize}
                  showCollected={!missingOnly}
                  wished={wished.has(it.id)}
                  onWish={onWish}
                  onOpen={setDetail}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      {detail && (
        <ItemDetailDrawer
          item={detail}
          onClose={() => setDetail(null)}
          onWish={onWish}
          wished={wished.has(detail.id)}
        />
      )}
    </div>
  );
}
