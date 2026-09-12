import { it, expect, vi, afterEach } from "vitest";
import { toGTItemView } from "./adapters";

afterEach(() => {
  vi.useRealTimers();
});

it("maps an APIItemView to a view-only GTItem", () => {
  const g = toGTItemView({
    itemHash: "55",
    name: "Mod",
    icon: "/i.png",
    itemType: "Mod",
    tierType: 5,
    rarity: "Legendary",
    description: "desc",
  });
  expect(g.id).toBe("55");
  expect(g.viewOnly).toBe(true);
  expect(g.collected).toBe(false);
  expect(g.rarity).toBe("legendary");
  expect(g.acquisitionSources).toEqual([]);
});
