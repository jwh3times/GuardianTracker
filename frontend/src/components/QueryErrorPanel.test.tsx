import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { QueryErrorPanel } from "./QueryErrorPanel";
import { ApiError } from "../lib/api";

describe("QueryErrorPanel", () => {
  it("shows the privacy affordance only for a privacy restriction", () => {
    const { unmount } = render(
      <QueryErrorPanel
        error={new ApiError("nope", 403, "PRIVACY_RESTRICTION")}
        onRetry={() => {}}
      />,
    );
    expect(screen.getByText(/profile is private/i)).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /bungie privacy settings/i }),
    ).toHaveAttribute(
      "href",
      "https://www.bungie.net/7/en/User/Account/Privacy",
    );
    unmount();

    render(
      <QueryErrorPanel
        error={new ApiError("boom", 502, "BUNGIE_ERROR")}
        onRetry={() => {}}
      />,
    );
    expect(screen.getByText(/bungie api unavailable/i)).toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: /bungie privacy settings/i }),
    ).not.toBeInTheDocument();
  });

  it("branches copy on the backend's machine-readable code", () => {
    const cases: Array<[ApiError | undefined, RegExp]> = [
      [new ApiError("x", 503, "MANIFEST_NOT_READY"), /warming up/i],
      [new ApiError("x", 403, "TIER_LOCKED"), /feature locked/i],
      [new ApiError("x", 404, "FEATURE_DISABLED"), /not available/i],
      // No error at all (a query that resolved with nothing usable) still gets
      // honest copy rather than an empty panel.
      [undefined, /couldn't load data/i],
    ];
    for (const [error, expected] of cases) {
      const { unmount } = render(
        <QueryErrorPanel error={error} onRetry={() => {}} />,
      );
      expect(screen.getByText(expected)).toBeInTheDocument();
      unmount();
    }
  });

  it("wires Retry to the caller's refetch", () => {
    const onRetry = vi.fn();
    render(<QueryErrorPanel error={undefined} onRetry={onRetry} />);
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
