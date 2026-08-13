import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { OnboardingTour } from "./OnboardingTour";

const mocks = vi.hoisted(() => ({
  complete: vi.fn(async () => {}),
  preferencesReady: true,
  onboardedAt: null as string | null,
}));

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    // This mock bypasses collectionsQuery's `select`, so it must supply the
    // adapted view shape directly: summary is an ordered array of
    // { count: [collected, total] }, not the raw four-key record.
    useQuery: () => ({
      data: {
        summary: [
          { id: "weapons", label: "Weapons", pct: 70, count: [7, 10] },
          { id: "armor", label: "Armor", pct: 100, count: [8, 8] },
          { id: "exotics", label: "Exotics", pct: 80, count: [4, 5] },
          { id: "cosmetics", label: "Cosmetics", pct: 75, count: [9, 12] },
        ],
      },
    }),
  };
});

vi.mock("../../contexts/AuthContext", () => ({
  useAuth: () => ({
    user: { membershipType: 3, membershipId: "member-1" },
  }),
}));

vi.mock("../../contexts/PreferencesContext", () => ({
  usePreferences: () => ({
    onboardedAt: mocks.onboardedAt,
    preferencesReady: mocks.preferencesReady,
    completeOnboarding: mocks.complete,
  }),
}));

describe("OnboardingTour", () => {
  beforeEach(() => {
    mocks.complete.mockClear();
    mocks.preferencesReady = true;
    mocks.onboardedAt = null;
  });

  it("frames the collection snapshot and walks through three steps", async () => {
    render(
      <MemoryRouter initialEntries={["/dashboard"]}>
        <OnboardingTour />
      </MemoryRouter>,
    );

    expect(
      screen.getByRole("dialog", { name: /your collection, with a plan/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/7 gaps across 3/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /show me around/i }));
    expect(screen.getByText(/step 1 of 3/i)).toBeInTheDocument();
    expect(screen.getByText(/collection at a glance/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByText(/step 2 of 3/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByText(/step 3 of 3/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Finish" }));
    await waitFor(() => expect(mocks.complete).toHaveBeenCalledOnce());
  });

  it("does not render before preferences resolve or after completion", () => {
    mocks.preferencesReady = false;
    const { rerender } = render(
      <MemoryRouter>
        <OnboardingTour />
      </MemoryRouter>,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    mocks.preferencesReady = true;
    mocks.onboardedAt = "2026-07-12T15:30:00Z";
    rerender(
      <MemoryRouter>
        <OnboardingTour />
      </MemoryRouter>,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
