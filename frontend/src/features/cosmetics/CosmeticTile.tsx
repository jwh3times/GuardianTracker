import type { GTItem } from "../../types/design";
import { Badge, ItemTile } from "../../components/primitives";

export function CosmeticTile({
  item,
  onOpen,
}: {
  item: GTItem;
  onOpen: (item: GTItem) => void;
}) {
  // Same rule as ItemCard on Collections: flag only what the player can act on
  // — obtainable right now and not already owned.
  const availBadge = item.obtainable && !item.collected;
  return (
    <button
      type="button"
      className="gt-cosmetic-tile"
      data-collected={item.collected}
      onClick={() => onOpen(item)}
      title={
        availBadge && item.availFrom
          ? `${item.name} — available now from ${item.availFrom}`
          : item.name
      }
    >
      <ItemTile rarity={item.rarity} type={item.type} icon={item.icon} />
      <span className="gt-cosmetic-tile-name">{item.name}</span>
      {availBadge && <Badge kind="avail-now" dot />}
    </button>
  );
}
