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
});
