import React, { useMemo, useState } from "react";
import { useMutation, useQuery } from "@apollo/client";
import {
  Badge,
  Button,
  Dropdown,
  EmptyState,
  Icon,
  ItemTile,
  PageHead,
} from "../components/kit";
import { FilterChip } from "../components/kit";
import { GET_WISH_LIST } from "../graphql/queries";
import {
  REMOVE_FROM_WISH_LIST,
  UPDATE_WISH_LIST_ITEM,
} from "../graphql/mutations";
import { useToast } from "../components/ui/Toast";
import { toWishlistEntry } from "../lib/adapters";
import { PRIORITY_LABEL, wishlist as MOCK_WISHLIST } from "../lib/mockData";
import type { Priority, WishlistEntry } from "../types/design";
import type { WishListItem } from "../types";

const PRIORITY_ORDER: Priority[] = ["urgent", "high", "medium", "low"];
const PRIORITY_ENUM: Record<Priority, string> = {
  urgent: "URGENT",
  high: "HIGH",
  medium: "MEDIUM",
  low: "LOW",
};
type SortKey = "availability" | "priority";
type FilterKey = "all" | Priority;

export function WishList() {
  const { showToast } = useToast();
  const { data, loading, refetch } = useQuery(GET_WISH_LIST);
  const [removeFromWishList] = useMutation(REMOVE_FROM_WISH_LIST);
  const [updateWishListItem] = useMutation(UPDATE_WISH_LIST_ITEM);

  const [filter, setFilter] = useState<FilterKey>("all");
  const [sort, setSort] = useState<SortKey>("availability");
  // Local copy used only when falling back to mock data (no live wishlist).
  const [mockList, setMockList] = useState<WishlistEntry[]>(MOCK_WISHLIST);

  const realRaw = data?.currentUser?.wishList as WishListItem[] | undefined;
  const usingReal = !!(realRaw && realRaw.length);
  const list: WishlistEntry[] = usingReal ? realRaw!.map(toWishlistEntry) : mockList;

  const counts = useMemo(() => {
    const c: Record<string, number> = { all: list.length };
    PRIORITY_ORDER.forEach((p) => (c[p] = list.filter((i) => i.priority === p).length));
    return c;
  }, [list]);

  const shown = useMemo(() => {
    const l = filter === "all" ? list.slice() : list.filter((i) => i.priority === filter);
    if (sort === "availability") l.sort((a, b) => (b.avail.now ? 1 : 0) - (a.avail.now ? 1 : 0));
    else if (sort === "priority")
      l.sort((a, b) => PRIORITY_ORDER.indexOf(a.priority) - PRIORITY_ORDER.indexOf(b.priority));
    return l;
  }, [list, filter, sort]);

  const setPriority = async (id: string, p: Priority) => {
    if (usingReal) {
      try {
        await updateWishListItem({ variables: { wishListItemId: id, priority: PRIORITY_ENUM[p] } });
        await refetch();
      } catch (err) {
        showToast(`Failed to update priority: ${(err as Error).message}`, "error");
      }
    } else {
      setMockList((l) => l.map((i) => (i.id === id ? { ...i, priority: p } : i)));
    }
  };

  const remove = async (id: string, name: string) => {
    if (usingReal) {
      try {
        await removeFromWishList({ variables: { wishListItemId: id } });
        await refetch();
        showToast(`Removed ${name}`, "info");
      } catch (err) {
        showToast(`Failed to remove item: ${(err as Error).message}`, "error");
      }
    } else {
      setMockList((l) => l.filter((i) => i.id !== id));
      showToast(`Removed ${name}`, "info");
    }
  };

  if (loading && !data) {
    return (
      <div className="gt-page">
        <PageHead title="Wishlist" />
        <div className="gt-itemlist">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="gt-skel" style={{ height: "5.5rem", borderRadius: "var(--r-lg)" }} />
          ))}
        </div>
      </div>
    );
  }

  if (list.length === 0) {
    return (
      <div className="gt-page">
        <PageHead title="Wishlist" />
        <div className="gt-card">
          <EmptyState
            icon="wishlist"
            color="var(--c-signal)"
            title="Your wishlist is empty"
            body="Track the items you're chasing. We'll tell you the moment they're available from a vendor or this week's activities."
            action={
              <a href="/collections">
                <Button variant="primary" icon="collections">
                  Browse Collections
                </Button>
              </a>
            }
          />
        </div>
      </div>
    );
  }

  return (
    <div className="gt-page">
      <PageHead
        title="Wishlist"
        sub={
          <span className="mono">
            {list.length} items · {list.filter((i) => i.avail.now).length} available now
          </span>
        }
        right={
          <Dropdown
            label="Sort: Availability"
            value={{ availability: "Sort: Availability", priority: "Sort: Priority" }[sort]}
            noClear
            options={[
              { v: "availability", l: "Sort: Availability" },
              { v: "priority", l: "Sort: Priority" },
            ]}
            onPick={(v) => v && setSort(v as SortKey)}
          />
        }
      />

      <div className="gt-filterbar gt-wl-filters">
        {(["all", ...PRIORITY_ORDER] as FilterKey[]).map((p) => (
          <FilterChip key={p} on={filter === p} onClick={() => setFilter(p)}>
            {p === "all" ? "All" : PRIORITY_LABEL[p]}{" "}
            <span className="mono" style={{ opacity: 0.6 }}>
              {counts[p]}
            </span>
          </FilterChip>
        ))}
      </div>

      <div className="gt-wl-list">
        {shown.map((i) => (
          <div key={i.id} className="gt-wl-item gt-card" data-rarity={i.rarity}>
            <ItemTile rarity={i.rarity} type={i.type} style={{ width: "3rem" }} />
            <div className="gt-wl-body">
              <div className="gt-wl-top">
                <div>
                  <div className="gt-item-name">{i.name}</div>
                  <div className="gt-item-type">
                    {i.type} · <Badge kind={i.rarity} dot />
                  </div>
                </div>
                <Badge kind={i.priority} solid>
                  {PRIORITY_LABEL[i.priority]}
                </Badge>
              </div>
              {i.avail.now ? (
                <div className="gt-wl-avail">
                  <Badge kind="avail-now" dot icon="bolt" />
                  <span className="gt-wl-where">{i.avail.where}</span>
                </div>
              ) : (
                <div className="gt-action-meta mono">Source: {i.avail.where}</div>
              )}
              {i.notes && <div className="gt-wl-notes">“{i.notes}”</div>}
              <div className="gt-wl-foot">
                <Dropdown
                  label="Priority"
                  value={PRIORITY_LABEL[i.priority]}
                  noClear
                  options={PRIORITY_ORDER.map((p) => ({ v: p, l: PRIORITY_LABEL[p] }))}
                  onPick={(p) => p && setPriority(i.id, p as Priority)}
                />
                <span className="gt-action-meta mono">Added {i.added}</span>
                <button className="gt-link gt-link--danger" onClick={() => remove(i.id, i.name)}>
                  <Icon name="close" size="0.8rem" /> Remove
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
