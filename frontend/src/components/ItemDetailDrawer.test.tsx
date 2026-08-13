import React from "react";
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
  availableNow: false,
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

describe("ItemDetailDrawer catalysts", () => {
  it("renders a catalyst section with name and description", () => {
    renderDrawer({
      catalysts: [
        {
          name: "Sunshot Catalyst",
          description:
            "Applying an elemental debuff to a target increases this weapon's reload speed, aim assist, and movement while aiming down sights for a short duration.",
        },
      ],
    });
    expect(screen.getByText("Catalyst")).toBeInTheDocument();
    expect(screen.getByText("Sunshot Catalyst")).toBeInTheDocument();
    expect(
      screen.getByText(/Applying an elemental debuff/i),
    ).toBeInTheDocument();
  });

  it("hides the catalyst section when catalysts is empty or absent", () => {
    renderDrawer({ catalysts: [] });
    expect(screen.queryByText("Catalyst")).not.toBeInTheDocument();

    renderDrawer({});
    expect(screen.queryByText("Catalyst")).not.toBeInTheDocument();
  });

  it("renders all entries for a multi-catalyst exotic", () => {
    renderDrawer({
      catalysts: [
        { name: "Catalyst One", description: "First effect." },
        { name: "Catalyst Two", description: "Second effect." },
        { name: "Catalyst Three", description: "" },
        { name: "Catalyst Four", description: "Fourth effect." },
      ],
    });
    expect(screen.getByText("Catalyst One")).toBeInTheDocument();
    expect(screen.getByText("First effect.")).toBeInTheDocument();
    expect(screen.getByText("Catalyst Two")).toBeInTheDocument();
    expect(screen.getByText("Second effect.")).toBeInTheDocument();
    expect(screen.getByText("Catalyst Three")).toBeInTheDocument();
    expect(screen.getByText("Catalyst Four")).toBeInTheDocument();
    expect(screen.getByText("Fourth effect.")).toBeInTheDocument();
  });

  it("shows just the name when a catalyst entry has an empty description", () => {
    const { container } = renderDrawer({
      catalysts: [{ name: "Duality Catalyst", description: "" }],
    });
    expect(screen.getByText("Duality Catalyst")).toBeInTheDocument();
    expect(container.querySelectorAll(".gt-catalyst-desc").length).toBe(0);
  });
});

describe("ItemDetailDrawer difficulty copy", () => {
  it("shows the unrated 'why' copy", async () => {
    renderDrawer({ item: { ...baseItem, diff: "unrated" } });
    await userEvent.click(screen.getByRole("button", { name: /why\?/i }));
    expect(
      screen.getByText(/couldn't determine difficulty/i),
    ).toBeInTheDocument();
  });

  it("shows the estimated 'why' copy for rated difficulties", async () => {
    renderDrawer({ item: { ...baseItem, diff: "moderate" } });
    await userEvent.click(screen.getByRole("button", { name: /why\?/i }));
    expect(screen.getByText(/estimated/i)).toBeInTheDocument();
  });

  it("shows farm-only note when farmOnly is set", () => {
    renderDrawer({ item: { ...baseItem, farmOnly: true } });
    expect(screen.getByText(/earn a fresh drop/i)).toBeInTheDocument();
  });

  it("omits farm-only note when farmOnly is not set", () => {
    renderDrawer({ item: { ...baseItem, farmOnly: false } });
    expect(screen.queryByText(/earn a fresh drop/i)).not.toBeInTheDocument();
  });
});
