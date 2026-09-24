import { Button, ItemTile, Textarea } from "../../components/primitives";
import { Icon } from "../../components/Icon";
import type { RollTarget, RollTargetNearMiss } from "../../types/design";
import { NearMissSummary } from "./NearMissSummary";

/**
 * One saved roll target's card: weapon icon + name (or the any-weapon
 * phrasing), its perks as chips, its note, and the management affordances
 * (select, edit notes, delete). Shared by the "Still chasing" grouped view
 * and the neutral "match status unknown" list a match-report failure falls
 * back to — the two differ only in framing (goal vs. neutral), not in what
 * a row can do.
 */
export function RollTargetRow({
  target,
  weaponName,
  icon,
  type,
  neutral,
  bestCopy,
  selectable,
  selected,
  onToggleSelect,
  editing,
  draftNotes,
  onStartEdit,
  onChangeDraft,
  onSaveNotes,
  onCancelEdit,
  onRemove,
}: {
  target: RollTarget;
  weaponName: string;
  icon?: string;
  type?: string;
  /** Renders a "Match status unknown" badge instead of any chasing framing. */
  neutral?: boolean;
  /** Backend-selected best partial copy; absent when no target perk matches. */
  bestCopy?: RollTargetNearMiss;
  selectable?: boolean;
  selected?: boolean;
  onToggleSelect?: () => void;
  editing: boolean;
  draftNotes: string;
  onStartEdit: () => void;
  onChangeDraft: (v: string) => void;
  onSaveNotes: () => void;
  onCancelEdit: () => void;
  onRemove: () => void;
}) {
  const heading = target.anyWeapon
    ? `Any weapon with ${target.perks.join(" + ")}`
    : weaponName;

  return (
    <div className="gt-rt-row gt-card">
      {selectable && (
        <input
          type="checkbox"
          checked={!!selected}
          onChange={onToggleSelect}
          aria-label={`Select ${heading}`}
          className="gt-rt-row-check"
        />
      )}
      <ItemTile
        rarity="legendary"
        type={type}
        icon={icon}
        style={{ width: "3rem" }}
      />
      <div className="gt-rt-row-body">
        <div className="gt-rt-row-top">
          <span className="gt-item-name">{heading}</span>
          {neutral && (
            <span className="gt-chip gt-rt-chip-neutral">
              Match status unknown
            </span>
          )}
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
        {!neutral && (
          <NearMissSummary
            bestCopy={bestCopy}
            targetPerkCount={target.perks.length}
          />
        )}
        {editing ? (
          <div className="gt-wl-notes-edit">
            <Textarea
              value={draftNotes}
              onChange={onChangeDraft}
              placeholder="Why you're chasing this roll…"
              maxLength={500}
              autoFocus
              ariaLabel={`Notes for ${heading}`}
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
