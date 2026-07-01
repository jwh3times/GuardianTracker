import { useEffect } from "react";
import type { GTItem } from "../../types/design";
import { ItemTile, Button } from "../../components/primitives";
import { Icon } from "../../components/Icon";

export function CosmeticDetail({
  item,
  onClose,
}: {
  item: GTItem | null;
  onClose: () => void;
}) {
  useEffect(() => {
    if (!item) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [item, onClose]);

  if (!item) return null;

  return (
    <div className="gt-drawer-scrim" onClick={onClose}>
      <aside
        className="gt-drawer"
        data-rarity={item.rarity}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
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
            <h2 className="gt-drawer-name">{item.name}</h2>
            <div className="gt-item-type">{item.type}</div>
            <div className="gt-cosmetic-detail-state">
              {item.collected ? "● Collected" : "○ Not collected"}
            </div>
          </div>
        </div>
        {item.desc && <p className="gt-drawer-desc">{item.desc}</p>}
        <div className="gt-drawer-actions">
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
