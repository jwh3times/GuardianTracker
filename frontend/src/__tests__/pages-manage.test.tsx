import React from "react";
import { describe, it, expect, beforeAll, afterAll, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, sampleWishlist, server } from "./testServer";
import { AuthProvider } from "../contexts/AuthContext";
import { PreferencesProvider } from "../contexts/PreferencesContext";
import { FlagsProvider } from "../contexts/FlagsContext";
import { ToastProvider } from "../components/ui/Toast";
import { WishList } from "../pages/WishList";
import { Settings } from "../pages/Settings";
import { Login } from "../pages/Login";

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
          <ToastProvider>
            <MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter>
          </ToastProvider>
        </PreferencesProvider>
      </AuthProvider>
    </QueryClientProvider>
  );
}

describe("WishList page", () => {
  it("renders items with available-now and source-only states", async () => {
    server.use(
      http.get(`${API}/api/wishlist`, () =>
        HttpResponse.json([
          ...sampleWishlist,
          {
            id: "2",
            itemHash: 88888,
            name: "Vex Mythoclast",
            itemType: "Fusion Rifle",
            rarity: "Exotic",
            icon: "",
            priority: "LOW",
            notes: "from VoG",
            sources: ["Vault of Glass"],
            availableNow: false,
            dateAdded: new Date().toISOString(),
          },
        ])
      )
    );
    renderPage(<WishList />);
    expect(await screen.findByText("Gjallarhorn")).toBeInTheDocument();
    // available-now item shows the where label; source-only item shows "Source:"
    expect(screen.getByText("Vex Mythoclast")).toBeInTheDocument();
    expect(screen.getByText(/Source:/)).toBeInTheDocument();
    expect(screen.getByText("2 items · 1 available now")).toBeInTheDocument();
    // notes render in quotes
    expect(screen.getByText('"from VoG"')).toBeInTheDocument();
  });

  it("shows the empty state when the wishlist has no items", async () => {
    server.use(http.get(`${API}/api/wishlist`, () => HttpResponse.json([])));
    renderPage(<WishList />);
    expect(await screen.findByText("Your wishlist is empty")).toBeInTheDocument();
  });

  it("renders the loading state before data arrives", async () => {
    const { container } = renderPage(<WishList />);
    // PageHead + spinner card render synchronously while the query is pending
    expect(container.querySelector(".gt-page")).not.toBeNull();
    expect(screen.queryByText(/available now/)).not.toBeInTheDocument();
  });

  it("filters by priority and reports per-priority counts", async () => {
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");
    // Gjallarhorn is HIGH priority; filtering to Low hides it
    fireEvent.click(screen.getByRole("button", { name: /Low/ }));
    expect(screen.queryByText("Gjallarhorn")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /All/ }));
    expect(screen.getByText("Gjallarhorn")).toBeInTheDocument();
  });

  it("removes an item optimistically", async () => {
    // Stateful handler so the post-delete refetch (onSettled) reflects the removal.
    let items = [...sampleWishlist];
    let deleted = false;
    server.use(
      http.get(`${API}/api/wishlist`, () => HttpResponse.json(items)),
      http.delete(`${API}/api/wishlist/:id`, ({ params }) => {
        items = items.filter((i) => i.id !== params.id);
        deleted = true;
        return new HttpResponse(null, { status: 204 });
      })
    );
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");
    fireEvent.click(screen.getByRole("button", { name: /Remove/ }));
    await waitFor(() => expect(screen.queryByText("Gjallarhorn")).not.toBeInTheDocument());
    expect(deleted).toBe(true);
  });

  it("rolls back and toasts when a delete fails", async () => {
    server.use(
      http.delete(`${API}/api/wishlist/:id`, () =>
        HttpResponse.json({ error: "boom" }, { status: 500 })
      )
    );
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");
    fireEvent.click(screen.getByRole("button", { name: /Remove/ }));
    expect(await screen.findByText("Failed to remove item")).toBeInTheDocument();
    // Optimistic removal rolled back → item returns
    expect(await screen.findByText("Gjallarhorn")).toBeInTheDocument();
  });

  it("edits notes and saves them", async () => {
    let putBody: unknown = null;
    server.use(
      http.put(`${API}/api/wishlist/:id`, async ({ request }) => {
        putBody = await request.json();
        return HttpResponse.json({ ...sampleWishlist[0], notes: "new note" });
      })
    );
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");

    fireEvent.click(screen.getByRole("button", { name: /Edit notes/ }));
    const textarea = screen.getByLabelText("Notes for Gjallarhorn");
    fireEvent.change(textarea, { target: { value: "new note" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(putBody).toEqual({ notes: "new note" }));
  });

  it("cancels a notes edit without saving", async () => {
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");
    fireEvent.click(screen.getByRole("button", { name: /Edit notes/ }));
    expect(screen.getByLabelText("Notes for Gjallarhorn")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.queryByLabelText("Notes for Gjallarhorn")).not.toBeInTheDocument();
  });

  it("changes priority through the row dropdown", async () => {
    let putBody: unknown = null;
    server.use(
      http.put(`${API}/api/wishlist/:id`, async ({ request }) => {
        putBody = await request.json();
        return HttpResponse.json({ ...sampleWishlist[0], priority: "URGENT" });
      })
    );
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");

    // Open the row's Priority dropdown. Its accessible name is exactly "High"
    // (the filter chip is "High 1", so an exact match disambiguates).
    fireEvent.click(screen.getByRole("button", { name: "High" }));
    fireEvent.click(screen.getByRole("button", { name: "Urgent" }));
    await waitFor(() => expect(putBody).toEqual({ priority: "URGENT" }));
  });
});

describe("Settings page", () => {
  function renderSettings(route = "/settings") {
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    return render(
      <QueryClientProvider client={qc}>
        <AuthProvider>
          <PreferencesProvider>
            <FlagsProvider>
              <ToastProvider>
                <MemoryRouter initialEntries={[route]}>
                  <Routes>
                    <Route path="/settings" element={<Settings />} />
                    <Route path="/admin" element={<div>admin-stub</div>} />
                    <Route path="/login" element={<div>login-stub</div>} />
                  </Routes>
                </MemoryRouter>
              </ToastProvider>
            </FlagsProvider>
          </PreferencesProvider>
        </AuthProvider>
      </QueryClientProvider>
    );
  }

  it("renders account info and the empty-characters state", async () => {
    renderSettings();
    expect(await screen.findByText("Settings")).toBeInTheDocument();
    expect(screen.getByText("TestGuardian")).toBeInTheDocument();
    // sampleUser.platform === "steam"
    expect(screen.getByText("steam")).toBeInTheDocument();
    expect(await screen.findByText("No characters loaded yet.")).toBeInTheDocument();
  });

  it("lists characters when the API returns them", async () => {
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([
          {
            characterId: "c1",
            classType: 1,
            className: "Hunter",
            raceName: "Awoken",
            light: 2010,
            emblemPath: "/e.png",
            emblemBackgroundPath: "/eb.png",
            dateLastPlayed: new Date().toISOString(),
          },
        ])
      )
    );
    renderSettings();
    expect(await screen.findByText(/Hunter/)).toBeInTheDocument();
    expect(screen.getByText("2010")).toBeInTheDocument();
  });

  it("toggles appearance preferences", async () => {
    let prefBody: unknown = null;
    server.use(
      http.put(`${API}/api/preferences`, async ({ request }) => {
        prefBody = await request.json();
        return HttpResponse.json({ cardStyle: "compact", personalize: true });
      })
    );
    renderSettings();
    await screen.findByText("Settings");
    fireEvent.click(screen.getByRole("radio", { name: "Compact" }));
    await waitFor(() => expect(prefBody).toEqual({ cardStyle: "compact" }));
  });

  it("triggers a data refresh", async () => {
    let refreshed = false;
    server.use(
      http.post(`${API}/api/collections/:type/:id/refresh`, () => {
        refreshed = true;
        return HttpResponse.json({ success: true, message: "ok" });
      })
    );
    renderSettings();
    await screen.findByText("Settings");
    fireEvent.click(screen.getByRole("button", { name: "Refresh data" }));
    await waitFor(() => expect(refreshed).toBe(true));
  });

  it("signs out and navigates to login", async () => {
    renderSettings();
    await screen.findByText("Settings");
    fireEvent.click(screen.getByRole("button", { name: "Sign out" }));
    expect(await screen.findByText("login-stub")).toBeInTheDocument();
    expect(localStorage.getItem("guardian_token")).toBeNull();
  });

  it("signs out of all devices and navigates to login", async () => {
    let calledAll = false;
    server.use(
      http.post(`${API}/api/auth/logout/all`, () => {
        calledAll = true;
        return HttpResponse.json({ message: "Signed out of all devices" });
      })
    );
    renderSettings();
    await screen.findByText("Settings");
    fireEvent.click(screen.getByRole("button", { name: "Sign out all devices" }));
    expect(await screen.findByText("login-stub")).toBeInTheDocument();
    expect(localStorage.getItem("guardian_token")).toBeNull();
    await waitFor(() => expect(calledAll).toBe(true));
  });

  it("opts into an early-access tier", async () => {
    // A standard-tier user can self-select Beta; the picker is interactive
    // (it's disabled only for admins, which the default flags handler returns).
    server.use(
      http.get(`${API}/api/flags`, () => HttpResponse.json({ role: "standard", flags: [] }))
    );
    let optInBody: unknown = null;
    server.use(
      http.put(`${API}/api/account/role`, async ({ request }) => {
        optInBody = await request.json();
        return HttpResponse.json({ role: "beta" });
      })
    );
    renderSettings();
    await screen.findByText("Membership & access");
    fireEvent.click(screen.getByRole("radio", { name: /Beta/ }));
    await waitFor(() => expect(optInBody).toEqual({ role: "beta" }));
  });
});

describe("Login page", () => {
  it("redirects to the Bungie auth URL on success", async () => {
    server.use(
      http.get(`${API}/api/auth/bungie`, () =>
        HttpResponse.json({ authUrl: "https://bungie.net/authorize", state: "xyz" })
      )
    );
    renderPage(<Login />);
    fireEvent.click(screen.getByRole("button", { name: /Sign in with Bungie/ }));
    // After success the button enters the redirecting (loading) state
    expect(await screen.findByText("Redirecting to Bungie.net…")).toBeInTheDocument();
  });

  it("shows an error when the auth request fails", async () => {
    server.use(
      http.get(`${API}/api/auth/bungie`, () => new HttpResponse(null, { status: 500 }))
    );
    renderPage(<Login />);
    fireEvent.click(screen.getByRole("button", { name: /Sign in with Bungie/ }));
    expect(
      await screen.findByText(/Failed to start authentication/)
    ).toBeInTheDocument();
  });

  it("shows an error when the response has no auth URL", async () => {
    server.use(
      http.get(`${API}/api/auth/bungie`, () => HttpResponse.json({ state: "xyz" }))
    );
    renderPage(<Login />);
    fireEvent.click(screen.getByRole("button", { name: /Sign in with Bungie/ }));
    expect(
      await screen.findByText(/No authorization URL received/)
    ).toBeInTheDocument();
  });
});
