import { describe, it, expect } from "vitest";
import {
  apiNodeToTreeNode,
  gatherItemHashes,
  findNodePath,
} from "./collectionTree";
import type { APICollectionNode } from "../../../types/api";

const tree: APICollectionNode[] = [
  {
    hash: "10",
    name: "Weapons",
    icon: "",
    collected: 1,
    total: 3,
    children: [
      {
        hash: "11",
        name: "Hand Cannons",
        icon: "",
        collected: 1,
        total: 2,
        items: ["100", "200"],
      },
      {
        hash: "12",
        name: "Bows",
        icon: "",
        collected: 0,
        total: 2,
        items: ["300", "100"], // "100" duplicates Hand Cannons — exercises Set dedup
      },
    ],
  },
];

describe("collectionTree helpers", () => {
  it("apiNodeToTreeNode maps counts + pct and recurses", () => {
    const tn = apiNodeToTreeNode(tree[0]);
    expect(tn.id).toBe("10");
    expect(tn.label).toBe("Weapons");
    expect(tn.count).toEqual([1, 3]);
    expect(tn.pct).toBe(33); // round(1/3*100)
    expect(tn.children?.[0].id).toBe("11");
  });

  it("gatherItemHashes collects all descendants deduped", () => {
    // "100" appears under both Hand Cannons and Bows — dedup must yield it once.
    const hashes = gatherItemHashes(tree[0]);
    expect(hashes.sort()).toEqual(["100", "200", "300"]);
    expect(hashes.filter((h) => h === "100").length).toBe(1);
  });

  it("findNodePath returns the path to the node holding an item", () => {
    expect(findNodePath(tree, "300")).toEqual(["10", "12"]);
    expect(findNodePath(tree, "999")).toBeNull();
  });
});
