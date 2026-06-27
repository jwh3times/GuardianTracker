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
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { PreferencesProvider } from "../../contexts/PreferencesContext";
import { ToastProvider } from "../../components/Toast";
import { Admin } from "./Admin";

// Harness copied from pages-manage.test.tsx (Settings/WishList tests).
// Admin uses useAuth (needs AuthProvider), useToast (needs ToastProvider),
// and React Query hooks (needs QueryClientProvider + MemoryRouter for any
// internal link rendering).

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  localStorage.clear();
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_refresh_token", "test-refresh");
  localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
});

function renderAdmin() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <AuthProvider>
        <PreferencesProvider>
          <ToastProvider>
            <MemoryRouter>
              <Admin />
            </MemoryRouter>
          </ToastProvider>
        </PreferencesProvider>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

describe("Admin Audit panel", () => {
  // Add a default audit handler for this describe block. The server's
  // defaultHandlers don't include /api/admin/audit, so we register it
  // before each test and let afterEach → server.resetHandlers() clean up.
  beforeEach(() => {
    server.use(
      http.get(`${API}/api/admin/audit`, ({ request }) => {
        const url = new URL(request.url);
        const type = url.searchParams.get("type") ?? "";
        return HttpResponse.json({
          entries: [
            {
              id: "1",
              eventType:
                type === "flag.update" ? "flag.update" : "login.success",
              outcome: "success",
              actor: { membershipId: "mid-1", displayName: "Tester" },
              details: {},
              createdAt: new Date().toISOString(),
              ip: "203.0.113.7",
            },
          ],
          nextCursor: "",
        });
      }),
    );
  });

  it("renders audit events after switching to the Audit Log tab", async () => {
    renderAdmin();
    // Click the "Audit Log" subtab — triggers the audit query (enabled: tab === "audit")
    fireEvent.click(screen.getByText(/Audit Log/i));
    // AuditTable maps "login.success" → "Login"
    await waitFor(() => expect(screen.getByText("Login")).toBeInTheDocument());
    // IP is rendered in the details cell as "ip: 203.0.113.7"
    expect(screen.getByText(/203\.0\.113\.7/)).toBeInTheDocument();
    expect(screen.getByText("Tester")).toBeInTheDocument();
  });

  it("Logouts chip requests the logout. family and labels both events", async () => {
    let requestedType: string | null = null;
    server.use(
      http.get(`${API}/api/admin/audit`, ({ request }) => {
        const type = new URL(request.url).searchParams.get("type") ?? "";
        if (type !== "logout.") {
          return HttpResponse.json({ entries: [], nextCursor: "" });
        }
        requestedType = type;
        return HttpResponse.json({
          entries: [
            {
              id: "10",
              eventType: "logout.session",
              outcome: "success",
              actor: { membershipId: "mid-1", displayName: "Tester" },
              details: {},
              createdAt: new Date().toISOString(),
            },
            {
              id: "11",
              eventType: "logout.all",
              outcome: "success",
              actor: { membershipId: "mid-1", displayName: "Tester" },
              details: {},
              createdAt: new Date().toISOString(),
            },
          ],
          nextCursor: "",
        });
      }),
    );

    renderAdmin();
    fireEvent.click(screen.getByText(/Audit Log/i));
    fireEvent.click(await screen.findByText("Logouts"));

    // The single-device event (logout.session) labels as "Logout"; the all-devices
    // event labels as "Logout (all devices)". Both must appear → the prefix matched.
    await waitFor(() =>
      expect(screen.getByText("Logout (all devices)")).toBeInTheDocument(),
    );
    expect(screen.getByText("Logout")).toBeInTheDocument();
    expect(requestedType).toBe("logout.");
  });

  it("refetches when the Flags filter chip is clicked", async () => {
    renderAdmin();
    fireEvent.click(screen.getByText(/Audit Log/i));
    // Wait for the initial "Login" row from login.success
    await screen.findByText("Login");
    // Click the "Flags" filter chip — sets auditType to "flag.update"
    fireEvent.click(screen.getByText("Flags"));
    // AuditTable maps "flag.update" → "Flag update"
    await waitFor(() =>
      expect(screen.getByText("Flag update")).toBeInTheDocument(),
    );
  });
});
