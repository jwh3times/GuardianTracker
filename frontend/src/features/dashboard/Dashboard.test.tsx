import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { CharacterProvider } from "../../contexts/CharacterContext";
import { PreferencesProvider } from "../../contexts/PreferencesContext";
import { ToastProvider } from "../../components/Toast";
import { Dashboard } from "./Dashboard";

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

  it("greets the current Guardian", async () => {
    renderDashboard();
    expect(await screen.findByText(/^Welcome, /)).toBeInTheDocument();
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
