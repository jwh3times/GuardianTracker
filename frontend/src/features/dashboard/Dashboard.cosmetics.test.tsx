import React from "react";
import {
  describe,
  it,
  expect,
  beforeAll,
  afterAll,
  afterEach,
  beforeEach,
} from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
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

function renderDashboard() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <AuthProvider>
        <PreferencesProvider>
          <CharacterProvider>
            <ToastProvider>
              <MemoryRouter initialEntries={["/dashboard"]}>
                <Routes>
                  <Route path="/dashboard" element={<Dashboard />} />
                  <Route
                    path="/cosmetics"
                    element={<div>COSMETICS PAGE</div>}
                  />
                </Routes>
              </MemoryRouter>
            </ToastProvider>
          </CharacterProvider>
        </PreferencesProvider>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

describe("Dashboard cosmetics deep-link", () => {
  it("navigates to /cosmetics when the Cosmetics hero bar is clicked", async () => {
    renderDashboard();
    fireEvent.click(await screen.findByRole("button", { name: /cosmetics/i }));
    expect(await screen.findByText("COSMETICS PAGE")).toBeInTheDocument();
  });
});
