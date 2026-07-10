import React from "react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "../test/testServer";
import App from "../App";
import { queryClient } from "../lib/api";
import { AppShell } from "./AppShell";
import { AuthProvider } from "../contexts/AuthContext";
import { CharacterProvider } from "../contexts/CharacterContext";
import { PreferencesProvider } from "../contexts/PreferencesContext";
import { FlagsProvider } from "../contexts/FlagsContext";
import { ToastProvider } from "./Toast";
import { ErrorBoundary } from "./ErrorBoundary";
import { LoadingSpinner } from "./LoadingSpinner";

const authed = () => {
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_refresh_token", "test-refresh");
  localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
  // Not an onboarding test — mark first-run done so the Dashboard greeting
  // reads "Welcome back" as these assertions expect.
  localStorage.setItem(
    `guardian_first_run_done:${sampleUser.membershipId}`,
    "2026-01-01",
  );
};

describe("App routing", () => {
  beforeEach(() => localStorage.clear());

  it("redirects an unauthenticated visitor to the login page", async () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <App />
      </MemoryRouter>,
    );
    expect(await screen.findByText("Sign in with Bungie")).toBeInTheDocument();
  });

  it("renders the protected shell and dashboard when authenticated", async () => {
    authed();
    render(
      <MemoryRouter initialEntries={["/dashboard"]}>
        <App />
      </MemoryRouter>,
    );
    // AppShell sidebar nav renders alongside the dashboard content
    expect(
      await screen.findByText(/Welcome back, TestGuardian/),
    ).toBeInTheDocument();
    expect(screen.getAllByText("Collections").length).toBeGreaterThan(0);
  });

  it("bounces an authenticated user away from /login to the dashboard", async () => {
    authed();
    render(
      <MemoryRouter initialEntries={["/login"]}>
        <App />
      </MemoryRouter>,
    );
    expect(
      await screen.findByText(/Welcome back, TestGuardian/),
    ).toBeInTheDocument();
  });
});

describe("AppShell interactions", () => {
  beforeEach(() => {
    localStorage.clear();
    authed();
  });

  function renderShell(route = "/dashboard") {
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    return render(
      <QueryClientProvider client={qc}>
        <AuthProvider>
          <PreferencesProvider>
            <MemoryRouter initialEntries={[route]}>
              <FlagsProvider>
                <CharacterProvider>
                  <ToastProvider>
                    <Routes>
                      <Route
                        path="/dashboard"
                        element={
                          <AppShell>
                            <div>shell-child</div>
                          </AppShell>
                        }
                      />
                      <Route path="/login" element={<div>login-stub</div>} />
                      <Route
                        path="/collections"
                        element={<div>collections-stub</div>}
                      />
                    </Routes>
                  </ToastProvider>
                </CharacterProvider>
              </FlagsProvider>
            </MemoryRouter>
          </PreferencesProvider>
        </AuthProvider>
      </QueryClientProvider>,
    );
  }

  it("renders the shell content and nav", async () => {
    renderShell();
    expect(screen.getByText("shell-child")).toBeInTheDocument();
    expect(screen.getByText("Triumphs & Seals")).toBeInTheDocument();
  });

  it("opens and closes the mobile nav drawer", async () => {
    const { container } = renderShell();
    // Sidebar has one "Sign out"; opening the drawer duplicates the nav → two.
    expect(screen.getAllByText("Sign out")).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: "Menu" }));
    expect(screen.getAllByText("Sign out")).toHaveLength(2);
    // Clicking the scrim closes the drawer
    fireEvent.click(container.querySelector(".gt-mobnav-scrim")!);
    await waitFor(() =>
      expect(screen.getAllByText("Sign out")).toHaveLength(1),
    );
  });

  it("shows the character switcher menu and switches guardians", async () => {
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([
          {
            characterId: "char-hunter",
            classType: 1,
            className: "Hunter",
            raceName: "Awoken",
            light: 2010,
            emblemPath: "",
            emblemBackgroundPath: "",
            dateLastPlayed: new Date().toISOString(),
          },
          {
            characterId: "char-warlock",
            classType: 2,
            className: "Warlock",
            raceName: "Human",
            light: 2005,
            emblemPath: "",
            emblemBackgroundPath: "",
            dateLastPlayed: new Date().toISOString(),
          },
        ]),
      ),
    );
    renderShell();
    // Switcher button shows the display name once characters load
    const switcher = await screen.findByRole("button", {
      name: /TestGuardian/,
    });
    fireEvent.click(switcher);
    expect(await screen.findByText("Switch Guardian")).toBeInTheDocument();
    fireEvent.click(screen.getByText(/Warlock/));
    // Menu closes after a pick
    await waitFor(() =>
      expect(screen.queryByText("Switch Guardian")).not.toBeInTheDocument(),
    );
  });

  it("runs a global search and navigates to the picked item", async () => {
    server.use(
      http.get(`${API}/api/items/search`, () =>
        HttpResponse.json([
          {
            hash: 555,
            name: "Gjallarhorn",
            icon: "/img/gjally.png",
            type: "Rocket Launcher",
            rarity: "Exotic",
          },
        ]),
      ),
    );
    const { container } = renderShell();
    const input = screen.getByPlaceholderText("Search items…");
    fireEvent.change(input, { target: { value: "gjall" } });
    // Debounced 250ms query → result appears
    expect(
      await screen.findByText("Gjallarhorn", {}, { timeout: 2000 }),
    ).toBeInTheDocument();
    // Result tile renders the Bungie CDN icon, not just the type glyph
    const icon = container.querySelector("img.gt-tile-img") as HTMLImageElement;
    expect(icon).not.toBeNull();
    expect(icon.src).toBe("https://www.bungie.net/img/gjally.png");
    fireEvent.click(screen.getByText("Gjallarhorn"));
    expect(await screen.findByText("collections-stub")).toBeInTheDocument();
  });

  it("shows the no-match state for an empty search", async () => {
    server.use(
      http.get(`${API}/api/items/search`, () => HttpResponse.json([])),
    );
    renderShell();
    const input = screen.getByPlaceholderText("Search items…");
    fireEvent.change(input, { target: { value: "zzz" } });
    expect(
      await screen.findByText(/No items match/, {}, { timeout: 2000 }),
    ).toBeInTheDocument();
  });

  it("labels the icon-only controls", () => {
    renderShell();
    expect(
      screen.getByRole("searchbox", { name: /search items/i }),
    ).toBeInTheDocument();
  });

  it("signs out from the sidebar", async () => {
    renderShell();
    fireEvent.click(screen.getAllByText("Sign out")[0]);
    expect(await screen.findByText("login-stub")).toBeInTheDocument();
    expect(localStorage.getItem("guardian_token")).toBeNull();
  });

  it("redirects to the dashboard when weekly-planner is disabled", async () => {
    queryClient.clear();
    server.use(
      http.get(`${API}/api/flags`, () =>
        HttpResponse.json({
          role: "admin",
          flags: [
            {
              key: "weekly-planner",
              name: "Weekly Planner",
              desc: "Weekly reset planning and recommendations.",
              category: "Planning",
              minTier: "standard",
              enabled: false,
              accessible: false,
              locked: false,
            },
          ],
        }),
      ),
    );
    render(
      <MemoryRouter initialEntries={["/this-week"]}>
        <App />
      </MemoryRouter>,
    );
    expect(
      await screen.findByText(/Welcome back, TestGuardian/),
    ).toBeInTheDocument();
  });

  it("renders the locked upsell for /this-week when weekly-planner is tier-locked", async () => {
    queryClient.clear();
    server.use(
      http.get(`${API}/api/flags`, () =>
        HttpResponse.json({
          role: "standard",
          flags: [
            {
              key: "weekly-planner",
              name: "Weekly Planner",
              desc: "Weekly reset planning and recommendations.",
              category: "Planning",
              minTier: "alpha",
              enabled: true,
              accessible: false,
              locked: true,
            },
          ],
        }),
      ),
    );
    render(
      <MemoryRouter initialEntries={["/this-week"]}>
        <App />
      </MemoryRouter>,
    );
    expect(await screen.findByText("Change access tier")).toBeInTheDocument();
    expect(screen.getByText("Weekly Planner")).toBeInTheDocument();
  });

  it("redirects to the dashboard when cosmetics is disabled", async () => {
    queryClient.clear();
    server.use(
      http.get(`${API}/api/flags`, () =>
        HttpResponse.json({
          role: "admin",
          flags: [
            {
              key: "cosmetics",
              name: "Cosmetics",
              desc: "Browsable cosmetics gallery.",
              category: "Collection",
              minTier: "standard",
              enabled: false,
              accessible: false,
              locked: false,
            },
          ],
        }),
      ),
    );
    render(
      <MemoryRouter initialEntries={["/cosmetics"]}>
        <App />
      </MemoryRouter>,
    );
    expect(
      await screen.findByText(/Welcome back, TestGuardian/),
    ).toBeInTheDocument();
  });
});

function Boom(): React.ReactElement {
  throw new Error("kaboom");
}

describe("ErrorBoundary", () => {
  it("renders the gt fallback and shows the dev error message", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    render(
      <ErrorBoundary>
        <Boom />
      </ErrorBoundary>,
    );
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(
      screen.getByText(
        "An unexpected error occurred. Reload the page to continue.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Error: kaboom")).toBeInTheDocument();
    spy.mockRestore();
  });

  it("renders a custom fallback when provided", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    render(
      <ErrorBoundary fallback={<div>custom-fallback</div>}>
        <Boom />
      </ErrorBoundary>,
    );
    expect(screen.getByText("custom-fallback")).toBeInTheDocument();
    spy.mockRestore();
  });

  it("renders children when nothing throws", () => {
    render(
      <ErrorBoundary>
        <div>safe-child</div>
      </ErrorBoundary>,
    );
    expect(screen.getByText("safe-child")).toBeInTheDocument();
  });

  it("wires the reload action", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    const reload = vi.fn();
    const original = window.location;
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...original, reload },
    });
    render(
      <ErrorBoundary>
        <Boom />
      </ErrorBoundary>,
    );
    fireEvent.click(screen.getByText("Reload"));
    expect(reload).toHaveBeenCalled();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: original,
    });
    spy.mockRestore();
  });
});

describe("LoadingSpinner", () => {
  it("applies the size class and merges a custom className", () => {
    const { container, rerender } = render(
      <LoadingSpinner size="sm" className="extra" />,
    );
    expect(container.firstChild).toHaveClass("h-4", "w-4", "extra");
    rerender(<LoadingSpinner size="lg" />);
    expect(container.firstChild).toHaveClass("h-12", "w-12");
    rerender(<LoadingSpinner />);
    expect(container.firstChild).toHaveClass("h-8", "w-8");
  });
});
