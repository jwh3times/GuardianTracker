import { describe, it, expect } from "vitest";
import { cosmeticItems, groupByType } from "./cosmeticItems";
import type { APIUserCollections } from "../../types/api";

const data = {
  tree: [
    {
      hash: "root",
      name: "Cosmetics",
      icon: "",
      collected: 1,
      total: 2,
      children: [
        {
          hash: "emb",
          name: "Emblems",
          icon: "",
          collected: 1,
          total: 4,
          items: ["100", "101", "102", "103"],
        },
        {
          hash: "wpn",
          name: "Weapons",
          icon: "",
          collected: 0,
          total: 1,
          items: ["200"],
        },
      ],
    },
  ],
  items: {
    "100": {
      itemHash: "100",
      name: "Emblem A",
      description: "",
      icon: "/a.png",
      itemType: "Emblem",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "",
      sources: [],
      isExotic: false,
    },
    "101": {
      itemHash: "101",
      name: "Emblem B",
      description: "",
      icon: "/b.png",
      itemType: "Emblem",
      tierType: 6,
      rarity: "Exotic",
      difficulty: "",
      sources: [],
      isExotic: true,
    },
    "102": {
      itemHash: "102",
      name: "Ornament A",
      description: "",
      icon: "/o.png",
      itemType: "Ornament",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "",
      sources: [],
      isExotic: false,
    },
    "103": {
      itemHash: "103",
      name: "Finisher A",
      description: "",
      icon: "/f.png",
      itemType: "Finisher",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "",
      sources: [],
      isExotic: false,
    },
    "200": {
      itemHash: "200",
      name: "Gun",
      description: "",
      icon: "/g.png",
      itemType: "Hand Cannon",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "Moderate",
      sources: [],
      isExotic: false,
    },
  },
  collectedHashes: ["100"],
  summary: {
    weapons: { total: 1, collected: 0 },
    armor: { total: 0, collected: 0 },
    exotics: { total: 0, collected: 0 },
    cosmetics: { total: 4, collected: 1 },
  },
  fetchedAt: "2026-06-30T00:00:00Z",
} as unknown as APIUserCollections;

describe("cosmeticItems", () => {
  it("keeps only cosmetic-typed items and tags collected state", () => {
    const items = cosmeticItems(data);
    expect(items.map((i) => i.name).sort()).toEqual([
      "Emblem A",
      "Emblem B",
      "Finisher A",
      "Ornament A",
    ]);
    expect(items.find((i) => i.id === "100")?.collected).toBe(true);
    expect(items.find((i) => i.id === "101")?.collected).toBe(false);
  });

  it("groups items by their type string", () => {
    const groups = groupByType(cosmeticItems(data));
    expect(groups.get("Emblem")).toHaveLength(2);
    expect(groups.get("Ornament")).toHaveLength(1);
    expect(groups.get("Finisher")).toHaveLength(1);
    expect(groups.has("Hand Cannon")).toBe(false);
  });
});
