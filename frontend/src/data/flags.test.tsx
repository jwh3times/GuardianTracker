import { useState } from "react";
import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useFlags } from "../contexts/FlagsContext";
import { useResolvedFlags } from "./flags";

/**
 * Contract tests for the Flags data-access module (ADR 0020) — key identity,
 * projection to the domain `Flag`, the role default, the one-minute reuse
 * window, and a refresh that really re-fetches.
 */

function flag(overrides: Record<string, unknown> = {}) {
  return {
    key: "gated",
    name: "Gated",
    desc: "A gated feature.",
    category: "Power",
    minTier: "alpha",
    enabled: true,
    accessible: false,
    locked: true,
    ...overrides,
  };
}

function ResolvedProbe() {
  const { role, flags, isLoading, refresh } = useResolvedFlags();
  return (
    <div>
      <button onClick={refresh}>refresh</button>
      <div data-testid="resolved">
        {isLoading
          ? "loading"
          : `${role}:${flags
              .map(
                (f) =>
                  `${f.key}|${f.name}|${f.desc}|${f.category}|${f.minTier}|${f.enabled}|${f.accessible}|${f.locked}`,
              )
              .join(",")}`}
      </div>
    </div>
  );
}

/** Reads the same resource through the gating context. */
function ContextProbe() {
  const { role } = useFlags();
  return <div data-testid="context">{role}</div>;
}

describe("flags query identity", () => {
  it("serves the gating context and the module hook from one request", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/flags`, () => {
        requests += 1;
        return HttpResponse.json({ role: "beta", flags: [] });
      }),
    );

    renderWithProviders(
      <>
        <ResolvedProbe />
        <ContextProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getByTestId("resolved")).toHaveTextContent("beta:"),
    );
    expect(screen.getByTestId("context")).toHaveTextContent("beta");
    expect(requests).toBe(1);
  });

  it("reuses resolved flags within a minute rather than asking again", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/flags`, () => {
        requests += 1;
        return HttpResponse.json({ role: "beta", flags: [] });
      }),
    );

    function Toggle() {
      const [open, setOpen] = useState(true);
      return (
        <>
          <button onClick={() => setOpen((v) => !v)}>toggle</button>
          {open && <ResolvedProbe />}
        </>
      );
    }

    renderWithProviders(<Toggle />);
    await waitFor(() =>
      expect(screen.getByTestId("resolved")).toHaveTextContent("beta:"),
    );

    await userEvent.click(screen.getByText("toggle"));
    await userEvent.click(screen.getByText("toggle"));

    expect(screen.getByTestId("resolved")).toHaveTextContent("beta:");
    // The FlagsProvider above keeps one observer mounted throughout, so this
    // guards the probe's own remount — not a second request from the context.
    expect(requests).toBe(1);
  });
});

describe("projection", () => {
  it("projects the role and every flag field", async () => {
    server.use(
      http.get(`${API}/api/flags`, () =>
        HttpResponse.json({
          role: "alpha",
          flags: [
            flag(),
            flag({
              key: "open",
              name: "Open",
              desc: "",
              category: "Completion",
              minTier: "standard",
              accessible: true,
              locked: false,
            }),
          ],
        }),
      ),
    );

    renderWithProviders(<ResolvedProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("resolved")).toHaveTextContent(
        "alpha:gated|Gated|A gated feature.|Power|alpha|true|false|true,open|Open||Completion|standard|true|true|false",
      ),
    );
  });

  it("treats an unrecognised gate tier as standard", async () => {
    server.use(
      http.get(`${API}/api/flags`, () =>
        HttpResponse.json({
          role: "standard",
          flags: [
            flag({ minTier: "legend" }),
            flag({ key: "t", minTier: "toString" }),
          ],
        }),
      ),
    );

    renderWithProviders(<ResolvedProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("resolved")).toHaveTextContent(
        "gated|Gated|A gated feature.|Power|standard|",
      ),
    );
    expect(screen.getByTestId("resolved")).toHaveTextContent(
      "t|Gated|A gated feature.|Power|standard|",
    );
  });

  it("tolerates a response with no flag list", async () => {
    server.use(
      http.get(`${API}/api/flags`, () => HttpResponse.json({ role: "beta" })),
    );

    renderWithProviders(<ResolvedProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("resolved")).toHaveTextContent(/^beta:$/),
    );
  });

  it("reports the standard role until the response arrives", () => {
    server.use(
      // Never resolves: flags are still in flight.
      http.get(`${API}/api/flags`, () => new Promise(() => {})),
    );

    function RoleProbe() {
      const { role } = useResolvedFlags();
      return <div data-testid="role">{role}</div>;
    }

    renderWithProviders(<RoleProbe />);

    expect(screen.getByTestId("role")).toHaveTextContent("standard");
  });
});

describe("refresh", () => {
  it("re-fetches and shows the new role", async () => {
    let role = "standard";
    let requests = 0;
    server.use(
      http.get(`${API}/api/flags`, () => {
        requests += 1;
        return HttpResponse.json({ role, flags: [] });
      }),
    );

    renderWithProviders(<ResolvedProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("resolved")).toHaveTextContent("standard:"),
    );

    role = "beta";
    await userEvent.click(screen.getByText("refresh"));

    // Inside the one-minute window, so only an actual invalidation re-fetches.
    await waitFor(() =>
      expect(screen.getByTestId("resolved")).toHaveTextContent("beta:"),
    );
    expect(requests).toBe(2);
  });
});
