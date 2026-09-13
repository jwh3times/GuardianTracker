import React, { useCallback, useMemo, useState } from "react";
import { Dropdown, PageHead } from "../../components/composite";
import { CategoryTree } from "./CategoryTree";
import { ItemDetailDrawer } from "./ItemDetailDrawer";
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
import { usePreferences } from "../../data/preferences";
import { QueryErrorPanel } from "../../components/QueryErrorPanel";
import { useCollectionsBrowser, type SortKey } from "./useCollectionsBrowser";
import { DIFFS, DIFF_LABEL, RARITIES, RARITY_LABEL } from "../../lib/constants";
import type { GTItem, Rarity, Difficulty, TreeNode } from "../../types/design";
import { useCollections } from "../../data/collections";
import { useItemPerks } from "../../data/items";
import { useMembershipRefresh } from "../../data/membershipRefresh";
import {
  useAddWishlistItem,
  useRemoveWishlistItem,
  useWishlist,
} from "../../data/wishlist";

export function Collections() {
  const { showToast } = useToast();
  const {
    values: { cardStyle, personalize },
  } = usePreferences();

  const { user } = useAuth();
  const membershipType = user?.membershipType;
  const membershipId = user?.membershipId;

  // The collections browser always loads the full dataset (collected + missing)
  // and filters the display client-side via the missing-only filter. Using one
  // stable query key avoids re-fetching when toggling the filter or following a
  // deep-link to a collected item.
  const {
    view: collections,
    isLoading: loading,
    error,
    retry: refetch,
  } = useCollections();

  const onItemUnavailable = useCallback(
    () => showToast("That item isn't in your trackable collections", "info"),
    [showToast],
  );
  const {
    filters: { node: active, q, rarity, diff, sort, view, avail, farm },
    filters: { missing: missingOnly },
    items,
    activeNode,
    expandPath,
    detail,
    searching,
    hasFilters,
    selectNode,
    setFilter,
    clearFilters,
    openItem,
    closeDetail,
  } = useCollectionsBrowser(collections, { onItemUnavailable });

  const {
    perkColumns,
    catalysts,
    isLoading: perksLoading,
  } = useItemPerks(detail?.id);

  const { entries: wishlistEntries } = useWishlist();

  const wished = useMemo(
    () => new Set(wishlistEntries.map((e) => e.itemId)),
    [wishlistEntries],
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

  const { add: addToWishlist } = useAddWishlistItem({
    onSuccess: (_result, vars) =>
      showToast(`${vars.name} added to wishlist`, "success"),
    onError: (err) => showToast(`Failed to add item: ${err.message}`, "error"),
    onSettled: (vars) => markPending(vars.itemId, false),
  });

  const { remove: removeFromWishlist } = useRemoveWishlistItem({
    onSuccess: (_result, vars) => showToast(`Removed ${vars.name}`, "info"),
    onError: (err) =>
      showToast(`Failed to remove item: ${err.message}`, "error"),
    onSettled: (vars) => markPending(vars.itemId, false),
  });

  const { refresh, isRefreshing } = useMembershipRefresh();

  const hasReal = !!collections;

  // The sidebar tree, the node lookup and the per-item collected state are all
  // derived once by the collections adapter — this page just reads them.
  const treeNodes: TreeNode[] = collections?.roots ?? [];

  const onWish = (item: GTItem) => {
    // Ignore clicks while a mutation for this item is still settling — `wished`
    // and `wishlistData` haven't caught up yet, so acting now would double-add
    // or silently skip the remove.
    if (pendingWish.has(item.id)) return;
    if (!wished.has(item.id)) {
      markPending(item.id, true);
      addToWishlist({ itemId: item.id, name: item.name });
      return;
    }
    const row = wishlistEntries.find((e) => e.itemId === item.id);
    if (!row) return; // wishlist cache not refreshed yet; wait for it
    markPending(item.id, true);
    removeFromWishlist({
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
            refreshing={isRefreshing}
            onRefresh={() => {
              if (membershipType != null && !!membershipId) {
                refresh();
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
            onSelect={selectNode}
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
                  onChange={(e) => setFilter({ q: e.target.value })}
                />
              </div>
              <FilterChip
                on={missingOnly}
                onClick={() => setFilter({ missing: !missingOnly })}
              >
                Missing only
              </FilterChip>
              <FilterChip
                on={avail}
                onClick={() => setFilter({ avail: !avail })}
              >
                Available now
              </FilterChip>
              <FilterChip on={farm} onClick={() => setFilter({ farm: !farm })}>
                Hide farm-only
              </FilterChip>
              <Dropdown
                label="Rarity"
                value={rarity ? RARITY_LABEL[rarity] : null}
                options={RARITIES.map((r) => ({ v: r, l: RARITY_LABEL[r] }))}
                onPick={(v) => setFilter({ rarity: v as Rarity | null })}
              />
              <Dropdown
                label="Difficulty"
                value={diff ? DIFF_LABEL[diff] : null}
                options={DIFFS.map((d) => ({ v: d, l: DIFF_LABEL[d] }))}
                onPick={(v) => setFilter({ diff: v as Difficulty | null })}
                note="estimate"
              />
              <Dropdown
                label="Sort"
                value={
                  {
                    rarity: "Rarity",
                    name: "Name",
                    avail: "Availability",
                  }[sort]
                }
                options={[
                  { v: "rarity", l: "Rarity" },
                  { v: "name", l: "Name" },
                  { v: "avail", l: "Availability" },
                ]}
                onPick={(v) => v && setFilter({ sort: v as SortKey })}
                noClear
              />
            </div>
            <div className="gt-viewtoggle">
              <button
                className="gt-iconbtn"
                data-on={view === "grid"}
                onClick={() => setFilter({ view: "grid" })}
                aria-label="Grid"
              >
                <Icon name="grid" size="1rem" />
              </button>
              <button
                className="gt-iconbtn"
                data-on={view === "list"}
                onClick={() => setFilter({ view: "list" })}
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
                  searching
                    ? `No items match "${q}"`
                    : hasFilters
                      ? "No items match these filters"
                      : "All caught up!"
                }
                body={
                  searching
                    ? "Try a different search term, or clear the search to see this category's full list."
                    : hasFilters
                      ? "Try loosening a filter to see more of this category."
                      : "You've collected everything in this category. Nice work."
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
                  onOpen={openItem}
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
                  onOpen={openItem}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      {detail && (
        <ItemDetailDrawer
          item={detail}
          perkColumns={perkColumns}
          perksLoading={perksLoading}
          catalysts={catalysts}
          onClose={closeDetail}
          onWish={onWish}
          wished={wished.has(detail.id)}
        />
      )}
    </div>
  );
}
