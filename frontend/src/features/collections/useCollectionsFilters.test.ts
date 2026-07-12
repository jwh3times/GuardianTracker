import { describe, expect, it, beforeEach } from "vitest";
import { parseFilters, serializeFilters } from "./useCollectionsFilters";

describe("collections filter (de)serialization", () => {
  it("parses defaults from an empty param set", () => {
    const f = parseFilters(new URLSearchParams(""));
    expect(f).toEqual({
      node: "",
      rarity: null,
      diff: null,
      sort: "rarity",
      view: "grid",
      missing: true,
      avail: false,
      farm: false,
    });
  });

  it("serializes only non-default values", () => {
    const p = serializeFilters({
      node: "",
      rarity: null,
      diff: null,
      sort: "rarity",
      view: "grid",
      missing: true,
      avail: false,
      farm: false,
    });
    expect(p.toString()).toBe(""); // all defaults → empty query
  });

  it("round-trips a fully-populated state", () => {
    const state = {
      node: "42",
      rarity: "exotic" as const,
      diff: "challenging" as const,
      sort: "name" as const,
      view: "list" as const,
      missing: false,
      avail: true,
      farm: true,
    };
    const p = serializeFilters(state);
    expect(parseFilters(p)).toEqual(state);
  });

  it("emits missing=0 only when off, avail/farm=1 only when on", () => {
    expect(
      serializeFilters({ ...defaults(), missing: false }).get("missing"),
    ).toBe("0");
    expect(
      serializeFilters({ ...defaults(), missing: true }).has("missing"),
    ).toBe(false);
    expect(serializeFilters({ ...defaults(), avail: true }).get("avail")).toBe(
      "1",
    );
    expect(serializeFilters({ ...defaults(), avail: false }).has("avail")).toBe(
      false,
    );
  });
});

function defaults() {
  return {
    node: "",
    rarity: null,
    diff: null,
    sort: "rarity" as const,
    view: "grid" as const,
    missing: true,
    avail: false,
    farm: false,
  };
}
