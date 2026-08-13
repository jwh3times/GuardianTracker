import { render, screen } from "@testing-library/react";
import { ItemCard } from "./ItemCard";
import type { GTItem } from "../../types/design";

function item(over: Partial<GTItem>): GTItem {
  return {
    id: "1",
    name: "Test",
    type: "Hand Cannon",
    slot: "",
    rarity: "legendary",
    diff: "unrated",
    source: "Unknown",
    sourceDetail: "",
    availableNow: false,
    collected: false,
    desc: "",
    ...over,
  };
}

it("renders the Unrated difficulty badge", () => {
  render(<ItemCard item={item({ diff: "unrated" })} />);
  expect(screen.getByText("Unrated")).toBeInTheDocument();
});

it("renders a Farm only chip when farmOnly is set", () => {
  render(<ItemCard item={item({ farmOnly: true })} />);
  expect(screen.getByText("Farm only")).toBeInTheDocument();
});

it("omits the Farm only chip otherwise", () => {
  render(<ItemCard item={item({ farmOnly: false })} />);
  expect(screen.queryByText("Farm only")).not.toBeInTheDocument();
});
