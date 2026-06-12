import React from "react";
import { describe, it, expect, beforeAll, afterAll, beforeEach, afterEach } from "vitest";
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
    </QueryClientProvider>
  );
}

describe("Collections", () => {
  it("renders missing items from the API", async () => {
    renderPage(<Collections />);
    expect(await screen.findByText("Fatebringer")).toBeInTheDocument();
  });

  // B2 regression: the add-to-wishlist POST must send { itemHash: <number> },
  // not the { itemId: <string> } shape that 400'd before the fix.
  it("posts the correct wishlist payload shape", async () => {
    let captured: unknown = null;
    server.use(
      http.get(`${API}/api/wishlist`, () => HttpResponse.json([])),
      http.post(`${API}/api/wishlist`, async ({ request }) => {
        captured = await request.json();
        return HttpResponse.json(
          {
            id: "10",
            itemHash: 12345,
            name: "Fatebringer",
            itemType: "Hand Cannon",
            rarity: "Legendary",
            icon: "",
            priority: "MEDIUM",
            notes: "",
            sources: [],
            availableNow: false,
            dateAdded: new Date().toISOString(),
          },
          { status: 201 }
        );
      })
    );

    renderPage(<Collections />);
    await screen.findByText("Fatebringer");

    fireEvent.click(screen.getAllByTitle("Add to wishlist")[0]);

    await waitFor(() => expect(captured).not.toBeNull());
    expect(captured).toEqual({ itemHash: 12345 });
  });

  it("shows the privacy error state on PRIVACY_RESTRICTION", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(
          { error: "User has their Destiny 2 profile set to private", code: "PRIVACY_RESTRICTION" },
          { status: 403 }
        )
      )
    );
    renderPage(<Collections />);
    expect(
      await screen.findByText("Your Destiny profile is private")
    ).toBeInTheDocument();
  });
});

describe("Dashboard", () => {
  it("renders real collection totals and honest wishlist availability", async () => {
    renderPage(<Dashboard />);
    expect(await screen.findByText(/Welcome back, TestGuardian/)).toBeInTheDocument();
    // 8 of 10 weapons collected in the fixture
    expect(await screen.findByText("8/10")).toBeInTheDocument();
    // One wishlist item, available now (fixture)
    expect(await screen.findByText(/of 1 wishlist items available now/)).toBeInTheDocument();
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
      })
    );

    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <React.StrictMode>
        <QueryClientProvider client={qc}>
          <AuthProvider>
            <MemoryRouter initialEntries={["/auth/callback?code=onetime&state=sig"]}>
              <Routes>
                <Route path="/auth/callback" element={<OAuthCallback />} />
                <Route path="/dashboard" element={<div>dashboard-stub</div>} />
              </Routes>
            </MemoryRouter>
          </AuthProvider>
        </QueryClientProvider>
      </React.StrictMode>
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
      </QueryClientProvider>
    );
    expect(await screen.findByText("Authentication error")).toBeInTheDocument();
  });
});
