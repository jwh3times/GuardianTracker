import { useState } from "react";
import { Button } from "../../components/primitives";
import { useToast } from "../../components/Toast";
import { useDeleteRollTargetImport } from "../../data/rolltargets";
import type { RollTarget } from "../../types/design";

interface ImportGroup {
  id: string;
  title?: string;
  date: string;
  count: number;
}

/** Groups the complete saved list, independently of match state and page filters. */
export function RollTargetImports({ targets }: { targets: RollTarget[] }) {
  const groups = new Map<string, ImportGroup>();
  for (const target of targets) {
    if (!target.importId) continue;
    const group = groups.get(target.importId);
    if (group) {
      group.count += 1;
      if (target.dateAdded < group.date) group.date = target.dateAdded;
    } else {
      groups.set(target.importId, {
        id: target.importId,
        title: target.importTitle,
        date: target.dateAdded,
        count: 1,
      });
    }
  }
  const [confirming, setConfirming] = useState<ImportGroup | null>(null);
  const [failed, setFailed] = useState(false);
  const { showToast } = useToast();
  const { deleteImport, isPending } = useDeleteRollTargetImport({
    onSuccess: (result) => {
      setConfirming(null);
      showToast(`${result.deleted} imported roll targets deleted`, "info");
    },
    onError: () => setFailed(true),
  });

  if (groups.size === 0 && !confirming) return null;
  return (
    <section
      className="gt-card gt-rt-imports"
      aria-label="Imported roll targets"
    >
      <h2 className="gt-section-title">Imported roll targets</h2>
      <p className="gt-action-meta">
        Each import groups only the targets it added. Already saved rolls keep
        their original source. To replace an import, delete it and import your
        file again.
      </p>
      <div className="gt-rt-list">
        {Array.from(groups.values()).map((group) => (
          <div className="gt-rt-group" key={group.id}>
            <div className="gt-rt-group-head">
              <span className="gt-item-name">
                {group.title || "Untitled DIM import"}
              </span>
              <span className="gt-action-meta">
                {group.count} saved target{group.count === 1 ? "" : "s"}
              </span>
              <Button
                variant="ghost"
                sm
                disabled={isPending}
                aria-label={`Delete import ${group.id}`}
                onClick={() => {
                  setConfirming(group);
                  setFailed(false);
                }}
              >
                Delete import…
              </Button>
            </div>
            <p className="gt-action-meta">
              <time dateTime={group.date}>
                {new Date(group.date).toLocaleString()}
              </time>
              {" · Import ID: "}
              <span className="mono">{group.id}</span>
            </p>
          </div>
        ))}
      </div>
      {confirming && (
        <div
          role="alertdialog"
          aria-label="Confirm delete import"
          aria-describedby="rt-import-confirm"
        >
          <p id="rt-import-confirm">
            Delete every remaining target from{" "}
            {confirming.title || "this DIM import"} ({confirming.id})? Other
            imports and individually saved targets are kept.
          </p>
          {failed && (
            <p role="alert">Could not delete this import. Try again.</p>
          )}
          <Button
            variant="primary"
            sm
            disabled={isPending}
            onClick={() => {
              setFailed(false);
              deleteImport(confirming.id);
            }}
          >
            {isPending ? "Deleting…" : "Yes, delete import"}
          </Button>
          <Button
            variant="ghost"
            sm
            disabled={isPending}
            onClick={() => setConfirming(null)}
          >
            Cancel
          </Button>
        </div>
      )}
    </section>
  );
}
