import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { XurModule } from "./XurModule";

describe("XurModule", () => {
  it("marks class armor for the active Guardian", () => {
    render(
      <XurModule
        activeClassName="Warlock"
        xur={{
          present: true,
          leavesIn: { d: 1 },
          items: [
            {
              hash: "101",
              name: "Test Warlock Robes",
              type: "Armor",
              icon: "",
              rarity: "exotic",
              missing: true,
              cost: "23 Strange Coins",
              className: "Warlock",
            },
          ],
        }}
      />,
    );

    expect(screen.getByText("For your Warlock")).toBeInTheDocument();
  });
});
