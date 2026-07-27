import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { PreferencesProvider } from "../../contexts/PreferencesContext";
import { FlagsProvider } from "../../contexts/FlagsContext";
import { ToastProvider } from "../../components/Toast";
import { Settings } from "./Settings";

beforeEach(() => {
  localStorage.clear();
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
});

describe("Settings page", () => {
  function renderSettings(route = "/settings") {
    const qc = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
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
      </QueryClientProvider>,
    );
  }

  it("renders account info and the empty-characters state", async () => {
    renderSettings();
    expect(await screen.findByText("Settings")).toBeInTheDocument();
    expect(screen.getByText("TestGuardian")).toBeInTheDocument();
    // sampleUser.platform === "steam"
    expect(screen.getByText("steam")).toBeInTheDocument();
    expect(
      await screen.findByText("No characters loaded yet."),
    ).toBeInTheDocument();
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
        ]),
      ),
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
      }),
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
      }),
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
      }),
    );
    renderSettings();
    await screen.findByText("Settings");
    fireEvent.click(
      screen.getByRole("button", { name: "Sign out all devices" }),
    );
    expect(await screen.findByText("login-stub")).toBeInTheDocument();
    expect(localStorage.getItem("guardian_token")).toBeNull();
    await waitFor(() => expect(calledAll).toBe(true));
  });

  it("opts into an early-access tier", async () => {
    // A standard-tier user can self-select Beta; the picker is interactive
    // (it's disabled only for admins, which the default flags handler returns).
    server.use(
      http.get(`${API}/api/flags`, () =>
        HttpResponse.json({ role: "standard", flags: [] }),
      ),
    );
    let optInBody: unknown = null;
    server.use(
      http.put(`${API}/api/account/role`, async ({ request }) => {
        optInBody = await request.json();
        return HttpResponse.json({ role: "beta" });
      }),
    );
    renderSettings();
    await screen.findByText("Membership & access");
    fireEvent.click(screen.getByRole("radio", { name: /Beta/ }));
    await waitFor(() => expect(optInBody).toEqual({ role: "beta" }));
  });
});
