import React from "react";
import { Icon } from "../../components/Icon";
import { Badge, Button, ItemTile } from "../../components/primitives";
import type { GTItem } from "../../types/design";

type CSS = React.CSSProperties & Record<`--${string}`, string | number>;

export type ItemCardDensity = "grid" | "list" | "compact";

interface ItemCardProps {
  item: GTItem;
  density?: ItemCardDensity;
  personalize?: "off" | "normal" | "aggressive";
  onOpen?: (item: GTItem) => void;
  onWish?: (item: GTItem) => void;
  wished?: boolean;
  showCollected?: boolean;
}

function acquisitionSourceSummary(item: GTItem): string | null {
  if (item.acquisitionSources.length === 1)
    return item.acquisitionSources[0].text;
  if (item.acquisitionSources.length > 1)
    return `${item.acquisitionSources.length} acquisition sources`;
  return null;
}

export function ItemCard({
  item,
  density = "grid",
  personalize = "normal",
  onOpen,
  onWish,
  wished,
  showCollected,
}: ItemCardProps) {
  const r = item.rarity;
  const availBadge = item.availableNow && !item.collected;
  const showFor = personalize !== "off";
  const aggressive = personalize === "aggressive";
  const sourceSummary = acquisitionSourceSummary(item);

  if (density === "list") {
    return (
      <div
        className="gt-item gt-item--list"
        data-rarity={r}
        data-collected={showCollected && item.collected}
        onClick={() => onOpen?.(item)}
      >
        <ItemTile rarity={r} type={item.type} icon={item.icon} />
        <div className="gt-item-head">
          <div className="gt-il-main">
            <div className="gt-item-name">{item.name}</div>
            <div className="gt-item-type">
              {item.type}
              {sourceSummary ? ` · ${sourceSummary}` : ""}
            </div>
          </div>
          <div className="gt-item-badges">
            <Badge kind={r} dot />
            {item.farmOnly && <Badge kind="farmonly">Farm only</Badge>}
            {showFor && availBadge && <Badge kind="avail-now" dot />}
          </div>
        </div>
        <div className="gt-item-actions">
          <button
            className="gt-iconbtn"
            data-on={wished}
            title="Wishlist"
            aria-label={
              wished
                ? `Remove ${item.name} from wishlist`
                : `Add ${item.name} to wishlist`
            }
            onClick={(e) => {
              e.stopPropagation();
              onWish?.(item);
            }}
          >
            <Icon
              name="wishlist"
              size="1rem"
              fill={wished ? "currentColor" : "none"}
            />
          </button>
          <button
            className="gt-iconbtn"
            title="Details"
            aria-label={`${item.name} details`}
            onClick={(e) => {
              e.stopPropagation();
              onOpen?.(item);
            }}
          >
            <Icon name="info" size="1rem" />
          </button>
        </div>
      </div>
    );
  }

  if (density === "compact") {
    return (
      <div
        className="gt-item gt-item--compact"
        data-rarity={r}
        data-collected={showCollected && item.collected}
        onClick={() => onOpen?.(item)}
      >
        <ItemTile rarity={r} type={item.type} icon={item.icon} />
        <div className="gt-item-head" style={{ flex: 1 }}>
          <div className="gt-item-name">{item.name}</div>
          <div className="gt-item-type">
            {item.type}
            {sourceSummary ? ` · ${sourceSummary}` : ""}
          </div>
        </div>
        {showFor && availBadge && <Badge kind="avail-now" dot />}
        <button
          className="gt-iconbtn"
          data-on={wished}
          aria-label={
            wished
              ? `Remove ${item.name} from wishlist`
              : `Add ${item.name} to wishlist`
          }
          onClick={(e) => {
            e.stopPropagation();
            onWish?.(item);
          }}
        >
          <Icon
            name="wishlist"
            size="0.95rem"
            fill={wished ? "currentColor" : "none"}
          />
        </button>
      </div>
    );
  }

  // grid
  return (
    <div
      className="gt-item gt-item--grid"
      data-rarity={r}
      data-collected={showCollected && item.collected}
      style={
        aggressive && availBadge
          ? { boxShadow: "0 0 0 0.09rem var(--c-signal-line)" }
          : undefined
      }
      onClick={() => onOpen?.(item)}
    >
      <div className="gt-item-top">
        <ItemTile rarity={r} type={item.type} icon={item.icon} />
        <div className="gt-item-head">
          <div className="gt-item-name">{item.name}</div>
          <div className="gt-item-type">
            {item.type}
            {item.slot ? ` · ${item.slot}` : ""}
          </div>
          <div
            className="gt-item-badges"
            style={{ marginTop: "var(--s-1)" } as CSS}
          >
            <Badge kind={r} dot />
            {item.farmOnly && <Badge kind="farmonly">Farm only</Badge>}
          </div>
        </div>
      </div>
      {sourceSummary && (
        <div className="gt-item-src">
          <Icon
            name="bolt"
            size="0.8rem"
            style={{ color: "var(--c-text-4)" }}
          />
          {sourceSummary}
        </div>
      )}
      {showFor && (availBadge || item.collected) && (
        <div className="gt-item-badges">
          {availBadge && <Badge kind="avail-now" dot icon="bolt" />}
          {item.collected && (
            <Badge kind="owned" dot>
              Collected
            </Badge>
          )}
        </div>
      )}
      <div className="gt-item-foot">
        <button
          className="gt-iconbtn"
          data-on={wished}
          title="Add to wishlist"
          aria-label={
            wished
              ? `Remove ${item.name} from wishlist`
              : `Add ${item.name} to wishlist`
          }
          onClick={(e) => {
            e.stopPropagation();
            onWish?.(item);
          }}
        >
          <Icon
            name="wishlist"
            size="1rem"
            fill={wished ? "currentColor" : "none"}
          />
        </button>
        <Button
          sm
          variant="ghost"
          icon="info"
          onClick={(e) => {
            e.stopPropagation();
            onOpen?.(item);
          }}
        >
          Details
        </Button>
      </div>
    </div>
  );
}
