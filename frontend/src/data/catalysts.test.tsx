import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API, sampleUser } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useCatalysts } from "./catalysts";
import { useMembershipRefresh } from "./membershipRefresh";

/**
 * Contract tests for the Catalysts data-access module (ADR 0020) — key
 * identity, the membership-scoped transport, projection to the domain
 * `Catalyst`, and its membership-refresh invalidation.
 */

function catalyst(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    name: `Catalyst ${id}`,
    type: "Hand Cannon",
    icon: `/${id}.png`,
    status: "in-progress",
    obj: { label: "Kills", cur: 10, max: 100 },
    source: "Crucible",
    effect: "Faster reload.",
    ...overrides,
  };
}

function envelope(items: unknown) {
  return { items, fetchedAt: "2026-09-01T00:00:00Z" };
}

function CatalystsProbe() {
  const { catalysts, isLoading, isError, retry } = useCatalysts();
  return (
    <div>
      <button onClick={retry}>retry</button>
      <div data-testid="catalysts">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : catalysts
                .map(
                  (c) =>
                    `${c.id}|${c.name}|${c.type}|${c.icon}|${c.status}|${
                      c.obj
                        ? `${c.obj.label}:${c.obj.cur}/${c.obj.max}`
                        : "no-obj"
                    }|${c.source}|${c.effect ?? "no-effect"}`,
                )
                .join(",") || "empty"}
      </div>
    </div>
  );
}

describe("catalysts query identity", () => {
  it("requests the signed-in membership once for every reader", async () => {
    const sent: string[] = [];
    server.use(
      http.get(`${API}/api/catalysts/:type/:id`, ({ params }) => {
        sent.push(`${String(params.type)}/${String(params.id)}`);
        return HttpResponse.json(envelope([catalyst("a")]));
      }),
    );

    renderWithProviders(
      <>
        <CatalystsProbe />
        <CatalystsProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getAllByTestId("catalysts")[1]).toHaveTextContent(
        "Catalyst a",
      ),
    );
    expect(sent).toEqual([
      `${sampleUser.membershipType}/${sampleUser.membershipId}`,
    ]);
  });
});

describe("projection", () => {
  it("projects every field, with and without an objective", async () => {
    server.use(
      http.get(`${API}/api/catalysts/:type/:id`, () =>
        HttpResponse.json(
          envelope([
            catalyst("a"),
            catalyst("b", {
              status: "complete",
              obj: null,
              type: "Bow",
              effect: "",
            }),
          ]),
        ),
      ),
    );

    renderWithProviders(<CatalystsProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("catalysts")).toHaveTextContent(
        "a|Catalyst a|Hand Cannon|/a.png|in-progress|Kills:10/100|Crucible|Faster reload.," +
          "b|Catalyst b|Bow|/b.png|complete|no-obj|Crucible|",
      ),
    );
  });

  it("treats a null item list as empty", async () => {
    server.use(
      http.get(`${API}/api/catalysts/:type/:id`, () =>
        // A nil Go slice serializes as null.
        HttpResponse.json(envelope(null)),
      ),
    );

    renderWithProviders(<CatalystsProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("catalysts")).toHaveTextContent("empty"),
    );
  });
});

describe("retry", () => {
  it("surfaces a failure and refetches on retry", async () => {
    let attempt = 0;
    server.use(
      http.get(`${API}/api/catalysts/:type/:id`, () => {
        attempt += 1;
        return attempt === 1
          ? HttpResponse.json({ error: "boom" }, { status: 500 })
          : HttpResponse.json(envelope([catalyst("a")]));
      }),
    );

    renderWithProviders(<CatalystsProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("catalysts")).toHaveTextContent("failed"),
    );

    await userEvent.click(screen.getByText("retry"));

    await waitFor(() =>
      expect(screen.getByTestId("catalysts")).toHaveTextContent("Catalyst a"),
    );
  });
});

function RefreshAndCatalystsProbe() {
  const { refresh } = useMembershipRefresh();
  const { catalysts } = useCatalysts();
  return (
    <div>
      <button onClick={refresh}>refresh</button>
      <div data-testid="catalysts">{catalysts.map((c) => c.id).join(",")}</div>
    </div>
  );
}

describe("invalidation", () => {
  it("a membership refresh actually refetches catalysts", async () => {
    let gets = 0;
    server.use(
      http.get(`${API}/api/catalysts/:type/:id`, () => {
        gets += 1;
        return HttpResponse.json(envelope([catalyst(`v${gets}`)]));
      }),
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ success: true, message: "ok" }),
      ),
    );

    renderWithProviders(<RefreshAndCatalystsProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("catalysts")).toHaveTextContent("v1"),
    );

    await userEvent.click(screen.getByText("refresh"));

    // A key head or an invalidation call would still pass with a reshaped root
    // key that matches no real entry. Only the refetched list proves the key.
    await waitFor(() =>
      expect(screen.getByTestId("catalysts")).toHaveTextContent("v2"),
    );
  });
});
