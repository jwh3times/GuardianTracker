import React from "react";
import { describe, expect, it } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { MemoryRouter } from "react-router";
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
      q: "",
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
      q: "",
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
      q: "hand cannon",
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

  it("round-trips q through parse/serialize; empty q is omitted from the URL", () => {
    const withQ = serializeFilters({ ...defaults(), q: "warlock" });
    expect(withQ.get("q")).toBe("warlock");
    expect(parseFilters(withQ).q).toBe("warlock");

    const withoutQ = serializeFilters({ ...defaults(), q: "" });
    expect(withoutQ.has("q")).toBe(false);
  });

  it("truncates an over-long q to exactly 100 characters", () => {
    const long = "a".repeat(150);
    const f = parseFilters(new URLSearchParams(`q=${long}`));
    expect(f.q).toHaveLength(100);
    expect(f.q).toBe("a".repeat(100));
  });

  // Finding D regression: serializeFilters is a write path independent of
  // parseFilters' read-path clamp — a future caller (e.g. setFilters({ q }))
  // could otherwise write an over-long q straight into the URL.
  it("clamps an over-long q to 100 characters on write, not just on read", () => {
    const long = "b".repeat(150);
    const p = serializeFilters({ ...defaults(), q: long });
    expect(p.get("q")).toHaveLength(100);
    expect(p.get("q")).toBe("b".repeat(100));
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

  it("treats the removed legacy difficulty sort as the default", () => {
    const f = parseFilters(new URLSearchParams("sort=difficulty"));
    expect(f.sort).toBe("rarity");
    expect(serializeFilters(f).has("sort")).toBe(false);
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

  it("keeps node from the URL even if a legacy localStorage payload contains node", () => {
    localStorage.clear();
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(
        MemoryRouter,
        { initialEntries: ["/?node=11"] },
        children,
      );
    const { result } = renderHook(() => useCollectionsFilters(), { wrapper });
    // Simulate a hand-edited / legacy stored payload that (wrongly) includes node.
    // The persist effect normally strips node, so seed it after mount.
    localStorage.setItem(
      "gt.collections.filters",
      JSON.stringify({ node: "999", rarity: "exotic" }),
    );
    act(() => {
      result.current.setRarity("legendary");
    });
    // write's base re-asserts node from the URL, so the stored "999" can't leak in.
    expect(result.current.node).toBe("11");
    localStorage.clear();
  });

  it("a lone ?q=foo still applies the user's stored filter defaults", () => {
    localStorage.clear();
    localStorage.setItem(
      "gt.collections.filters",
      JSON.stringify({ sort: "name" }),
    );
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(
        MemoryRouter,
        { initialEntries: ["/?q=foo"] },
        children,
      );
    const { result } = renderHook(() => useCollectionsFilters(), { wrapper });
    // A lone `q` param must not be treated as "the URL carries filter params" —
    // the stored non-default sort must still apply.
    expect(result.current.q).toBe("foo");
    expect(result.current.sort).toBe("name");
    localStorage.clear();
  });

  it("migrates the removed persisted difficulty sort to rarity", () => {
    localStorage.clear();
    localStorage.setItem(
      "gt.collections.filters",
      JSON.stringify({ sort: "difficulty", view: "list" }),
    );
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(MemoryRouter, null, children);

    const { result } = renderHook(() => useCollectionsFilters(), { wrapper });

    expect(result.current.sort).toBe("rarity");
    expect(result.current.view).toBe("list");
    expect(
      JSON.parse(localStorage.getItem("gt.collections.filters") ?? "{}"),
    ).toMatchObject({ sort: "rarity", view: "list" });
    localStorage.clear();
  });

  it("never persists q to localStorage", () => {
    localStorage.clear();
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(MemoryRouter, null, children);
    const { result } = renderHook(() => useCollectionsFilters(), { wrapper });

    act(() => {
      result.current.setQ("hunter");
    });

    const stored = JSON.parse(
      localStorage.getItem("gt.collections.filters") ?? "{}",
    );
    expect(stored).not.toHaveProperty("q");
    localStorage.clear();
  });

  it("does not leak a stray q from localStorage into state", () => {
    localStorage.clear();
    localStorage.setItem(
      "gt.collections.filters",
      JSON.stringify({ q: "leaked", sort: "name" }),
    );
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(MemoryRouter, null, children);
    const { result } = renderHook(() => useCollectionsFilters(), { wrapper });
    // q is URL-only: a hand-edited/legacy stored payload can't leak it in,
    // even though the other stored field (sort) still applies.
    expect(result.current.q).toBe("");
    expect(result.current.sort).toBe("name");
    localStorage.clear();
  });

  it("preserves q when the selected category changes", () => {
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(
        MemoryRouter,
        { initialEntries: ["/?q=foo"] },
        children,
      );
    const { result } = renderHook(() => useCollectionsFilters(), { wrapper });

    act(() => {
      result.current.setNode("7");
    });

    expect(result.current.node).toBe("7");
    expect(result.current.q).toBe("foo");
  });

  // Finding B regression: a whitespace-only q must not count as "the user
  // has filters set" — otherwise a fully-collected category with a stray
  // space in the search box shows a misleading "no items match" empty state
  // instead of "All caught up!" (see Collections.tsx's mirrored guard).
  it("hasFilters is false for a whitespace-only q", () => {
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(
        MemoryRouter,
        { initialEntries: ["/?q=%20%20"] },
        children,
      );
    const { result } = renderHook(() => useCollectionsFilters(), { wrapper });

    expect(result.current.q).toBe("  ");
    expect(result.current.hasFilters).toBe(false);
  });

  it("clearFilters clears q; hasFilters is true for a search-only state", () => {
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(
        MemoryRouter,
        { initialEntries: ["/?q=foo"] },
        children,
      );
    const { result } = renderHook(() => useCollectionsFilters(), { wrapper });

    expect(result.current.hasFilters).toBe(true);

    act(() => {
      result.current.clearFilters();
    });

    expect(result.current.q).toBe("");
  });
});

function defaults() {
  return {
    node: "",
    q: "",
    rarity: null,
    diff: null,
    sort: "rarity" as const,
    view: "grid" as const,
    missing: true,
    avail: false,
    farm: false,
  };
}
