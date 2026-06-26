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
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { OAuthCallback } from "./pages/OAuthCallback";

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  localStorage.clear();
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_refresh_token", "test-refresh");
  localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
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
