import { describe, it, expect, vi, afterEach } from "vitest";
import { toWishlistEntry, toCharacter, toGTItemView } from "./adapters";
import type { APICharacter, WishListItem } from "../types/api";

afterEach(() => {
  vi.useRealTimers();
});

const apiWish: WishListItem = {
  id: "7",
  itemHash: 99999,
  name: "Gjallarhorn",
  itemType: "Rocket Launcher",
  rarity: "Exotic",
  icon: "/icons/gj.png",
  priority: "URGENT",
  notes: "the classic",
  sources: ["Exotic quest"],
  availableNow: true,
  availableFrom: "Xûr",
  dateAdded: new Date().toISOString(),
};

describe("toWishlistEntry", () => {
  it("maps availability from the API (B6)", () => {
    const entry = toWishlistEntry(apiWish);
    expect(entry.avail.now).toBe(true);
    expect(entry.avail.where).toBe("Xûr");
    expect(entry.priority).toBe("urgent");
    expect(entry.icon).toBe("/icons/gj.png");
  });

  it("shows the source when not available now", () => {
    const entry = toWishlistEntry({
      ...apiWish,
      availableNow: false,
      availableFrom: undefined,
    });
    expect(entry.avail.now).toBe(false);
    expect(entry.avail.where).toBe("Exotic quest");
  });

  it("tolerates missing availability fields (older API)", () => {
    const legacy = { ...apiWish } as Partial<WishListItem>;
    delete legacy.availableNow;
    const entry = toWishlistEntry(legacy as WishListItem);
    expect(entry.avail.now).toBe(false);
  });
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
