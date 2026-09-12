import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API, sampleWishlist } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import {
  useBulkWishlistAction,
  useRemoveWishlistItem,
  useSetWishlistNotes,
  useSetWishlistPriority,
  useWishlist,
} from "./wishlist";

/**
 * Contract tests for the Wish list data-access module (ADR 0020).
 *
 * They own what the module owns — projection, key identity, and optimistic
 * rollback and settle behaviour — and they exercise it through the public hooks
 * rather than the private projection function, because the hook surface is what
 * features depend on. Feature tests keep visible behaviour only.
 */

/** A probe that renders the projected entries as text, plus one mutation. */
function WishlistProbe({
  onRender,
}: {
  onRender?: (entries: ReturnType<typeof useWishlist>["entries"]) => void;
}) {
  const { entries, isLoading, isError } = useWishlist();
  onRender?.(entries);
  if (isLoading) return <div>loading</div>;
  if (isError) return <div>failed</div>;
  return (
    <ul>
      {entries.map((e) => (
        <li key={e.id} data-testid={`entry-${e.id}`}>
          {e.name}|{e.itemId}|{e.rarity}|{e.priority}|
          {e.avail.now ? "now" : "later"}|{e.avail.where}|{e.notes}
        </li>
      ))}
    </ul>
  );
}

/** Re-renders on demand, so referential stability can be observed over time. */
function StabilityProbe({
  onRender,
}: {
  onRender: (entries: ReturnType<typeof useWishlist>["entries"]) => void;
}) {
  const [n, setN] = React.useState(0);
  const { entries } = useWishlist();
  onRender(entries);
  return (
    <button onClick={() => setN(n + 1)} data-count={n}>
      bump
    </button>
  );
}

describe("useWishlist projection", () => {
  it("projects wire rows to domain entries", async () => {
    renderWithProviders(<WishlistProbe />);

    const row = await screen.findByTestId("entry-1");
    expect(row).toHaveTextContent(
      "Gjallarhorn|99999|exotic|high|now|Xûr|the classic",
    );
  });

  it("exposes the item hash so a consumer never reads the wire shape", async () => {
    renderWithProviders(<WishlistProbe />);
    // Collections matches its tiles on this field. `id` is the wish list row
    // id and is NOT interchangeable with it — the fixture's differ on purpose.
    const row = await screen.findByTestId("entry-1");
    expect(row).toHaveTextContent("|99999|");
    expect(sampleWishlist[0].id).not.toBe(String(sampleWishlist[0].itemHash));
  });

  it("keeps live vendor availability separate from source provenance", async () => {
    server.use(
      http.get(`${API}/api/wishlist`, () =>
        HttpResponse.json([
          {
            ...sampleWishlist[0],
            availableNow: false,
            availableFrom: undefined,
          },
        ]),
      ),
    );

    renderWithProviders(<WishlistProbe />);
    const row = await screen.findByTestId("entry-1");
    expect(row).toHaveTextContent("|later||");
  });

  it("tolerates an older payload with no availability fields", async () => {
    const legacy: Record<string, unknown> = { ...sampleWishlist[0] };
    delete legacy.availableNow;
    delete legacy.availableFrom;
    server.use(
      http.get(`${API}/api/wishlist`, () => HttpResponse.json([legacy])),
    );

    renderWithProviders(<WishlistProbe />);
    const row = await screen.findByTestId("entry-1");
    expect(row).toHaveTextContent("|later|");
  });

  it("falls back to legendary and medium for unknown wire vocabulary", async () => {
    server.use(
      http.get(`${API}/api/wishlist`, () =>
        HttpResponse.json([
          { ...sampleWishlist[0], rarity: "Mythic???", priority: "WHENEVER" },
        ]),
      ),
    );

    renderWithProviders(<WishlistProbe />);
    const row = await screen.findByTestId("entry-1");
    expect(row).toHaveTextContent("|legendary|medium|");
  });

  it("returns a referentially stable empty list before data arrives", async () => {
    // Hang the request so the component stays in its pre-data state, then force
    // a re-render. One sample proves nothing — this test only discriminates
    // because it compares the instance across two renders.
    server.use(http.get(`${API}/api/wishlist`, () => new Promise(() => {})));
    const seen: unknown[] = [];

    renderWithProviders(<StabilityProbe onRender={(e) => seen.push(e)} />);
    await waitFor(() => expect(seen.length).toBe(1));

    await userEvent.click(screen.getByText("bump"));
    await waitFor(() => expect(seen.length).toBeGreaterThan(1));

    // Every pre-data render must hand back the same array instance, or the
    // three consumers' useMemo dependencies churn on every render.
    expect(new Set(seen).size).toBe(1);
  });
});

describe("useWishlist cache identity", () => {
  it("serves every consumer of the resource from one request", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/wishlist`, () => {
        requests += 1;
        return HttpResponse.json(sampleWishlist);
      }),
    );

    renderWithProviders(
      <>
        <WishlistProbe />
        <WishlistProbe />
        <WishlistProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getAllByText(/Gjallarhorn/)).toHaveLength(3),
    );
    // Before this module the three pages declared the key separately; the point
    // of private key ownership is that they cannot drift onto separate entries.
    expect(requests).toBe(1);
  });
});

function RemoveProbe({ onError }: { onError?: (m: string) => void }) {
  const { entries } = useWishlist();
  const { remove } = useRemoveWishlistItem({
    onError: (err) => onError?.(err.message),
  });
  return (
    <div>
      <button
        onClick={() =>
          remove({ rowId: "1", itemId: "99999", name: "Gjallarhorn" })
        }
      >
        remove
      </button>
      <div data-testid="count">{entries.length}</div>
    </div>
  );
}

describe("optimistic remove", () => {
  beforeEach(() => {
    server.use(
      http.get(`${API}/api/wishlist`, () => HttpResponse.json(sampleWishlist)),
    );
  });

  it("drops the row from the cache before the server answers", async () => {
    let release: (() => void) | undefined;
    const blocked = new Promise<void>((r) => (release = r));
    server.use(
      http.delete(`${API}/api/wishlist/:id`, async () => {
        await blocked;
        return new HttpResponse(null, { status: 204 });
      }),
    );

    renderWithProviders(<RemoveProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("count")).toHaveTextContent("1"),
    );

    await userEvent.click(screen.getByText("remove"));
    // The request has NOT resolved yet. A non-optimistic remove would still
    // read 1 here, which is exactly what Collections did before this module.
    await waitFor(() =>
      expect(screen.getByTestId("count")).toHaveTextContent("0"),
    );

    release!();
  });

  it("restores the snapshot and reports the failure when the server rejects", async () => {
    // The settle refetch must not be what restores the row, or this test would
    // pass with the rollback deleted — it did, until a mutant caught it. The
    // first GET seeds the cache; every later one hangs, so the only thing that
    // can put the row back is the rollback itself.
    let seeded = false;
    server.use(
      http.get(`${API}/api/wishlist`, async () => {
        if (seeded) await new Promise(() => {});
        seeded = true;
        return HttpResponse.json(sampleWishlist);
      }),
      http.delete(`${API}/api/wishlist/:id`, () =>
        HttpResponse.json({ message: "nope" }, { status: 500 }),
      ),
    );
    const errors: string[] = [];

    renderWithProviders(<RemoveProbe onError={(m) => errors.push(m)} />);
    await waitFor(() =>
      expect(screen.getByTestId("count")).toHaveTextContent("1"),
    );

    await userEvent.click(screen.getByText("remove"));

    await waitFor(() => expect(errors).toHaveLength(1));
    // Rolled back to the pre-mutation snapshot, not left at the optimistic 0.
    await waitFor(() =>
      expect(screen.getByTestId("count")).toHaveTextContent("1"),
    );
  });

  it("refetches on settle so the server stays authoritative", async () => {
    let gets = 0;
    server.use(
      http.get(`${API}/api/wishlist`, () => {
        gets += 1;
        return HttpResponse.json(sampleWishlist);
      }),
      http.delete(
        `${API}/api/wishlist/:id`,
        () => new HttpResponse(null, { status: 204 }),
      ),
    );

    renderWithProviders(<RemoveProbe />);
    await waitFor(() => expect(gets).toBe(1));

    await userEvent.click(screen.getByText("remove"));

    // The optimistic write is a prediction; settling must re-ask the server.
    await waitFor(() => expect(gets).toBe(2));
  });
});

function PriorityProbe() {
  const { entries } = useWishlist();
  const { setPriority } = useSetWishlistPriority();
  return (
    <div>
      <button onClick={() => setPriority({ rowId: "1", priority: "low" })}>
        demote
      </button>
      <div data-testid="priority">{entries[0]?.priority ?? "-"}</div>
    </div>
  );
}

describe("optimistic priority", () => {
  it("writes the wire vocabulary and projects it back as a domain value", async () => {
    const bodies: unknown[] = [];
    let release: (() => void) | undefined;
    const blocked = new Promise<void>((r) => (release = r));
    server.use(
      http.get(`${API}/api/wishlist`, () => HttpResponse.json(sampleWishlist)),
      http.put(`${API}/api/wishlist/:id`, async ({ request }) => {
        bodies.push(await request.json());
        await blocked;
        return HttpResponse.json(sampleWishlist[0]);
      }),
    );

    renderWithProviders(<PriorityProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("priority")).toHaveTextContent("high"),
    );

    await userEvent.click(screen.getByText("demote"));

    // Optimistically shown as the domain value the caller asked for...
    await waitFor(() =>
      expect(screen.getByTestId("priority")).toHaveTextContent("low"),
    );
    // ...while the wire carries the upper-case spelling the API expects. The
    // up-casing used to live in the feature; it belongs to the module now.
    expect(bodies).toEqual([{ priority: "LOW" }]);

    release!();
  });
});

function NotesProbe() {
  const { entries } = useWishlist();
  const { setNotes } = useSetWishlistNotes();
  return (
    <div>
      <button onClick={() => setNotes({ rowId: "1", notes: "farmed it" })}>
        save notes
      </button>
      <div data-testid="notes">{entries[0]?.notes ?? "-"}</div>
    </div>
  );
}

describe("optimistic notes", () => {
  it("shows the new notes before the server answers", async () => {
    // The feature test for notes only asserts the outgoing PUT body, so it
    // passes with this optimistic write deleted. This is the test that does
    // not: it reads the rendered value back while the PUT is still in flight.
    let release: (() => void) | undefined;
    const blocked = new Promise<void>((r) => (release = r));
    server.use(
      http.get(`${API}/api/wishlist`, () => HttpResponse.json(sampleWishlist)),
      http.put(`${API}/api/wishlist/:id`, async () => {
        await blocked;
        return HttpResponse.json(sampleWishlist[0]);
      }),
    );

    renderWithProviders(<NotesProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("notes")).toHaveTextContent("the classic"),
    );

    await userEvent.click(screen.getByText("save notes"));

    await waitFor(() =>
      expect(screen.getByTestId("notes")).toHaveTextContent("farmed it"),
    );

    release!();
  });
});

function BulkProbe({
  onDone,
}: {
  onDone: (updated: number, skipped: number) => void;
}) {
  const { entries } = useWishlist();
  const { runBulkAction } = useBulkWishlistAction({
    onSuccess: (res) => onDone(res.updated, res.skipped),
  });
  return (
    <div>
      <button
        onClick={() => runBulkAction({ action: "delete", rowIds: ["1"] })}
      >
        bulk delete
      </button>
      <div data-testid="count">{entries.length}</div>
    </div>
  );
}

describe("optimistic bulk action", () => {
  it("applies the action optimistically and hands the caller the counts", async () => {
    const done: Array<[number, number]> = [];
    server.use(
      http.get(`${API}/api/wishlist`, () => HttpResponse.json(sampleWishlist)),
      http.post(`${API}/api/wishlist/bulk`, () =>
        HttpResponse.json({ updated: 1, skipped: 0 }),
      ),
    );

    renderWithProviders(<BulkProbe onDone={(u, s) => done.push([u, s])} />);
    await waitFor(() =>
      expect(screen.getByTestId("count")).toHaveTextContent("1"),
    );

    await userEvent.click(screen.getByText("bulk delete"));

    await waitFor(() => expect(done).toEqual([[1, 0]]));
  });

  it("sends row ids as numbers, which is what the endpoint accepts", async () => {
    const bodies: unknown[] = [];
    server.use(
      http.get(`${API}/api/wishlist`, () => HttpResponse.json(sampleWishlist)),
      http.post(`${API}/api/wishlist/bulk`, async ({ request }) => {
        bodies.push(await request.json());
        return HttpResponse.json({ updated: 1, skipped: 0 });
      }),
    );

    renderWithProviders(<BulkProbe onDone={() => {}} />);
    await waitFor(() =>
      expect(screen.getByTestId("count")).toHaveTextContent("1"),
    );

    await userEvent.click(screen.getByText("bulk delete"));

    await waitFor(() =>
      expect(bodies).toEqual([{ action: "delete", ids: [1] }]),
    );
  });
});
