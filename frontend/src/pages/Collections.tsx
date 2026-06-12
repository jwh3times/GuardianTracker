import React, { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
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
import { useToast } from "../components/ui/Toast";
import { useAuth } from "../contexts/AuthContext";
import { usePreferences } from "../contexts/PreferencesContext";
import { apiFetch } from "../lib/api";
import { errorState } from "../lib/errorState";
import { collectionsQuery } from "../lib/queries";
import { toGTItem } from "../lib/adapters";
import { DIFFS, DIFF_LABEL, RARITIES, RARITY_LABEL } from "../lib/constants";
import type {
  Difficulty,
  GTItem,
  Rarity,
  TreeNode,
} from "../types/design";
import type {
  ProfileResponse,
  APICollectionSummary,
  APICacheRefreshResponse,
  WishListItem,
} from "../types/api";

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
  if (id === "armor" || id.startsWith("a-")) return "armor";
  if (id === "exotics" || id.startsWith("e-")) return "exotics";
  if (id === "cosmetics" || id.startsWith("c-")) return "cosmetics";
  return "weapons";
}

const TOP_CATEGORIES: { id: TopCategory; label: string }[] = [
  { id: "weapons", label: "Weapons" },
  { id: "armor", label: "Armor" },
  { id: "exotics", label: "Exotics" },
  { id: "cosmetics", label: "Cosmetics" },
];

export function Collections() {
  const { showToast } = useToast();
  const { cardStyle, personalize } = usePreferences();

  const [active, setActive] = useState("weapons");
  const [rarity, setRarity] = useState<Rarity | null>(null);
  const [diff, setDiff] = useState<Difficulty | null>(null);
  const [sort, setSort] = useState<SortKey>("rarity");
  const [view, setView] = useState<"grid" | "list">("grid");
  const [missingOnly, setMissingOnly] = useState(true);
  const [detail, setDetail] = useState<GTItem | null>(null);

  const { user } = useAuth();
  const [searchParams, setSearchParams] = useSearchParams();
  const itemParam = searchParams.get("item");

  const { data: profileData } = useQuery({
    queryKey: ["currentUser"],
    queryFn: () => apiFetch<ProfileResponse>("/api/auth/profile"),
  });

  const membershipType = profileData?.user.membershipType ?? user?.membershipType;
  const membershipId = profileData?.user.membershipId ?? user?.membershipId;

  // The collections browser always loads the full dataset (collected + missing)
  // and filters the display client-side via `missingOnly`. Using one stable
  // query key avoids re-fetching when toggling the filter or following a
  // deep-link to a collected item.
  const {
    data: collections,
    isLoading: loading,
    isError,
    error,
    refetch,
  } = useQuery(collectionsQuery(membershipType, membershipId, true));

  const queryClient = useQueryClient();

  const { data: wishlistData } = useQuery({
    queryKey: ["wishlist"],
    queryFn: () => apiFetch<WishListItem[]>("/api/wishlist"),
  });

  const wished = useMemo(
    () => new Set(wishlistData?.map((w) => String(w.itemHash)) ?? []),
    [wishlistData]
  );

  // Items with an add/remove mutation in flight. Guards against a rapid second
  // click acting on the pre-mutation `wished` snapshot (double-add, or a silent
  // no-op remove before the wishlist query has refetched).
  const [pendingWish, setPendingWish] = useState<Set<string>>(() => new Set());
  const markPending = (id: string, on: boolean) =>
    setPendingWish((prev) => {
      const next = new Set(prev);
      if (on) next.add(id);
      else next.delete(id);
      return next;
    });

  const addWishlistMutation = useMutation({
    mutationFn: (item: GTItem) =>
      apiFetch("/api/wishlist", {
        method: "POST",
        body: JSON.stringify({ itemHash: Number(item.id) }),
      }),
    onSuccess: (_data, item) => {
      void queryClient.invalidateQueries({ queryKey: ["wishlist"] });
      showToast(`${item.name} added to wishlist`, "success");
    },
    onError: (err: Error) =>
      showToast(`Failed to add item: ${err.message}`, "error"),
    onSettled: (_data, _err, item) => markPending(item.id, false),
  });

  const removeWishlistMutation = useMutation({
    mutationFn: ({ rowId }: { rowId: string; name: string; itemId: string }) =>
      apiFetch<void>(`/api/wishlist/${rowId}`, { method: "DELETE" }),
    onSuccess: (_data, { name }) => {
      void queryClient.invalidateQueries({ queryKey: ["wishlist"] });
      showToast(`Removed ${name}`, "info");
    },
    onError: (err: Error) =>
      showToast(`Failed to remove item: ${err.message}`, "error"),
    onSettled: (_data, _err, { itemId }) => markPending(itemId, false),
  });

  const refreshMutation = useMutation({
    mutationFn: () =>
      apiFetch<APICacheRefreshResponse>(
        `/api/collections/${membershipType}/${membershipId}/refresh`,
        { method: "POST" }
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["collections"] });
      void queryClient.invalidateQueries({ queryKey: ["characters"] });
      void queryClient.invalidateQueries({ queryKey: ["weekly"] });
      void queryClient.invalidateQueries({ queryKey: ["catalysts"] });
      void queryClient.invalidateQueries({ queryKey: ["crafting"] });
      void queryClient.invalidateQueries({ queryKey: ["seals"] });
    },
  });

  // Search deep-link (?item=<hash>): once data is loaded, locate the item in
  // any bucket (missing or collected), open its drawer, and clear the param.
  // The URL is the external system being synchronized here; the one-off
  // cascading render on deep-link navigation is intentional.
  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    if (!itemParam || !collections) return;
    const buckets: { id: TopCategory; b: APICollectionSummary }[] = [
      { id: "weapons", b: collections.weapons },
      { id: "armor", b: collections.armor },
      { id: "exotics", b: collections.exotics },
      { id: "cosmetics", b: collections.cosmetics },
    ];
    let found: GTItem | null = null;
    let bucketId: TopCategory | null = null;
    for (const { id, b } of buckets) {
      const miss = b?.missing?.find((d) => d.itemHash === itemParam);
      if (miss) {
        found = toGTItem(miss);
        bucketId = id;
        break;
      }
      const coll = b?.collectedItems?.find((d) => d.itemHash === itemParam);
      if (coll) {
        found = { ...toGTItem(coll), collected: true };
        bucketId = id;
        break;
      }
    }
    if (found && bucketId) {
      setActive(bucketId);
      if (found.collected) setMissingOnly(false);
      setDetail(found);
    } else {
      showToast("That item isn't in your trackable collections", "info");
    }
    setSearchParams({}, { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [itemParam, collections]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const top = topOf(active);
  const realBucket = collections?.[top];
  const hasReal = !!collections;

  const treeNodes: TreeNode[] = useMemo(() => {
    if (!collections) return [];
    return TOP_CATEGORIES.map(({ id, label }) => {
      const b = collections[id];
      const total = b?.total ?? 0;
      const collected = b?.collected ?? 0;
      return {
        id,
        label,
        pct: total ? Math.round((collected / total) * 100) : 0,
        count: [collected, total] as [number, number],
      };
    });
  }, [collections]);

  const baseItems: GTItem[] = useMemo(() => {
    if (!realBucket) return [];
    const missing = (realBucket.missing ?? []).map(toGTItem);
    if (missingOnly) return missing;
    const collected = (realBucket.collectedItems ?? []).map((d) => ({
      ...toGTItem(d),
      collected: true,
    }));
    return [...missing, ...collected];
  }, [realBucket, missingOnly]);

  const items = useMemo(() => {
    let list = baseItems.slice();
    if (rarity) list = list.filter((i) => i.rarity === rarity);
    if (diff) list = list.filter((i) => i.diff === diff);
    if (sort === "rarity") list.sort((a, b) => RARITY_RANK[a.rarity] - RARITY_RANK[b.rarity]);
    else if (sort === "name") list.sort((a, b) => a.name.localeCompare(b.name));
    else if (sort === "difficulty") list.sort((a, b) => DIFF_RANK[a.diff] - DIFF_RANK[b.diff]);
    else if (sort === "avail")
      list.sort((a, b) => (b.obtainable ? 1 : 0) - (a.obtainable ? 1 : 0));
    return list;
  }, [baseItems, rarity, diff, sort]);

  const clearFilters = () => {
    setRarity(null);
    setDiff(null);
  };
  const hasFilters = !!(rarity || diff);

  const onWish = (item: GTItem) => {
    // Ignore clicks while a mutation for this item is still settling — `wished`
    // and `wishlistData` haven't caught up yet, so acting now would double-add
    // or silently skip the remove.
    if (pendingWish.has(item.id)) return;
    if (!wished.has(item.id)) {
      markPending(item.id, true);
      addWishlistMutation.mutate(item);
      return;
    }
    const row = wishlistData?.find((w) => String(w.itemHash) === item.id);
    if (!row) return; // wishlist cache not refreshed yet; wait for it
    markPending(item.id, true);
    removeWishlistMutation.mutate({ rowId: row.id, name: item.name, itemId: item.id });
  };

  const total = realBucket?.total ?? 0;
  const collected = realBucket?.collected ?? 0;
  const missing = Math.max(total - collected, 0);

  const errState = errorState(error);

  return (
    <div className="gt-page gt-collections">
      <PageHead
        title="Collections"
        sub={<span className="mono">Track what you're missing across every category</span>}
        right={
          <DataFreshnessChip
            updatedAt={collections?.fetchedAt}
            refreshing={refreshMutation.isPending}
            onRefresh={() => {
              if (membershipType != null && !!membershipId) {
                refreshMutation.mutate();
              } else {
                void refetch();
              }
            }}
          />
        }
      />

      <div className="gt-coll-layout">
        {/* CATEGORY TREE */}
        <aside className="gt-coll-aside">
          <div className="gt-section-title" style={{ marginBottom: "var(--s-3)" }}>
            Categories
          </div>
          <CategoryTree nodes={treeNodes} activeId={active} onSelect={setActive} />
        </aside>

        {/* MAIN */}
        <div className="gt-coll-main">
          {/* FILTER BAR */}
          <div className="gt-coll-toolbar">
            <div className="gt-filterbar">
              <FilterChip on={missingOnly} onClick={() => setMissingOnly((v) => !v)}>
                Missing only
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
          {loading && !hasReal ? (
            <div className="gt-itemgrid">
              {Array.from({ length: 8 }).map((_, i) => (
                <ItemCardSkeleton key={i} />
              ))}
            </div>
          ) : !hasReal ? (
            <div className="gt-card">
              <EmptyState
                icon={errState.icon}
                color="var(--c-text-3)"
                title={errState.title}
                body={errState.body}
                action={
                  <div style={{ display: "flex", gap: "var(--s-2)" }}>
                    {isError && errState.privacyLink && (
                      <a
                        href="https://www.bungie.net/7/en/User/Account/Privacy"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        <Button variant="outline" sm icon="external">
                          Bungie privacy settings
                        </Button>
                      </a>
                    )}
                    <Button variant="outline" sm onClick={() => { void refetch(); }}>
                      Retry
                    </Button>
                  </div>
                }
              />
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
