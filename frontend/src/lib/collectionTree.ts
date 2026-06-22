import type { APICollectionNode } from "../types/api";
import type { TreeNode } from "../types/design";

/** Adapt an API collection node (and its descendants) to the design TreeNode shape. */
export function apiNodeToTreeNode(n: APICollectionNode): TreeNode {
  return {
    id: n.hash,
    label: n.name,
    pct: n.total ? Math.round((n.collected / n.total) * 100) : 0,
    count: [n.collected, n.total],
    children: n.children?.map(apiNodeToTreeNode),
  };
}

/** All leaf item hashes under a node (recursive, deduped). */
export function gatherItemHashes(n: APICollectionNode): string[] {
  const out = new Set<string>();
  const walk = (node: APICollectionNode) => {
    for (const h of node.items ?? []) out.add(h);
    for (const c of node.children ?? []) walk(c);
  };
  walk(n);
  return [...out];
}

/** Node-hash path (root→node) to the node whose own `items` include itemHash. */
export function findNodePath(
  tree: APICollectionNode[],
  itemHash: string,
): string[] | null {
  const dfs = (node: APICollectionNode, trail: string[]): string[] | null => {
    const here = [...trail, node.hash];
    if ((node.items ?? []).includes(itemHash)) return here;
    for (const c of node.children ?? []) {
      const found = dfs(c, here);
      if (found) return found;
    }
    return null;
  };
  for (const root of tree) {
    const found = dfs(root, []);
    if (found) return found;
  }
  return null;
}
