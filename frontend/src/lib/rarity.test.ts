import { describe, it, expect } from "vitest";
import { toRarity } from "./rarity";

describe("toRarity", () => {
  it("maps every manifest tier name", () => {
    expect(
      ["Exotic", "Legendary", "Rare", "Uncommon", "Common"].map(toRarity),
    ).toEqual(["exotic", "legendary", "rare", "uncommon", "common"]);
  });

  it("falls back to legendary for unknown, empty or missing input", () => {
    expect(toRarity("Mythic")).toBe("legendary");
    expect(toRarity("")).toBe("legendary");
    expect(toRarity(undefined)).toBe("legendary");
    expect(toRarity(null)).toBe("legendary");
  });

  it("does not resolve inherited object keys", () => {
    expect(toRarity("toString")).toBe("legendary");
    expect(toRarity("constructor")).toBe("legendary");
  });
});
