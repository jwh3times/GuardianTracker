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
    acquisitionSources: [],
    availableNow: false,
    collected: false,
    desc: "",
    ...over,
  };
}

it("shows the source text when the item has exactly one acquisition source", () => {
  render(
    <ItemCard
      item={item({
        acquisitionSources: [
          {
            text: "Vault of Glass",
            difficulty: "challenging",
            raidDungeon: true,
          },
        ],
      })}
    />,
  );
  expect(screen.getByText("Vault of Glass")).toBeInTheDocument();
  expect(screen.queryByText("Challenging")).not.toBeInTheDocument();
});

it("summarizes multiple sources without an aggregate difficulty badge", () => {
  render(
    <ItemCard
      item={item({
        acquisitionSources: [
          {
            text: "Vault of Glass",
            difficulty: "challenging",
            raidDungeon: true,
          },
          {
            text: "Monument to Lost Lights",
            difficulty: "easy",
            raidDungeon: false,
          },
        ],
      })}
    />,
  );
  expect(screen.getByText("2 acquisition sources")).toBeInTheDocument();
  expect(screen.queryByText("Challenging")).not.toBeInTheDocument();
  expect(screen.queryByText("Easy")).not.toBeInTheDocument();
});

it("renders a Farm only chip when farmOnly is set", () => {
  render(<ItemCard item={item({ farmOnly: true })} />);
  expect(screen.getByText("Farm only")).toBeInTheDocument();
});

it("omits the Farm only chip otherwise", () => {
  render(<ItemCard item={item({ farmOnly: false })} />);
  expect(screen.queryByText("Farm only")).not.toBeInTheDocument();
});
