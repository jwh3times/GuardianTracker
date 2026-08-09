import type {
  APICategoryCount,
  APICollectionNode,
  APIDestinyItem,
  APIUserCollections,
} from "../types/api";
import type {
  Difficulty,
  GTItem,
  Rarity,
  SummaryCategory,
  TreeNode,
} from "../types/design";

/**
 * The adapter for the collections payload — the largest thing the API returns,
 * and previously the only major one that crossed into feature modules raw.
 *
 * Five modules read it directly and three of them independently joined the same
 * three sources together: an item's detail from `items`, its ownership from
 * `collectedHashes`, and its live vendor from `availableNow`. The copies drifted
 * — the cosmetics one never applied the vendor half, so an item Xûr was actively
 * selling showed no "Available now" badge there while the identical item did on
 * Collections. The join now happens once, here, and every GTItem handed out is
 * already complete.
 *
 * Two views because the endpoint has two shapes. `?include=all` carries items;
 * the default carries only the tree counts and the summary. A single view whose
 * item accessors returned empty for the lighter payload would make "there are no
 * items" and "you fetched the wrong variant" indistinguishable — so the lighter
 * one simply has no item surface, and the compiler enforces it.
 */

const RARITY_MAP: Record<string, Rarity> = {
  Exotic: "exotic",
  Legendary: "legendary",
  Rare: "rare",
  Uncommon: "uncommon",
  Common: "common",
};

const DIFF_MAP: Record<string, Difficulty> = {
  Easy: "easy",
  Moderate: "moderate",
  Challenging: "challenging",
  Unrated: "unrated",
};

/** A node's counts, flattened for the callers that only need the numbers. */
export interface CollectionNodeView {
  hash: string;
  total: number;
  collected: number;
  missing: number;
}

/** What the default (counts-only) collections payload supports. */
export interface CollectionsSummaryView {
  /** The sidebar tree, already in design-system shape. */
  roots: TreeNode[];
  /** Root node hashes in payload order; `roots[0]` is the default selection. */
  rootHashes: string[];
  /** Ordered weapons, armor, exotics, cosmetics — the hero's render order. */
  summary: SummaryCategory[];
  /** Mean of the four category percentages (not weighted by size). */
  overallPct: number;
  /** RFC3339, left raw: formatting it here would bake a timestamp into a memo. */
  fetchedAt: string;
  node(hash: string): CollectionNodeView | null;
  /** Node-hash path root→node, for revealing a node in the sidebar. */
  pathToNode(hash: string): string[] | null;
}

/** Everything above, plus the item surface that `?include=all` makes possible. */
export interface CollectionsView extends CollectionsSummaryView {
  /** Fully joined: ownership and live vendor already applied. */
  itemByHash(hash: string): GTItem | null;
  /** Every item at or below a node, deduped. Computed on first ask, then cached. */
  itemsUnder(nodeHash: string): GTItem[];
  /** Every item reachable from any root, deduped, in first-seen order. */
  allItems(): GTItem[];
  /** Node-hash path root→the node whose own items include this hash. */
  pathToItem(itemHash: string): string[] | null;
}

/**
 * Wire item → domain item. Private on purpose: it cannot know about ownership or
 * vendor availability, so anything it returns is half-built. Letting callers
 * reach it directly is what allowed the three joins to drift apart.
 */
function toItem(d: APIDestinyItem): GTItem {
  const sources = d.sources ?? [];
  return {
    id: d.itemHash,
    name: d.name,
    type: d.itemType,
    slot: "",
    rarity: RARITY_MAP[d.rarity] ?? "legendary",
    diff: DIFF_MAP[d.difficulty] ?? "unrated",
    farmOnly: d.farmOnly ?? false,
    source: sources[0] ?? "Unknown source",
    sourceDetail: sources.slice(1).join(" · ") || sources[0] || "",
    obtainable: false,
    collected: false,
    desc: d.description ?? "",
    icon: d.icon,
  };
}

function toTreeNode(n: APICollectionNode): TreeNode {
  return {
    id: n.hash,
    label: n.name,
    pct: n.total ? Math.round((n.collected / n.total) * 100) : 0,
    count: [n.collected, n.total],
    children: n.children?.map(toTreeNode),
  };
}

const CATEGORIES = [
  { id: "weapons", label: "Weapons" },
  { id: "armor", label: "Armor" },
  { id: "exotics", label: "Exotics" },
  { id: "cosmetics", label: "Cosmetics" },
] as const;

function toSummary(raw: APIUserCollections): SummaryCategory[] {
  return CATEGORIES.map(({ id, label }) => {
    const c: APICategoryCount | undefined = raw.summary?.[id];
    // total === 0 is both "nothing to show" and the divide-by-zero guard.
    if (!c || c.total <= 0) {
      return { id, label, pct: 0, count: [0, 0] as [number, number] };
    }
    return {
      id,
      label,
      pct: Math.round((c.collected / c.total) * 100),
      count: [c.collected, c.total] as [number, number],
    };
  });
}

interface Walked {
  nodes: Map<string, APICollectionNode>;
  nodeViews: Map<string, CollectionNodeView>;
  pathByNode: Map<string, string[]>;
  pathByItem: Map<string, string[]>;
  /** Tree-reachable item hashes, deduped, in first-seen order. */
  reachable: string[];
}

/**
 * One depth-first pass produces everything path- and node-shaped. Doing it once
 * is the point: Collections used to run three separate walks (a node map, a
 * node-path search, an item-path search) over the same tree on every render.
 */
function walk(tree: APICollectionNode[]): Walked {
  const out: Walked = {
    nodes: new Map(),
    nodeViews: new Map(),
    pathByNode: new Map(),
    pathByItem: new Map(),
    reachable: [],
  };
  const seenItem = new Set<string>();

  const visit = (n: APICollectionNode, trail: string[]) => {
    const here = [...trail, n.hash];
    out.nodes.set(n.hash, n);
    out.nodeViews.set(n.hash, {
      hash: n.hash,
      total: n.total,
      collected: n.collected,
      missing: Math.max(n.total - n.collected, 0),
    });
    if (!out.pathByNode.has(n.hash)) out.pathByNode.set(n.hash, here);
    for (const h of n.items ?? []) {
      // First match wins, matching the original depth-first search.
      if (!out.pathByItem.has(h)) out.pathByItem.set(h, here);
      if (!seenItem.has(h)) {
        seenItem.add(h);
        out.reachable.push(h);
      }
    }
    for (const c of n.children ?? []) visit(c, here);
  };

  for (const root of tree) visit(root, []);
  return out;
}

function summaryView(
  raw: APIUserCollections,
  walked: Walked,
): CollectionsSummaryView {
  const summary = toSummary(raw);
  const tree = raw.tree ?? [];
  return {
    roots: tree.map(toTreeNode),
    rootHashes: tree.map((n) => n.hash),
    summary,
    overallPct: Math.round(
      summary.reduce((sum, c) => sum + c.pct, 0) / summary.length,
    ),
    fetchedAt: raw.fetchedAt,
    node: (hash) => walked.nodeViews.get(hash) ?? null,
    pathToNode: (hash) => walked.pathByNode.get(hash) ?? null,
  };
}

/** Adapt the default (counts-only) collections payload. */
export function toCollectionsSummary(
  raw: APIUserCollections,
): CollectionsSummaryView {
  return summaryView(raw, walk(raw.tree ?? []));
}

/** Adapt the `?include=all` collections payload, items and all. */
export function toCollections(raw: APIUserCollections): CollectionsView {
  const walked = walk(raw.tree ?? []);
  const base = summaryView(raw, walked);

  const details = raw.items ?? {};
  const collected = new Set(raw.collectedHashes ?? []);
  const availableNow = raw.availableNow ?? {};

  // The join, in the one place it happens. Every GTItem below leaves here
  // complete — callers never patch `collected`, `obtainable` or `availFrom`.
  const items = new Map<string, GTItem>();
  for (const [hash, detail] of Object.entries(details)) {
    const vendor = availableNow[hash];
    items.set(hash, {
      ...toItem(detail),
      collected: collected.has(hash),
      obtainable: !!vendor,
      availFrom: vendor,
    });
  }

  // itemsUnder is a tree walk per node. Collections asks for one node at a time,
  // Cosmetics sweeps everything — computing on first ask and caching serves both
  // without either paying for the other's access pattern.
  const underCache = new Map<string, GTItem[]>();

  function itemsUnder(nodeHash: string): GTItem[] {
    const cached = underCache.get(nodeHash);
    if (cached) return cached;
    const node = walked.nodes.get(nodeHash);
    if (!node) return [];
    const seen = new Set<string>();
    const out: GTItem[] = [];
    const collect = (n: APICollectionNode) => {
      for (const h of n.items ?? []) {
        if (seen.has(h)) continue;
        seen.add(h);
        const item = items.get(h);
        if (item) out.push(item);
      }
      for (const c of n.children ?? []) collect(c);
    };
    collect(node);
    underCache.set(nodeHash, out);
    return out;
  }

  let all: GTItem[] | null = null;

  return {
    ...base,
    itemByHash: (hash) => items.get(hash) ?? null,
    itemsUnder,
    allItems: () => {
      all ??= walked.reachable
        .map((h) => items.get(h))
        .filter((i): i is GTItem => !!i);
      return all;
    },
    pathToItem: (itemHash) => walked.pathByItem.get(itemHash) ?? null,
  };
}
