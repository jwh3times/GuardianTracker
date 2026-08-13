import { describe, it, expect } from "vitest";
import { toCollections, toCollectionsSummary } from "./collectionsView";
import type { APIDestinyItem, APIMembershipCollections } from "../types/api";

function item(over: Partial<APIDestinyItem> & { itemHash: string }) {
  return {
    name: "Item " + over.itemHash,
    description: "",
    icon: "",
    itemType: "Hand Cannon",
    tierType: 5,
    rarity: "Legendary",
    difficulty: "Challenging",
    sources: [],
    isExotic: false,
    ...over,
  } as APIDestinyItem;
}

// Weapons(10) ─ Hand Cannons(11): items 100 (collected), 200 (on sale)
//             └ Shotguns(12):     item 300
// Armor(20):                      item 400, and 100 again (shared)
const full = {
  tree: [
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
          name: "Shotguns",
          icon: "",
          collected: 0,
          total: 1,
          items: ["300"],
        },
      ],
    },
    {
      hash: "20",
      name: "Armor",
      icon: "",
      collected: 0,
      total: 2,
      items: ["400", "100"],
    },
  ],
  items: {
    "100": item({ itemHash: "100", name: "Fatebringer" }),
    "200": item({ itemHash: "200", name: "Imperial Decree" }),
    "300": item({ itemHash: "300", name: "Found Verdict" }),
    "400": item({ itemHash: "400", name: "Helm of Saint-14" }),
  },
  collectedHashes: ["100"],
  availableNow: { "200": "Banshee-44" },
  summary: {
    weapons: { total: 4, collected: 1 },
    armor: { total: 2, collected: 0 },
    exotics: { total: 0, collected: 0 },
    cosmetics: { total: 10, collected: 5 },
  },
  fetchedAt: "2026-08-09T00:00:00Z",
} as unknown as APIMembershipCollections;

/** The counts-only variant the backend serves without ?include=all. */
const lightweight = {
  tree: [
    { hash: "10", name: "Weapons", icon: "", collected: 1, total: 3 },
    { hash: "20", name: "Armor", icon: "", collected: 0, total: 2 },
  ],
  summary: full.summary,
  fetchedAt: full.fetchedAt,
} as unknown as APIMembershipCollections;

// The reason this module exists. Three modules used to join item detail,
// ownership and vendor availability independently, and the copies drifted.
describe("the join", () => {
  it("resolves ownership and live vendor onto every item", () => {
    const v = toCollections(full);

    const owned = v.itemByHash("100")!;
    expect(owned.collected).toBe(true);
    expect(owned.availableNow).toBe(false);
    expect(owned.availFrom).toBeUndefined();

    const onSale = v.itemByHash("200")!;
    expect(onSale.collected).toBe(false);
    expect(onSale.availableNow).toBe(true);
    expect(onSale.availFrom).toBe("Banshee-44");

    const neither = v.itemByHash("300")!;
    expect(neither.collected).toBe(false);
    expect(neither.availableNow).toBe(false);
  });

  it("applies the same join however the item is reached", () => {
    const v = toCollections(full);
    const direct = v.itemByHash("200")!;
    const underNode = v.itemsUnder("11").find((i) => i.id === "200")!;
    const swept = v.allItems().find((i) => i.id === "200")!;
    // Identity, not just equality: one construction, shared everywhere.
    expect(underNode).toBe(direct);
    expect(swept).toBe(direct);
  });

  it("returns null for an unknown item rather than a half-built one", () => {
    expect(toCollections(full).itemByHash("999")).toBeNull();
  });
});

describe("wire → domain mapping", () => {
  it("maps rarity, difficulty and sources", () => {
    const v = toCollections({
      ...full,
      items: {
        "100": item({
          itemHash: "100",
          rarity: "Exotic",
          difficulty: "Moderate",
          sources: ["Vault of Glass raid", "Master VoG"],
          icon: "/icons/fb.png",
        }),
      },
    } as unknown as APIMembershipCollections);
    const i = v.itemByHash("100")!;
    expect(i.rarity).toBe("exotic");
    expect(i.diff).toBe("moderate");
    expect(i.source).toBe("Vault of Glass raid");
    expect(i.sourceDetail).toBe("Master VoG");
    expect(i.icon).toBe("/icons/fb.png");
  });

  it("falls back on unknown rarity/difficulty and empty sources", () => {
    const v = toCollections({
      ...full,
      items: {
        "100": item({
          itemHash: "100",
          rarity: "Mythic???",
          difficulty: "Impossible",
          sources: [],
        }),
      },
    } as unknown as APIMembershipCollections);
    const i = v.itemByHash("100")!;
    expect(i.rarity).toBe("legendary");
    expect(i.diff).toBe("unrated");
    expect(i.source).toBe("Unknown source");
  });

  it("carries farmOnly and maps Unrated", () => {
    const v = toCollections({
      ...full,
      items: {
        "100": item({ itemHash: "100", difficulty: "Unrated", farmOnly: true }),
      },
    } as unknown as APIMembershipCollections);
    expect(v.itemByHash("100")!.diff).toBe("unrated");
    expect(v.itemByHash("100")!.farmOnly).toBe(true);
  });
});

describe("tree", () => {
  it("gathers items from a node and all its descendants, deduped", () => {
    const v = toCollections(full);
    expect(v.itemsUnder("11").map((i) => i.id)).toEqual(["100", "200"]);
    // The Weapons root rolls up both child nodes.
    expect(v.itemsUnder("10").map((i) => i.id)).toEqual(["100", "200", "300"]);
    // 100 appears under both roots but is listed once per node it belongs to.
    expect(v.itemsUnder("20").map((i) => i.id)).toEqual(["400", "100"]);
  });

  it("caches a node's items so repeat reads are the same list", () => {
    const v = toCollections(full);
    expect(v.itemsUnder("10")).toBe(v.itemsUnder("10"));
  });

  it("sweeps every reachable item once, in first-seen order", () => {
    expect(
      toCollections(full)
        .allItems()
        .map((i) => i.id),
    ).toEqual(["100", "200", "300", "400"]);
  });

  it("resolves the path to a node and to an item's owning node", () => {
    const v = toCollections(full);
    expect(v.pathToNode("12")).toEqual(["10", "12"]);
    expect(v.pathToNode("10")).toEqual(["10"]);
    expect(v.pathToNode("nope")).toBeNull();
    expect(v.pathToItem("300")).toEqual(["10", "12"]);
    expect(v.pathToItem("999")).toBeNull();
  });

  it("exposes node counts with missing derived", () => {
    const v = toCollections(full);
    expect(v.node("11")).toEqual({
      hash: "11",
      total: 2,
      collected: 1,
      missing: 1,
    });
    expect(v.node("nope")).toBeNull();
  });

  it("adapts roots to the design tree shape", () => {
    const v = toCollections(full);
    expect(v.rootHashes).toEqual(["10", "20"]);
    expect(v.roots[0].label).toBe("Weapons");
    expect(v.roots[0].pct).toBe(33); // 1/3
    expect(v.roots[0].children?.[0].id).toBe("11");
  });
});

describe("summary", () => {
  it("orders categories for the hero and derives percentages", () => {
    const v = toCollections(full);
    expect(v.summary.map((c) => c.id)).toEqual([
      "weapons",
      "armor",
      "exotics",
      "cosmetics",
    ]);
    expect(v.summary[0]).toEqual({
      id: "weapons",
      label: "Weapons",
      pct: 25,
      count: [1, 4],
    });
  });

  it("treats an empty category as zero rather than dividing by it", () => {
    const exotics = toCollections(full).summary[2];
    expect(exotics).toEqual({
      id: "exotics",
      label: "Exotics",
      pct: 0,
      count: [0, 0],
    });
  });

  it("averages the four percentages unweighted", () => {
    // 25 + 0 + 0 + 50 = 75, / 4 = 18.75 → 19. Deliberately not weighted by
    // category size; this preserves the Dashboard's existing behaviour.
    expect(toCollections(full).overallPct).toBe(19);
  });

  it("leaves fetchedAt raw so it is not frozen into a memo", () => {
    expect(toCollections(full).fetchedAt).toBe("2026-08-09T00:00:00Z");
  });
});

describe("the counts-only variant", () => {
  it("supports the whole summary surface without items", () => {
    const v = toCollectionsSummary(lightweight);
    expect(v.rootHashes).toEqual(["10", "20"]);
    expect(v.node("10")?.missing).toBe(2);
    expect(v.summary[0].pct).toBe(25);
    expect(v.overallPct).toBe(19);
    expect(v.pathToNode("20")).toEqual(["20"]);
  });

  it("tolerates a payload with no tree at all", () => {
    const empty = toCollectionsSummary({
      summary: full.summary,
      fetchedAt: full.fetchedAt,
    } as unknown as APIMembershipCollections);
    expect(empty.roots).toEqual([]);
    expect(empty.rootHashes).toEqual([]);
  });
});

describe("absent optional fields", () => {
  it("treats a missing availableNow as nothing on sale", () => {
    const v = toCollections({
      ...full,
      availableNow: undefined,
    } as unknown as APIMembershipCollections);
    expect(v.allItems().every((i) => !i.availableNow)).toBe(true);
    expect(v.allItems().every((i) => i.availFrom === undefined)).toBe(true);
  });

  it("treats a missing collectedHashes as nothing owned", () => {
    const v = toCollections({
      ...full,
      collectedHashes: undefined,
    } as unknown as APIMembershipCollections);
    expect(v.allItems().every((i) => !i.collected)).toBe(true);
  });

  it("skips tree hashes with no matching item detail", () => {
    const v = toCollections({
      ...full,
      items: { "100": full.items!["100"] },
    } as unknown as APIMembershipCollections);
    expect(v.itemsUnder("11").map((i) => i.id)).toEqual(["100"]);
    expect(v.itemByHash("200")).toBeNull();
  });
});
