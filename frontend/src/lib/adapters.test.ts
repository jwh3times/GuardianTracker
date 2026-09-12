import { describe, it, expect, vi, afterEach } from "vitest";
import { toCharacter, toGTItemView } from "./adapters";
import type { APICharacter } from "../types/api";

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

describe("toCharacter", () => {
  it("maps the API character", () => {
    const c: APICharacter = {
      characterId: "char-1",
      classType: 1,
      className: "Hunter",
      raceName: "Awoken",
      light: 2010,
      emblemPath: "/emblem.png",
      emblemBackgroundPath: "/emblem-bg.png",
      dateLastPlayed: new Date().toISOString(),
    };
    const out = toCharacter(c);
    expect(out.id).toBe("char-1");
    expect(out.cls).toBe("Hunter");
    expect(out.power).toBe(2010);
    expect(out.emblemUrl).toBe("/emblem.png");
  });
});
