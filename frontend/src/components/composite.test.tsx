import { describe, it, expect } from "vitest";
import { render, screen, fireEvent, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CategoryTree, SealCard, XurModule } from "./composite";
import type { Seal, TreeNode } from "../types/design";

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

describe("SealCard", () => {
  const sealWithObjectives: Seal = {
    id: "seal-test",
    name: "Test Seal",
    pct: 40,
    gilded: 0,
    left: "3 triumphs left",
    triumphs: [
      { label: "Plain triumph", done: false, cur: 1, max: 3 },
      {
        label: "Multi-objective triumph",
        done: false,
        cur: 1,
        max: 2,
        objectives: [
          { label: "Objective A", done: true, cur: 1, max: 1 },
          { label: "Objective B", done: false, cur: 0, max: 1 },
        ],
      },
      {
        label: "Single-objective triumph",
        done: false,
        cur: 2,
        max: 5,
        objectives: [{ label: "Objective solo", done: false, cur: 2, max: 5 }],
      },
    ],
  };

  it("renders a triumph without objectives with no disclosure toggle", () => {
    render(<SealCard seal={sealWithObjectives} expanded onToggle={() => {}} />);
    expect(screen.getByText("Plain triumph")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Plain triumph/i }),
    ).not.toBeInTheDocument();
  });

  it("renders a collapsed disclosure by default for a triumph with objectives", () => {
    render(<SealCard seal={sealWithObjectives} expanded onToggle={() => {}} />);
    const toggle = screen.getByRole("button", {
      name: /Multi-objective triumph/i,
    });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Objective A")).not.toBeInTheDocument();
    expect(screen.queryByText("Objective B")).not.toBeInTheDocument();
  });

  it("is keyboard operable: Tab reaches the toggle and Enter expands it", async () => {
    const user = userEvent.setup();
    render(<SealCard seal={sealWithObjectives} expanded onToggle={() => {}} />);
    const toggle = screen.getByRole("button", {
      name: /Multi-objective triumph/i,
    });

    await user.tab(); // seal head
    // "Plain triumph" has no toggle (its check is aria-hidden/tabIndex=-1),
    // so the next stop is the multi-objective toggle button.
    await user.tab();
    expect(toggle).toHaveFocus();

    await user.keyboard("{Enter}");
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("Objective A")).toBeInTheDocument();
  });

  it("shows a completed/total count summary for a multi-objective triumph", () => {
    render(<SealCard seal={sealWithObjectives} expanded onToggle={() => {}} />);
    const toggle = screen.getByRole("button", {
      name: /Multi-objective triumph/i,
    });
    // 1 of 2 objectives is done
    expect(within(toggle).getByText("1/2")).toBeInTheDocument();
  });

  it("shows the objective's own progress (not a count) for a single-objective triumph", () => {
    render(<SealCard seal={sealWithObjectives} expanded onToggle={() => {}} />);
    const toggle = screen.getByRole("button", {
      name: /Single-objective triumph/i,
    });
    // Not "0/1" or "1/1" — the objective's own cur/max
    expect(within(toggle).getByText("2/5")).toBeInTheDocument();
  });

  it("shows each objective's label and exact progress when expanded", () => {
    render(<SealCard seal={sealWithObjectives} expanded onToggle={() => {}} />);
    fireEvent.click(
      screen.getByRole("button", { name: /Multi-objective triumph/i }),
    );

    expect(screen.getByText("Objective A")).toBeInTheDocument();
    expect(screen.getByText("Objective B")).toBeInTheDocument();
    expect(screen.getByText("1/1")).toBeInTheDocument();
    expect(screen.getByText("0/1")).toBeInTheDocument();
  });

  it("expands one triumph's disclosure without expanding another", () => {
    render(<SealCard seal={sealWithObjectives} expanded onToggle={() => {}} />);
    const multiToggle = screen.getByRole("button", {
      name: /Multi-objective triumph/i,
    });
    const singleToggle = screen.getByRole("button", {
      name: /Single-objective triumph/i,
    });

    fireEvent.click(multiToggle);

    expect(multiToggle).toHaveAttribute("aria-expanded", "true");
    expect(singleToggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("Objective A")).toBeInTheDocument();
    expect(screen.queryByText("Objective solo")).not.toBeInTheDocument();
  });
});
