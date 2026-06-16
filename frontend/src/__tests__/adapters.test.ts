import { describe, it, expect, vi, afterEach } from "vitest";
import {
  relTime,
  toGTItem,
  toWishlistEntry,
  toCharacter,
} from "../lib/adapters";
import type { APICharacter, APIDestinyItem, WishListItem } from "../types/api";

afterEach(() => {
  vi.useRealTimers();
});

describe("relTime", () => {
  const at = (msAgo: number) => new Date(Date.now() - msAgo).toISOString();

  it("returns 'just now' under a minute", () => {
    expect(relTime(at(30_000))).toBe("just now");
  });
  it("returns minutes under an hour", () => {
    expect(relTime(at(5 * 60_000))).toBe("5m ago");
    expect(relTime(at(59 * 60_000))).toBe("59m ago");
  });
  it("returns hours under a day", () => {
    expect(relTime(at(3 * 3_600_000))).toBe("3h ago");
  });
  it("returns days under a month", () => {
    expect(relTime(at(4 * 86_400_000))).toBe("4d ago");
  });
  it("returns months beyond 30 days", () => {
    expect(relTime(at(65 * 86_400_000))).toBe("2mo ago");
  });
});

const apiItem: APIDestinyItem = {
  itemHash: "12345",
  name: "Fatebringer",
  description: "Legendary hand cannon",
  icon: "/icons/fb.png",
  itemType: "Hand Cannon",
  tierType: 5,
  rarity: "Legendary",
  difficulty: "Challenging",
  sources: ["Vault of Glass raid", "Master VoG"],
  isExotic: false,
};

describe("toGTItem", () => {
  it("maps fields and carries the icon", () => {
    const item = toGTItem(apiItem);
    expect(item.id).toBe("12345");
    expect(item.rarity).toBe("legendary");
    expect(item.diff).toBe("challenging");
    expect(item.source).toBe("Vault of Glass raid");
    expect(item.sourceDetail).toBe("Master VoG");
    expect(item.icon).toBe("/icons/fb.png");
  });

  it("falls back on unknown rarity/difficulty and empty sources", () => {
    const item = toGTItem({
      ...apiItem,
      rarity: "Mythic???",
      difficulty: "Impossible",
      sources: [],
    });
    expect(item.rarity).toBe("legendary");
    expect(item.diff).toBe("moderate");
    expect(item.source).toBe("Unknown source");
  });
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
