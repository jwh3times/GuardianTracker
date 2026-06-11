import React from "react";
import { Icon } from "./Icon";
import { Badge, Button, ItemTile } from "./primitives";
import { DIFF_LABEL } from "../../lib/constants";
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
  const availBadge = item.obtainable && !item.collected;
  const showFor = personalize !== "off";
  const aggressive = personalize === "aggressive";

  if (density === "list") {
    return (
      <div
        className="gt-item gt-item--list"
        data-rarity={r}
        data-collected={showCollected && item.collected}
        onClick={() => onOpen?.(item)}
      >
        <ItemTile rarity={r} type={item.type} />
        <div className="gt-item-head">
          <div className="gt-il-main">
            <div className="gt-item-name">{item.name}</div>
            <div className="gt-item-type">
              {item.type} · {item.source}
            </div>
          </div>
          <div className="gt-item-badges">
            <Badge kind={r} dot />
            <Badge kind={item.diff}>{DIFF_LABEL[item.diff]}</Badge>
            {showFor && availBadge && <Badge kind="avail-now" dot />}
          </div>
        </div>
        <div className="gt-item-actions">
          <button
            className="gt-iconbtn"
            data-on={wished}
            title="Wishlist"
            onClick={(e) => {
              e.stopPropagation();
              onWish?.(item);
            }}
          >
            <Icon name="wishlist" size="1rem" fill={wished ? "currentColor" : "none"} />
          </button>
          <button
            className="gt-iconbtn"
            title="Details"
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
        <ItemTile rarity={r} type={item.type} />
        <div className="gt-item-head" style={{ flex: 1 }}>
          <div className="gt-item-name">{item.name}</div>
          <div className="gt-item-type">{item.type}</div>
        </div>
        {showFor && availBadge && <Badge kind="avail-now" dot />}
        <button
          className="gt-iconbtn"
          data-on={wished}
          onClick={(e) => {
            e.stopPropagation();
            onWish?.(item);
          }}
        >
          <Icon name="wishlist" size="0.95rem" fill={wished ? "currentColor" : "none"} />
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
        <ItemTile rarity={r} type={item.type} />
        <div className="gt-item-head">
          <div className="gt-item-name">{item.name}</div>
          <div className="gt-item-type">
            {item.type}
            {item.slot ? ` · ${item.slot}` : ""}
          </div>
          <div className="gt-item-badges" style={{ marginTop: "var(--s-1)" } as CSS}>
            <Badge kind={r} dot />
            <Badge kind={item.diff}>{DIFF_LABEL[item.diff]}</Badge>
          </div>
        </div>
      </div>
      <div className="gt-item-src">
        <Icon name="bolt" size="0.8rem" style={{ color: "var(--c-text-4)" }} />
        {item.source}
      </div>
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
          onClick={(e) => {
            e.stopPropagation();
            onWish?.(item);
          }}
        >
          <Icon name="wishlist" size="1rem" fill={wished ? "currentColor" : "none"} />
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
