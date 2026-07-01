import type { GTItem } from "../../types/design";
import { ItemTile } from "../../components/primitives";

export function CosmeticTile({
  item,
  onOpen,
}: {
  item: GTItem;
  onOpen: (item: GTItem) => void;
}) {
  return (
    <button
      type="button"
      className="gt-cosmetic-tile"
      data-collected={item.collected}
      onClick={() => onOpen(item)}
      title={item.name}
    >
      <ItemTile rarity={item.rarity} type={item.type} icon={item.icon} />
      <span className="gt-cosmetic-tile-name">{item.name}</span>
    </button>
  );
}
