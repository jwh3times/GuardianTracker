import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "../test/testServer";
import { queryClient } from "./api";
import App from "../App";
import { Admin } from "../features/admin/Admin";
import { FlagsProvider, useFlag } from "../contexts/FlagsContext";
import { LockedFeature } from "../features/admin/AdminKit";
import { AuthProvider } from "../contexts/AuthContext";
import { ToastProvider } from "../components/Toast";

const authed = () => {
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_refresh_token", "test-refresh");
  localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
  // Not an onboarding test — mark first-run done so the Dashboard greeting
};

beforeEach(() => {
  localStorage.clear();
  // App uses the singleton queryClient; clear it so a cached ["flags"] role from
  // a prior test doesn't leak into the next.
  queryClient.clear();
  authed();
});

function wrap(ui: React.ReactNode) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <AuthProvider>
        <ToastProvider>
          <MemoryRouter>
            <FlagsProvider>{ui}</FlagsProvider>
          </MemoryRouter>
        </ToastProvider>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

function FlagProbe({ flagKey }: { flagKey: string }) {
  const st = useFlag(flagKey);
  return <div>{`${flagKey}=${st.enabled}/${st.accessible}/${st.locked}`}</div>;
}

describe("useFlag", () => {
  it("resolves an accessible flag and fails open for unknown keys", async () => {
    wrap(
      <>
        <FlagProbe flagKey="catalysts-crafting" />
        <FlagProbe flagKey="totally-unknown" />
      </>,
    );
    expect(
      await screen.findByText("catalysts-crafting=true/true/false"),
    ).toBeInTheDocument();
    // Unknown key fails open: enabled + accessible, never locked.
    expect(
      screen.getByText("totally-unknown=true/true/false"),
    ).toBeInTheDocument();
  });

  it("reports locked and hidden states from the server", async () => {
    server.use(
      http.get(`${API}/api/flags`, () =>
        HttpResponse.json({
          role: "standard",
          flags: [
            {
              key: "gated",
              name: "Gated",
              desc: "",
              category: "",
              minTier: "alpha",
              enabled: true,
              accessible: false,
              locked: true,
            },
            {
              key: "off",
              name: "Off",
              desc: "",
              category: "",
              minTier: "beta",
              enabled: false,
              accessible: false,
              locked: false,
            },
          ],
        }),
      ),
    );
    wrap(
      <>
        <FlagProbe flagKey="gated" />
        <FlagProbe flagKey="off" />
      </>,
    );
    expect(
      await screen.findByText("gated=true/false/true"),
    ).toBeInTheDocument();
    expect(screen.getByText("off=false/false/false")).toBeInTheDocument();
  });
});

describe("LockedFeature", () => {
  it("renders the upsell and fires the actions", () => {
    let changed = false;
    let back = false;
    render(
      <LockedFeature
        flag={{
          key: "god-roll",
          name: "God-roll insights",
          desc: "Per-weapon owned rolls.",
          category: "Power",
          minTier: "alpha",
          enabled: true,
          accessible: false,
          locked: true,
        }}
        onChangeTier={() => (changed = true)}
        onBack={() => (back = true)}
      />,
    );
    expect(screen.getByText("God-roll insights")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Change access tier"));
    fireEvent.click(screen.getByText("Back to Dashboard"));
    expect(changed).toBe(true);
    expect(back).toBe(true);
  });
});

describe("Admin console", () => {
  it("lists members and toggles a feature flag", async () => {
    let flagPutHit = false;
    server.use(
      http.put(`${API}/api/admin/flags/:key`, ({ params }) => {
        flagPutHit = true;
        return HttpResponse.json({
          key: params.key as string,
          name: "Catalyst & Crafting tracker",
          description: "",
          category: "Completion",
          minTier: "standard",
          enabled: false,
          updatedAt: new Date().toISOString(),
        });
      }),
    );
    wrap(<Admin />);

    expect(await screen.findByText("Admin Console")).toBeInTheDocument();
    // Users tab: the seeded user rows render.
    expect(await screen.findByText("Vega")).toBeInTheDocument();

    // Switch to the Feature Flags tab and toggle the first flag's switch.
    fireEvent.click(screen.getByText("Feature Flags"));
    const toggle = await screen.findByRole("switch", {
      name: /Enable Catalyst & Crafting tracker/i,
    });
    fireEvent.click(toggle);
    await waitFor(() => expect(flagPutHit).toBe(true));
  });
});

describe("admin route guard", () => {
  it("redirects a non-admin away from /admin", async () => {
    server.use(
      http.get(`${API}/api/flags`, () =>
        HttpResponse.json({ role: "standard", flags: [] }),
      ),
    );
    render(
      <MemoryRouter initialEntries={["/admin"]}>
        <App />
      </MemoryRouter>,
    );
    // Falls back to the dashboard hero rather than the admin console.
    expect(
      await screen.findByText(/Welcome, TestGuardian/),
    ).toBeInTheDocument();
    expect(screen.queryByText("Admin Console")).not.toBeInTheDocument();
  });

  it("renders the console for an admin at /admin", async () => {
    render(
      <MemoryRouter initialEntries={["/admin"]}>
        <App />
      </MemoryRouter>,
    );
    // The lazy-loaded console heading (h1) — distinct from the sidebar nav item.
    expect(
      await screen.findByRole(
        "heading",
        { name: /Admin Console/, level: 1 },
        { timeout: 3000 },
      ),
    ).toBeInTheDocument();
  });
});
