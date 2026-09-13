import { seedBrowserSession } from "../test/browserSession";
import React from "react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  render,
  screen,
  fireEvent,
  renderHook,
  act,
  waitFor,
} from "@testing-library/react";
import { ApiError } from "./api";
import { errorState } from "./errorState";
import { ItemCard } from "../features/collections/ItemCard";
import { ToastProvider, useToast } from "../components/Toast";
import { AuthProvider, useAuth } from "../contexts/AuthContext";
import type { GTItem } from "../types/design";

beforeEach(() => localStorage.clear());

/* ---------------- errorState ---------------- */
describe("errorState", () => {
  it("maps PRIVACY_RESTRICTION to the private-profile copy with a privacy link", () => {
    const s = errorState(new ApiError("x", 403, "PRIVACY_RESTRICTION"));
    expect(s.title).toMatch(/private/i);
    expect(s.privacyLink).toBe(true);
  });

  it("maps MANIFEST_NOT_READY and 503 to the warming-up copy", () => {
    expect(
      errorState(new ApiError("x", 500, "MANIFEST_NOT_READY")).title,
    ).toMatch(/Warming up/);
    expect(errorState(new ApiError("x", 503)).title).toMatch(/Warming up/);
  });

  it("maps BUNGIE_ERROR and 502 to the Bungie-unavailable copy", () => {
    expect(errorState(new ApiError("x", 500, "BUNGIE_ERROR")).title).toMatch(
      /Bungie API unavailable/,
    );
    expect(errorState(new ApiError("x", 502)).title).toMatch(
      /Bungie API unavailable/,
    );
  });

  it("falls back to the generic copy for unknown errors", () => {
    expect(errorState(new Error("boom")).title).toMatch(/Couldn't load data/);
    expect(errorState(new ApiError("x", 400, "SOMETHING_ELSE")).title).toMatch(
      /Couldn't load data/,
    );
  });
});

/* ---------------- ItemCard ---------------- */
const baseItem: GTItem = {
  id: "i1",
  name: "Fatebringer",
  type: "Hand Cannon",
  slot: "Kinetic",
  rarity: "legendary",
  acquisitionSources: [
    {
      text: "Vault of Glass",
      difficulty: "moderate",
      raidDungeon: true,
    },
  ],
  availableNow: true,
  collected: false,
  desc: "A legendary hand cannon.",
  icon: "/icons/fb.png",
};

describe("ItemCard", () => {
  it("renders the grid density with an availability badge and fires callbacks", () => {
    const opened: GTItem[] = [];
    const wished: GTItem[] = [];
    render(
      <ItemCard
        item={baseItem}
        density="grid"
        personalize="aggressive"
        onOpen={(i) => opened.push(i)}
        onWish={(i) => wished.push(i)}
      />,
    );
    expect(screen.getByText("Fatebringer")).toBeInTheDocument();
    expect(screen.getByText("Avail now")).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText("Add Fatebringer to wishlist"));
    expect(wished).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: /Details/ }));
    expect(opened).toHaveLength(1);
  });

  it("renders the list density and shows the collected badge", () => {
    render(
      <ItemCard
        item={{ ...baseItem, collected: true, availableNow: false }}
        density="list"
        showCollected
        wished
      />,
    );
    expect(screen.getByText(/Vault of Glass/)).toBeInTheDocument();
    // wished=true → remove label
    expect(
      screen.getByLabelText("Remove Fatebringer from wishlist"),
    ).toBeInTheDocument();
  });

  it("renders the compact density and hides for-you badges when personalize is off", () => {
    render(<ItemCard item={baseItem} density="compact" personalize="off" />);
    expect(screen.getByText("Fatebringer")).toBeInTheDocument();
    expect(screen.queryByText("Avail now")).not.toBeInTheDocument();
  });

  it("shows the Collected badge in grid for an owned item", () => {
    render(
      <ItemCard
        item={{ ...baseItem, collected: true }}
        density="grid"
        showCollected
      />,
    );
    expect(screen.getByText("Collected")).toBeInTheDocument();
  });
});

/* ---------------- Toast ---------------- */
function ToastTrigger({ type }: { type: "success" | "error" | "info" }) {
  const { showToast } = useToast();
  return <button onClick={() => showToast(`msg-${type}`, type)}>fire</button>;
}

describe("Toast", () => {
  it("shows and dismisses toasts of each type", () => {
    const { rerender } = render(
      <ToastProvider>
        <ToastTrigger type="success" />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByText("fire"));
    expect(screen.getByText("msg-success")).toBeInTheDocument();
    // Dismiss via the × button
    fireEvent.click(screen.getByText("×"));
    expect(screen.queryByText("msg-success")).not.toBeInTheDocument();

    rerender(
      <ToastProvider>
        <ToastTrigger type="error" />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByText("fire"));
    expect(screen.getByText("msg-error")).toBeInTheDocument();
    // Dismiss by clicking the toast body
    fireEvent.click(screen.getByText("msg-error"));
    expect(screen.queryByText("msg-error")).not.toBeInTheDocument();
  });

  it("throws when used outside a provider", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => renderHook(() => useToast())).toThrow(
      /within a ToastProvider/,
    );
    spy.mockRestore();
  });
});

/* ---------------- AuthContext: refresh + recovery ---------------- */
describe("AuthContext refresh & recovery", () => {
  it("clears corrupt stored user data and stays unauthenticated", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    localStorage.setItem("guardian_token", "tok");
    localStorage.setItem("guardian_user", "{bad json");
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });
    expect(result.current.isAuthenticated).toBe(false);
    expect(localStorage.getItem("guardian_token")).toBeNull();
    spy.mockRestore();
  });

  it("adopts a newer browser projection", async () => {
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });
    expect(result.current.isAuthenticated).toBe(false);
    act(() => {
      seedBrowserSession();
    });
    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));
  });
});
