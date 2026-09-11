import { Icon } from "../../components/Icon";
import { Badge } from "../../components/primitives";
import { DIFF_LABEL } from "../../lib/constants";
import type { RecommendedAction } from "../../types/design";

export function ActionList({
  items,
  onToggle,
}: {
  items: RecommendedAction[];
  onToggle: (id: string) => void;
}) {
  return (
    <ul className="gt-actions">
      {items.map((a) => (
        <li key={a.id} className="gt-action" data-done={a.done}>
          <button
            className="gt-check"
            data-on={a.done}
            onClick={() => onToggle(a.id)}
            aria-label="Mark done"
          >
            {a.done && <Icon name="check" size="0.85rem" stroke={3} />}
          </button>
          <div className="gt-action-body">
            <div className="gt-action-top">
              <span className="gt-action-text">{a.text}</span>
              {a.badge && <Badge kind={a.badge} dot />}
            </div>
            <div className="gt-action-meta mono">
              {a.detail} ·{" "}
              <span style={{ color: "var(--c-text-2)" }}>{a.time}</span>
            </div>
          </div>
          <Badge kind={a.diff} style={{ flex: "none" }}>
            {DIFF_LABEL[a.diff]}
          </Badge>
        </li>
      ))}
    </ul>
  );
}
