import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CosmeticDetail } from "./CosmeticDetail";
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
  desc: "A regal emblem.",
};

describe("CosmeticDetail", () => {
  it("renders nothing when item is null", () => {
    const { container } = render(
      <CosmeticDetail item={null} onClose={() => {}} />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("renders details and closes on scrim click", () => {
    const onClose = vi.fn();
    render(<CosmeticDetail item={item} onClose={onClose} />);
    const dialog = screen.getByRole("dialog", { name: "Calus Selected" });
    expect(dialog).toBeInTheDocument();
    expect(screen.getByText("Emblem")).toBeInTheDocument();
    expect(screen.getByText(/Collected/)).toBeInTheDocument();
    fireEvent.click(dialog.parentElement as HTMLElement); // the scrim
    expect(onClose).toHaveBeenCalled();
  });

  it("does not react to Escape when item is null", () => {
    const onClose = vi.fn();
    render(<CosmeticDetail item={null} onClose={onClose} />);
    fireEvent.keyDown(document.body, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
  });

  it("closes on Escape when open", () => {
    const onClose = vi.fn();
    render(<CosmeticDetail item={item} onClose={onClose} />);
    fireEvent.keyDown(document.body, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });
});
