import React, { useEffect, useState } from "react";
import { Icon } from "../../components/Icon";
import { Badge, Button, ItemTile } from "../../components/primitives";
import { DIFF_LABEL } from "../../lib/constants";
import type { GTItem, ItemCatalyst, PerkColumn } from "../../types/design";

type CSS = React.CSSProperties & Record<`--${string}`, string | number>;

export function ItemDetailDrawer({
  item,
  onClose,
  onWish,
  wished,
  perkColumns,
  perksLoading,
  catalysts,
}: {
  item: GTItem | null;
  onClose: () => void;
  onWish: (item: GTItem) => void;
  wished: boolean;
  perkColumns?: PerkColumn[];
  perksLoading?: boolean;
  catalysts?: ItemCatalyst[];
}) {
  const [showWhy, setShowWhy] = useState(false);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);
  if (!item) return null;
  const viewOnly = !!item.viewOnly;
  return (
    <div className="gt-drawer-scrim" onClick={onClose}>
      <aside
        className="gt-drawer"
        data-rarity={item.rarity}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-label={item.name}
      >
        <button
          className="gt-drawer-close gt-iconbtn"
          onClick={onClose}
          aria-label="Close"
        >
          <Icon name="close" size="1.1rem" />
        </button>
        <div className="gt-drawer-hero">
          <ItemTile
            rarity={item.rarity}
            type={item.type}
            icon={item.icon}
            style={{ width: "4.6rem" }}
          />
          <div>
            <div
              className="gt-item-badges"
              style={{ marginBottom: "var(--s-1)" } as CSS}
            >
              <Badge kind={item.rarity} dot lg />
            </div>
            <h2 className="gt-drawer-name">{item.name}</h2>
            <div className="gt-item-type">
              {item.type}
              {item.slot ? ` · ${item.slot}` : ""}
            </div>
          </div>
        </div>

        {viewOnly && (
          <p className="gt-drawer-desc" style={{ color: "var(--c-text-3)" }}>
            Not in your trackable collections — view only.
          </p>
        )}

        {!viewOnly && item.availableNow && !item.collected && (
          <div className="gt-drawer-avail">
            <Badge kind="avail-now" dot icon="bolt" lg />
            <span>
              {item.availFrom
                ? `Available now — ${item.availFrom}`
                : "Available right now — see source below."}
            </span>
          </div>
        )}

        <p className="gt-drawer-desc">{item.desc}</p>

        {!viewOnly && (
          <div className="gt-drawer-block">
            <div className="gt-section-title">
              Acquisition sources
              {item.farmOnly && <Badge kind="farmonly">Farm only</Badge>}
              {item.acquisitionSources.length > 0 && (
                <button
                  className="gt-why"
                  onClick={() => setShowWhy((v) => !v)}
                >
                  why?{" "}
                  <Icon
                    name="chevronDown"
                    size="0.7rem"
                    style={{ transform: showWhy ? "rotate(180deg)" : "none" }}
                  />
                </button>
              )}
            </div>
            {item.acquisitionSources.length === 0 ? (
              <p className="gt-why-text">No acquisition sources reported.</p>
            ) : (
              item.acquisitionSources.map((source, index) => (
                <div
                  className="gt-drawer-src"
                  data-diff={source.difficulty}
                  key={`${source.text}-${index}`}
                >
                  <Icon
                    name="bolt"
                    size="0.9rem"
                    style={{ color: "var(--rarity)" }}
                  />
                  <div>
                    <div style={{ color: "var(--c-text)" }}>{source.text}</div>
                    <Badge kind={source.difficulty}>
                      {DIFF_LABEL[source.difficulty]}
                    </Badge>
                    {showWhy && (
                      <p className="gt-why-text">
                        {source.difficulty === "unrated"
                          ? "No difficulty tier was inferred from this source."
                          : "Difficulty is estimated from this source; treat it as a guide, not a guarantee."}
                      </p>
                    )}
                  </div>
                </div>
              ))
            )}
            {item.farmOnly && (
              <p className="gt-why-text">
                Random perks — earn a fresh drop from its source; this item
                can't be pulled from Collections.
              </p>
            )}
          </div>
        )}

        {(perksLoading || (perkColumns && perkColumns.length > 0)) && (
          <div className="gt-drawer-block">
            <div className="gt-section-title">Possible perks / rolls</div>
            {perksLoading ? (
              <div className="gt-perks-loading">Loading perks…</div>
            ) : (
              <div className="gt-perk-cols">
                {perkColumns!.map((col) => (
                  <div key={col.label} className="gt-perk-col">
                    <div className="gt-perk-col-label">{col.label}</div>
                    <div className="gt-perks">
                      {col.perks.map((p) => (
                        <span key={p} className="gt-perk">
                          {p}
                        </span>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {catalysts && catalysts.length > 0 && (
          <div className="gt-drawer-block">
            <div className="gt-section-title">Catalyst</div>
            <ul className="gt-catalyst-list">
              {catalysts.map((c) => (
                <li key={c.name} className="gt-catalyst">
                  <div className="gt-catalyst-name">{c.name}</div>
                  {c.description && (
                    <p className="gt-catalyst-desc">{c.description}</p>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}

        <div className="gt-drawer-actions">
          {!viewOnly && (
            <Button
              variant="primary"
              icon="wishlist"
              onClick={() => onWish(item)}
              style={{ flex: 1 }}
            >
              {wished ? "On wishlist" : "Add to Wishlist"}
            </Button>
          )}
          <Button
            variant="outline"
            icon="external"
            onClick={() =>
              window.open(
                `https://www.light.gg/db/items/${item.id}`,
                "_blank",
                "noopener,noreferrer",
              )
            }
          >
            View on light.gg ↗
          </Button>
        </div>
      </aside>
    </div>
  );
}
