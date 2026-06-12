import React from "react";
import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import {
  Badge,
  DataFreshnessChip,
  EmptyState,
  FilterChip,
  ItemTile,
  ProgressBar,
  StatTile,
  Textarea,
} from "../components/kit";

describe("ItemTile", () => {
  it("renders the Bungie icon when an icon path is given", () => {
    const { container } = render(<ItemTile rarity="exotic" type="Hand Cannon" icon="/icons/x.png" />);
    const img = container.querySelector("img.gt-tile-img");
    expect(img).toHaveAttribute("src", "https://www.bungie.net/icons/x.png");
    expect(container.querySelector(".gt-tile-glyph")).toBeNull();
  });

  it("falls back to the type glyph when the image fails to load", () => {
    const { container } = render(<ItemTile rarity="legendary" type="Hand Cannon" icon="/broken.png" />);
    fireEvent.error(container.querySelector("img.gt-tile-img")!);
    expect(container.querySelector(".gt-tile-glyph")).toHaveTextContent("HC");
  });

  it("renders the glyph when no icon is provided", () => {
    const { container } = render(<ItemTile rarity="rare" type="Auto Rifle" />);
    expect(container.querySelector(".gt-tile-glyph")).toHaveTextContent("AR");
  });
});

describe("DataFreshnessChip", () => {
  it("shows relative time from updatedAt", () => {
    const fiveMinAgo = new Date(Date.now() - 5 * 60_000).toISOString();
    render(<DataFreshnessChip updatedAt={fiveMinAgo} onRefresh={() => undefined} />);
    expect(screen.getByText("Updated 5m ago")).toBeInTheDocument();
  });

  it("shows a placeholder when no timestamp exists", () => {
    render(<DataFreshnessChip />);
    expect(screen.getByText("Updated —")).toBeInTheDocument();
  });

  it("shows the refreshing state while a refresh is in flight", () => {
    render(<DataFreshnessChip updatedAt={new Date().toISOString()} refreshing onRefresh={() => undefined} />);
    expect(screen.getByText("Refreshing…")).toBeInTheDocument();
  });

  it("fires onRefresh on click but not while refreshing", () => {
    let calls = 0;
    const { rerender } = render(
      <DataFreshnessChip updatedAt={new Date().toISOString()} onRefresh={() => calls++} />
    );
    fireEvent.click(screen.getByRole("button"));
    expect(calls).toBe(1);

    rerender(
      <DataFreshnessChip updatedAt={new Date().toISOString()} refreshing onRefresh={() => calls++} />
    );
    fireEvent.click(screen.getByRole("button"));
    expect(calls).toBe(1);
  });
});

describe("ProgressBar", () => {
  it("shows value/max when showVal is set", () => {
    const { container } = render(<ProgressBar value={3} max={4} showVal />);
    expect(container.textContent).toContain("3/4");
  });
});

describe("Badge", () => {
  it("renders the default label for known kinds", () => {
    render(<Badge kind="avail-now" />);
    expect(screen.getByText("Avail now")).toBeInTheDocument();
  });
  it("renders custom children", () => {
    render(<Badge kind="missing">3 missing</Badge>);
    expect(screen.getByText("3 missing")).toBeInTheDocument();
  });
});

describe("FilterChip", () => {
  it("reflects on-state and handles clicks", () => {
    let on = false;
    render(<FilterChip on={on} onClick={() => (on = true)}>Missing only</FilterChip>);
    fireEvent.click(screen.getByText("Missing only"));
    expect(on).toBe(true);
  });
});

describe("Textarea", () => {
  it("propagates changes and respects maxLength", () => {
    let value = "";
    render(
      <Textarea value={value} onChange={(v) => (value = v)} maxLength={500} ariaLabel="Notes" />
    );
    const el = screen.getByLabelText("Notes");
    fireEvent.change(el, { target: { value: "god roll" } });
    expect(value).toBe("god roll");
    expect(el).toHaveAttribute("maxlength", "500");
  });
});

describe("StatTile / EmptyState", () => {
  it("renders stat values", () => {
    render(<StatTile num="1,234" label="Total" mono />);
    expect(screen.getByText("1,234")).toBeInTheDocument();
  });
  it("renders empty-state copy and actions", () => {
    render(<EmptyState icon="filter" title="Nothing here" body="No matches." action={<button>Clear</button>} />);
    expect(screen.getByText("Nothing here")).toBeInTheDocument();
    expect(screen.getByText("Clear")).toBeInTheDocument();
  });
});
