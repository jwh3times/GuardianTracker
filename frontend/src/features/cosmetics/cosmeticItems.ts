import type { APIUserCollections } from "../../types/api";
import type { GTItem } from "../../types/design";
import { toGTItem } from "../../lib/adapters";
import { gatherItemHashes } from "../collections/collectionTree";
import { COSMETIC_TYPE_SET } from "./cosmeticBuckets";

/**
 * Build the cosmetic GTItem list from an include=all collections payload:
 * walk every tree root for leaf hashes, map details via toGTItem, keep only
 * cosmetic itemTypes, and tag `collected` from collectedHashes.
 */
export function cosmeticItems(data: APIUserCollections): GTItem[] {
  const itemsMap = data.items ?? {};
  const collected = new Set(data.collectedHashes ?? []);
  const seen = new Set<string>();
  const out: GTItem[] = [];
  for (const root of data.tree) {
    for (const hash of gatherItemHashes(root)) {
      if (seen.has(hash)) continue;
      seen.add(hash);
      const detail = itemsMap[hash];
      if (!detail) continue;
      const gt = toGTItem(detail);
      if (!COSMETIC_TYPE_SET.has(gt.type)) continue;
      gt.collected = collected.has(hash);
      out.push(gt);
    }
  }
  return out;
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
