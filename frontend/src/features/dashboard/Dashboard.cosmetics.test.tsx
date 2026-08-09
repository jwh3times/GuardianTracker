import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { Routes, Route } from "react-router";
import { renderWithProviders } from "../../test/renderWithProviders";
import { Dashboard } from "./Dashboard";

beforeEach(() => {
  localStorage.clear();
});

function renderDashboard() {
  return renderWithProviders(
    <Routes>
      <Route path="/dashboard" element={<Dashboard />} />
      <Route path="/cosmetics" element={<div>COSMETICS PAGE</div>} />
    </Routes>,
    { route: "/dashboard" },
  );
}

describe("Dashboard cosmetics deep-link", () => {
  it("navigates to /cosmetics when the Cosmetics hero bar is clicked", async () => {
    renderDashboard();
    fireEvent.click(await screen.findByRole("button", { name: /cosmetics/i }));
    expect(await screen.findByText("COSMETICS PAGE")).toBeInTheDocument();
  });
});
