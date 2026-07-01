import React from "react";
import {
  describe,
  it,
  expect,
  vi,
  beforeAll,
  afterAll,
  afterEach,
  beforeEach,
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
import { Cosmetics } from "./Cosmetics";

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
beforeAll(() => {
  server.listen({ onUnhandledRequest: "error" });
  vi.stubGlobal("ResizeObserver", ResizeObserverStub);
  Object.defineProperty(HTMLElement.prototype, "clientWidth", {
    configurable: true,
    get: () => 1000,
  });
  // @tanstack/react-virtual v3.14.5 measures the scroll container via
  // offsetHeight (not getBoundingClientRect). Without this stub the virtualizer
  // sees a zero-height viewport and renders no tiles.
  Object.defineProperty(HTMLElement.prototype, "offsetHeight", {
    configurable: true,
    get: () => 800,
  });
  HTMLElement.prototype.getBoundingClientRect = () =>
    ({
      width: 1000,
      height: 800,
      top: 0,
      left: 0,
      right: 1000,
      bottom: 800,
      x: 0,
      y: 0,
      toJSON: () => {},
    }) as DOMRect;
});
afterEach(() => server.resetHandlers());
afterAll(() => {
  server.close();
  vi.unstubAllGlobals();
});

beforeEach(() => {
  localStorage.clear();
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_refresh_token", "test-refresh");
  localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
});

const cosmeticsData = {
  tree: [
    {
      hash: "c",
      name: "Cosmetics",
      icon: "",
      collected: 1,
      total: 2,
      children: [
        {
          hash: "emb",
          name: "Emblems",
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
      name: "Calus Selected",
      description: "",
      icon: "/a.png",
      itemType: "Emblem",
      tierType: 5,
      rarity: "Legendary",
      difficulty: "",
      sources: [],
      isExotic: false,
    },
    "200": {
      itemHash: "200",
      name: "Neon Mareld",
      description: "",
      icon: "/b.png",
      itemType: "Emblem",
      tierType: 6,
      rarity: "Exotic",
      difficulty: "",
      sources: [],
      isExotic: true,
    },
  },
  collectedHashes: ["100"],
  summary: {
    weapons: { total: 0, collected: 0 },
    armor: { total: 0, collected: 0 },
    exotics: { total: 0, collected: 0 },
    cosmetics: { total: 2, collected: 1 },
  },
  fetchedAt: "2026-06-30T00:00:00Z",
};

function renderCosmetics() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <AuthProvider>
        <PreferencesProvider>
          <CharacterProvider>
            <ToastProvider>
              <MemoryRouter initialEntries={["/cosmetics"]}>
                <Cosmetics />
              </MemoryRouter>
            </ToastProvider>
          </CharacterProvider>
        </PreferencesProvider>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

describe("Cosmetics", () => {
  it("shows a tab per cosmetic type and lists owned + missing under 'All'", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(cosmeticsData),
      ),
    );
    renderCosmetics();
    expect(
      await screen.findByRole("tab", { name: "Emblem" }),
    ).toBeInTheDocument();
    expect(await screen.findByText("Calus Selected")).toBeInTheDocument();
    expect(screen.getByText("Neon Mareld")).toBeInTheDocument();
    expect(screen.getByText("1/2 collected")).toBeInTheDocument();
  });

  it("filters to owned only", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(cosmeticsData),
      ),
    );
    renderCosmetics();
    await screen.findByText("Calus Selected");
    fireEvent.click(screen.getByRole("button", { name: "Owned" }));
    expect(screen.getByText("Calus Selected")).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.queryByText("Neon Mareld")).not.toBeInTheDocument(),
    );
  });

  it("opens the detail drawer when a tile is clicked", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(cosmeticsData),
      ),
    );
    renderCosmetics();
    fireEvent.click(await screen.findByText("Calus Selected"));
    expect(
      await screen.findByRole("dialog", { name: "Calus Selected" }),
    ).toBeInTheDocument();
  });

  it("shows an empty state when no cosmetics are present", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json({
          tree: [],
          items: {},
          collectedHashes: [],
          summary: {
            weapons: { total: 0, collected: 0 },
            armor: { total: 0, collected: 0 },
            exotics: { total: 0, collected: 0 },
            cosmetics: { total: 0, collected: 0 },
          },
          fetchedAt: "2026-06-30T00:00:00Z",
        }),
      ),
    );
    renderCosmetics();
    expect(
      await screen.findByText("Cosmetics data unavailable."),
    ).toBeInTheDocument();
  });
});
