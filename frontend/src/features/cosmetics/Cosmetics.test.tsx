import React from "react";
import {
  describe,
  it,
  expect,
  vi,
  beforeAll,
  afterAll,
  beforeEach,
} from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { API, server } from "../../test/testServer";
import { renderWithProviders } from "../../test/renderWithProviders";
import { Cosmetics } from "./Cosmetics";

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
beforeAll(() => {
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
afterAll(() => {
  vi.unstubAllGlobals();
});

beforeEach(() => {
  localStorage.clear();
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
          total: 4,
          items: ["100", "200", "300", "400"],
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
      acquisitionSources: [],
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
      acquisitionSources: [],
      isExotic: true,
    },
    "300": {
      itemHash: "300",
      name: "Test Ornament",
      description: "",
      icon: "/o.png",
      itemType: "Ornament",
      tierType: 5,
      rarity: "Legendary",
      acquisitionSources: [],
      isExotic: false,
    },
    "400": {
      itemHash: "400",
      name: "Test Finisher",
      description: "",
      icon: "/f.png",
      itemType: "Finisher",
      tierType: 5,
      rarity: "Legendary",
      acquisitionSources: [],
      isExotic: false,
    },
  },
  collectedHashes: ["100"],
  summary: {
    weapons: { total: 0, collected: 0 },
    armor: { total: 0, collected: 0 },
    exotics: { total: 0, collected: 0 },
    cosmetics: { total: 4, collected: 1 },
  },
  fetchedAt: "2026-06-30T00:00:00Z",
};

function renderCosmetics() {
  return renderWithProviders(<Cosmetics />, { route: "/cosmetics" });
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
    expect(screen.getByRole("tab", { name: "Ornament" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Finisher" })).toBeInTheDocument();
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
    expect(await screen.findByText("No cosmetics data")).toBeInTheDocument();
  });

  it("shows the privacy error state when the collections request is forbidden", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(
          { error: "Profile is private", code: "PRIVACY_RESTRICTION" },
          { status: 403 },
        ),
      ),
    );
    renderCosmetics();
    expect(
      await screen.findByText(/your destiny profile is private/i),
    ).toBeInTheDocument();
    expect(screen.getByText("Retry")).toBeInTheDocument();
  });

  it("shows the manifest-warming error state when collections return 503", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(
          { error: "Manifest not ready", code: "MANIFEST_NOT_READY" },
          { status: 503 },
        ),
      ),
    );
    renderCosmetics();
    expect(await screen.findByText(/warming up/i)).toBeInTheDocument();
  });

  it("wires tabs to the tabpanel", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(cosmeticsData),
      ),
    );
    renderCosmetics();
    const tab = await screen.findByRole("tab", { selected: true });
    expect(tab).toHaveAttribute("aria-controls", "cosmetics-panel");
    expect(screen.getByRole("tabpanel")).toHaveAttribute(
      "id",
      "cosmetics-panel",
    );
  });
});
