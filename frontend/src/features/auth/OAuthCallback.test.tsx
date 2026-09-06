import { browserSessionClient } from "../../lib/browserSessionBrowser";
import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { OAuthCallback } from "./OAuthCallback";

beforeEach(() => {
  localStorage.clear();
  sessionStorage.clear();
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
});

describe("OAuthCallback", () => {
  // B13 regression: React StrictMode double-invokes effects in dev — the
  // single-use auth code must only be submitted once.
  it("submits the auth code exactly once under StrictMode", async () => {
    let callbackPosts = 0;
    let callbackCredentials: RequestCredentials | undefined;
    server.use(
      http.post(`${API}/api/auth/bungie/callback`, ({ request }) => {
        callbackPosts++;
        callbackCredentials = request.credentials;
        return HttpResponse.json({
          token: "new-token",
          user: sampleUser,
        });
      }),
    );

    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <React.StrictMode>
        <QueryClientProvider client={qc}>
          <AuthProvider>
            <MemoryRouter
              initialEntries={["/auth/callback?code=onetime&state=sig"]}
            >
              <Routes>
                <Route path="/auth/callback" element={<OAuthCallback />} />
                <Route path="/dashboard" element={<div>dashboard-stub</div>} />
              </Routes>
            </MemoryRouter>
          </AuthProvider>
        </QueryClientProvider>
      </React.StrictMode>,
    );

    expect(await screen.findByText("dashboard-stub")).toBeInTheDocument();
    expect(callbackPosts).toBe(1);
    expect(callbackCredentials).toBe("include");
    expect(localStorage.getItem("guardian_refresh_token")).toBeNull();
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
      </QueryClientProvider>,
    );
    expect(await screen.findByText("Authentication error")).toBeInTheDocument();
  });

  it("uses the authenticated reconnect endpoint and returns to the saved app route", async () => {
    sessionStorage.setItem("guardian_bungie_reconnect", "1");
    sessionStorage.setItem(
      "guardian_bungie_reconnect_return_to",
      "/collections?node=10",
    );
    let reconnectAuthorization: string | null = null;
    let reconnectBody = "";
    let normalCallbackPosts = 0;
    server.use(
      http.post(`${API}/api/auth/bungie/reconnect`, async ({ request }) => {
        reconnectAuthorization = request.headers.get("authorization");
        reconnectBody = await request.text();
        return new HttpResponse(null, { status: 204 });
      }),
      http.post(`${API}/api/auth/bungie/callback`, () => {
        normalCallbackPosts++;
        return HttpResponse.json({ token: "wrong-path", user: sampleUser });
      }),
    );

    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const collectionsKey = ["collections", 3, sampleUser.membershipId, "all"];
    qc.setQueryData(collectionsKey, { degraded: true });
    render(
      <QueryClientProvider client={qc}>
        <AuthProvider>
          <MemoryRouter
            initialEntries={["/auth/callback?code=reconnect-code&state=sig"]}
          >
            <Routes>
              <Route path="/auth/callback" element={<OAuthCallback />} />
              <Route
                path="/collections"
                element={<div>collections-stub</div>}
              />
            </Routes>
          </MemoryRouter>
        </AuthProvider>
      </QueryClientProvider>,
    );

    expect(await screen.findByText("collections-stub")).toBeInTheDocument();
    expect(reconnectAuthorization).toBe("Bearer test-token");
    expect(new URLSearchParams(reconnectBody).get("code")).toBe(
      "reconnect-code",
    );
    expect(new URLSearchParams(reconnectBody).get("state")).toBe("sig");
    expect(normalCallbackPosts).toBe(0);
    expect(qc.getQueryState(collectionsKey)?.isInvalidated).toBe(true);
    expect(sessionStorage.getItem("guardian_bungie_reconnect")).toBeNull();
    expect(browserSessionClient.getSnapshot().status).toBe("authenticated");
  });

  it("falls back to normal login when the Guardian Tracker session is gone", async () => {
    localStorage.clear();
    sessionStorage.setItem("guardian_bungie_reconnect", "1");
    sessionStorage.setItem(
      "guardian_bungie_reconnect_return_to",
      "/collections",
    );
    let reconnectPosts = 0;
    let normalCallbackPosts = 0;
    server.use(
      http.post(`${API}/api/auth/bungie/reconnect`, () => {
        reconnectPosts++;
        return new HttpResponse(null, { status: 204 });
      }),
      http.post(`${API}/api/auth/bungie/callback`, () => {
        normalCallbackPosts++;
        return HttpResponse.json({ token: "new-login", user: sampleUser });
      }),
    );

    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={qc}>
        <AuthProvider>
          <MemoryRouter
            initialEntries={["/auth/callback?code=login-code&state=sig"]}
          >
            <Routes>
              <Route path="/auth/callback" element={<OAuthCallback />} />
              <Route path="/dashboard" element={<div>dashboard-stub</div>} />
            </Routes>
          </MemoryRouter>
        </AuthProvider>
      </QueryClientProvider>,
    );

    expect(await screen.findByText("dashboard-stub")).toBeInTheDocument();
    expect(normalCallbackPosts).toBe(1);
    expect(reconnectPosts).toBe(0);
    expect(sessionStorage.getItem("guardian_bungie_reconnect")).toBeNull();
    expect(browserSessionClient.getSnapshot().status).toBe("authenticated");
  });
});
