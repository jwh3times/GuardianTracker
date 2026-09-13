import React from "react";
import { describe, expect, it, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, server } from "../../test/testServer";
import { toCollections } from "../../lib/collectionsView";
import type { APIDifficulty, APIMembershipCollections } from "../../types/api";
import {
  effectiveFilters,
  intentWrite,
  nextParams,
  parseFilters,
  reconcile,
  revealPath,
  serializeFilters,
  useCollectionsBrowser,
  visibleItems,
  type CollectionsFilters,
} from "./useCollectionsBrowser";

beforeEach(() => {
  localStorage.clear();
});

// Weapons (10) holds Auto Rifles (12) directly and Hand Cannons (11), which
// holds Precision Frames (13). Items: 100 collected under 11, 200 missing
// under 11, 300 missing under 13.
const raw: APIMembershipCollections = {
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
          total: 3,
          items: ["100", "200"],
          children: [
            {
              hash: "13",
              name: "Precision Frames",
              icon: "",
              collected: 0,
              total: 1,
              items: ["300"],
            },
          ],
        },
      ],
    },
    { hash: "20", name: "Armor", icon: "", collected: 0, total: 0 },
  ],
  items: {
    "100": item("100", "Fatebringer", "Legendary", "Challenging"),
    "200": item("200", "Imperial Decree", "Legendary", "Moderate"),
    "300": item("300", "Ace of Spades", "Exotic", "Easy"),
  },
  collectedHashes: ["100"],
  availableNow: { "300": "Xûr" },
  summary: {
    weapons: { total: 3, collected: 1 },
    armor: { total: 0, collected: 0 },
    exotics: { total: 1, collected: 0 },
    cosmetics: { total: 0, collected: 0 },
  },
  fetchedAt: "2026-09-13T00:00:00Z",
};
const collections = toCollections(raw);

function item(
  itemHash: string,
  name: string,
  rarity: string,
  difficulty: APIDifficulty,
) {
  return {
    itemHash,
    name,
    description: "",
    icon: "",
    itemType: "Hand Cannon",
    tierType: rarity === "Exotic" ? 6 : 5,
    rarity,
    acquisitionSources: [{ text: "Somewhere", difficulty, raidDungeon: false }],
    isExotic: rarity === "Exotic",
  };
}

function params(search: string) {
  return new URLSearchParams(search);
}

function defaults(): CollectionsFilters {
  return parseFilters(params(""));
}

describe("collections filter (de)serialization", () => {
  it("parses defaults from an empty param set", () => {
    expect(defaults()).toEqual({
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
    expect(serializeFilters(defaults()).toString()).toBe("");
  });

  it("round-trips a fully-populated state", () => {
    const state: CollectionsFilters = {
      node: "42",
      q: "hand cannon",
      rarity: "exotic",
      diff: "challenging",
      sort: "name",
      view: "list",
      missing: false,
      avail: true,
      farm: true,
    };
    expect(parseFilters(serializeFilters(state))).toEqual(state);
  });

  it("omits an empty q from the URL", () => {
    expect(serializeFilters({ ...defaults(), q: "" }).has("q")).toBe(false);
  });

  it("truncates an over-long q to exactly 100 characters on read", () => {
    const f = parseFilters(params(`q=${"a".repeat(150)}`));
    expect(f.q).toBe("a".repeat(100));
  });

  // serializeFilters is a write path independent of parseFilters' read-path
  // clamp — a patch could otherwise write an over-long q straight into the URL.
  it("clamps an over-long q to 100 characters on write, not just on read", () => {
    const p = serializeFilters({ ...defaults(), q: "b".repeat(150) });
    expect(p.get("q")).toBe("b".repeat(100));
  });

  it("emits missing=0 only when off, avail/farm=1 only when on", () => {
    expect(
      serializeFilters({ ...defaults(), missing: false }).get("missing"),
    ).toBe("0");
    expect(serializeFilters(defaults()).has("missing")).toBe(false);
    expect(serializeFilters({ ...defaults(), avail: true }).get("avail")).toBe(
      "1",
    );
    expect(serializeFilters({ ...defaults(), farm: true }).get("farm")).toBe(
      "1",
    );
  });

  it("drops an invalid rarity or diff instead of round-tripping garbage", () => {
    expect(parseFilters(params("rarity=oops")).rarity).toBeNull();
    expect(parseFilters(params("diff=nope")).diff).toBeNull();
    expect(parseFilters(params("rarity=exotic")).rarity).toBe("exotic");
  });

  it("treats the removed legacy difficulty sort as the default", () => {
    const f = parseFilters(params("sort=difficulty"));
    expect(f.sort).toBe("rarity");
    expect(serializeFilters(f).has("sort")).toBe(false);
  });
});

describe("effectiveFilters", () => {
  it.each([
    {
      name: "stored defaults apply when the URL has no filter params",
      search: "node=11",
      stored: { sort: "name" as const },
      expected: { node: "11", sort: "name" },
    },
    {
      name: "a lone q is not a filter param, so stored defaults still apply",
      search: "q=foo",
      stored: { sort: "name" as const },
      expected: { q: "foo", sort: "name" },
    },
    {
      name: "any filter param makes the URL the whole source of truth",
      search: "avail=1",
      stored: { sort: "name" as const },
      expected: { avail: true, sort: "rarity" },
    },
    {
      name: "a stored payload can never leak node or q",
      search: "node=11",
      stored: { node: "999", q: "leaked" } as never,
      expected: { node: "11", q: "" },
    },
  ])("$name", ({ search, stored, expected }) => {
    expect(effectiveFilters(params(search), stored)).toMatchObject(expected);
  });
});

describe("intent writes", () => {
  it.each([
    {
      name: "selecting a node pushes and keeps the search term",
      prev: "node=11&q=foo",
      intent: { type: "selectNode", node: "10" } as const,
      search: "node=10&q=foo",
      replace: false,
    },
    {
      name: "a filter change replaces and keeps the category",
      prev: "node=11",
      intent: { type: "setFilter", patch: { rarity: "exotic" } } as const,
      search: "node=11&rarity=exotic",
      replace: true,
    },
    {
      name: "clearing filters resets them and the search, keeping sort and view",
      prev: "node=11&q=foo&rarity=exotic&diff=easy&avail=1&farm=1&sort=name&view=list",
      intent: { type: "clearFilters" } as const,
      search: "node=11&sort=name&view=list",
      replace: true,
    },
    {
      name: "params the browser does not own survive a write",
      prev: "node=11&tab=perks",
      intent: { type: "setFilter", patch: { avail: true } } as const,
      search: "node=11&avail=1&tab=perks",
      replace: true,
    },
  ])("$name", ({ prev, intent, search, replace }) => {
    const write = intentWrite(intent);
    expect(write.replace).toBe(replace);
    expect(nextParams(params(prev), null, write).toString()).toBe(search);
  });

  it("builds on stored defaults when the URL carries none", () => {
    const write = intentWrite({ type: "setFilter", patch: { avail: true } });
    expect(
      nextParams(params("node=11"), { sort: "name" }, write).toString(),
    ).toBe("node=11&sort=name&avail=1");
  });
});

describe("reconcile", () => {
  it("does nothing until collections data arrives", () => {
    expect(reconcile(params("item=100"), undefined)).toBeNull();
  });

  it.each([
    {
      name: "a collected deep link selects its owning node and reveals collected items",
      search: "item=100&avail=1",
      type: "openDeepLink",
      next: "node=11&missing=0&avail=1",
    },
    {
      name: "a missing deep link selects its owning node only",
      search: "node=10&item=300",
      type: "openDeepLink",
      next: "node=13",
    },
    {
      name: "a deep link outside the tree is consumed and looked up",
      search: "node=11&item=999",
      type: "lookUpDeepLink",
      next: "node=11",
    },
    {
      name: "a bare load seeds the first root",
      search: "",
      type: "seedRoot",
      next: "node=10",
    },
    {
      name: "a node the tree no longer has seeds the first root",
      search: "node=gone&q=foo",
      type: "seedRoot",
      next: "node=10&q=foo",
    },
  ])("$name", ({ search, type, next }) => {
    const step = reconcile(params(search), collections);
    expect(step?.type).toBe(type);
    expect(step?.write.replace).toBe(true);
    expect(nextParams(params(search), null, step!.write).toString()).toBe(next);
  });

  it("opens the deep-linked item itself", () => {
    const step = reconcile(params("item=200"), collections);
    expect(step).toMatchObject({
      type: "openDeepLink",
      item: { id: "200", name: "Imperial Decree" },
    });
  });

  it("a pending deep link wins over seeding, so the two never both write", () => {
    expect(reconcile(params("item=999"), collections)?.type).toBe(
      "lookUpDeepLink",
    );
  });

  it("leaves a URL that already agrees with the data alone", () => {
    expect(reconcile(params("node=13&rarity=exotic"), collections)).toBeNull();
  });
});

describe("visibleItems", () => {
  const names = (search: string) =>
    visibleItems(collections, parseFilters(params(search))).map((i) => i.name);

  it.each([
    { search: "", expected: [] },
    { search: "node=11", expected: ["Ace of Spades", "Imperial Decree"] },
    {
      search: "node=11&missing=0&sort=name",
      expected: ["Ace of Spades", "Fatebringer", "Imperial Decree"],
    },
    { search: "node=10&rarity=legendary", expected: ["Imperial Decree"] },
    { search: "node=10&diff=easy", expected: ["Ace of Spades"] },
    { search: "node=10&avail=1", expected: ["Ace of Spades"] },
    { search: "node=10&q=%20ACE%20", expected: ["Ace of Spades"] },
    {
      search: "node=10&q=%20%20",
      expected: ["Ace of Spades", "Imperial Decree"],
    },
    {
      search: "node=10&sort=avail&missing=0",
      expected: ["Ace of Spades", "Fatebringer", "Imperial Decree"],
    },
  ])("?$search shows $expected", ({ search, expected }) => {
    expect(names(search)).toEqual(expected);
  });

  it("hides farm-only items when asked", () => {
    const farmed = toCollections({
      ...raw,
      items: { ...raw.items, "300": { ...raw.items!["300"], farmOnly: true } },
    });
    expect(
      visibleItems(farmed, parseFilters(params("node=10&farm=1"))).map(
        (i) => i.id,
      ),
    ).toEqual(["200"]);
  });
});

describe("revealPath", () => {
  it.each([
    { node: "", expected: [] },
    { node: "10", expected: [] },
    { node: "11", expected: ["10"] },
    { node: "13", expected: ["10", "11"] },
    { node: "gone", expected: [] },
  ])("node '$node' opens $expected", ({ node, expected }) => {
    expect(revealPath(collections, node)).toEqual(expected);
  });
});

/** Pass `null` as `data` for "collections not loaded yet". */
function renderBrowser(
  route: string,
  data: typeof collections | null = collections,
  onItemUnavailable?: () => void,
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const seen = { search: "" };
  function Probe() {
    const { search } = useLocation();
    React.useLayoutEffect(() => {
      seen.search = search;
    });
    return null;
  }
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <Probe />
        {children}
      </MemoryRouter>
    </QueryClientProvider>
  );
  const hook = renderHook(
    () => useCollectionsBrowser(data ?? undefined, { onItemUnavailable }),
    { wrapper },
  );
  return { ...hook, search: () => seen.search };
}

describe("useCollectionsBrowser", () => {
  it("composes intents issued in the same tick instead of dropping all but the last", () => {
    const { result, search } = renderBrowser("/collections?node=11");

    act(() => {
      result.current.setFilter({ rarity: "exotic" });
      result.current.setFilter({ avail: true });
      result.current.selectNode("13");
    });

    expect(new URLSearchParams(search()).toString()).toBe(
      "node=13&rarity=exotic&avail=1",
    );
    expect(result.current.filters).toMatchObject({
      node: "13",
      rarity: "exotic",
      avail: true,
    });
  });

  it("opens a deep-linked collectible and consumes the param in one write", async () => {
    const { result, search } = renderBrowser("/collections?item=100");

    await waitFor(() => expect(result.current.detail?.id).toBe("100"));
    expect(new URLSearchParams(search()).toString()).toBe("node=11&missing=0");
    expect(result.current.activeNode?.hash).toBe("11");
    expect(result.current.expandPath).toEqual(["10"]);
  });

  it("looks up a deep link outside the tree and opens it view-only", async () => {
    server.use(
      http.get(`${API}/api/items/:hash`, () =>
        HttpResponse.json({
          itemHash: "999",
          name: "Vendor Mod",
          icon: "",
          itemType: "Mod",
          tierType: 5,
          rarity: "Legendary",
          description: "",
        }),
      ),
    );
    const { result, search } = renderBrowser("/collections?item=999");

    await waitFor(() => expect(result.current.detail?.name).toBe("Vendor Mod"));
    expect(search()).not.toContain("item=");
  });

  it("reports an unknown deep link once and opens nothing", async () => {
    server.use(
      http.get(`${API}/api/items/:hash`, () =>
        HttpResponse.json({ error: "not found" }, { status: 404 }),
      ),
    );
    let reports = 0;
    const { result } = renderBrowser(
      "/collections?item=999",
      collections,
      () => {
        reports += 1;
      },
    );

    await waitFor(() => expect(reports).toBe(1));
    expect(result.current.detail).toBeNull();
  });

  it("opens and closes the drawer on request", () => {
    const { result } = renderBrowser("/collections?node=11");
    const target = collections.itemByHash("200")!;

    act(() => result.current.openItem(target));
    expect(result.current.detail).toBe(target);

    act(() => result.current.closeDetail());
    expect(result.current.detail).toBeNull();
  });

  it("persists filter defaults but never node or q", () => {
    const { result } = renderBrowser("/collections?node=11");

    act(() => result.current.setFilter({ q: "hunter", view: "list" }));

    const stored = JSON.parse(
      localStorage.getItem("gt.collections.filters") ?? "{}",
    ) as Record<string, unknown>;
    expect(stored).toMatchObject({ view: "list" });
    expect(stored).not.toHaveProperty("q");
    expect(stored).not.toHaveProperty("node");
  });

  it("migrates the removed persisted difficulty sort to rarity", () => {
    localStorage.setItem(
      "gt.collections.filters",
      JSON.stringify({ sort: "difficulty", view: "list" }),
    );
    const { result } = renderBrowser("/collections?node=11");

    expect(result.current.filters).toMatchObject({
      sort: "rarity",
      view: "list",
    });
    expect(
      JSON.parse(localStorage.getItem("gt.collections.filters") ?? "{}"),
    ).toMatchObject({ sort: "rarity", view: "list" });
  });

  it.each([
    { search: "node=11&q=%20%20", hasFilters: false, searching: false },
    { search: "node=11&q=foo", hasFilters: true, searching: true },
    { search: "node=11&farm=1", hasFilters: true, searching: false },
    { search: "node=11&sort=name", hasFilters: false, searching: false },
  ])(
    "?$search → hasFilters $hasFilters, searching $searching",
    ({ search, hasFilters, searching }) => {
      const { result } = renderBrowser(`/collections?${search}`);
      expect(result.current.hasFilters).toBe(hasFilters);
      expect(result.current.searching).toBe(searching);
    },
  );

  it("writes nothing before collections data arrives", () => {
    const { search } = renderBrowser("/collections?item=100", null);
    expect(search()).toBe("?item=100");
  });
});
