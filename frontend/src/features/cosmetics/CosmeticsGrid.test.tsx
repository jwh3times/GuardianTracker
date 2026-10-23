import { describe, it, expect, vi, beforeAll, afterAll } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CosmeticsGrid } from "./CosmeticsGrid";
import type { GTItem } from "../../types/design";

// jsdom has no layout/ResizeObserver; stub them so the virtualizer measures a
// non-zero viewport and renders a window of rows.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
beforeAll(() => {
  vi.stubGlobal("ResizeObserver", ResizeObserverStub);
  Object.defineProperty(HTMLElement.prototype, "clientWidth", {
    configurable: true,
    get: () => 1000,
  });
  // @tanstack/virtual-core v3.19.6 uses clientWidth/clientHeight (via getRect)
  // rather than getBoundingClientRect to measure the scroll container.
  // Stub clientHeight so the virtualizer sees a non-zero viewport height.
  Object.defineProperty(HTMLElement.prototype, "clientHeight", {
    configurable: true,
    get: () => 800,
  });
  HTMLElement.prototype.getBoundingClientRect = () => ({
    width: 1000,
    height: 800,
    top: 0,
    left: 0,
    right: 1000,
    bottom: 800,
    x: 0,
    y: 0,
    toJSON: () => {},
  });
});
afterAll(() => vi.unstubAllGlobals());

function makeItems(n: number): GTItem[] {
  return Array.from({ length: n }, (_, i) => ({
    id: String(i),
    name: `Item ${i}`,
    type: "Emblem",
    slot: "",
    rarity: "legendary",
    acquisitionSources: [],
    availableNow: false,
    collected: i % 2 === 0,
    desc: "",
  }));
}

describe("CosmeticsGrid", () => {
  it("renders visible tiles and fires onOpen when one is clicked", () => {
    const onOpen = vi.fn<(item: GTItem) => void>();
    render(<CosmeticsGrid items={makeItems(30)} onOpen={onOpen} />);
    fireEvent.click(screen.getByText("Item 0"));
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ id: "0" }));
  });
});
