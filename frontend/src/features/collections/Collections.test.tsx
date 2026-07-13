import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, useLocation, useNavigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { CharacterProvider } from "../../contexts/CharacterContext";
import { PreferencesProvider } from "../../contexts/PreferencesContext";
import { ToastProvider } from "../../components/Toast";
import { Collections } from "./Collections";

beforeEach(() => {
  localStorage.clear();
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
});

function renderPage(ui: React.ReactNode, route = "/") {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <AuthProvider>
        <PreferencesProvider>
          <CharacterProvider>
            <ToastProvider>
              <MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter>
            </ToastProvider>
          </CharacterProvider>
        </PreferencesProvider>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

function renderCollections(search = "") {
  return renderPage(<Collections />, `/collections${search}`);
}

// Renders the live router search string so tests can assert on it directly
// (e.g. that a deep-link's item= param was stripped in the same write that
// preserved node/avail).
function LocationProbe() {
  const loc = useLocation();
  return <div data-testid="search">{loc.search}</div>;
}

// Renders the live router pathname — used alongside a BackButton to prove a
// URL write was a history *replace* (Back skips over it) rather than a push
// (Back would land on it).
function PathProbe() {
  const loc = useLocation();
  return <div data-testid="path">{loc.pathname}</div>;
}

function BackButton() {
  const nav = useNavigate();
  return (
    <button onClick={() => nav(-1)} type="button">
      go back
    </button>
  );
}

// Tree-shaped ?include=all payload: a "Weapons" root with a single
// "Hand Cannons" leaf node holding two items, one collected (100) and one
// missing (200). The shared sampleCollections fixture is the pre-tree flat
// shape (still consumed by the Dashboard test, migrated in Task 11), so the
// Collections tests override the handler with the new shape locally.
const treeCollections = {
  tree: [
    {
      hash: "10",
      name: "Weapons",
      icon: "",
      collected: 1,
      total: 2,
      children: [
        {
          hash: "11",
          name: "Hand Cannons",
          icon: "",
          collected: 1,
          total: 2,
          items: ["100", "200"],
        },
      ],
    },
  ],
  items: {
    "100": {
      itemHash: "100",
      name: "Fatebringer",
      description: "",
      icon: "/i/fb.png",
      itemType: "Hand Cannon",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "Challenging",
      sources: ["Vault of Glass"],
      isExotic: false,
    },
    "200": {
      itemHash: "200",
      name: "Imperial Decree",
      description: "",
      icon: "/i/id.png",
      itemType: "Shotgun",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "Moderate",
      sources: ["Menagerie"],
      isExotic: false,
    },
  },
  collectedHashes: ["100"],
  summary: {
    weapons: { total: 2, collected: 1 },
    armor: { total: 0, collected: 0 },
    exotics: { total: 0, collected: 0 },
    cosmetics: { total: 0, collected: 0 },
  },
  fetchedAt: new Date().toISOString(),
};

const treeCollectionsHandler = http.get(
  `${API}/api/collections/:type/:id`,
  () => HttpResponse.json(treeCollections),
);

describe("Collections", () => {
  it("navigates the tree and filters the grid to the selected node", async () => {
    server.use(treeCollectionsHandler);
    renderPage(<Collections />);

    // Default selection is the first root ("Weapons"); expand it and pick the
    // "Hand Cannons" leaf. Under the default missing-only filter the missing
    // item (Imperial Decree) shows and the collected one (Fatebringer) hides.
    await screen.findByText("Weapons");
    fireEvent.click(screen.getByRole("button", { name: /expand weapons/i }));
    fireEvent.click(screen.getByText("Hand Cannons"));

    expect(await screen.findByText("Imperial Decree")).toBeInTheDocument();
    expect(screen.queryByText("Fatebringer")).not.toBeInTheDocument();
  });

  // B2 regression: the add-to-wishlist POST must send { itemHash: <number> },
  // not the { itemId: <string> } shape that 400'd before the fix.
  it("posts the correct wishlist payload shape", async () => {
    let captured: unknown = null;
    server.use(
      treeCollectionsHandler,
      http.get(`${API}/api/wishlist`, () => HttpResponse.json([])),
      http.post(`${API}/api/wishlist`, async ({ request }) => {
        captured = await request.json();
        return HttpResponse.json(
          {
            id: "10",
            itemHash: 200,
            name: "Imperial Decree",
            itemType: "Shotgun",
            rarity: "Legendary",
            icon: "",
            priority: "MEDIUM",
            notes: "",
            sources: [],
            availableNow: false,
            dateAdded: new Date().toISOString(),
          },
          { status: 201 },
        );
      }),
    );

    renderPage(<Collections />);
    await screen.findByText("Weapons");
    fireEvent.click(screen.getByRole("button", { name: /expand weapons/i }));
    fireEvent.click(screen.getByText("Hand Cannons"));
    await screen.findByText("Imperial Decree");

    fireEvent.click(screen.getAllByTitle("Add to wishlist")[0]);

    await waitFor(() => expect(captured).not.toBeNull());
    expect(captured).toEqual({ itemHash: 200 });
  });

  it("deep-links to an item: opens its drawer and reveals its nested node", async () => {
    server.use(
      treeCollectionsHandler,
      http.get(`${API}/api/wishlist`, () => HttpResponse.json([])),
    );
    // Item "200" (Imperial Decree) lives under "Hand Cannons" (node 11),
    // nested below "Weapons" (node 10). Deep-linking must open the detail
    // drawer AND expand the sidebar to reveal that nested node.
    renderPage(<Collections />, "/collections?item=200");

    // Drawer opens for the deep-linked item.
    expect(
      await screen.findByRole("dialog", { name: "Imperial Decree" }),
    ).toBeInTheDocument();

    // The nested "Hand Cannons" node is revealed in the sidebar without any
    // manual expand click (the deep-link seeds the tree's open set).
    expect(await screen.findByText("Hand Cannons")).toBeInTheDocument();
  });

  it("shows the 'Available now' vendor on an obtainable item", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json({
          ...treeCollections,
          availableNow: { "200": "Banshee-44" },
        }),
      ),
      http.get(`${API}/api/wishlist`, () => HttpResponse.json([])),
    );
    renderPage(<Collections />);
    await screen.findByText("Weapons");
    fireEvent.click(screen.getByRole("button", { name: /expand weapons/i }));
    fireEvent.click(screen.getByText("Hand Cannons"));

    // Imperial Decree (200) is missing and listed; open its drawer.
    fireEvent.click(await screen.findByText("Imperial Decree"));

    expect(
      await screen.findByText(/Available now — Banshee-44/i),
    ).toBeInTheDocument();
  });

  it("deep-links to a weapon: fetches and renders its perk columns", async () => {
    server.use(
      treeCollectionsHandler,
      http.get(`${API}/api/wishlist`, () => HttpResponse.json([])),
      http.get(`${API}/api/items/:hash/perks`, () =>
        HttpResponse.json({
          itemHash: "200",
          perkColumns: [
            { role: "barrel", label: "Barrel", perks: ["Full Bore"] },
            { role: "trait", label: "Trait 1", perks: ["Frenzy"] },
          ],
        }),
      ),
    );
    renderPage(<Collections />, "/collections?item=200");

    // Drawer opens for the deep-linked item, then its perks load and render.
    expect(
      await screen.findByRole("dialog", { name: "Imperial Decree" }),
    ).toBeInTheDocument();
    expect(
      await screen.findByText("Possible perks / rolls"),
    ).toBeInTheDocument();
    expect(screen.getByText("Barrel")).toBeInTheDocument();
    expect(screen.getByText("Full Bore")).toBeInTheDocument();
    expect(screen.getByText("Trait 1")).toBeInTheDocument();
    expect(screen.getByText("Frenzy")).toBeInTheDocument();
  });

  it("shows the privacy error state on PRIVACY_RESTRICTION", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(
          {
            error: "User has their Destiny 2 profile set to private",
            code: "PRIVACY_RESTRICTION",
          },
          { status: 403 },
        ),
      ),
    );
    renderPage(<Collections />);
    expect(
      await screen.findByText("Your Destiny profile is private"),
    ).toBeInTheDocument();
  });

  it("opens a view-only drawer for a deep-linked non-collectible item", async () => {
    server.use(
      treeCollectionsHandler,
      http.get(`${API}/api/items/:hash`, () =>
        HttpResponse.json({
          itemHash: "999",
          name: "Vendor Mod",
          icon: "",
          itemType: "Mod",
          tierType: 5,
          rarity: "Legendary",
          description: "A mod.",
        }),
      ),
    );
    renderCollections("?item=999");
    expect(
      await screen.findByRole("dialog", { name: "Vendor Mod" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/view only/i)).toBeInTheDocument();
    // collectible-only affordances are hidden:
    expect(
      screen.queryByRole("button", { name: /add to wishlist/i }),
    ).not.toBeInTheDocument();
  });

  it("toasts on a genuine 404 for a deep-linked non-collectible item", async () => {
    server.use(
      treeCollectionsHandler,
      http.get(`${API}/api/items/:hash`, () =>
        HttpResponse.json({ error: "not found" }, { status: 404 }),
      ),
    );
    renderCollections("?item=999");
    expect(
      await screen.findByText(/that item isn't in your trackable collections/i),
    ).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});

describe("Collections filters persistence + obtainability", () => {
  // Serve the Hand Cannons leaf with 100 obtainable-now and 200 farm-only, both missing.
  function serveFilterFixture() {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json({
          ...treeCollections,
          items: {
            "100": { ...treeCollections.items["100"], farmOnly: false },
            "200": { ...treeCollections.items["200"], farmOnly: true },
          },
          collectedHashes: [], // both missing → both visible under missing-only
          availableNow: { "100": "Banshee-44" }, // 100 obtainable now
        }),
      ),
    );
  }

  it("restores the Available-now filter from the URL on load", async () => {
    serveFilterFixture();
    renderCollections("?node=11&avail=1");
    // avail=1 keeps only the obtainable item (Fatebringer); Imperial Decree drops.
    expect(await screen.findByText("Fatebringer")).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.queryByText("Imperial Decree")).not.toBeInTheDocument(),
    );
  });

  it("Hide farm-only excludes farm-only items when toggled", async () => {
    serveFilterFixture();
    renderCollections("?node=11");
    // Both visible by default.
    await screen.findByText("Fatebringer");
    expect(screen.getByText("Imperial Decree")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /hide farm-only/i }));
    await waitFor(() =>
      expect(screen.queryByText("Imperial Decree")).not.toBeInTheDocument(),
    );
    expect(screen.getByText("Fatebringer")).toBeInTheDocument();
  });

  it("preserves filter params when an ?item= deep-link opens the drawer", async () => {
    serveFilterFixture();
    renderCollections("?node=11&avail=1&item=100");
    // Drawer opens for the deep-linked item…
    await screen.findByText("Fatebringer");
    // …and avail=1 survives (the deep-link only strips item=), so Imperial Decree
    // stays filtered out rather than reappearing.
    await waitFor(() =>
      expect(screen.queryByText("Imperial Decree")).not.toBeInTheDocument(),
    );
  });

  it("reveals the restored node in the sidebar on ?node= load", async () => {
    server.use(treeCollectionsHandler);
    renderCollections("?node=11");
    // Without the reveal (findPathToNode + expandPath), the "Weapons" root
    // stays collapsed and "Hand Cannons" is absent.
    expect(await screen.findByText("Hand Cannons")).toBeInTheDocument();
  });

  it("strips item= from the URL while keeping node/avail after the deep-link drawer opens", async () => {
    serveFilterFixture();
    renderPage(
      <>
        <Collections />
        <LocationProbe />
      </>,
      "/collections?node=11&avail=1&item=100",
    );
    // Drawer opens for the deep-linked item…
    await screen.findByText("Fatebringer");
    // …and the same write that strips item= keeps node/avail intact.
    await waitFor(() => {
      const search = screen.getByTestId("search").textContent ?? "";
      expect(search).toContain("node=11");
      expect(search).toContain("avail=1");
      expect(search).not.toContain("item=");
    });
  });
});

// Regression coverage for the "default to first root" effect racing the
// `?item=` deep-link effect: both fire on a bare `/collections?item=<hash>`
// load, and react-router's functional `setSearchParams` updater snapshots
// `prev` per call — so an unguarded default-root write can silently
// re-preserve `item=` and stomp the owning node the deep-link effect just
// selected. The fix guards the default-root effect on a pending `itemParam`
// and writes via `setFilters(..., { replace: true })` instead of the node
// setter's push.
describe("Collections default-root seeding (regression)", () => {
  it("a bare ?item= deep-link consumes item and selects the OWNING node, not the root", async () => {
    server.use(treeCollectionsHandler);
    renderPage(
      <>
        <Collections />
        <LocationProbe />
      </>,
      "/collections?item=100",
    );

    // Drawer opens for the deep-linked item (100 = Fatebringer, owned by leaf
    // node "11", nested under root "10").
    expect(
      await screen.findByRole("dialog", { name: "Fatebringer" }),
    ).toBeInTheDocument();

    await waitFor(() => {
      const search = screen.getByTestId("search").textContent ?? "";
      // Owning node wins — NOT the root the buggy default-root effect used to
      // stomp it with.
      expect(search).toContain("node=11");
      expect(search).not.toContain("node=10");
      // The deep-link's own write consumed item= — the default-root effect
      // must not have re-added it.
      expect(search).not.toContain("item=");
    });
  });

  it("a cold /collections load canonicalizes to the first root via a REPLACE (Back isn't trapped)", async () => {
    server.use(treeCollectionsHandler);
    const qc = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    render(
      <QueryClientProvider client={qc}>
        <AuthProvider>
          <PreferencesProvider>
            <CharacterProvider>
              <ToastProvider>
                <MemoryRouter
                  initialEntries={["/elsewhere", "/collections"]}
                  initialIndex={1}
                >
                  <Collections />
                  <LocationProbe />
                  <PathProbe />
                  <BackButton />
                </MemoryRouter>
              </ToastProvider>
            </CharacterProvider>
          </PreferencesProvider>
        </AuthProvider>
      </QueryClientProvider>,
    );

    // No node/filters in the URL, so the default-root effect seeds node=10
    // (the first, and only, root in the fixture).
    await waitFor(() => {
      const search = screen.getByTestId("search").textContent ?? "";
      expect(search).toContain("node=10");
    });

    // Prove the seed was a history REPLACE, not a push: from here, Back should
    // land on the entry that was already behind "/collections" before the
    // effect ran ("/elsewhere"). A push would have inserted a new entry ahead
    // of the paramless "/collections", so Back would land back on paramless
    // "/collections" instead — re-triggering (and re-pushing) the same
    // canonicalization effect.
    fireEvent.click(screen.getByText("go back"));

    await waitFor(() => {
      expect(screen.getByTestId("path").textContent).toBe("/elsewhere");
    });
  });

  it("falls back to the first root when ?node= points at an unknown node", async () => {
    server.use(treeCollectionsHandler);
    renderPage(
      <>
        <Collections />
        <LocationProbe />
      </>,
      "/collections?node=doesnotexist",
    );
    // A stale/tampered node hash is replaced with the first root (10) so the grid
    // isn't stuck empty with no way to recover.
    await waitFor(() => {
      const search = screen.getByTestId("search").textContent ?? "";
      expect(search).toContain("node=10");
      expect(search).not.toContain("doesnotexist");
    });
  });
});

// Fixture with a mismatched-rarity third item so a rarity+search test can
// assert a true intersection (not just the search result on its own).
const mixedRarityCollections = {
  tree: [
    {
      hash: "10",
      name: "Weapons",
      icon: "",
      collected: 0,
      total: 3,
      children: [
        {
          hash: "11",
          name: "Hand Cannons",
          icon: "",
          collected: 0,
          total: 3,
          items: ["100", "200", "300"],
        },
      ],
    },
  ],
  items: {
    "100": {
      itemHash: "100",
      name: "Fatebringer",
      description: "",
      icon: "/i/fb.png",
      itemType: "Hand Cannon",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "Challenging",
      sources: ["Vault of Glass"],
      isExotic: false,
    },
    "200": {
      itemHash: "200",
      name: "Imperial Decree",
      description: "",
      icon: "/i/id.png",
      itemType: "Shotgun",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "Moderate",
      sources: ["Menagerie"],
      isExotic: false,
    },
    "300": {
      itemHash: "300",
      name: "Imperial Threat",
      description: "",
      icon: "/i/it.png",
      itemType: "Hand Cannon",
      tierType: 6,
      rarity: "Exotic",
      difficulty: "Challenging",
      sources: ["Trials"],
      isExotic: true,
    },
  },
  collectedHashes: [],
  summary: {
    weapons: { total: 3, collected: 0 },
    armor: { total: 0, collected: 0 },
    exotics: { total: 0, collected: 0 },
    cosmetics: { total: 0, collected: 0 },
  },
  fetchedAt: new Date().toISOString(),
};

// Fixture whose node gathers items in reverse-alphabetical order, so a
// sort=name test proves sorting actually reordered the filtered set rather
// than happening to already match gather order.
const sortOrderCollections = {
  tree: [
    {
      hash: "10",
      name: "Weapons",
      icon: "",
      collected: 0,
      total: 3,
      children: [
        {
          hash: "11",
          name: "Hand Cannons",
          icon: "",
          collected: 0,
          total: 3,
          // Deliberately includes a non-matching item (Fatebringer) so this
          // fixture also proves the search predicate actually filtered
          // something, not just that sort ran over an already-filtered set.
          items: ["300", "200", "100"],
        },
      ],
    },
  ],
  items: {
    "100": {
      itemHash: "100",
      name: "Fatebringer",
      description: "",
      icon: "",
      itemType: "Hand Cannon",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "Challenging",
      sources: ["Vault of Glass"],
      isExotic: false,
    },
    "200": {
      itemHash: "200",
      name: "Imperial Decree",
      description: "",
      icon: "",
      itemType: "Shotgun",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "Moderate",
      sources: ["Menagerie"],
      isExotic: false,
    },
    "300": {
      itemHash: "300",
      name: "Imperial Threat",
      description: "",
      icon: "",
      itemType: "Hand Cannon",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "Challenging",
      sources: ["Trials"],
      isExotic: false,
    },
  },
  collectedHashes: [],
  summary: {
    weapons: { total: 3, collected: 0 },
    armor: { total: 0, collected: 0 },
    exotics: { total: 0, collected: 0 },
    cosmetics: { total: 0, collected: 0 },
  },
  fetchedAt: new Date().toISOString(),
};

describe("Collections search field", () => {
  it("has an accessible name of 'Search this category…'", async () => {
    server.use(treeCollectionsHandler);
    renderPage(<Collections />);
    expect(
      await screen.findByRole("searchbox", { name: "Search this category…" }),
    ).toBeInTheDocument();
  });

  it("filters the grid to case-insensitive name-substring matches while typing", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json({ ...treeCollections, collectedHashes: [] }),
      ),
    );
    renderCollections("?node=11");
    await screen.findByText("Fatebringer");
    expect(screen.getByText("Imperial Decree")).toBeInTheDocument();

    const search = screen.getByRole("searchbox", {
      name: "Search this category…",
    });
    fireEvent.change(search, { target: { value: "FATE" } });

    await waitFor(() =>
      expect(screen.queryByText("Imperial Decree")).not.toBeInTheDocument(),
    );
    expect(screen.getByText("Fatebringer")).toBeInTheDocument();
  });

  it("composes the search term with an active rarity filter (intersection, not union)", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(mixedRarityCollections),
      ),
    );
    renderCollections("?node=11&rarity=legendary&q=imperial");

    // Matches both the rarity filter and the search term.
    expect(await screen.findByText("Imperial Decree")).toBeInTheDocument();
    // Matches the search term but not the rarity filter — excluded.
    expect(screen.queryByText("Imperial Threat")).not.toBeInTheDocument();
    // Matches the rarity filter but not the search term — excluded.
    expect(screen.queryByText("Fatebringer")).not.toBeInTheDocument();
  });

  it("composes the search term with a non-default sort, preserving sort order", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(sortOrderCollections),
      ),
    );
    renderCollections("?node=11&sort=name&q=imperial");

    const matches = await screen.findAllByText(/^Imperial/);
    expect(matches.map((el) => el.textContent)).toEqual([
      "Imperial Decree",
      "Imperial Threat",
    ]);
    // Fatebringer doesn't match the search term — proves this list was
    // actually filtered, not just sorted.
    expect(screen.queryByText("Fatebringer")).not.toBeInTheDocument();
  });

  it("keeps the search term across a category (node) change", async () => {
    server.use(treeCollectionsHandler);
    renderPage(
      <>
        <Collections />
        <LocationProbe />
      </>,
      "/collections?node=11&q=imperial",
    );
    await screen.findByText("Imperial Decree");

    fireEvent.click(screen.getByText("Weapons"));

    await waitFor(() => {
      const search = screen.getByTestId("search").textContent ?? "";
      expect(search).toContain("node=10");
      expect(search).toContain("q=imperial");
    });
    expect(
      screen.getByRole("searchbox", { name: "Search this category…" }),
    ).toHaveValue("imperial");
  });

  it("shows a search-specific empty state naming the term when nothing matches", async () => {
    server.use(treeCollectionsHandler);
    renderCollections("?node=11&q=nonexistentitem");

    expect(
      await screen.findByText(/no items match "nonexistentitem"/i),
    ).toBeInTheDocument();
  });

  it("Clear filters clears the search field and restores the full grid", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json({ ...treeCollections, collectedHashes: [] }),
      ),
    );
    renderCollections("?node=11&q=nonexistentitem");
    await screen.findByText(/no items match "nonexistentitem"/i);

    fireEvent.click(screen.getByRole("button", { name: /clear filters/i }));

    await screen.findByText("Fatebringer");
    expect(screen.getByText("Imperial Decree")).toBeInTheDocument();
    expect(
      screen.getByRole("searchbox", { name: "Search this category…" }),
    ).toHaveValue("");
  });
});
