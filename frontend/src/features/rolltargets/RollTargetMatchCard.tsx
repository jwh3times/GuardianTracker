import { Button, ItemTile, Textarea } from "../../components/primitives";
import { Icon } from "../../components/Icon";
import { isTargetPerk } from "./rollTargetsView";
import type { MatchGroup } from "./rollTargetsView";

/**
 * One target's "You have it" (or informational "marked unwanted") card: the
 * target's own heading/perks/note, then each owned copy that satisfies it —
 * "Copy 1", "Copy 2", … — with the copy's full perks and the target's own
 * perks highlighted among them. Copies carry no location or power (the match
 * report doesn't have it), so the label is the only thing distinguishing them.
 */
export function RollTargetMatchCard({
  group,
  editing,
  draftNotes,
  onStartEdit,
  onChangeDraft,
  onSaveNotes,
  onCancelEdit,
  onRemove,
}: {
  group: MatchGroup;
  editing: boolean;
  draftNotes: string;
  onStartEdit: () => void;
  onChangeDraft: (v: string) => void;
  onSaveNotes: () => void;
  onCancelEdit: () => void;
  onRemove: () => void;
}) {
  const { target } = group;
  return (
    <div className="gt-rt-row gt-card">
      <ItemTile
        rarity="legendary"
        type={group.type}
        icon={group.icon}
        style={{ width: "3rem" }}
      />
      <div className="gt-rt-row-body">
        <div className="gt-rt-row-top">
          <span className="gt-item-name">{group.heading}</span>
        </div>
        {!target.anyWeapon && target.perks.length > 0 && (
          <div className="gt-rt-perks">
            {target.perks.map((p) => (
              <span key={p} className="gt-chip">
                {p}
              </span>
            ))}
          </div>
        )}
        {editing ? (
          <div className="gt-wl-notes-edit">
            <Textarea
              value={draftNotes}
              onChange={onChangeDraft}
              placeholder="Why you're chasing this roll…"
              maxLength={500}
              autoFocus
              ariaLabel={`Notes for ${group.heading}`}
            />
            <div
              style={{
                display: "flex",
                gap: "var(--s-2)",
                marginTop: "var(--s-1)",
              }}
            >
              <Button variant="primary" sm onClick={onSaveNotes}>
                Save
              </Button>
              <Button variant="ghost" sm onClick={onCancelEdit}>
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          target.notes && <div className="gt-wl-notes">"{target.notes}"</div>
        )}

        <ul className="gt-rt-copies">
          {group.copies.map((copy, i) => (
            <li key={copy.instanceId} className="gt-rt-copy">
              <span className="gt-rt-copy-label mono">Copy {i + 1}</span>
              <div className="gt-rt-perks">
                {copy.perks.map((p) => (
                  <span
                    key={p}
                    className="gt-chip"
                    data-highlight={isTargetPerk(p, copy.targetPerks)}
                  >
                    {p}
                  </span>
                ))}
              </div>
            </li>
          ))}
        </ul>

        <div className="gt-wl-foot">
          <button className="gt-link" onClick={onStartEdit}>
            <Icon name="settings" size="0.8rem" />{" "}
            {target.notes ? "Edit notes" : "Add notes"}
          </button>
          <button className="gt-link gt-link--danger" onClick={onRemove}>
            <Icon name="close" size="0.8rem" /> Delete
          </button>
        </div>
      </div>
    </div>
  );
}
