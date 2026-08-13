import type { CollectionsView } from "../../lib/collectionsView";
import type { GTItem } from "../../types/design";
import { COSMETIC_TYPE_SET } from "./cosmeticBuckets";

/**
 * The cosmetic slice of a Destiny membership's collections.
 *
 * This used to walk the tree and join ownership and vendor availability itself —
 * the third copy of that join, and the one that was incomplete: it set
 * `collected` but never `availableNow` or `availFrom`, so a cosmetic a vendor was
 * actively selling showed no "Available now" badge here while the identical item
 * did on Collections. The adapter owns the join now, and what is genuinely
 * cosmetics-specific — which item types count as cosmetics — stays here.
 */
export function cosmeticItems(view: CollectionsView): GTItem[] {
  return view.allItems().filter((i) => COSMETIC_TYPE_SET.has(i.type));
}

/** Group cosmetics by their type string (insertion order = first-seen order). */
export function groupByType(items: GTItem[]): Map<string, GTItem[]> {
  const groups = new Map<string, GTItem[]>();
  for (const it of items) {
    const arr = groups.get(it.type);
    if (arr) arr.push(it);
    else groups.set(it.type, [it]);
  }
  return groups;
}
