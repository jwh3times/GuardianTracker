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
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { sampleUser, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { CharacterProvider } from "../../contexts/CharacterContext";
import { PreferencesProvider } from "../../contexts/PreferencesContext";
import { ToastProvider } from "../../components/Toast";
import { Dashboard } from "./Dashboard";

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

describe("Dashboard page", () => {
  function renderDashboard() {
    return renderPage(<Dashboard />);
  }

  it("shows the top recommended action as the best thing to do today", async () => {
    renderDashboard();
    expect(await screen.findByText("Run Vault of Glass")).toBeInTheDocument();
    expect(screen.getByText(/Best thing to do today/i)).toBeInTheDocument();
  });
});
