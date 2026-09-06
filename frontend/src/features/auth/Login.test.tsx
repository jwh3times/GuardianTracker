import { browserSessionClient } from "../../lib/browserSessionBrowser";
import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { API, server } from "../../test/testServer";
import { renderWithProviders } from "../../test/renderWithProviders";
import { Login } from "./Login";

beforeEach(() => {
  localStorage.clear();
  sessionStorage.clear();
});

function renderPage(ui: React.ReactNode, route = "/") {
  return renderWithProviders(ui, { route });
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
      await screen.findByText(
        /Authorization setup returned an invalid response/,
      ),
    ).toBeInTheDocument();
  });

  it("starts the same Bungie authorization flow in reconnect mode without clearing app auth", async () => {
    server.use(
      http.get(`${API}/api/auth/bungie`, () =>
        HttpResponse.json({
          authUrl: "https://bungie.net/authorize",
          state: "xyz",
        }),
      ),
    );

    renderPage(<Login mode="reauthorize" />);
    expect(
      screen.getByRole("heading", { name: /Reconnect Bungie to continue/i }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Reconnect Bungie/i }));

    expect(
      await screen.findByText("Redirecting to Bungie.net…"),
    ).toBeInTheDocument();
    expect(sessionStorage.getItem("guardian_bungie_reconnect")).toBe("1");
    expect(browserSessionClient.getSnapshot().status).toBe("authenticated");
  });
});
