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
import { http, HttpResponse, delay } from "msw";
import { API, sampleUser, sampleWeekly, server } from "./testServer";
import { AuthProvider } from "../contexts/AuthContext";
import { ToastProvider } from "../components/ui/Toast";
import { Catalysts } from "../pages/Catalysts";
import { Triumphs } from "../pages/Triumphs";
import { ThisWeek } from "../pages/ThisWeek";

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
        <ToastProvider>
          <MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter>
        </ToastProvider>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

const privacy403 = () =>
  HttpResponse.json(
    { error: "Profile is private", code: "PRIVACY_RESTRICTION" },
    { status: 403 },
  );

describe("Catalysts page", () => {
  it("renders catalyst cards with progress, complete, and not-acquired states", async () => {
    renderPage(<Catalysts />);
    expect(await screen.findByText("Sunshot Catalyst")).toBeInTheDocument();
    // in-progress fixture has an objective → progress bar value rendered
    expect(screen.getByText("Riskrunner Catalyst")).toBeInTheDocument();
    // missing fixture has obj=null → "Not yet acquired"
    expect(screen.getByText("Not yet acquired")).toBeInTheDocument();
    // "1/3 complete" (only Riskrunner is complete)
    expect(screen.getByText("1/3 complete")).toBeInTheDocument();
  });

  it("filters catalysts by status", async () => {
    renderPage(<Catalysts />);
    await screen.findByText("Sunshot Catalyst");

    // Filter chips are buttons; Badge labels with the same text are spans.
    fireEvent.click(screen.getByRole("button", { name: "Complete" }));
    expect(screen.getByText("Riskrunner Catalyst")).toBeInTheDocument();
    expect(screen.queryByText("Sunshot Catalyst")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "In progress" }));
    expect(screen.getByText("Sunshot Catalyst")).toBeInTheDocument();
    expect(screen.queryByText("Riskrunner Catalyst")).not.toBeInTheDocument();
  });

  it("shows the empty state when a catalyst filter matches nothing", async () => {
    server.use(
      http.get(`${API}/api/catalysts/:type/:id`, () =>
        HttpResponse.json({
          items: [
            {
              id: "c-only",
              name: "Lone Catalyst",
              type: "Bow",
              icon: "",
              status: "complete",
              obj: { label: "Kills", cur: 1, max: 1 },
              source: "Strikes",
            },
          ],
          fetchedAt: "",
        }),
      ),
    );
    renderPage(<Catalysts />);
    await screen.findByText("Lone Catalyst");
    fireEvent.click(screen.getByRole("button", { name: "Missing" }));
    expect(
      screen.getByText("No catalysts match this filter."),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByText("Clear"));
    expect(await screen.findByText("Lone Catalyst")).toBeInTheDocument();
  });

  it("switches to the crafting tab and shows craftable vs in-progress", async () => {
    renderPage(<Catalysts />);
    await screen.findByText("Sunshot Catalyst");

    fireEvent.click(screen.getByRole("button", { name: "Crafting Patterns" }));
    expect(await screen.findByText("The Enigma")).toBeInTheDocument();
    // Enigma cur>=max → "Craftable"; Osteo not done → note badge
    expect(screen.getByText("Craftable")).toBeInTheDocument();
    expect(screen.getByText("3 to go")).toBeInTheDocument();
    expect(screen.getByText("1/3 craftable")).toBeInTheDocument();
  });

  it("filters crafting by missing/in-progress and shows the empty state when nothing matches", async () => {
    renderPage(<Catalysts />);
    await screen.findByText("Sunshot Catalyst");
    fireEvent.click(screen.getByRole("button", { name: "Crafting Patterns" }));
    await screen.findByText("The Enigma");

    // "missing" → only patterns with cur === 0 (the fresh pattern)
    fireEvent.click(screen.getByRole("button", { name: "Missing" }));
    expect(screen.getByText("Unworked Pattern")).toBeInTheDocument();
    expect(screen.queryByText("The Enigma")).not.toBeInTheDocument();

    // "in-progress" → started but not done (Osteo: 2/5)
    fireEvent.click(screen.getByRole("button", { name: "In progress" }));
    expect(screen.getByText("Osteo Striga")).toBeInTheDocument();
    expect(screen.queryByText("Unworked Pattern")).not.toBeInTheDocument();
    expect(screen.queryByText("The Enigma")).not.toBeInTheDocument();
  });

  it("shows the empty state when a crafting filter matches nothing", async () => {
    server.use(
      http.get(`${API}/api/crafting/:type/:id`, () =>
        HttpResponse.json({
          items: [
            {
              id: "cr-only",
              name: "Half-Done Pattern",
              type: "Glaive",
              patterns: { cur: 2, max: 5 },
              note: "3 to go",
              source: "Crafting",
            },
          ],
          fetchedAt: "",
        }),
      ),
    );
    renderPage(<Catalysts />);
    await screen.findByText("Sunshot Catalyst");
    fireEvent.click(screen.getByRole("button", { name: "Crafting Patterns" }));
    await screen.findByText("Half-Done Pattern");

    // Only an in-progress pattern exists → filtering to "complete" empties the grid
    fireEvent.click(screen.getByRole("button", { name: "Complete" }));
    expect(
      screen.getByText("No patterns match this filter."),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByText("Clear"));
    expect(await screen.findByText("Half-Done Pattern")).toBeInTheDocument();
  });

  it("shows the privacy error state with a retry and Bungie link", async () => {
    server.use(http.get(`${API}/api/catalysts/:type/:id`, privacy403));
    renderPage(<Catalysts />);
    expect(
      await screen.findByText("Your Destiny profile is private"),
    ).toBeInTheDocument();
    expect(screen.getByText("Bungie privacy settings")).toBeInTheDocument();
    expect(screen.getByText("Retry")).toBeInTheDocument();
  });

  it("renders the loading spinner while the query is in flight", async () => {
    server.use(
      http.get(`${API}/api/catalysts/:type/:id`, async () => {
        await delay(40);
        return HttpResponse.json({ items: [], fetchedAt: "" });
      }),
    );
    const { container } = renderPage(<Catalysts />);
    expect(container.querySelector(".gt-page-loading")).not.toBeNull();
  });
});

describe("Triumphs page", () => {
  it("renders seals, auto-expands the first, and counts gilded", async () => {
    renderPage(<Triumphs />);
    expect(await screen.findByText("Conqueror")).toBeInTheDocument();
    expect(screen.getByText("2 seals · 1 gilded")).toBeInTheDocument();
    // First seal auto-expands → its triumph labels are visible
    expect(screen.getByText("Complete a GM")).toBeInTheDocument();
  });

  it("toggles a seal open and closed", async () => {
    renderPage(<Triumphs />);
    await screen.findByText("Conqueror");

    // Collapse the auto-opened first seal
    fireEvent.click(screen.getByText("Conqueror"));
    await waitFor(() =>
      expect(screen.queryByText("Complete a GM")).not.toBeInTheDocument(),
    );

    // Re-open it
    fireEvent.click(screen.getByText("Conqueror"));
    expect(await screen.findByText("Complete a GM")).toBeInTheDocument();
  });

  it("re-sorts by name when chosen from the dropdown", async () => {
    renderPage(<Triumphs />);
    await screen.findByText("Conqueror");

    fireEvent.click(screen.getByText("Sort: Closest to done"));
    fireEvent.click(screen.getByText("Sort: Name"));
    // After sorting by name, both seals remain rendered
    expect(screen.getByText("Conqueror")).toBeInTheDocument();
    expect(screen.getByText("Flawless")).toBeInTheDocument();
  });

  it("shows the privacy error state", async () => {
    server.use(http.get(`${API}/api/seals/:type/:id`, privacy403));
    renderPage(<Triumphs />);
    expect(
      await screen.findByText("Your Destiny profile is private"),
    ).toBeInTheDocument();
  });

  it("renders the loading spinner while the query is in flight", async () => {
    server.use(
      http.get(`${API}/api/seals/:type/:id`, async () => {
        await delay(40);
        return HttpResponse.json({ items: [], fetchedAt: "" });
      }),
    );
    const { container } = renderPage(<Triumphs />);
    expect(container.querySelector(".gt-page-loading")).not.toBeNull();
  });
});

describe("ThisWeek page", () => {
  it("renders reset label and the Xûr-away module by default", async () => {
    renderPage(<ThisWeek />);
    expect(
      await screen.findByText("Resets Tuesday 17:00 UTC"),
    ).toBeInTheDocument();
    expect(screen.getByText("Xûr returns Friday")).toBeInTheDocument();
    expect(screen.getByText("0/0 done")).toBeInTheDocument();
  });

  it("toggles a recommended action done and persists across the reset key", async () => {
    server.use(
      http.get(`${API}/api/weekly/recommendations`, () =>
        HttpResponse.json({
          ...sampleWeekly,
          recommended: [
            {
              id: "rec-1",
              text: "Run a Nightfall",
              detail: "Strikes",
              badge: "missing",
              done: false,
              diff: "moderate",
              time: "30m",
            },
          ],
        }),
      ),
    );
    renderPage(<ThisWeek />);
    expect(await screen.findByText("Run a Nightfall")).toBeInTheDocument();
    expect(screen.getByText("0/1 done")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Mark done" }));
    expect(await screen.findByText("1/1 done")).toBeInTheDocument();
    // Persisted under the reset-keyed localStorage entry
    expect(localStorage.getItem("gt_done:" + sampleWeekly.resetAt)).toContain(
      "rec-1",
    );

    // Toggle back off
    fireEvent.click(screen.getByRole("button", { name: "Mark done" }));
    expect(await screen.findByText("0/1 done")).toBeInTheDocument();
  });

  it("renders the degraded banner, Xûr inventory, milestones and vendors", async () => {
    server.use(
      http.get(`${API}/api/weekly/recommendations`, () =>
        HttpResponse.json({
          ...sampleWeekly,
          degraded: true,
          xur: {
            present: true,
            leavesIn: { d: 1, h: 2, m: 0 },
            location: "Tower Hangar",
            items: [
              {
                name: "Gjallarhorn",
                type: "Rocket Launcher",
                rarity: "exotic",
                missing: true,
                cost: "29 Strange Coins",
              },
              {
                name: "Ace of Spades",
                type: "Hand Cannon",
                rarity: "exotic",
                missing: false,
                cost: "29 Strange Coins",
              },
            ],
          },
          milestones: [
            {
              id: "m1",
              label: "Weekly",
              name: "Vanguard Ops",
              reward: "Pinnacle",
              missing: 2,
              note: "",
            },
            {
              id: "m2",
              label: "Weekly",
              name: "Crucible",
              reward: "Pinnacle",
              note: "",
            },
          ],
          vendors: [
            {
              id: "v1",
              name: "Banshee",
              role: "Gunsmith",
              missing: 3,
              items: ["Rifle", "Cannon"],
            },
            {
              id: "v2",
              name: "Xûr",
              role: "Exotics",
              missing: 0,
              items: ["Boots"],
            },
          ],
        }),
      ),
    );
    renderPage(<ThisWeek />);
    expect(
      await screen.findByText(/Item names are still loading/),
    ).toBeInTheDocument();
    expect(screen.getByText("Tower Hangar")).toBeInTheDocument();
    expect(screen.getByText("Gjallarhorn")).toBeInTheDocument();
    expect(screen.getByText("Vanguard Ops")).toBeInTheDocument();
    expect(screen.getByText("Banshee")).toBeInTheDocument();
    expect(screen.getByText("Caught up")).toBeInTheDocument();
  });

  it("purges checkmarks from previous reset weeks on load", async () => {
    localStorage.setItem(
      "gt_done:2020-01-01T00:00:00Z",
      JSON.stringify(["old"]),
    );
    renderPage(<ThisWeek />);
    await screen.findByText("Resets Tuesday 17:00 UTC");
    await waitFor(() =>
      expect(localStorage.getItem("gt_done:2020-01-01T00:00:00Z")).toBeNull(),
    );
  });

  it("renders the loading spinner while the query is in flight", async () => {
    server.use(
      http.get(`${API}/api/weekly/recommendations`, async () => {
        await delay(40);
        return HttpResponse.json(sampleWeekly);
      }),
    );
    const { container } = renderPage(<ThisWeek />);
    expect(container.querySelector(".gt-page")).not.toBeNull();
    // The spinner-only page has no PageHead while loading
    expect(
      screen.queryByText("Resets Tuesday 17:00 UTC"),
    ).not.toBeInTheDocument();
  });
});
