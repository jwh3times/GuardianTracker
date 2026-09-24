import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { Button, EmptyState, FilterChip } from "../../components/primitives";
import { Icon } from "../../components/Icon";
import { PageHead } from "../../components/composite";
import { LoadingSpinner } from "../../components/LoadingSpinner";
import { QueryErrorPanel } from "../../components/QueryErrorPanel";
import { useToast } from "../../components/Toast";
import { ApiError } from "../../lib/api";
import { errorState } from "../../lib/errorState";
import {
  MAX_BULK_DELETE,
  useBulkDeleteRollTargets,
  useDeleteAllRollTargets,
  useRemoveRollTarget,
  useUpdateRollTargetNotes,
} from "../../data/rolltargets";
import type { RollTarget } from "../../types/design";
import { RollTargetImport } from "./RollTargetImport";
import { RollTargetMatchCard } from "./RollTargetMatchCard";
import { RollTargetRow } from "./RollTargetRow";
import { fallbackWeaponName } from "./rollTargetsView";
import { useRollTargetsBrowser } from "./useRollTargetsBrowser";

/**
 * The Roll targets page (ADR 0020, slice 5b). Reads `data/rolltargets.ts`
 * through `useRollTargetsBrowser` for query state and grouping/filtering;
 * owns only local UI state here (search/toggle live in the browser hook,
 * selection/notes-editing/confirmation live here, matching `WishList.tsx`'s
 * split).
 */
export function RollTargets() {
  const navigate = useNavigate();
  const { showToast } = useToast();

  const {
    targets,
    targetsLoading,
    targetsError,
    targetsErrorObj,
    retryTargets,
    matchesLoading,
    matchesError,
    matchesErrorObj,
    retryMatches,
    matchesReady,
    collectionsError,
    lookup,
    showAll,
    setShowAll,
    search,
    setSearch,
    stillChasing,
    wantedGroups,
    unwantedGroups,
  } = useRollTargetsBrowser();

  // Inline notes editor: id of the row being edited + its draft text —
  // shared across every section, matching WishList's single-editor pattern.
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftNotes, setDraftNotes] = useState("");

  const startEditNotes = (id: string, current: string) => {
    setEditingId(id);
    setDraftNotes(current);
  };

  const { setNotes: mutateNotes } = useUpdateRollTargetNotes({
    onError: () => showToast("Failed to save notes", "error"),
  });
  const saveNotes = () => {
    if (editingId == null) return;
    mutateNotes({ id: editingId, notes: draftNotes.trim() });
    setEditingId(null);
  };

  const { remove } = useRemoveRollTarget({
    onError: () => showToast("Failed to delete roll target", "error"),
  });
  const removeTarget = (target: RollTarget) => {
    remove({ id: target.id });
    showToast("Roll target deleted", "info");
  };

  const [selectMode, setSelectMode] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(() => new Set());
  const exitSelectMode = () => {
    setSelectMode(false);
    setSelected(new Set());
  };
  const toggleSelected = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else if (next.size >= MAX_BULK_DELETE) {
        showToast(`You can select up to ${MAX_BULK_DELETE} at a time`, "info");
        return prev;
      } else {
        next.add(id);
      }
      return next;
    });

  const { bulkDelete } = useBulkDeleteRollTargets({
    onSuccess: (result) => {
      showToast(
        result.skipped > 0
          ? `${result.deleted} deleted, ${result.skipped} skipped`
          : `${result.deleted} deleted`,
        "info",
      );
      exitSelectMode();
    },
    onError: () => showToast("Bulk delete failed", "error"),
  });
  const runBulkDelete = () => {
    if (selected.size === 0) return;
    bulkDelete(Array.from(selected));
  };

  const [confirmingDeleteAll, setConfirmingDeleteAll] = useState(false);
  const { deleteAll } = useDeleteAllRollTargets({
    onSuccess: () => showToast("All roll targets deleted", "info"),
    onError: () => showToast("Delete all failed", "error"),
  });

  // Rows shown when the match report failed: every saved target, neutral —
  // never "Still chasing" — per the owner decision on #350.
  const neutralTargets = useMemo(() => targets, [targets]);

  if (targetsLoading) {
    return (
      <div className="gt-page">
        <PageHead title="Roll targets" />
        <div
          className="gt-card"
          style={{ display: "flex", justifyContent: "center", padding: "3rem" }}
        >
          <LoadingSpinner />
        </div>
      </div>
    );
  }

  if (targetsError) {
    return (
      <div className="gt-page">
        <PageHead title="Roll targets" />
        <QueryErrorPanel error={targetsErrorObj} onRetry={retryTargets} />
      </div>
    );
  }

  const rowProps = (
    target: RollTarget,
    weaponName: string,
    icon?: string,
    type?: string,
  ) => ({
    target,
    weaponName,
    icon,
    type,
    editing: editingId === target.id,
    draftNotes,
    onStartEdit: () => startEditNotes(target.id, target.notes),
    onChangeDraft: setDraftNotes,
    onSaveNotes: saveNotes,
    onCancelEdit: () => setEditingId(null),
    onRemove: () => removeTarget(target),
  });

  return (
    <div className="gt-page">
      <PageHead
        title="Roll targets"
        sub={
          <span className="mono">
            {targets.length} saved{targets.length === 1 ? "" : ""}
          </span>
        }
      />

      <RollTargetImport />

      {targets.length === 0 ? (
        <div className="gt-card">
          <EmptyState
            icon="star"
            color="var(--c-signal)"
            title="Save the rolls you're chasing"
            body="A roll target is a specific perk combination on a weapon (or on any weapon that can roll it). Save one from the Collections item drawer, or import a DIM-format wish list above to add several at once."
          />
        </div>
      ) : (
        <>
          <div
            className="gt-card"
            role="region"
            aria-label="Manage roll targets"
            style={{
              display: "flex",
              alignItems: "center",
              gap: "var(--s-2)",
              flexWrap: "wrap",
              padding: "var(--s-2)",
              marginBottom: "var(--s-2)",
            }}
          >
            <Button
              variant={selectMode ? "primary" : "ghost"}
              sm
              onClick={() =>
                selectMode ? exitSelectMode() : setSelectMode(true)
              }
            >
              {selectMode ? "Done" : "Select"}
            </Button>
            {selectMode && (
              <>
                <span className="mono">{selected.size} selected</span>
                <button
                  className="gt-link gt-link--danger"
                  onClick={runBulkDelete}
                  disabled={selected.size === 0}
                >
                  <Icon name="close" size="0.8rem" /> Delete selected
                </button>
              </>
            )}
            <div style={{ marginLeft: "auto" }}>
              {!confirmingDeleteAll ? (
                <Button
                  variant="ghost"
                  sm
                  onClick={() => setConfirmingDeleteAll(true)}
                >
                  Delete all…
                </Button>
              ) : (
                <span
                  role="alertdialog"
                  aria-label="Confirm delete all roll targets"
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "var(--s-2)",
                  }}
                >
                  <span className="gt-action-meta">
                    Delete all {targets.length} roll targets?
                  </span>
                  <Button
                    variant="primary"
                    sm
                    onClick={() => {
                      deleteAll();
                      setConfirmingDeleteAll(false);
                    }}
                  >
                    Yes, delete all
                  </Button>
                  <Button
                    variant="ghost"
                    sm
                    onClick={() => setConfirmingDeleteAll(false)}
                  >
                    Cancel
                  </Button>
                </span>
              )}
            </div>
          </div>

          {matchesError ? (
            <>
              <MatchFailureBanner
                error={matchesErrorObj}
                onRetry={retryMatches}
                onReconnect={() => navigate("/reauthorize")}
              />
              <section aria-labelledby="rt-neutral-title">
                <h2 id="rt-neutral-title" className="gt-section-title">
                  Your roll targets
                </h2>
                <div className="gt-rt-list">
                  {neutralTargets.map((target) => (
                    <RollTargetRow
                      key={target.id}
                      {...rowProps(
                        target,
                        target.itemHash
                          ? lookup.name(target.itemHash) ||
                              fallbackWeaponName(target.itemHash)
                          : "",
                        target.itemHash
                          ? lookup.icon(target.itemHash)
                          : undefined,
                        target.itemHash
                          ? lookup.type(target.itemHash)
                          : undefined,
                      )}
                      neutral
                      selectable={selectMode}
                      selected={selected.has(target.id)}
                      onToggleSelect={() => toggleSelected(target.id)}
                    />
                  ))}
                </div>
              </section>
            </>
          ) : matchesLoading || !matchesReady || !stillChasing ? (
            <div
              className="gt-card"
              style={{
                display: "flex",
                justifyContent: "center",
                padding: "2rem",
              }}
            >
              <LoadingSpinner />
            </div>
          ) : (
            <>
              {collectionsError && (
                <p className="gt-action-meta" role="note">
                  Weapon names and the default filter may be limited while
                  Collections data is unavailable.
                </p>
              )}

              <section aria-labelledby="rt-chasing-title">
                <div className="gt-rt-section-head">
                  <h2 id="rt-chasing-title" className="gt-section-title">
                    Still chasing
                  </h2>
                  <div className="gt-filterbar">
                    <div className="gt-search gt-rt-search">
                      <Icon
                        name="search"
                        size="1rem"
                        style={{ color: "var(--c-text-3)" }}
                      />
                      <input
                        className="gt-search-input"
                        type="search"
                        aria-label="Search still-chasing roll targets"
                        placeholder="Search weapon or perk…"
                        maxLength={100}
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                      />
                    </div>
                    <FilterChip
                      on={showAll}
                      onClick={() => setShowAll(!showAll)}
                    >
                      Show all
                    </FilterChip>
                  </div>
                </div>
                {!showAll && stillChasing.hiddenCount > 0 && (
                  <p className="gt-action-meta">
                    {stillChasing.hiddenCount} roll target
                    {stillChasing.hiddenCount === 1 ? "" : "s"} hidden — not yet
                    collected or on your wish list.
                  </p>
                )}
                {stillChasing.groups.length === 0 ? (
                  <div className="gt-card">
                    <EmptyState
                      icon="star"
                      title={
                        search
                          ? `No roll targets match "${search}"`
                          : "Nothing left to chase"
                      }
                      body={
                        search
                          ? undefined
                          : "Every saved roll target is either matched below or hidden by the default filter."
                      }
                    />
                  </div>
                ) : (
                  <div className="gt-rt-groups">
                    {stillChasing.groups.map((group) => (
                      <div key={group.key} className="gt-rt-group gt-card">
                        <div className="gt-rt-group-head">
                          <span className="gt-item-name">{group.name}</span>
                          <span className="gt-action-meta mono">
                            {group.targets.length} roll
                            {group.targets.length === 1 ? "" : "s"}
                          </span>
                        </div>
                        <div className="gt-rt-list">
                          {group.targets.map((target) => (
                            <RollTargetRow
                              key={target.id}
                              {...rowProps(
                                target,
                                group.name,
                                group.icon,
                                group.type,
                              )}
                              bestCopy={target.bestCopy}
                              selectable={selectMode}
                              selected={selected.has(target.id)}
                              onToggleSelect={() => toggleSelected(target.id)}
                            />
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </section>

              <section aria-labelledby="rt-wanted-title">
                <h2 id="rt-wanted-title" className="gt-section-title">
                  You have it
                </h2>
                {wantedGroups.length === 0 ? (
                  <div className="gt-card">
                    <EmptyState
                      icon="check"
                      title="No owned weapon matches a wanted roll yet"
                    />
                  </div>
                ) : (
                  <div className="gt-rt-list">
                    {wantedGroups.map((group) => (
                      <RollTargetMatchCard
                        key={group.target.id}
                        group={group}
                        editing={editingId === group.target.id}
                        draftNotes={draftNotes}
                        onStartEdit={() =>
                          startEditNotes(group.target.id, group.target.notes)
                        }
                        onChangeDraft={setDraftNotes}
                        onSaveNotes={saveNotes}
                        onCancelEdit={() => setEditingId(null)}
                        onRemove={() => removeTarget(group.target)}
                      />
                    ))}
                  </div>
                )}
              </section>

              <UnwantedDisclosure
                groups={unwantedGroups}
                editingId={editingId}
                draftNotes={draftNotes}
                onStartEdit={startEditNotes}
                onChangeDraft={setDraftNotes}
                onSaveNotes={saveNotes}
                onCancelEdit={() => setEditingId(null)}
                onRemove={removeTarget}
              />
            </>
          )}
        </>
      )}
    </div>
  );
}

function MatchFailureBanner({
  error,
  onRetry,
  onReconnect,
}: {
  error: unknown;
  onRetry: () => void;
  onReconnect: () => void;
}) {
  const isReauth =
    error instanceof ApiError && error.code === "BUNGIE_REAUTH_REQUIRED";
  const copy = errorState(error);
  return (
    <div
      className="gt-card"
      role="alert"
      style={{ marginBottom: "var(--s-3)" }}
    >
      <EmptyState
        icon={copy.icon}
        color="var(--c-text-3)"
        title={isReauth ? "Reconnect Bungie to see matches" : copy.title}
        body={
          isReauth
            ? "Your saved roll targets are shown below with an unknown match status until you reconnect."
            : `${copy.body} Your saved roll targets are shown below with an unknown match status.`
        }
        action={
          <Button
            variant="outline"
            sm
            onClick={isReauth ? onReconnect : onRetry}
          >
            {isReauth ? "Reconnect" : "Retry"}
          </Button>
        }
      />
    </div>
  );
}

function UnwantedDisclosure({
  groups,
  editingId,
  draftNotes,
  onStartEdit,
  onChangeDraft,
  onSaveNotes,
  onCancelEdit,
  onRemove,
}: {
  groups: ReturnType<typeof useRollTargetsBrowser>["unwantedGroups"];
  editingId: string | null;
  draftNotes: string;
  onStartEdit: (id: string, notes: string) => void;
  onChangeDraft: (v: string) => void;
  onSaveNotes: () => void;
  onCancelEdit: () => void;
  onRemove: (target: RollTarget) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  if (groups.length === 0) return null;
  return (
    <section aria-labelledby="rt-unwanted-title" className="gt-rt-disclosure">
      <button
        className="gt-rt-disclosure-head"
        onClick={() => setExpanded((e) => !e)}
        aria-expanded={expanded}
      >
        <h2 id="rt-unwanted-title" className="gt-section-title">
          Matches a roll you marked unwanted ({groups.length})
        </h2>
        <Icon
          name="chevronDown"
          size="1rem"
          style={{
            color: "var(--c-text-3)",
            transform: expanded ? "rotate(180deg)" : "none",
            transition: "transform var(--dur)",
          }}
        />
      </button>
      {expanded && (
        <div className="gt-rt-list">
          {groups.map((group) => (
            <RollTargetMatchCard
              key={group.target.id}
              group={group}
              editing={editingId === group.target.id}
              draftNotes={draftNotes}
              onStartEdit={() =>
                onStartEdit(group.target.id, group.target.notes)
              }
              onChangeDraft={onChangeDraft}
              onSaveNotes={onSaveNotes}
              onCancelEdit={onCancelEdit}
              onRemove={() => onRemove(group.target)}
            />
          ))}
        </div>
      )}
    </section>
  );
}
