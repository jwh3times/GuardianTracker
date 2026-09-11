import {
  Badge,
  CountdownChip,
  EmptyState,
  ItemTile,
} from "../../components/primitives";
import { Panel } from "../../components/composite";
import type { Xur } from "../../types/design";

export function XurModule({
  xur,
  activeClassName,
  onSelect,
}: {
  xur: Xur | null | undefined;
  activeClassName?: string;
  onSelect?: (hash: string) => void;
}) {
  if (!xur || !xur.present) {
    return (
      <Panel title="Xûr — Agent of Nine" icon="bungie">
        <EmptyState
          icon="bungie"
          title="Xûr returns Friday"
          body="The weekend exotic vendor is away. Check back in 3d 2h."
        />
      </Panel>
    );
  }
  return (
    <Panel
      title="Xûr — weekend exotics"
      icon="bungie"
      accent="var(--c-exotic)"
      right={
        <CountdownChip prefix="Leaves" time={xur.leavesIn} soon icon="clock" />
      }
    >
      {xur.location ? (
        <div className="gt-xur-loc mono">{xur.location}</div>
      ) : null}
      {xur.items.length === 0 ? (
        <EmptyState
          icon="bungie"
          title="Inventory unavailable"
          body="Xûr is in the system, but his stock could not be loaded."
        />
      ) : (
        <ul className="gt-vendor-list">
          {xur.items.map((it) => {
            const body = (
              <>
                <ItemTile
                  rarity={it.rarity}
                  type={it.type}
                  icon={it.icon}
                  style={{ width: "2.4rem" }}
                />
                <div className="gt-vendor-main">
                  <div className="gt-item-name">{it.name}</div>
                  <div className="gt-item-type">
                    {it.type} · <span className="mono">{it.cost}</span>
                  </div>
                  {it.className && (
                    <div
                      className="gt-item-badges"
                      style={{ marginTop: "var(--s-1)" }}
                    >
                      <Badge kind="for-you" dot>
                        {it.className === activeClassName
                          ? `For your ${it.className}`
                          : `${it.className} armor`}
                      </Badge>
                    </div>
                  )}
                </div>
                {it.missing ? (
                  <Badge kind="missing" dot icon="bolt" />
                ) : (
                  <Badge kind="owned" dot>
                    Owned
                  </Badge>
                )}
              </>
            );
            // The row stays a listitem and the click target is a real button.
            // role="button" on the <li> replaces its implicit listitem role,
            // which leaves the <ul> with no listitem children (axe "list").
            // A native button also brings Enter/Space and focus for free.
            return (
              <li
                key={it.hash}
                className="gt-vendor-item"
                data-rarity={it.rarity}
              >
                {onSelect ? (
                  <button
                    type="button"
                    className="gt-vendor-item-btn"
                    onClick={() => onSelect(it.hash)}
                  >
                    {body}
                  </button>
                ) : (
                  body
                )}
              </li>
            );
          })}
        </ul>
      )}
    </Panel>
  );
}
