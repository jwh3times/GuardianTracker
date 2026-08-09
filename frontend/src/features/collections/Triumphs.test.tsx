import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse, delay } from "msw";
import { API, server } from "../../test/testServer";
import { renderWithProviders } from "../../test/renderWithProviders";
import { Triumphs } from "./Triumphs";

beforeEach(() => {
  localStorage.clear();
});

function renderPage(ui: React.ReactNode, route = "/") {
  return renderWithProviders(ui, { route });
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

  it("expands a triumph's own objective disclosure to show exact per-objective progress", async () => {
    renderPage(<Triumphs />);
    await screen.findByText("Conqueror");

    // "Complete a GM" carries no objective data — it stays a flat row.
    expect(
      screen.queryByRole("button", { name: /Complete a GM/i }),
    ).not.toBeInTheDocument();

    // "Flawless card" carries objective data — it gets a disclosure toggle,
    // collapsed by default.
    const toggle = screen.getByRole("button", { name: /Flawless card/i });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Win 7 Trials matches")).not.toBeInTheDocument();

    fireEvent.click(toggle);

    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(await screen.findByText("Win 7 Trials matches")).toBeInTheDocument();
    expect(screen.getByText("No deaths in the run")).toBeInTheDocument();
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
