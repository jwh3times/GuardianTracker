import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CosmeticTile } from "./CosmeticTile";
import type { GTItem } from "../../types/design";

const item: GTItem = {
  id: "1",
  name: "Calus Selected",
  type: "Emblem",
  slot: "",
  rarity: "legendary",
  diff: "unrated",
  source: "",
  sourceDetail: "",
  availableNow: false,
  collected: true,
  desc: "",
};

describe("CosmeticTile", () => {
  it("shows the name and marks collected state", () => {
    render(<CosmeticTile item={item} onOpen={() => {}} />);
    expect(screen.getByText("Calus Selected")).toBeInTheDocument();
    expect(screen.getByRole("button")).toHaveAttribute(
      "data-collected",
      "true",
    );
  });

  it("calls onOpen when clicked", () => {
    const onOpen = vi.fn();
    render(
      <CosmeticTile item={{ ...item, collected: false }} onOpen={onOpen} />,
    );
    fireEvent.click(screen.getByRole("button"));
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ id: "1" }));
  });

  it("flags an item a vendor is selling, but only when not already collected", () => {
    const onSale = { ...item, availableNow: true, availFrom: "Xûr" };

    const { unmount } = render(
      <CosmeticTile item={{ ...onSale, collected: false }} onOpen={() => {}} />,
    );
    expect(screen.getByRole("button")).toHaveAttribute(
      "title",
      "Calus Selected — available now from Xûr",
    );
    unmount();

    // Already owned: nothing to act on, so no badge and no availability title.
    render(<CosmeticTile item={onSale} onOpen={() => {}} />);
    expect(screen.getByRole("button")).toHaveAttribute(
      "title",
      "Calus Selected",
    );
  });

  it("renders the Bungie CDN icon image when the item has an icon", () => {
    const { container } = render(
      <CosmeticTile
        item={{ ...item, icon: "/common/destiny2_content/icons/calus.png" }}
        onOpen={() => {}}
      />,
    );
    const img = container.querySelector("img.gt-tile-img") as HTMLImageElement;
    expect(img).not.toBeNull();
    expect(img.src).toBe(
      "https://www.bungie.net/common/destiny2_content/icons/calus.png",
    );
  });
});
