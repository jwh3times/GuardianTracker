import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { QueryClient } from "@tanstack/react-query";
import { server, API, sampleCollections, sampleUser } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useCollections, useCollectionsSummary } from "./collections";
import { useMembershipRefresh } from "./membershipRefresh";

/**
 * Contract tests for the Collections and membership-refresh data-access
 * modules (ADR 0020) — key identity, the variant split, the caller-composed
 * gate, and the refresh fan-out.
 *
 * Projection is NOT retested here. `toCollections` and `toCollectionsSummary`
 * stay in `lib/collectionsView.ts` with the ADR 0018 survivor tests that own
 * them; duplicating those assertions would be the "tests exercise the new seam
 * rather than duplicating its data" failure AGENTS.md names.
 */

function SummaryProbe({ enabled }: { enabled?: boolean }) {
  const { view } = useCollectionsSummary(
    enabled === undefined ? undefined : { enabled },
  );
  return <div data-testid="summary">{view ? view.summary.length : "none"}</div>;
}

function FullProbe() {
  const { view, isLoading, isError, retry } = useCollections();
  return (
    <div>
      <button onClick={retry}>retry</button>
      <div data-testid="full">
        {isLoading ? "loading" : isError ? "failed" : (view?.fetchedAt ?? "-")}
      </div>
    </div>
  );
}

describe("collections query identity", () => {
  it("serves every counts-only consumer from one request", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/collections/:type/:id`, ({ request }) => {
        requests += 1;
        expect(new URL(request.url).searchParams.get("include")).toBeNull();
        return HttpResponse.json(collectionsPayload());
      }),
    );

    renderWithProviders(
      <>
        <SummaryProbe />
        <SummaryProbe />
        <SummaryProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getAllByTestId("summary")[0]).toHaveTextContent("4"),
    );
    // Dashboard, Settings and the onboarding tour previously declared this
    // query separately; private key ownership is what stops them drifting apart.
    expect(requests).toBe(1);
  });

  it("keeps the two variants on separate cache entries", async () => {
    const seen: (string | null)[] = [];
    server.use(
      http.get(`${API}/api/collections/:type/:id`, ({ request }) => {
        seen.push(new URL(request.url).searchParams.get("include"));
        return HttpResponse.json(collectionsPayload());
      }),
    );

    renderWithProviders(
      <>
        <SummaryProbe />
        <FullProbe />
      </>,
    );

    await waitFor(() => expect(seen).toHaveLength(2));
    // The full variant must actually ask for every item. Sharing one entry
    // would silently serve the counts-only payload to the item surfaces.
    expect(new Set(seen)).toEqual(new Set([null, "all"]));
  });
});

function RefreshProbe() {
  const { refresh, isRefreshing } = useMembershipRefresh();
  return (
    <div>
      <button onClick={refresh}>refresh</button>
      <div data-testid="pending">{isRefreshing ? "yes" : "no"}</div>
    </div>
  );
}

describe("membership routing", () => {
  it("addresses the signed-in membership, type before id", async () => {
    // Every other handler here matches the wildcard `:type/:id`, so a swapped,
    // hard-coded or dropped segment would route identically and pass. This is
    // the test that reads the path values back.
    const paths: string[] = [];
    server.use(
      http.get(`${API}/api/collections/:type/:id`, ({ params }) => {
        paths.push(`${String(params.type)}/${String(params.id)}`);
        return HttpResponse.json(collectionsPayload());
      }),
    );

    renderWithProviders(<SummaryProbe />);

    await waitFor(() => expect(paths).toHaveLength(1));
    expect(paths[0]).toBe(
      `${sampleUser.membershipType}/${sampleUser.membershipId}`,
    );
  });

  it("addresses the same membership when refreshing", async () => {
    const paths: string[] = [];
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(collectionsPayload()),
      ),
      http.post(`${API}/api/collections/:type/:id/refresh`, ({ params }) => {
        paths.push(`${String(params.type)}/${String(params.id)}`);
        return HttpResponse.json({ success: true, message: "ok" });
      }),
    );

    renderWithProviders(<RefreshProbe />);
    await userEvent.click(screen.getByText("refresh"));

    await waitFor(() => expect(paths).toHaveLength(1));
    expect(paths[0]).toBe(
      `${sampleUser.membershipType}/${sampleUser.membershipId}`,
    );
  });
});

describe("caller-composed gate", () => {
  it("does not fetch while the caller's gate is closed", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () => {
        requests += 1;
        return HttpResponse.json(collectionsPayload());
      }),
    );

    renderWithProviders(<SummaryProbe enabled={false} />);

    // The onboarding tour holds this closed until saved preferences load.
    // Settle the microtask queue so a request would have been issued by now.
    await waitFor(() =>
      expect(screen.getByTestId("summary")).toHaveTextContent("none"),
    );
    expect(requests).toBe(0);
  });

  it("fetches once the caller's gate opens", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () => {
        requests += 1;
        return HttpResponse.json(collectionsPayload());
      }),
    );

    renderWithProviders(<SummaryProbe enabled={true} />);

    await waitFor(() => expect(requests).toBe(1));
  });
});

describe("retry", () => {
  it("refetches after a failure", async () => {
    let attempt = 0;
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () => {
        attempt += 1;
        return attempt === 1
          ? HttpResponse.json({ message: "nope" }, { status: 500 })
          : HttpResponse.json(collectionsPayload());
      }),
    );

    renderWithProviders(<FullProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("full")).toHaveTextContent("failed"),
    );

    await userEvent.click(screen.getByText("retry"));

    await waitFor(() => expect(attempt).toBe(2));
  });
});

describe("membership refresh", () => {
  it("invalidates every membership-scoped resource ADR 0018 names", async () => {
    const refetched = new Set<string>();
    const track = (name: string) => () => {
      refetched.add(name);
      return HttpResponse.json(collectionsPayload());
    };
    server.use(
      http.get(`${API}/api/collections/:type/:id`, track("collections")),
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ refreshed: true }),
      ),
    );

    const client = trackingClient();
    renderWithProviders(
      <>
        <FullProbe />
        <RefreshProbe />
      </>,
      { client },
    );
    await waitFor(() => expect(refetched.has("collections")).toBe(true));
    client.invalidated.length = 0;

    await userEvent.click(screen.getByText("refresh"));

    await waitFor(() => expect(client.invalidated.length).toBeGreaterThan(0));
    // Dropping any of these silently leaves a stale page after a refresh —
    // the exact failure ADR 0018 cares about.
    expect([...client.invalidated].sort()).toEqual([
      "catalysts",
      "characters",
      "collections",
      "crafting",
      "seals",
      "weekly",
    ]);
  });

  it("does not invalidate anything when the refresh fails", async () => {
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () =>
        HttpResponse.json(collectionsPayload()),
      ),
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ message: "nope" }, { status: 500 }),
      ),
    );

    const client = trackingClient();
    renderWithProviders(<RefreshProbe />, { client });

    await userEvent.click(screen.getByText("refresh"));

    await waitFor(() =>
      expect(screen.getByTestId("pending")).toHaveTextContent("no"),
    );
    expect(client.invalidated).toEqual([]);
  });
});

/**
 * A QueryClient that records the key families it was asked to invalidate.
 * Asserting on invalidation directly is the only way to see the fan-out: the
 * unmigrated resources have no mounted observer here, so nothing refetches.
 */
function trackingClient() {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  const invalidated: string[] = [];
  const original = client.invalidateQueries.bind(client);
  client.invalidateQueries = ((filters?: { queryKey?: readonly unknown[] }) => {
    const head = filters?.queryKey?.[0];
    if (typeof head === "string") invalidated.push(head);
    return original(filters as never);
  }) as typeof client.invalidateQueries;
  return Object.assign(client, { invalidated });
}

/** The shared fixture; this module owns transport, not payload shape. */
function collectionsPayload() {
  return sampleCollections;
}
