import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { act, screen, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleWeekly, server } from "../../test/testServer";
import { renderWithProviders } from "../../test/renderWithProviders";
import { Dashboard } from "./Dashboard";

beforeEach(() => {
  localStorage.clear();
});

function renderPage(ui: React.ReactNode, route = "/") {
  return renderWithProviders(ui, { route });
}

describe("Dashboard", () => {
  it("renders real collection totals and honest wishlist availability", async () => {
    renderPage(<Dashboard />);
    expect(
      await screen.findByText(/Welcome, TestGuardian/),
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

describe("Dashboard page", () => {
  function renderDashboard() {
    return renderPage(<Dashboard />);
  }

  it("shows the top recommended action as the best thing to do today", async () => {
    renderDashboard();
    expect(await screen.findByText("Run Vault of Glass")).toBeInTheDocument();
    expect(screen.getByText(/Best thing to do today/i)).toBeInTheDocument();
  });

  it("greets the current Guardian Tracker user", async () => {
    renderDashboard();
    expect(await screen.findByText(/^Welcome, /)).toBeInTheDocument();
  });

  it("omits the weekly reset countdown while weekly data is loading", async () => {
    let weeklyRequested = false;
    let releaseWeekly: ((response: Response) => void) | undefined;
    server.use(
      http.get(`${API}/api/weekly/recommendations`, () => {
        weeklyRequested = true;
        return new Promise<Response>((resolve) => {
          releaseWeekly = resolve;
        });
      }),
    );

    renderDashboard();
    await waitFor(() => expect(weeklyRequested).toBe(true));

    try {
      expect(screen.queryByText("Weekly reset")).not.toBeInTheDocument();
    } finally {
      releaseWeekly?.(HttpResponse.json(sampleWeekly));
    }

    const resetLabel = await screen.findByText("Weekly reset");
    expect(resetLabel.closest(".gt-chip--count")).toHaveTextContent(
      /Weekly reset\s*2d 4h/,
    );
  });

  it("shows the actual weekly reset countdown when weekly data loads", async () => {
    renderDashboard();

    const resetLabel = await screen.findByText("Weekly reset");
    expect(resetLabel.closest(".gt-chip--count")).toHaveTextContent(
      /Weekly reset\s*2d 4h/,
    );
  });

  it("omits a cached weekly reset countdown when a refetch fails", async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    renderWithProviders(<Dashboard />, { client });
    expect(await screen.findByText("Weekly reset")).toBeInTheDocument();

    server.use(
      http.get(`${API}/api/weekly/recommendations`, () =>
        HttpResponse.json({ error: "boom" }, { status: 500 }),
      ),
    );
    await act(async () => {
      await client.refetchQueries({ queryKey: ["weekly", null] });
    });

    expect(
      await screen.findAllByText(/couldn't load this week's data/i),
    ).toHaveLength(2);
    expect(screen.queryByText("Weekly reset")).not.toBeInTheDocument();
  });

  it("shows the privacy error state when collections is blocked", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(
          { error: "restricted", code: "PRIVACY_RESTRICTION" },
          { status: 403 },
        ),
      ),
    );
    renderDashboard();
    expect(
      await screen.findByText(/your destiny profile is private/i),
    ).toBeInTheDocument();
  });

  it("shows the warming state on manifest 503", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(
          { error: "not ready", code: "MANIFEST_NOT_READY" },
          { status: 503 },
        ),
      ),
    );
    renderDashboard();
    expect(await screen.findByText(/warming up/i)).toBeInTheDocument();
  });

  it("degrades the weekly panel without zeroing the page", async () => {
    server.use(
      http.get(`${API}/api/weekly/recommendations`, () =>
        HttpResponse.json({ error: "boom" }, { status: 500 }),
      ),
    );
    renderDashboard();
    // The muted degraded-state row appears in both the "Do this today" panel
    // and the "This week — preview" list (task spec: both surfaces degrade
    // independently), so this is a multi-match query rather than findByText.
    const degraded = await screen.findAllByText(
      /couldn't load this week's data/i,
    );
    expect(degraded).toHaveLength(2);
    expect(screen.queryByText("Weekly reset")).not.toBeInTheDocument();
    // Collections hero still renders real numbers:
    expect(await screen.findByText(/overall/i)).toBeInTheDocument();
  });

  it("shows a muted row and hides the count header when wishlist fails", async () => {
    server.use(
      http.get(`${API}/api/wishlist`, () =>
        HttpResponse.json({ error: "boom" }, { status: 500 }),
      ),
    );
    renderDashboard();
    expect(
      await screen.findByText(/couldn't load your wishlist/i),
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/wishlist items available now/i),
    ).not.toBeInTheDocument();
  });
});

describe("Dashboard — since your last visit (ADR 0023)", () => {
  function renderDashboard() {
    return renderPage(<Dashboard />);
  }

  it("shows 'tracking starts now' on a first visit, never an empty digest", async () => {
    server.use(
      http.get(`${API}/api/digest/:type/:id`, () =>
        HttpResponse.json({
          status: "first-visit",
          visitStartedAt: "2026-09-18T18:00:00Z",
          acquired: [],
        }),
      ),
    );
    renderDashboard();

    expect(await screen.findByText(/tracking starts now/i)).toBeInTheDocument();
    expect(
      screen.queryByText(/nothing new since your last visit/i),
    ).not.toBeInTheDocument();
  });

  it("distinguishes a genuinely empty ready digest from a first visit", async () => {
    server.use(
      http.get(`${API}/api/digest/:type/:id`, () =>
        HttpResponse.json({
          status: "ready",
          visitStartedAt: "2026-09-18T18:00:00Z",
          previousVisitAt: "2026-09-17T12:00:00Z",
          acquired: [],
        }),
      ),
    );
    renderDashboard();

    expect(
      await screen.findByText(/nothing new since your last visit/i),
    ).toBeInTheDocument();
    expect(screen.queryByText(/tracking starts now/i)).not.toBeInTheDocument();
  });

  it("renders acquired items for a ready digest with something new", async () => {
    server.use(
      http.get(`${API}/api/digest/:type/:id`, () =>
        HttpResponse.json({
          status: "ready",
          visitStartedAt: "2026-09-18T18:00:00Z",
          previousVisitAt: "2026-09-17T12:00:00Z",
          acquired: [
            {
              itemHash: 99999,
              name: "Gjallarhorn",
              icon: "/icons/gj.png",
              itemType: "Rocket Launcher",
            },
          ],
        }),
      ),
    );
    renderDashboard();

    expect(await screen.findAllByText("Gjallarhorn")).not.toHaveLength(0);
    expect(
      screen.queryByText(/nothing new since your last visit/i),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/tracking starts now/i)).not.toBeInTheDocument();
  });

  it("shows a quiet, non-alarming state when the digest is unavailable", async () => {
    server.use(
      http.get(`${API}/api/digest/:type/:id`, () =>
        HttpResponse.json({
          status: "unavailable",
          visitStartedAt: "2026-09-18T18:00:00Z",
          acquired: [],
        }),
      ),
    );
    renderDashboard();

    expect(
      await screen.findByText(/digest unavailable right now/i),
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/nothing new since your last visit/i),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/tracking starts now/i)).not.toBeInTheDocument();
  });

  it("shows a muted row when the digest request itself fails", async () => {
    server.use(
      http.get(`${API}/api/digest/:type/:id`, () =>
        HttpResponse.json({ error: "boom" }, { status: 500 }),
      ),
    );
    renderDashboard();

    expect(
      await screen.findByText(/couldn't load your visit digest/i),
    ).toBeInTheDocument();
  });
});
