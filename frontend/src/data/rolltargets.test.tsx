import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import {
  useBulkDeleteRollTargets,
  useDeleteAllRollTargets,
  useImportRollTargets,
  useRemoveRollTarget,
  useRollTargetMatches,
  useRollTargets,
  useUpdateRollTargetNotes,
} from "./rolltargets";
import { useMembershipRefresh } from "./membershipRefresh";

/**
 * Contract tests for the Roll targets data-access module (ADR 0020, slice
 * 5b): query identity/shape, projection (including the explicit
 * any-weapon-target null and the unresolved-perk import shape), optimistic
 * mutation/rollback behavior, and membership-refresh invalidation.
 */

function target(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    itemHash: 100,
    anyWeapon: false,
    wanted: true,
    perks: ["Arrowhead Brake", "Explosive Payload"],
    notes: "",
    dateAdded: "2026-09-01T00:00:00Z",
    ...overrides,
  };
}

function matchReport(overrides: Record<string, unknown> = {}) {
  return { wanted: [], unwanted: [], unmatchedTargets: [], ...overrides };
}

function TargetsProbe() {
  const { targets, isLoading, isError, retry } = useRollTargets();
  return (
    <div>
      <button onClick={retry}>retry-targets</button>
      <div data-testid="targets">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : targets
                .map(
                  (t) =>
                    `${t.id}|${t.itemHash ?? "any"}|${t.anyWeapon}|${t.wanted}|${t.perks.join(",")}|${t.notes}`,
                )
                .join(";") || "empty"}
      </div>
    </div>
  );
}

function MatchesProbe() {
  const { matches, isLoading, isError, retry } = useRollTargetMatches();
  return (
    <div>
      <button onClick={retry}>retry-matches</button>
      <div data-testid="matches">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : matches
              ? `wanted:${matches.wanted.length}|unwanted:${matches.unwanted.length}|unmatched:${matches.unmatchedTargets.length}|best:${matches.unmatchedTargets[0]?.bestCopy?.itemHash ?? "none"}:${matches.unmatchedTargets[0]?.bestCopy?.matchedPerks.join(",") ?? "none"}`
              : "no-data"}
      </div>
    </div>
  );
}

describe("query identity and projection", () => {
  it("requests the flat list endpoint and projects an any-weapon target's null hash", async () => {
    let gets = 0;
    server.use(
      http.get(`${API}/api/rolltargets`, () => {
        gets += 1;
        return HttpResponse.json([
          target("1"),
          target("2", { itemHash: null, anyWeapon: true, perks: ["Rampage"] }),
        ]);
      }),
    );

    renderWithProviders(
      <>
        <TargetsProbe />
        <TargetsProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getAllByTestId("targets")[1]).toHaveTextContent(
        "1|100|false|true|Arrowhead Brake,Explosive Payload|;2|any|true|true|Rampage|",
      ),
    );
    // One request shared by both readers proves the query key is stable.
    expect(gets).toBe(1);
  });

  it("treats missing perks/notes as empty, not undefined", async () => {
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([
          {
            id: "1",
            itemHash: 100,
            anyWeapon: false,
            wanted: true,
            dateAdded: "2026-09-01T00:00:00Z",
          },
        ]),
      ),
    );

    renderWithProviders(<TargetsProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("targets")).toHaveTextContent(
        "1|100|false|true||",
      ),
    );
  });

  it("projects the match report's three buckets", async () => {
    server.use(
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          matchReport({
            wanted: [
              {
                targetId: "1",
                itemHash: 100,
                instanceId: "inst-1",
                perks: ["Arrowhead Brake"],
                targetPerks: ["Arrowhead Brake"],
              },
            ],
            unmatchedTargets: [
              target("2", {
                bestCopy: {
                  itemHash: 100,
                  instanceId: "near-1",
                  perks: ["Arrowhead Brake", "Firefly"],
                  matchedPerks: ["Arrowhead Brake"],
                },
              }),
            ],
          }),
        ),
      ),
    );

    renderWithProviders(<MatchesProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("matches")).toHaveTextContent(
        "wanted:1|unwanted:0|unmatched:1|best:100:Arrowhead Brake",
      ),
    );
  });

  it("surfaces a matches failure independently of the target list", async () => {
    server.use(
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json({ error: "boom" }, { status: 503 }),
      ),
    );

    renderWithProviders(
      <>
        <TargetsProbe />
        <MatchesProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getByTestId("matches")).toHaveTextContent("failed"),
    );
    // The target list is unaffected by the matches failure.
    await waitFor(() =>
      expect(screen.getByTestId("targets")).not.toHaveTextContent("loading"),
    );
    expect(screen.getByTestId("targets")).not.toHaveTextContent("failed");
  });
});

function GatedProbe({ enabled }: { enabled: boolean }) {
  const { targets } = useRollTargets({ enabled });
  const { matches } = useRollTargetMatches({ enabled });
  return (
    <div data-testid="gated">
      {targets.length}|{matches ? "loaded" : "none"}
    </div>
  );
}

describe("caller-supplied enabled gate", () => {
  it("fetches neither query while the caller's gate is closed", async () => {
    let listRequests = 0;
    let matchesRequests = 0;
    server.use(
      http.get(`${API}/api/rolltargets`, () => {
        listRequests += 1;
        return HttpResponse.json([target("1")]);
      }),
      http.get(`${API}/api/rolltargets/matches`, () => {
        matchesRequests += 1;
        return HttpResponse.json(matchReport());
      }),
    );

    renderWithProviders(<GatedProbe enabled={false} />);

    // Settle the microtask queue so a request would have been issued by now.
    await waitFor(() =>
      expect(screen.getByTestId("gated")).toHaveTextContent("0|none"),
    );
    expect(listRequests).toBe(0);
    expect(matchesRequests).toBe(0);
  });

  it("fetches both once the caller's gate opens", async () => {
    let listRequests = 0;
    let matchesRequests = 0;
    server.use(
      http.get(`${API}/api/rolltargets`, () => {
        listRequests += 1;
        return HttpResponse.json([target("1")]);
      }),
      http.get(`${API}/api/rolltargets/matches`, () => {
        matchesRequests += 1;
        return HttpResponse.json(matchReport());
      }),
    );

    renderWithProviders(<GatedProbe enabled={true} />);

    await waitFor(() => expect(listRequests).toBe(1));
    await waitFor(() => expect(matchesRequests).toBe(1));
  });
});

function NotesEditProbe() {
  const { targets } = useRollTargets();
  const { setNotes } = useUpdateRollTargetNotes();
  return (
    <div>
      <button onClick={() => setNotes({ id: "1", notes: "updated" })}>
        save-notes
      </button>
      <div data-testid="notes">{targets[0]?.notes ?? ""}</div>
    </div>
  );
}

describe("notes mutation", () => {
  it("updates optimistically, then settles on the server's value", async () => {
    // Stateful so the settle-time invalidation's refetch reflects the write,
    // matching the real endpoint — a static GET would otherwise stomp the
    // optimistic value back to "original" the instant it refetches.
    let notes = "original";
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([target("1", { notes })]),
      ),
      http.patch(`${API}/api/rolltargets/:id`, () => {
        notes = "updated";
        return HttpResponse.json(target("1", { notes }));
      }),
    );

    renderWithProviders(<NotesEditProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("notes")).toHaveTextContent("original"),
    );

    await userEvent.click(screen.getByText("save-notes"));
    // Optimistic write lands before the PATCH resolves.
    await waitFor(() =>
      expect(screen.getByTestId("notes")).toHaveTextContent("updated"),
    );
  });

  it("rolls back on failure", async () => {
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([target("1", { notes: "original" })]),
      ),
      http.patch(`${API}/api/rolltargets/:id`, () =>
        HttpResponse.json({ error: "nope" }, { status: 500 }),
      ),
    );

    renderWithProviders(<NotesEditProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("notes")).toHaveTextContent("original"),
    );

    await userEvent.click(screen.getByText("save-notes"));
    await waitFor(() =>
      expect(screen.getByTestId("notes")).toHaveTextContent("original"),
    );
  });
});

function RemoveProbe() {
  const { targets } = useRollTargets();
  const { remove } = useRemoveRollTarget();
  return (
    <div>
      <button onClick={() => remove({ id: "1" })}>remove</button>
      <div data-testid="ids">{targets.map((t) => t.id).join(",")}</div>
    </div>
  );
}

describe("remove mutation", () => {
  it("removes the row optimistically", async () => {
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([target("1"), target("2")]),
      ),
      http.delete(
        `${API}/api/rolltargets/:id`,
        () => new HttpResponse(null, { status: 204 }),
      ),
    );

    renderWithProviders(<RemoveProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("ids")).toHaveTextContent("1,2"),
    );

    await userEvent.click(screen.getByText("remove"));
    await waitFor(() =>
      expect(screen.getByTestId("ids")).toHaveTextContent("2"),
    );
  });
});

function BulkDeleteProbe() {
  const { targets } = useRollTargets();
  const { bulkDelete } = useBulkDeleteRollTargets();
  const { deleteAll } = useDeleteAllRollTargets();
  return (
    <div>
      <button onClick={() => bulkDelete(["1", "2"])}>bulk-delete</button>
      <button onClick={deleteAll}>delete-all</button>
      <div data-testid="ids">
        {targets.map((t) => t.id).join(",") || "empty"}
      </div>
    </div>
  );
}

describe("bulk delete and delete all", () => {
  it("bulk delete removes only the named ids, optimistically", async () => {
    // Stateful so the settle-time invalidation's refetch agrees with the
    // optimistic write instead of restoring the deleted rows.
    let store = [target("1"), target("2"), target("3")];
    server.use(
      http.get(`${API}/api/rolltargets`, () => HttpResponse.json(store)),
      http.post(`${API}/api/rolltargets/bulk`, async ({ request }) => {
        const body = (await request.json()) as { ids: string[] };
        const ids = new Set(body.ids);
        store = store.filter((r) => !ids.has(r.id));
        return HttpResponse.json({ deleted: ids.size, skipped: 0 });
      }),
    );

    renderWithProviders(<BulkDeleteProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("ids")).toHaveTextContent("1,2,3"),
    );

    await userEvent.click(screen.getByText("bulk-delete"));
    await waitFor(() =>
      expect(screen.getByTestId("ids")).toHaveTextContent("3"),
    );
  });

  it("delete all clears every row, optimistically", async () => {
    let store = [target("1"), target("2")];
    server.use(
      http.get(`${API}/api/rolltargets`, () => HttpResponse.json(store)),
      http.post(`${API}/api/rolltargets/bulk`, () => {
        const deleted = store.length;
        store = [];
        return HttpResponse.json({ deleted, skipped: 0 });
      }),
    );

    renderWithProviders(<BulkDeleteProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("ids")).toHaveTextContent("1,2"),
    );

    await userEvent.click(screen.getByText("delete-all"));
    await waitFor(() =>
      expect(screen.getByTestId("ids")).toHaveTextContent("empty"),
    );
  });
});

function ImportProbe() {
  const { importDIM } = useImportRollTargets({
    onSuccess: (report) => {
      document.title = `imported:${report.imported}`;
    },
  });
  return <button onClick={() => importDIM("dim text")}>import</button>;
}

describe("DIM import", () => {
  it("sends the raw text body and projects an unresolved-perk line", async () => {
    let receivedBody = "";
    let receivedContentType = "";
    server.use(
      http.post(`${API}/api/rolltargets/import`, async ({ request }) => {
        receivedBody = await request.text();
        receivedContentType = request.headers.get("content-type") ?? "";
        return HttpResponse.json({
          imported: 1,
          counts: { imported: 1, "unresolved perk": 1 },
          lines: [
            {
              line: 1,
              outcome: "imported",
              wanted: true,
              itemHash: 100,
              perks: ["Arrowhead Brake"],
            },
            {
              line: 2,
              outcome: "unresolved perk",
              detail: 'this weapon cannot roll "Fake Perk"',
              unresolved: {
                perkHash: 999,
                perkName: "Fake Perk",
                reason: "not-in-pool",
              },
              wanted: true,
            },
          ],
        });
      }),
    );

    renderWithProviders(<ImportProbe />);
    await userEvent.click(screen.getByText("import"));

    await waitFor(() => expect(receivedBody).toBe("dim text"));
    expect(receivedContentType).toContain("text/plain");
    await waitFor(() => expect(document.title).toBe("imported:1"));
  });
});

function RefreshAndTargetsProbe() {
  const { refresh } = useMembershipRefresh();
  const { targets } = useRollTargets();
  return (
    <div>
      <button onClick={refresh}>refresh</button>
      <div data-testid="targets">{targets.map((t) => t.id).join(",")}</div>
    </div>
  );
}

describe("invalidation", () => {
  it("a membership refresh actually refetches roll targets", async () => {
    let gets = 0;
    server.use(
      http.get(`${API}/api/rolltargets`, () => {
        gets += 1;
        return HttpResponse.json([target(`v${gets}`)]);
      }),
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ success: true, message: "ok" }),
      ),
    );

    renderWithProviders(<RefreshAndTargetsProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("targets")).toHaveTextContent("v1"),
    );

    await userEvent.click(screen.getByText("refresh"));

    await waitFor(() =>
      expect(screen.getByTestId("targets")).toHaveTextContent("v2"),
    );
  });
});
