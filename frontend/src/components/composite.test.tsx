import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CategoryTree, XurModule } from "./composite";
import type { TreeNode } from "../types/design";

const nodes: TreeNode[] = [
  {
    id: "10",
    label: "Weapons",
    pct: 50,
    count: [1, 2],
    children: [
      {
        id: "11",
        label: "Hand Cannons",
        pct: 50,
        count: [1, 2],
        children: [{ id: "12", label: "Kinetic HCs", pct: 0, count: [0, 1] }],
      },
    ],
  },
];

describe("CategoryTree", () => {
  it("renders nested children when expanded and selects on click", () => {
    let selected = "";
    render(
      <CategoryTree
        nodes={nodes}
        activeId="10"
        onSelect={(id) => (selected = id)}
      />,
    );

    // Top node visible; grandchild hidden until each level is expanded.
    expect(screen.getByText("Weapons")).toBeInTheDocument();
    expect(screen.queryByText("Kinetic HCs")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /expand weapons/i }));
    expect(screen.getByText("Hand Cannons")).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: /expand hand cannons/i }),
    );
    expect(screen.getByText("Kinetic HCs")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Kinetic HCs"));
    expect(selected).toBe("12");
  });

  it("reveals a deep descendant via the expand seed without any click", () => {
    render(
      <CategoryTree
        nodes={nodes}
        activeId="12"
        onSelect={() => {}}
        expand={["10", "11"]}
      />,
    );
    // Seeding the ancestor path ("10" → "11") opens both levels on mount, so
    // the depth-2 grandchild is in the document without a manual expand click.
    expect(screen.getByText("Kinetic HCs")).toBeInTheDocument();
  });

  it("exposes tree a11y roles", () => {
    render(<CategoryTree nodes={nodes} activeId="10" onSelect={() => {}} />);
    expect(screen.getByRole("tree")).toBeInTheDocument();
    // Only the top-level node ("Weapons") renders without expansion; children
    // are conditionally mounted, so exactly 1 treeitem is present initially.
    expect(screen.getAllByRole("treeitem").length).toBe(1);
  });
});

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
