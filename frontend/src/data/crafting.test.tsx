import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API, sampleUser } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useCraftingPatterns } from "./crafting";
import { useMembershipRefresh } from "./membershipRefresh";

/**
 * Contract tests for the Crafting data-access module (ADR 0020) — key
 * identity, the membership-scoped transport, projection to the domain
 * `CraftPattern`, and its membership-refresh invalidation.
 */

function pattern(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    name: `Pattern ${id}`,
    type: "Glaive",
    icon: `/${id}.png`,
    patterns: { cur: 2, max: 5 },
    note: "3 to go",
    source: "Throne World",
    ...overrides,
  };
}

function envelope(items: unknown) {
  return { items, fetchedAt: "2026-09-01T00:00:00Z" };
}

function PatternsProbe() {
  const { patterns, isLoading, isError, retry } = useCraftingPatterns();
  return (
    <div>
      <button onClick={retry}>retry</button>
      <div data-testid="patterns">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : patterns
                .map(
                  (p) =>
                    `${p.id}|${p.name}|${p.type}|${p.icon ?? "no-icon"}|${p.patterns.cur}/${p.patterns.max}|${p.note}|${p.source}`,
                )
                .join(",") || "empty"}
      </div>
    </div>
  );
}

describe("crafting query identity", () => {
  it("requests the signed-in membership once for every reader", async () => {
    const sent: string[] = [];
    server.use(
      http.get(`${API}/api/crafting/:type/:id`, ({ params }) => {
        sent.push(`${String(params.type)}/${String(params.id)}`);
        return HttpResponse.json(envelope([pattern("a")]));
      }),
    );

    renderWithProviders(
      <>
        <PatternsProbe />
        <PatternsProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getAllByTestId("patterns")[1]).toHaveTextContent(
        "Pattern a",
      ),
    );
    expect(sent).toEqual([
      `${sampleUser.membershipType}/${sampleUser.membershipId}`,
    ]);
  });
});

describe("projection", () => {
  it("projects every field, with and without an icon", async () => {
    server.use(
      http.get(`${API}/api/crafting/:type/:id`, () =>
        HttpResponse.json(
          envelope([
            pattern("a"),
            pattern("b", {
              icon: undefined,
              type: "Bow",
              patterns: { cur: 5, max: 5 },
              note: "Done",
              source: "Crafting",
            }),
          ]),
        ),
      ),
    );

    renderWithProviders(<PatternsProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("patterns")).toHaveTextContent(
        "a|Pattern a|Glaive|/a.png|2/5|3 to go|Throne World," +
          "b|Pattern b|Bow|no-icon|5/5|Done|Crafting",
      ),
    );
  });

  it("treats a null item list as empty", async () => {
    server.use(
      http.get(`${API}/api/crafting/:type/:id`, () =>
        HttpResponse.json(envelope(null)),
      ),
    );

    renderWithProviders(<PatternsProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("patterns")).toHaveTextContent("empty"),
    );
  });
});

describe("retry", () => {
  it("surfaces a failure and refetches on retry", async () => {
    let attempt = 0;
    server.use(
      http.get(`${API}/api/crafting/:type/:id`, () => {
        attempt += 1;
        return attempt === 1
          ? HttpResponse.json({ error: "boom" }, { status: 500 })
          : HttpResponse.json(envelope([pattern("a")]));
      }),
    );

    renderWithProviders(<PatternsProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("patterns")).toHaveTextContent("failed"),
    );

    await userEvent.click(screen.getByText("retry"));

    await waitFor(() =>
      expect(screen.getByTestId("patterns")).toHaveTextContent("Pattern a"),
    );
  });
});

function RefreshAndPatternsProbe() {
  const { refresh } = useMembershipRefresh();
  const { patterns } = useCraftingPatterns();
  return (
    <div>
      <button onClick={refresh}>refresh</button>
      <div data-testid="patterns">{patterns.map((p) => p.id).join(",")}</div>
    </div>
  );
}

describe("invalidation", () => {
  it("a membership refresh actually refetches crafting patterns", async () => {
    let gets = 0;
    server.use(
      http.get(`${API}/api/crafting/:type/:id`, () => {
        gets += 1;
        return HttpResponse.json(envelope([pattern(`v${gets}`)]));
      }),
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ success: true, message: "ok" }),
      ),
    );

    renderWithProviders(<RefreshAndPatternsProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("patterns")).toHaveTextContent("v1"),
    );

    await userEvent.click(screen.getByText("refresh"));

    // Only the refetched list proves the root key matches the real entry.
    await waitFor(() =>
      expect(screen.getByTestId("patterns")).toHaveTextContent("v2"),
    );
  });
});
