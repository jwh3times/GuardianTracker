import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API, sampleUser } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useSeals } from "./seals";
import { useMembershipRefresh } from "./membershipRefresh";

/**
 * Contract tests for the Seals data-access module (ADR 0020) — key identity,
 * the membership-scoped transport, projection to the domain `Seal` (including
 * the absent-versus-empty objectives rule), and its membership-refresh
 * invalidation.
 */

function seal(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    name: `Seal ${id}`,
    pct: 50,
    gilded: 1,
    left: "3 triumphs left",
    triumphs: [{ label: "Do a thing", done: true, cur: 1, max: 1 }],
    ...overrides,
  };
}

function envelope(items: unknown) {
  return { items, fetchedAt: "2026-09-01T00:00:00Z" };
}

function SealsProbe() {
  const { seals, isLoading, isError, retry } = useSeals();
  return (
    <div>
      <button onClick={retry}>retry</button>
      <div data-testid="seals">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : seals
                .map(
                  (s) =>
                    `${s.id}|${s.name}|${s.pct}|${s.gilded}|${s.left}|[${s.triumphs
                      .map(
                        (t) =>
                          `${t.label}:${t.done}:${t.cur}/${t.max}:${
                            t.objectives === undefined
                              ? "absent"
                              : t.objectives
                                  .map(
                                    (o) =>
                                      `${o.label}=${o.done}:${o.cur}/${o.max}`,
                                  )
                                  .join("+") || "empty"
                          }`,
                      )
                      .join(";")}]`,
                )
                .join(",") || "none"}
      </div>
    </div>
  );
}

describe("seals query identity", () => {
  it("requests the signed-in membership once for every reader", async () => {
    const sent: string[] = [];
    server.use(
      http.get(`${API}/api/seals/:type/:id`, ({ params }) => {
        sent.push(`${String(params.type)}/${String(params.id)}`);
        return HttpResponse.json(envelope([seal("a")]));
      }),
    );

    renderWithProviders(
      <>
        <SealsProbe />
        <SealsProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getAllByTestId("seals")[1]).toHaveTextContent("Seal a"),
    );
    expect(sent).toEqual([
      `${sampleUser.membershipType}/${sampleUser.membershipId}`,
    ]);
  });
});

describe("projection", () => {
  it("projects every seal, triumph and objective field", async () => {
    server.use(
      http.get(`${API}/api/seals/:type/:id`, () =>
        HttpResponse.json(
          envelope([
            seal("a", {
              pct: 80,
              gilded: 0,
              left: "2 triumphs left",
              triumphs: [
                { label: "Flat", done: false, cur: 0, max: 1 },
                {
                  label: "Nested",
                  done: false,
                  cur: 1,
                  max: 2,
                  objectives: [
                    { label: "First", done: true, cur: 3, max: 3 },
                    { label: "Second", done: false, cur: 1, max: 5 },
                  ],
                },
                { label: "Listed", done: true, cur: 1, max: 1, objectives: [] },
              ],
            }),
          ]),
        ),
      ),
    );

    renderWithProviders(<SealsProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("seals")).toHaveTextContent(
        "a|Seal a|80|0|2 triumphs left|[" +
          "Flat:false:0/1:absent;" +
          "Nested:false:1/2:First=true:3/3+Second=false:1/5;" +
          "Listed:true:1/1:empty]",
      ),
    );
  });

  it("treats a null seal list as empty", async () => {
    server.use(
      http.get(`${API}/api/seals/:type/:id`, () =>
        HttpResponse.json(envelope(null)),
      ),
    );

    renderWithProviders(<SealsProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("seals")).toHaveTextContent("none"),
    );
  });
});

describe("retry", () => {
  it("surfaces a failure and refetches on retry", async () => {
    let attempt = 0;
    server.use(
      http.get(`${API}/api/seals/:type/:id`, () => {
        attempt += 1;
        return attempt === 1
          ? HttpResponse.json({ error: "boom" }, { status: 500 })
          : HttpResponse.json(envelope([seal("a")]));
      }),
    );

    renderWithProviders(<SealsProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("seals")).toHaveTextContent("failed"),
    );

    await userEvent.click(screen.getByText("retry"));

    await waitFor(() =>
      expect(screen.getByTestId("seals")).toHaveTextContent("Seal a"),
    );
  });
});

function RefreshAndSealsProbe() {
  const { refresh } = useMembershipRefresh();
  const { seals } = useSeals();
  return (
    <div>
      <button onClick={refresh}>refresh</button>
      <div data-testid="seals">{seals.map((s) => s.id).join(",")}</div>
    </div>
  );
}

describe("invalidation", () => {
  it("a membership refresh actually refetches seals", async () => {
    let gets = 0;
    server.use(
      http.get(`${API}/api/seals/:type/:id`, () => {
        gets += 1;
        return HttpResponse.json(envelope([seal(`v${gets}`)]));
      }),
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ success: true, message: "ok" }),
      ),
    );

    renderWithProviders(<RefreshAndSealsProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("seals")).toHaveTextContent("v1"),
    );

    await userEvent.click(screen.getByText("refresh"));

    // Only the refetched list proves the root key matches the real entry.
    await waitFor(() =>
      expect(screen.getByTestId("seals")).toHaveTextContent("v2"),
    );
  });
});
