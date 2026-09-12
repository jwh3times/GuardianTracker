import React, { useMemo, useState } from "react";
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
import { QueryErrorPanel } from "../../components/QueryErrorPanel";
import {
  useBulkWishlistAction,
  useRemoveWishlistItem,
  useSetWishlistNotes,
  useSetWishlistPriority,
  useWishlist,
} from "../../data/wishlist";
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

function acquisitionSourceSummary(item: WishlistEntry): string {
  if (item.acquisitionSources.length === 1)
    return item.acquisitionSources[0].text;
  if (item.acquisitionSources.length > 1)
    return `${item.acquisitionSources.length} acquisition sources`;
  return "No acquisition sources reported.";
}

export function WishList() {
  const { showToast } = useToast();
  const [filter, setFilter] = useState<FilterKey>("all");
  const [sort, setSort] = useState<SortKey>("availability");
  // Inline notes editor: id of the row being edited + its draft text.
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftNotes, setDraftNotes] = useState("");

  const { entries: list, isLoading, isError, error, retry } = useWishlist();

  const { remove } = useRemoveWishlistItem({
    onError: () => showToast("Failed to remove item", "error"),
  });

  const { setPriority: mutatePriority } = useSetWishlistPriority({
    onError: () => showToast("Failed to update priority", "error"),
  });

  const { setNotes: mutateNotes } = useSetWishlistNotes({
    onError: () => showToast("Failed to save notes", "error"),
  });

  const startEditNotes = (id: string, current: string) => {
    setEditingId(id);
    setDraftNotes(current);
  };
  const saveNotes = () => {
    if (editingId == null) return;
    mutateNotes({ rowId: editingId, notes: draftNotes.trim() });
    setEditingId(null);
  };

  const setPriority = (id: string, p: Priority) => {
    mutatePriority({ rowId: id, priority: p });
  };

  const removeRow = (id: string, name: string) => {
    remove({ rowId: id, name });
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

  const { runBulkAction } = useBulkWishlistAction({
    onError: () => showToast("Bulk action failed", "error"),
    onSuccess: (res, vars) => {
      const verb = vars.action === "delete" ? "removed" : "updated";
      showToast(
        res.skipped > 0
          ? `${res.updated} ${verb}, ${res.skipped} skipped`
          : `${res.updated} ${verb}`,
        "info",
      );
      exitSelectMode();
    },
  });

  const bulkDelete = () => {
    if (selected.size === 0) return;
    runBulkAction({ action: "delete", rowIds: Array.from(selected) });
  };
  const bulkSetPriority = (p: Priority) => {
    if (selected.size === 0) return;
    runBulkAction({
      action: "set_priority",
      rowIds: Array.from(selected),
      priority: p,
    });
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

  // Must precede the empty-state branch: `data` defaults to [] on failure, so a
  // failed fetch would otherwise render "Your wishlist is empty" and invite the
  // user to start over.
  if (isError) {
    return (
      <div className="gt-page">
        <PageHead title="Wishlist" />
        <QueryErrorPanel error={error} onRetry={retry} />
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
            disabled={selected.size === 0}
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
              {i.avail.now && (
                <div className="gt-wl-avail">
                  <Badge kind="avail-now" dot icon="bolt" />
                  <span className="gt-wl-where">{i.avail.where}</span>
                </div>
              )}
              <div className="gt-action-meta mono">
                {i.acquisitionSources.length === 1 && (
                  <span>Acquisition source: </span>
                )}
                <span>{acquisitionSourceSummary(i)}</span>
              </div>
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
                  onClick={() => removeRow(i.id, i.name)}
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
