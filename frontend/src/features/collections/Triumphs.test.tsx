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
import { http, HttpResponse, delay } from "msw";
import { API, sampleUser, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { ToastProvider } from "../../components/Toast";
import { Triumphs } from "./Triumphs";

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  localStorage.clear();
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_refresh_token", "test-refresh");
  localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
});

function renderPage(ui: React.ReactNode, route = "/") {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <AuthProvider>
        <ToastProvider>
          <MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter>
        </ToastProvider>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

const privacy403 = () =>
  HttpResponse.json(
    { error: "Profile is private", code: "PRIVACY_RESTRICTION" },
    { status: 403 },
  );

describe("Triumphs page", () => {
  it("renders seals, auto-expands the first, and counts gilded", async () => {
    renderPage(<Triumphs />);
    expect(await screen.findByText("Conqueror")).toBeInTheDocument();
    expect(screen.getByText("2 seals · 1 gilded")).toBeInTheDocument();
    // First seal auto-expands → its triumph labels are visible
    expect(screen.getByText("Complete a GM")).toBeInTheDocument();
  });

  it("toggles a seal open and closed", async () => {
    renderPage(<Triumphs />);
    await screen.findByText("Conqueror");

    // Collapse the auto-opened first seal
    fireEvent.click(screen.getByText("Conqueror"));
    await waitFor(() =>
      expect(screen.queryByText("Complete a GM")).not.toBeInTheDocument(),
    );

    // Re-open it
    fireEvent.click(screen.getByText("Conqueror"));
    expect(await screen.findByText("Complete a GM")).toBeInTheDocument();
  });

  it("re-sorts by name when chosen from the dropdown", async () => {
    renderPage(<Triumphs />);
    await screen.findByText("Conqueror");

    fireEvent.click(screen.getByText("Sort: Closest to done"));
    fireEvent.click(screen.getByText("Sort: Name"));
    // After sorting by name, both seals remain rendered
    expect(screen.getByText("Conqueror")).toBeInTheDocument();
    expect(screen.getByText("Flawless")).toBeInTheDocument();
  });

  it("shows the privacy error state", async () => {
    server.use(http.get(`${API}/api/seals/:type/:id`, privacy403));
    renderPage(<Triumphs />);
    expect(
      await screen.findByText("Your Destiny profile is private"),
    ).toBeInTheDocument();
  });

  it("renders the loading spinner while the query is in flight", async () => {
    server.use(
      http.get(`${API}/api/seals/:type/:id`, async () => {
        await delay(40);
        return HttpResponse.json({ items: [], fetchedAt: "" });
      }),
    );
    const { container } = renderPage(<Triumphs />);
    expect(container.querySelector(".gt-page-loading")).not.toBeNull();
  });
});
