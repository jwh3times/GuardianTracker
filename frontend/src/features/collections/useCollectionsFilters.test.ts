import React from "react";
import { describe, expect, it } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import {
  parseFilters,
  serializeFilters,
  useCollectionsFilters,
} from "./useCollectionsFilters";

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

  it("drops an invalid rarity instead of round-tripping garbage", () => {
    const f = parseFilters(new URLSearchParams("rarity=oops"));
    expect(f.rarity).toBeNull();
  });

  it("drops an invalid diff instead of round-tripping garbage", () => {
    const f = parseFilters(new URLSearchParams("diff=nope"));
    expect(f.diff).toBeNull();
  });

  it("still parses a valid rarity", () => {
    const f = parseFilters(new URLSearchParams("rarity=exotic"));
    expect(f.rarity).toBe("exotic");
  });
});

describe("useCollectionsFilters — atomic setFilters", () => {
  it("applies a multi-field patch without losing either field (lost-update guard)", () => {
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(MemoryRouter, null, children);

    const { result } = renderHook(() => useCollectionsFilters(), { wrapper });

    act(() => {
      result.current.setFilters({ node: "5", missing: false });
    });

    expect(result.current.node).toBe("5");
    expect(result.current.missing).toBe(false);
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
