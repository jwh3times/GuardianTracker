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
  obtainable: false,
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
});
