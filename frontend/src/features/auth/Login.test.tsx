import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { PreferencesProvider } from "../../contexts/PreferencesContext";
import { ToastProvider } from "../../components/Toast";
import { Login } from "./Login";

beforeEach(() => {
  localStorage.clear();
  localStorage.setItem("guardian_token", "test-token");
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
    </QueryClientProvider>,
  );
}

describe("Login page", () => {
  it("redirects to the Bungie auth URL on success", async () => {
    server.use(
      http.get(`${API}/api/auth/bungie`, () =>
        HttpResponse.json({
          authUrl: "https://bungie.net/authorize",
          state: "xyz",
        }),
      ),
    );
    renderPage(<Login />);
    fireEvent.click(
      screen.getByRole("button", { name: /Sign in with Bungie/ }),
    );
    // After success the button enters the redirecting (loading) state
    expect(
      await screen.findByText("Redirecting to Bungie.net…"),
    ).toBeInTheDocument();
  });

  it("shows an error when the auth request fails", async () => {
    server.use(
      http.get(
        `${API}/api/auth/bungie`,
        () => new HttpResponse(null, { status: 500 }),
      ),
    );
    renderPage(<Login />);
    fireEvent.click(
      screen.getByRole("button", { name: /Sign in with Bungie/ }),
    );
    expect(
      await screen.findByText(/Failed to start authentication/),
    ).toBeInTheDocument();
  });

  it("shows an error when the response has no auth URL", async () => {
    server.use(
      http.get(`${API}/api/auth/bungie`, () =>
        HttpResponse.json({ state: "xyz" }),
      ),
    );
    renderPage(<Login />);
    fireEvent.click(
      screen.getByRole("button", { name: /Sign in with Bungie/ }),
    );
    expect(
      await screen.findByText(/No authorization URL received/),
    ).toBeInTheDocument();
  });
});
