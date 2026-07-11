import React, { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Badge,
  Button,
  EmptyState,
  FilterChip,
  ItemTile,
  Textarea,
} from "../../components/primitives";
import { Dropdown, PageHead } from "../../components/composite";
import { Icon } from "../../components/Icon";
import { useToast } from "../../components/Toast";
import { LoadingSpinner } from "../../components/LoadingSpinner";
import { apiFetch } from "../../lib/api";
import { toWishlistEntry } from "../../lib/adapters";
import type { WishListItem } from "../../types/api";
import type { Priority, WishlistEntry } from "../../types/design";

const PRIORITY_LABEL: Record<Priority, string> = {
  urgent: "Urgent",
  high: "High",
  medium: "Medium",
  low: "Low",
};

const PRIORITY_ORDER: Priority[] = ["urgent", "high", "medium", "low"];
type SortKey = "availability" | "priority";
type FilterKey = "all" | Priority;

export function WishList() {
  const { showToast } = useToast();
  const queryClient = useQueryClient();
  const [filter, setFilter] = useState<FilterKey>("all");
  const [sort, setSort] = useState<SortKey>("availability");
  // Inline notes editor: id of the row being edited + its draft text.
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftNotes, setDraftNotes] = useState("");

  const { data: rawItems = [], isLoading } = useQuery({
    queryKey: ["wishlist"],
    queryFn: () => apiFetch<WishListItem[]>("/api/wishlist"),
  });

  const list: WishlistEntry[] = useMemo(
    () => rawItems.map(toWishlistEntry),
    [rawItems],
  );

  const deleteMutation = useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/api/wishlist/${id}`, { method: "DELETE" }),
    onMutate: async (id: string) => {
      await queryClient.cancelQueries({ queryKey: ["wishlist"] });
      const previous = queryClient.getQueryData<WishListItem[]>(["wishlist"]);
      queryClient.setQueryData<WishListItem[]>(
        ["wishlist"],
        (old) => old?.filter((i) => i.id !== id) ?? [],
      );
      return { previous };
    },
    onError: (_err, _id, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["wishlist"], context.previous);
      }
      showToast("Failed to remove item", "error");
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: ["wishlist"] }),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, priority }: { id: string; priority: string }) =>
      apiFetch<WishListItem>(`/api/wishlist/${id}`, {
        method: "PUT",
        body: JSON.stringify({ priority }),
      }),
    onMutate: async ({ id, priority }) => {
      await queryClient.cancelQueries({ queryKey: ["wishlist"] });
      const previous = queryClient.getQueryData<WishListItem[]>(["wishlist"]);
      queryClient.setQueryData<WishListItem[]>(
        ["wishlist"],
        (old) => old?.map((i) => (i.id === id ? { ...i, priority } : i)) ?? [],
      );
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["wishlist"], context.previous);
      }
      showToast("Failed to update priority", "error");
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: ["wishlist"] }),
  });

  const notesMutation = useMutation({
    mutationFn: ({ id, notes }: { id: string; notes: string }) =>
      apiFetch<WishListItem>(`/api/wishlist/${id}`, {
        method: "PUT",
        body: JSON.stringify({ notes }),
      }),
    onMutate: async ({ id, notes }) => {
      await queryClient.cancelQueries({ queryKey: ["wishlist"] });
      const previous = queryClient.getQueryData<WishListItem[]>(["wishlist"]);
      queryClient.setQueryData<WishListItem[]>(
        ["wishlist"],
        (old) => old?.map((i) => (i.id === id ? { ...i, notes } : i)) ?? [],
      );
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["wishlist"], context.previous);
      }
      showToast("Failed to save notes", "error");
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: ["wishlist"] }),
  });

  const startEditNotes = (id: string, current: string) => {
    setEditingId(id);
    setDraftNotes(current);
  };
  const saveNotes = () => {
    if (editingId == null) return;
    notesMutation.mutate({ id: editingId, notes: draftNotes.trim() });
    setEditingId(null);
  };

  const setPriority = (id: string, p: Priority) => {
    updateMutation.mutate({ id, priority: p.toUpperCase() });
  };

  const remove = (id: string, name: string) => {
    deleteMutation.mutate(id);
    showToast(`Removed ${name}`, "info");
  };

  const [selectMode, setSelectMode] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(() => new Set());

  const toggleSelected = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  const exitSelectMode = () => {
    setSelectMode(false);
    setSelected(new Set());
  };

  const bulkMutation = useMutation({
    mutationFn: (vars: {
      action: "delete" | "set_priority";
      ids: string[];
      priority?: string;
    }) =>
      apiFetch<{ updated: number; skipped: number }>("/api/wishlist/bulk", {
        method: "POST",
        body: JSON.stringify({
          action: vars.action,
          ids: vars.ids.map(Number),
          ...(vars.priority ? { priority: vars.priority } : {}),
        }),
      }),
    onMutate: async (vars) => {
      await queryClient.cancelQueries({ queryKey: ["wishlist"] });
      const previous = queryClient.getQueryData<WishListItem[]>(["wishlist"]);
      const ids = new Set(vars.ids);
      queryClient.setQueryData<WishListItem[]>(["wishlist"], (old) => {
        if (!old) return [];
        if (vars.action === "delete") return old.filter((i) => !ids.has(i.id));
        return old.map((i) =>
          ids.has(i.id) ? { ...i, priority: vars.priority! } : i,
        );
      });
      return { previous };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) queryClient.setQueryData(["wishlist"], ctx.previous);
      showToast("Bulk action failed", "error");
    },
    onSuccess: (res, vars) => {
      const verb = vars.action === "delete" ? "removed" : "updated";
      showToast(
        res.skipped > 0
          ? `${res.updated} ${verb}, ${res.skipped} skipped`
          : `${res.updated} ${verb}`,
        "info",
      );
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: ["wishlist"] }),
  });

  const bulkDelete = () => {
    if (selected.size === 0) return;
    bulkMutation.mutate({ action: "delete", ids: Array.from(selected) });
    exitSelectMode();
  };
  const bulkSetPriority = (p: Priority) => {
    if (selected.size === 0) return;
    bulkMutation.mutate({
      action: "set_priority",
      ids: Array.from(selected),
      priority: p.toUpperCase(),
    });
    exitSelectMode();
  };

  const counts = useMemo(() => {
    const c: Record<string, number> = { all: list.length };
    PRIORITY_ORDER.forEach(
      (p) => (c[p] = list.filter((i) => i.priority === p).length),
    );
    return c;
  }, [list]);

  const shown = useMemo(() => {
    const l =
      filter === "all"
        ? list.slice()
        : list.filter((i) => i.priority === filter);
    if (sort === "availability")
      l.sort((a, b) => (b.avail.now ? 1 : 0) - (a.avail.now ? 1 : 0));
    else if (sort === "priority")
      l.sort(
        (a, b) =>
          PRIORITY_ORDER.indexOf(a.priority) -
          PRIORITY_ORDER.indexOf(b.priority),
      );
    return l;
  }, [list, filter, sort]);

  if (isLoading) {
    return (
      <div className="gt-page">
        <PageHead title="Wishlist" />
        <div
          className="gt-card"
          style={{ display: "flex", justifyContent: "center", padding: "3rem" }}
        >
          <LoadingSpinner />
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
            {list.length} items · {list.filter((i) => i.avail.now).length}{" "}
            available now
          </span>
        }
        right={
          <>
            <Button
              variant={selectMode ? "primary" : "ghost"}
              sm
              onClick={() =>
                selectMode ? exitSelectMode() : setSelectMode(true)
              }
            >
              {selectMode ? "Done" : "Select"}
            </Button>
            <Dropdown
              label="Sort: Availability"
              value={
                {
                  availability: "Sort: Availability",
                  priority: "Sort: Priority",
                }[sort]
              }
              noClear
              options={[
                { v: "availability", l: "Sort: Availability" },
                { v: "priority", l: "Sort: Priority" },
              ]}
              onPick={(v) => v && setSort(v as SortKey)}
            />
          </>
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

      {selectMode && (
        <div
          className="gt-card"
          role="region"
          aria-label="Bulk actions"
          style={{
            display: "flex",
            alignItems: "center",
            gap: "var(--s-2)",
            padding: "var(--s-2)",
            marginBottom: "var(--s-2)",
          }}
        >
          <label
            style={{ display: "flex", alignItems: "center", gap: "var(--s-1)" }}
          >
            <input
              type="checkbox"
              checked={shown.length > 0 && selected.size === shown.length}
              onChange={(e) =>
                setSelected(
                  e.target.checked
                    ? new Set(shown.map((i) => i.id))
                    : new Set(),
                )
              }
              aria-label="Select all shown"
            />
            <span className="mono">{selected.size} selected</span>
          </label>
          <Dropdown
            label="Set priority"
            value="Set priority"
            noClear
            options={PRIORITY_ORDER.map((p) => ({
              v: p,
              l: PRIORITY_LABEL[p],
            }))}
            onPick={(p) => p && bulkSetPriority(p as Priority)}
          />
          <button
            className="gt-link gt-link--danger"
            onClick={bulkDelete}
            disabled={selected.size === 0}
          >
            <Icon name="close" size="0.8rem" /> Delete
          </button>
        </div>
      )}

      <div className="gt-wl-list">
        {shown.map((i) => (
          <div key={i.id} className="gt-wl-item gt-card" data-rarity={i.rarity}>
            {selectMode && (
              <input
                type="checkbox"
                checked={selected.has(i.id)}
                onChange={() => toggleSelected(i.id)}
                aria-label={`Select ${i.name}`}
                style={{ alignSelf: "center", marginRight: "var(--s-1)" }}
              />
            )}
            <ItemTile
              rarity={i.rarity}
              type={i.type}
              icon={i.icon}
              style={{ width: "3rem" }}
            />
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
                <div className="gt-action-meta mono">
                  Source: {i.avail.where}
                </div>
              )}
              {editingId === i.id ? (
                <div className="gt-wl-notes-edit">
                  <Textarea
                    value={draftNotes}
                    onChange={setDraftNotes}
                    placeholder="Roll you're chasing, why you want it…"
                    maxLength={500}
                    autoFocus
                    ariaLabel={`Notes for ${i.name}`}
                  />
                  <div
                    style={{
                      display: "flex",
                      gap: "var(--s-2)",
                      marginTop: "var(--s-1)",
                    }}
                  >
                    <Button variant="primary" sm onClick={saveNotes}>
                      Save
                    </Button>
                    <Button
                      variant="ghost"
                      sm
                      onClick={() => setEditingId(null)}
                    >
                      Cancel
                    </Button>
                  </div>
                </div>
              ) : (
                i.notes && <div className="gt-wl-notes">"{i.notes}"</div>
              )}
              <div className="gt-wl-foot">
                <Dropdown
                  label="Priority"
                  value={PRIORITY_LABEL[i.priority]}
                  noClear
                  options={PRIORITY_ORDER.map((p) => ({
                    v: p,
                    l: PRIORITY_LABEL[p],
                  }))}
                  onPick={(p) => p && setPriority(i.id, p as Priority)}
                />
                <span className="gt-action-meta mono">Added {i.added}</span>
                <button
                  className="gt-link"
                  onClick={() => startEditNotes(i.id, i.notes)}
                >
                  <Icon name="settings" size="0.8rem" />{" "}
                  {i.notes ? "Edit notes" : "Add notes"}
                </button>
                <button
                  className="gt-link gt-link--danger"
                  onClick={() => remove(i.id, i.name)}
                >
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
