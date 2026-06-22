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
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "./testServer";
import { AuthProvider } from "../contexts/AuthContext";
import { CharacterProvider } from "../contexts/CharacterContext";
import { PreferencesProvider } from "../contexts/PreferencesContext";
import { ToastProvider } from "../components/ui/Toast";
import { Collections } from "../pages/Collections";
import { Dashboard } from "../pages/Dashboard";
import { OAuthCallback } from "../pages/OAuthCallback";

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
        HttpResponse.json({ ...treeCollections, availableNow: { "200": "Banshee-44" } }),
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

describe("Dashboard", () => {
  it("renders real collection totals and honest wishlist availability", async () => {
    renderPage(<Dashboard />);
    expect(
      await screen.findByText(/Welcome back, TestGuardian/),
    ).toBeInTheDocument();
    // 8 of 10 weapons collected in the fixture
    expect(await screen.findByText("8/10")).toBeInTheDocument();
    // One wishlist item, available now (fixture)
    expect(
      await screen.findByText(/of 1 wishlist items available now/),
    ).toBeInTheDocument();
    expect(await screen.findByText("Gjallarhorn")).toBeInTheDocument();
  });
});

describe("OAuthCallback", () => {
  // B13 regression: React StrictMode double-invokes effects in dev — the
  // single-use auth code must only be submitted once.
  it("submits the auth code exactly once under StrictMode", async () => {
    let callbackPosts = 0;
    server.use(
      http.post(`${API}/api/auth/bungie/callback`, () => {
        callbackPosts++;
        return HttpResponse.json({
          token: "new-token",
          refreshToken: "new-refresh",
          user: sampleUser,
        });
      }),
    );

    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <React.StrictMode>
        <QueryClientProvider client={qc}>
          <AuthProvider>
            <MemoryRouter
              initialEntries={["/auth/callback?code=onetime&state=sig"]}
            >
              <Routes>
                <Route path="/auth/callback" element={<OAuthCallback />} />
                <Route path="/dashboard" element={<div>dashboard-stub</div>} />
              </Routes>
            </MemoryRouter>
          </AuthProvider>
        </QueryClientProvider>
      </React.StrictMode>,
    );

    expect(await screen.findByText("dashboard-stub")).toBeInTheDocument();
    expect(callbackPosts).toBe(1);
  });

  it("shows an error when Bungie returns one", async () => {
    const qc = new QueryClient();
    render(
      <QueryClientProvider client={qc}>
        <AuthProvider>
          <MemoryRouter initialEntries={["/auth/callback?error=access_denied"]}>
            <Routes>
              <Route path="/auth/callback" element={<OAuthCallback />} />
              <Route path="/login" element={<div>login-stub</div>} />
            </Routes>
          </MemoryRouter>
        </AuthProvider>
      </QueryClientProvider>,
    );
    expect(await screen.findByText("Authentication error")).toBeInTheDocument();
  });
});
