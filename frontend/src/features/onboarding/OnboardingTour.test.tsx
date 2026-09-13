import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PreferencesResolution } from "../../data/preferences";
import { OnboardingTour } from "./OnboardingTour";

const REQUIRED: PreferencesResolution = {
  status: "resolved",
  persisted: true,
  onboarding: "required",
};

const mocks = vi.hoisted(() => ({
  complete: vi.fn<() => Promise<void>>(async () => {}),
  resolution: null as unknown as PreferencesResolution,
}));

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    // This mock bypasses useCollectionsSummary's `select`, so it must supply the
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

// Only the hook is replaced; the real fail-closed gate decides visibility.
vi.mock("../../data/preferences", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../../data/preferences")>();
  return {
    ...actual,
    usePreferences: () => ({
      resolution: mocks.resolution,
      completeOnboarding: mocks.complete,
    }),
  };
});

function renderTour() {
  return render(
    <MemoryRouter initialEntries={["/dashboard"]}>
      <OnboardingTour />
    </MemoryRouter>,
  );
}

describe("OnboardingTour", () => {
  beforeEach(() => {
    mocks.complete.mockClear();
    mocks.resolution = REQUIRED;
  });

  it("frames the collection snapshot and walks through three steps", async () => {
    renderTour();

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

  it("shows the save error and stays open when completion fails", async () => {
    mocks.complete.mockRejectedValueOnce(new Error("down"));
    renderTour();

    fireEvent.click(screen.getByRole("button", { name: /skip tour/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      /could not save that choice/i,
    );
    expect(
      screen.getByRole("dialog", { name: /your collection, with a plan/i }),
    ).toBeInTheDocument();
  });

  it.each<[string, PreferencesResolution]>([
    ["anonymous", { status: "anonymous" }],
    ["unresolved from cache", { status: "unresolved", source: "cache" }],
    ["unresolved from defaults", { status: "unresolved", source: "defaults" }],
    [
      "degraded",
      { status: "resolved", persisted: false, onboarding: "required" },
    ],
    [
      "already completed",
      {
        status: "resolved",
        persisted: true,
        onboarding: { completedAt: "2026-07-12T15:30:00Z" },
      },
    ],
  ])("does not render when preferences are %s", (_label, resolution) => {
    mocks.resolution = resolution;
    renderTour();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
