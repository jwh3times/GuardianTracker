import React from "react";
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ItemDetailDrawer } from "./composite";
import type { GTItem } from "../types/design";

const baseItem: GTItem = {
  id: "1000",
  name: "Test Auto Rifle",
  type: "Auto Rifle",
  slot: "",
  rarity: "legendary",
  diff: "moderate",
  source: "Vanguard",
  sourceDetail: "",
  obtainable: false,
  collected: false,
  desc: "A test weapon.",
  icon: undefined,
};

const noop = () => {};

function renderDrawer(
  extra: Partial<React.ComponentProps<typeof ItemDetailDrawer>>,
) {
  return render(
    <ItemDetailDrawer
      item={baseItem}
      onClose={noop}
      onWish={noop}
      wished={false}
      {...extra}
    />,
  );
}

describe("ItemDetailDrawer perks", () => {
  it("renders perk columns with labels and chips", () => {
    renderDrawer({
      perkColumns: [
        {
          role: "barrel",
          label: "Barrel",
          perks: ["Full Bore", "Arrowhead Brake"],
        },
        { role: "trait", label: "Trait 1", perks: ["Rampage"] },
      ],
    });
    expect(screen.getByText("Possible perks / rolls")).toBeInTheDocument();
    expect(screen.getByText("Barrel")).toBeInTheDocument();
    expect(screen.getByText("Trait 1")).toBeInTheDocument();
    expect(screen.getByText("Full Bore")).toBeInTheDocument();
    expect(screen.getByText("Arrowhead Brake")).toBeInTheDocument();
    expect(screen.getByText("Rampage")).toBeInTheDocument();
  });

  it("shows a loading state while perks load", () => {
    renderDrawer({ perksLoading: true });
    expect(screen.getByText(/loading perks/i)).toBeInTheDocument();
  });

  it("hides the perks block when there are no columns", () => {
    renderDrawer({ perkColumns: [] });
    expect(
      screen.queryByText("Possible perks / rolls"),
    ).not.toBeInTheDocument();
  });

  it("relabels the external link to light.gg", () => {
    renderDrawer({});
    expect(
      screen.getByRole("button", { name: /light\.gg/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /where to farm/i }),
    ).not.toBeInTheDocument();
  });
});
