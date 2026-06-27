import React from "react";
import {
  describe,
  it,
  expect,
  beforeAll,
  afterAll,
  beforeEach,
  afterEach,
} from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { CharacterProvider } from "../../contexts/CharacterContext";
import { PreferencesProvider } from "../../contexts/PreferencesContext";
import { ToastProvider } from "../../components/Toast";
import { Collections } from "./Collections";

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  localStorage.clear();
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_refresh_token", "test-refresh");
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
});
