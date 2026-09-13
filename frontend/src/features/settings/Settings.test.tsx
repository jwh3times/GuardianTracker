import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router";
import { http, HttpResponse } from "msw";
import { API, server } from "../../test/testServer";
import { renderWithProviders } from "../../test/renderWithProviders";
import { Settings } from "./Settings";

beforeEach(() => {
  localStorage.clear();
});

describe("Settings page", () => {
  function renderSettings(route = "/settings") {
    return renderWithProviders(
      <Routes>
        <Route path="/settings" element={<Settings />} />
        <Route path="/admin" element={<div>admin-stub</div>} />
        <Route path="/login" element={<div>login-stub</div>} />
      </Routes>,
      { route },
    );
  }

  it("renders Destiny membership info and the empty-characters state", async () => {
    renderSettings();
    expect(await screen.findByText("Settings")).toBeInTheDocument();
    expect(screen.getByText("Destiny membership")).toBeInTheDocument();
    expect(screen.getByText("Display name")).toBeInTheDocument();
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

  it("puts a preference back and says so when saving fails", async () => {
    server.use(
      http.put(`${API}/api/preferences`, () =>
        HttpResponse.json(
          { error: "database unavailable", code: "DB_UNAVAILABLE" },
          { status: 503 },
        ),
      ),
    );
    renderSettings();
    await screen.findByText("Settings");
    const compact = screen.getByRole("radio", { name: "Compact" });

    fireEvent.click(compact);

    expect(
      await screen.findByText(/couldn't save that preference/i),
    ).toBeInTheDocument();
    // The control shows what the server has, not what was clicked.
    expect(compact).toHaveAttribute("aria-checked", "false");
    expect(screen.getByRole("radio", { name: "Framed" })).toHaveAttribute(
      "aria-checked",
      "true",
    );
  });

  it("does not repeat an earlier save failure when Settings is opened again", async () => {
    server.use(
      http.put(`${API}/api/preferences`, () =>
        HttpResponse.json(
          { error: "database unavailable", code: "DB_UNAVAILABLE" },
          { status: 503 },
        ),
      ),
    );
    function Reopenable() {
      const [open, setOpen] = React.useState(true);
      return (
        <>
          <button onClick={() => setOpen((value) => !value)}>
            toggle settings
          </button>
          {open && <Settings />}
        </>
      );
    }
    renderWithProviders(<Reopenable />, { route: "/settings" });
    await screen.findByText("Settings");
    fireEvent.click(screen.getByRole("radio", { name: "Compact" }));
    expect(
      await screen.findAllByText(/couldn't save that preference/i),
    ).toHaveLength(1);

    fireEvent.click(screen.getByText("toggle settings"));
    fireEvent.click(screen.getByText("toggle settings"));
    await screen.findByText("Settings");
    await new Promise((resolve) => setTimeout(resolve, 50));

    // The shared client still holds that failure; reopening must not re-announce it.
    expect(screen.getAllByText(/couldn't save that preference/i)).toHaveLength(
      1,
    );
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

  it("locks the tier picker while an opt-in is in flight", async () => {
    server.use(
      http.get(`${API}/api/flags`, () =>
        HttpResponse.json({ role: "standard", flags: [] }),
      ),
      // Never resolves: the opt-in is still in flight.
      http.put(`${API}/api/account/role`, () => new Promise(() => {})),
    );
    renderSettings();
    await screen.findByText("Membership & access");
    const beta = await screen.findByRole("radio", { name: /Beta/ });
    await waitFor(() => expect(beta).toBeEnabled());

    fireEvent.click(beta);

    // A second pick before the first lands would race it on the server.
    await waitFor(() =>
      expect(screen.getByRole("radio", { name: /Alpha/ })).toBeDisabled(),
    );
  });

  it("says why an opt-in was refused", async () => {
    server.use(
      http.get(`${API}/api/flags`, () =>
        HttpResponse.json({ role: "standard", flags: [] }),
      ),
      http.put(`${API}/api/account/role`, () =>
        HttpResponse.json(
          { error: "Early access is closed right now" },
          { status: 403 },
        ),
      ),
    );
    renderSettings();
    await screen.findByText("Membership & access");
    const beta = await screen.findByRole("radio", { name: /Beta/ });
    await waitFor(() => expect(beta).toBeEnabled());

    fireEvent.click(beta);

    expect(
      await screen.findByText("Early access is closed right now"),
    ).toBeInTheDocument();
  });

  it("opts into an early-access tier", async () => {
    // A standard-tier user can self-select Beta; the picker is interactive
    // (it's disabled only for admins, which the default flags handler returns).
    let flagGets = 0;
    server.use(
      http.get(`${API}/api/flags`, () => {
        flagGets += 1;
        return HttpResponse.json({ role: "standard", flags: [] });
      }),
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
    await waitFor(() => expect(flagGets).toBe(1));
    fireEvent.click(screen.getByRole("radio", { name: /Beta/ }));
    await waitFor(() => expect(optInBody).toEqual({ role: "beta" }));
    // The new tier must reach gated navigation, so flags are asked for again —
    // once. This page used to both refetch and invalidate, firing it twice.
    await waitFor(() => expect(flagGets).toBe(2));
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(flagGets).toBe(2);
  });
});
