import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { GetStartedPanel } from "./GetStartedPanel";

describe("GetStartedPanel", () => {
  it("renders the three starting points and dismisses", () => {
    const onDismiss = vi.fn();
    render(
      <MemoryRouter>
        <GetStartedPanel onDismiss={onDismiss} />
      </MemoryRouter>,
    );
    expect(screen.getByText(/see what you're missing/i)).toBeInTheDocument();
    expect(screen.getByText(/plan your week/i)).toBeInTheDocument();
    expect(screen.getByText(/track your chase/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /got it/i }));
    expect(onDismiss).toHaveBeenCalledOnce();
  });
});
