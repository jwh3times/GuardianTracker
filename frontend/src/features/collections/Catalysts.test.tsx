import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse, delay } from "msw";
import { API, sampleUser, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { ToastProvider } from "../../components/Toast";
import { Catalysts } from "./Catalysts";

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

  it("shows a complete indicator (not 'Not yet acquired') for a completed catalyst, per backend contract obj:null on complete", async () => {
    server.use(
      http.get(`${API}/api/catalysts/:type/:id`, () =>
        HttpResponse.json({
          items: [
            {
              id: "c-complete",
              name: "Complete Catalyst",
              type: "Bow",
              icon: "",
              status: "complete",
              obj: null,
              source: "Strikes",
            },
            {
              id: "c-missing",
              name: "Missing Catalyst",
              type: "Auto Rifle",
              icon: "",
              status: "missing",
              obj: null,
              source: "World drops",
            },
            {
              id: "c-progress",
              name: "Progress Catalyst",
              type: "Hand Cannon",
              icon: "",
              status: "in-progress",
              obj: { label: "Kills", cur: 10, max: 100 },
              source: "Crucible",
            },
          ],
          fetchedAt: "",
        }),
      ),
    );
    const { container } = renderPage(<Catalysts />);

    // Complete catalyst: shows a completion indicator, never "Not yet acquired"
    await screen.findByText("Complete Catalyst");
    expect(screen.getByText("Catalyst complete")).toBeInTheDocument();
    expect(
      container.querySelector('.gt-prog-fill[data-complete="true"]'),
    ).not.toBeNull();

    // Missing catalyst: still "Not yet acquired", and only one such element
    expect(screen.getAllByText("Not yet acquired")).toHaveLength(1);

    // In-progress catalyst: still renders its progress bar with real values
    expect(screen.getByText("10/100")).toBeInTheDocument();
  });

  it("shows the catalyst effect text when present, and nothing extra when empty", async () => {
    server.use(
      http.get(`${API}/api/catalysts/:type/:id`, () =>
        HttpResponse.json({
          items: [
            {
              id: "c-effect",
              name: "Effect Catalyst",
              type: "Hand Cannon",
              icon: "",
              status: "missing",
              obj: null,
              source: "World drops",
              effect: "Kills with this weapon create a burst of solar light.",
            },
            {
              id: "c-noeffect",
              name: "No Effect Catalyst",
              type: "Bow",
              icon: "",
              status: "missing",
              obj: null,
              source: "World drops",
              effect: "",
            },
          ],
          fetchedAt: "",
        }),
      ),
    );
    const { container } = renderPage(<Catalysts />);
    expect(await screen.findByText("Effect Catalyst")).toBeInTheDocument();
    expect(
      screen.getByText("Kills with this weapon create a burst of solar light."),
    ).toBeInTheDocument();
    expect(screen.getByText("No Effect Catalyst")).toBeInTheDocument();
    // Exactly one card renders effect copy — the empty-effect card adds nothing extra.
    expect(container.querySelectorAll(".gt-catalyst-effect")).toHaveLength(1);
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
              obj: null,
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
