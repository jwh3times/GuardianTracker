import React, { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  CategoryTree,
  Dropdown,
  ItemDetailDrawer,
  PageHead,
} from "../../components/composite";
import {
  Button,
  DataFreshnessChip,
  EmptyState,
  FilterChip,
  ItemCardSkeleton,
  StatTile,
} from "../../components/primitives";
import { Icon } from "../../components/Icon";
import { ItemCard } from "./ItemCard";
import { useToast } from "../../components/Toast";
import { useAuth } from "../../contexts/AuthContext";
import { usePreferences } from "../../contexts/PreferencesContext";
import { apiFetch } from "../../lib/api";
import { QueryErrorPanel } from "../../components/QueryErrorPanel";
import {
  collectionsQuery,
  itemPerksQuery,
  itemByHashQuery,
} from "../../lib/queries";
import { toGTItem, toGTItemView } from "../../lib/adapters";
import {
  apiNodeToTreeNode,
  gatherItemHashes,
  findNodePath,
} from "./collectionTree";
import { useCollectionsFilters, type SortKey } from "./useCollectionsFilters";
import { DIFFS, DIFF_LABEL, RARITIES, RARITY_LABEL } from "../../lib/constants";
import type { GTItem, Rarity, Difficulty, TreeNode } from "../../types/design";
import type {
  ProfileResponse,
  APICollectionNode,
  APICacheRefreshResponse,
  WishListItem,
} from "../../types/api";

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
  unrated: 3,
};

// Root→node hash path to the node whose own hash is `nodeHash` (distinct from
// `findNodePath`, which locates the node owning an *item* hash). Used to
// reveal a category restored directly from a `?node=` URL. Root-level nodes
// are always visible in the sidebar without opening anything, so callers only
// need to act on paths longer than one entry.
function findPathToNode(
  nodes: APICollectionNode[],
  nodeHash: string,
): string[] | null {
  const dfs = (node: APICollectionNode, trail: string[]): string[] | null => {
    const here = [...trail, node.hash];
    if (node.hash === nodeHash) return here;
    for (const c of node.children ?? []) {
      const found = dfs(c, here);
      if (found) return found;
    }
    return null;
  };
  for (const root of nodes) {
    const found = dfs(root, []);
    if (found) return found;
  }
  return null;
}

export function Collections() {
  const { showToast } = useToast();
  const { cardStyle, personalize } = usePreferences();

  // Ancestor node-hash path to reveal in the sidebar tree (deep-link seed, or a
  // node restored from a persisted/URL selection).
  const [expandPath, setExpandPath] = useState<string[]>([]);
  const [detail, setDetail] = useState<GTItem | null>(null);

  const {
    node: active,
    q,
    rarity,
    diff,
    sort,
    view,
    missing: missingOnly,
    avail,
    farm,
    setNode: setActive,
    setQ,
    setRarity,
    setDiff,
    setSort,
    setView,
    setMissing: setMissingOnly,
    setAvail,
    setFarm,
    setFilters,
    clearFilters,
    hasFilters,
  } = useCollectionsFilters();

  // Whitespace-only search input behaves as no search at all — both for item
  // matching below and for the empty-state branch (see hasFilters in
  // useCollectionsFilters, which applies the same trim).
  const qTrimmed = q.trim().toLowerCase();

  const { user } = useAuth();
  const [searchParams] = useSearchParams();
  const itemParam = searchParams.get("item");

  const { data: profileData } = useQuery({
    queryKey: ["currentUser"],
    queryFn: () => apiFetch<ProfileResponse>("/api/auth/profile"),
  });

  const membershipType =
    profileData?.user.membershipType ?? user?.membershipType;
  const membershipId = profileData?.user.membershipId ?? user?.membershipId;

  // The collections browser always loads the full dataset (collected + missing)
  // and filters the display client-side via `missingOnly`. Using one stable
  // query key avoids re-fetching when toggling the filter or following a
  // deep-link to a collected item.
  const {
    data: collections,
    isLoading: loading,
    error,
    refetch,
  } = useQuery(collectionsQuery(membershipType, membershipId, true));

  const perksQuery = useQuery(itemPerksQuery(detail?.id));

  const [viewOnlyHash, setViewOnlyHash] = useState<string | null>(null);
  const itemViewQuery = useQuery(itemByHashQuery(viewOnlyHash));

  const queryClient = useQueryClient();

  const { data: wishlistData } = useQuery({
    queryKey: ["wishlist"],
    queryFn: () => apiFetch<WishListItem[]>("/api/wishlist"),
  });

  const wished = useMemo(
    () => new Set(wishlistData?.map((w) => String(w.itemHash)) ?? []),
    [wishlistData],
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
        { method: "POST" },
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
  // the tree, select its owning node, open its drawer, and clear the param.
  // The URL is the external system being synchronized here; the one-off
  // cascading render on deep-link navigation is intentional.
  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    if (!itemParam || !collections?.items) return;
    const d = collections.items[itemParam];
    const path = findNodePath(collections.tree, itemParam);
    if (d && path) {
      const owningNode = path[path.length - 1];
      setExpandPath(path);
      const isCollected = (collections.collectedHashes ?? []).includes(
        itemParam,
      );
      const vendor = collections.availableNow?.[itemParam];
      setDetail({
        ...toGTItem(d),
        collected: isCollected,
        obtainable: !!vendor,
        availFrom: vendor,
      });
      // One atomic URL write: select the owning node (reveal collected items too)
      // and consume the item param, preserving existing filters.
      setFilters(
        isCollected
          ? { node: owningNode, missing: false }
          : { node: owningNode },
        { replace: true, drop: ["item"] },
      );
    } else {
      setViewOnlyHash(itemParam);
      setFilters({}, { replace: true, drop: ["item"] });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [itemParam, collections]);
  /* eslint-enable react-hooks/set-state-in-effect */

  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    if (!viewOnlyHash) return;
    if (itemViewQuery.data) {
      setDetail(toGTItemView(itemViewQuery.data));
      setViewOnlyHash(null);
    } else if (itemViewQuery.isError) {
      showToast("That item isn't in your trackable collections", "info");
      setViewOnlyHash(null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [viewOnlyHash, itemViewQuery.data, itemViewQuery.isError]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const hasReal = !!collections;

  // Sidebar tree built from the API forest.
  const treeNodes: TreeNode[] = useMemo(
    () => (collections?.tree ?? []).map(apiNodeToTreeNode),
    [collections],
  );

  // hash→node map for item gathering + the selected node's rolled-up counts.
  const nodeByHash = useMemo(() => {
    const map = new Map<string, APICollectionNode>();
    const walk = (n: APICollectionNode) => {
      map.set(n.hash, n);
      (n.children ?? []).forEach(walk);
    };
    (collections?.tree ?? []).forEach(walk);
    return map;
  }, [collections]);

  // Owned item hashes — per-item collected state for the missing-only filter.
  const collectedSet = useMemo(
    () => new Set(collections?.collectedHashes ?? []),
    [collections],
  );

  // Default the selection to the first root once data arrives. Guarded on
  // `itemParam` so this doesn't race the deep-link effect above: without the
  // guard, both effects can fire on the same bare `?item=` load, and since
  // react-router's functional updater snapshots `prev` per call, whichever
  // effect's `setSearchParams` call runs second wins with a stale snapshot —
  // silently reintroducing `item=` after the deep-link effect already
  // consumed it. Uses `setFilters(..., { replace: true })` (not `setActive`,
  // which pushes) because seeding a default is URL canonicalization, not a
  // user navigation — it shouldn't create a Back-button stop.
  useEffect(() => {
    if (itemParam || !collections?.tree?.length) return;
    // Seed the first root when nothing is selected, OR when a restored/shared
    // `?node=` points at a hash that isn't in the current tree (stale or tampered
    // URL) — otherwise the grid would be stuck empty with no way to recover.
    if (!active || !nodeByHash.has(active)) {
      setFilters({ node: collections.tree[0].hash }, { replace: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [collections, active, itemParam, nodeByHash]);

  // Reveal a node restored directly from a `?node=` URL (a bookmarked link, or
  // a persisted/shared filter state) so the sidebar opens down to it — mirrors
  // the deep-link effect's own `setExpandPath(path)`. Root-level selections
  // (including the "default to first root" effect above) need no reveal since
  // roots are already visible without opening anything.
  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    if (!active || expandPath.length > 0 || !collections?.tree?.length) return;
    const path = findPathToNode(collections.tree, active);
    if (path && path.length > 1) setExpandPath(path);
  }, [active, expandPath, collections]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const activeNode = active ? nodeByHash.get(active) : undefined;

  const baseItems: GTItem[] = useMemo(() => {
    if (!activeNode || !collections?.items) return [];
    const out: GTItem[] = [];
    for (const h of gatherItemHashes(activeNode)) {
      const d = collections.items[h];
      if (!d) continue;
      const collected = collectedSet.has(h);
      if (missingOnly && collected) continue;
      const vendor = collections.availableNow?.[h];
      out.push({
        ...toGTItem(d),
        collected,
        obtainable: !!vendor,
        availFrom: vendor,
      });
    }
    return out;
  }, [activeNode, collections, missingOnly, collectedSet]);

  const items = useMemo(() => {
    let list = baseItems.slice();
    if (rarity) list = list.filter((i) => i.rarity === rarity);
    if (diff) list = list.filter((i) => i.diff === diff);
    if (avail) list = list.filter((i) => i.obtainable);
    if (farm) list = list.filter((i) => !i.farmOnly);
    if (qTrimmed)
      list = list.filter((i) => i.name.toLowerCase().includes(qTrimmed));
    if (sort === "rarity")
      list.sort((a, b) => RARITY_RANK[a.rarity] - RARITY_RANK[b.rarity]);
    else if (sort === "name") list.sort((a, b) => a.name.localeCompare(b.name));
    else if (sort === "difficulty")
      list.sort((a, b) => DIFF_RANK[a.diff] - DIFF_RANK[b.diff]);
    else if (sort === "avail")
      list.sort((a, b) => (b.obtainable ? 1 : 0) - (a.obtainable ? 1 : 0));
    return list;
  }, [baseItems, rarity, diff, avail, farm, qTrimmed, sort]);

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
    removeWishlistMutation.mutate({
      rowId: row.id,
      name: item.name,
      itemId: item.id,
    });
  };

  // Stat tiles use the selected node's rolled-up counts.
  const total = activeNode?.total ?? 0;
  const collected = activeNode?.collected ?? 0;
  const missing = Math.max(total - collected, 0);

  return (
    <div
      className="gt-page gt-collections"
      data-onboarding-target="collections"
    >
      <PageHead
        title="Collections"
        sub={
          <span className="mono">
            Track what you're missing across every category
          </span>
        }
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
          <div
            className="gt-section-title"
            style={{ marginBottom: "var(--s-3)" }}
          >
            Categories
          </div>
          <CategoryTree
            nodes={treeNodes}
            activeId={active}
            onSelect={setActive}
            expand={expandPath}
          />
        </aside>

        {/* MAIN */}
        <div className="gt-coll-main">
          {/* FILTER BAR */}
          <div className="gt-coll-toolbar">
            <div className="gt-filterbar">
              <div className="gt-search gt-coll-search">
                <Icon
                  name="search"
                  size="1rem"
                  style={{ color: "var(--c-text-3)" }}
                />
                <input
                  className="gt-search-input"
                  type="search"
                  aria-label="Search this category…"
                  placeholder="Search this category…"
                  maxLength={100}
                  value={q}
                  onChange={(e) => setQ(e.target.value)}
                />
              </div>
              <FilterChip
                on={missingOnly}
                onClick={() => setMissingOnly(!missingOnly)}
              >
                Missing only
              </FilterChip>
              <FilterChip on={avail} onClick={() => setAvail(!avail)}>
                Available now
              </FilterChip>
              <FilterChip on={farm} onClick={() => setFarm(!farm)}>
                Hide farm-only
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
                value={
                  {
                    rarity: "Rarity",
                    name: "Name",
                    difficulty: "Difficulty",
                    avail: "Availability",
                  }[sort]
                }
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
            <StatTile
              num={collected.toLocaleString()}
              label="Collected"
              mono
              color="var(--c-complete)"
            />
            <StatTile
              num={missing.toLocaleString()}
              label="Missing"
              mono
              color="var(--c-signal)"
            />
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
            <QueryErrorPanel
              error={error}
              onRetry={() => {
                void refetch();
              }}
            />
          ) : items.length === 0 ? (
            <div className="gt-card">
              <EmptyState
                icon={hasFilters ? "filter" : "check"}
                color={hasFilters ? "var(--c-text-3)" : "var(--c-complete)"}
                title={
                  qTrimmed
                    ? `No items match "${q}"`
                    : hasFilters
                      ? "No items match these filters"
                      : "All caught up!"
                }
                body={
                  qTrimmed
                    ? "Try a different search term, or clear the search to see this category's full list."
                    : hasFilters
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
          perkColumns={perksQuery.data?.perkColumns}
          perksLoading={perksQuery.isLoading}
          catalysts={perksQuery.data?.catalysts}
          onClose={() => setDetail(null)}
          onWish={onWish}
          wished={wished.has(detail.id)}
        />
      )}
    </div>
  );
}
