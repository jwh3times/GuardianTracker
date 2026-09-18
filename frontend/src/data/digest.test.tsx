import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API, sampleUser } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useDigest } from "./digest";
import { useMembershipRefresh } from "./membershipRefresh";

/**
 * Contract tests for the Digest data-access module (ADR 0023) — key identity,
 * the membership-scoped transport, projection of every status branch
 * (including the first-visit/unavailable states never being confused with an
 * empty `ready` digest), and its membership-refresh invalidation.
 */

function digest(overrides: Record<string, unknown> = {}) {
  return {
    status: "ready",
    visitStartedAt: "2026-09-18T18:00:00Z",
    previousVisitAt: "2026-09-17T12:00:00Z",
    acquired: [],
    ...overrides,
  };
}

function DigestProbe() {
  const { digest: data, isLoading, isError, retry } = useDigest();
  return (
    <div>
      <button onClick={retry}>retry</button>
      <div data-testid="digest">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : !data
              ? "none"
              : `${data.status}|${data.visitStartedAt}|${
                  data.previousVisitAt ?? "-"
                }|[${data.acquired
                  .map(
                    (a) => `${a.itemHash}:${a.name}:${a.type}:${a.icon ?? "-"}`,
                  )
                  .join(",")}]`}
      </div>
    </div>
  );
}

describe("digest query identity", () => {
  it("requests the signed-in membership once for every reader", async () => {
    const sent: string[] = [];
    server.use(
      http.get(`${API}/api/digest/:type/:id`, ({ params }) => {
        sent.push(`${String(params.type)}/${String(params.id)}`);
        return HttpResponse.json(digest());
      }),
    );

    renderWithProviders(
      <>
        <DigestProbe />
        <DigestProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getAllByTestId("digest")[1]).toHaveTextContent("ready"),
    );
    expect(sent).toEqual([
      `${sampleUser.membershipType}/${sampleUser.membershipId}`,
    ]);
  });
});

describe("projection", () => {
  it("projects a ready digest with acquired items", async () => {
    server.use(
      http.get(`${API}/api/digest/:type/:id`, () =>
        HttpResponse.json(
          digest({
            acquired: [
              {
                itemHash: 123,
                name: "Gjallarhorn",
                icon: "/icons/gj.png",
                itemType: "Rocket Launcher",
              },
              {
                itemHash: 456,
                name: "No Icon Item",
                icon: "",
                itemType: "Auto Rifle",
              },
            ],
          }),
        ),
      ),
    );

    renderWithProviders(<DigestProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("digest")).toHaveTextContent(
        "ready|2026-09-18T18:00:00Z|2026-09-17T12:00:00Z|" +
          "[123:Gjallarhorn:Rocket Launcher:/icons/gj.png,456:No Icon Item:Auto Rifle:-]",
      ),
    );
  });

  it("projects a first-visit digest with no previousVisitAt", async () => {
    server.use(
      http.get(`${API}/api/digest/:type/:id`, () =>
        HttpResponse.json({
          status: "first-visit",
          visitStartedAt: "2026-09-18T18:00:00Z",
          acquired: [],
        }),
      ),
    );

    renderWithProviders(<DigestProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("digest")).toHaveTextContent(
        "first-visit|2026-09-18T18:00:00Z|-|[]",
      ),
    );
  });

  it("projects an unavailable digest distinctly from an empty ready digest", async () => {
    server.use(
      http.get(`${API}/api/digest/:type/:id`, () =>
        HttpResponse.json(digest({ status: "unavailable", acquired: [] })),
      ),
    );

    renderWithProviders(<DigestProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("digest")).toHaveTextContent(
        "unavailable|2026-09-18T18:00:00Z|2026-09-17T12:00:00Z|[]",
      ),
    );
  });

  it("treats a null acquired list as empty", async () => {
    server.use(
      http.get(`${API}/api/digest/:type/:id`, () =>
        HttpResponse.json(digest({ acquired: null })),
      ),
    );

    renderWithProviders(<DigestProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("digest")).toHaveTextContent(
        "ready|2026-09-18T18:00:00Z|2026-09-17T12:00:00Z|[]",
      ),
    );
  });
});

describe("retry", () => {
  it("surfaces a failure and refetches on retry", async () => {
    let attempt = 0;
    server.use(
      http.get(`${API}/api/digest/:type/:id`, () => {
        attempt += 1;
        return attempt === 1
          ? HttpResponse.json({ error: "boom" }, { status: 500 })
          : HttpResponse.json(digest());
      }),
    );

    renderWithProviders(<DigestProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("digest")).toHaveTextContent("failed"),
    );

    await userEvent.click(screen.getByText("retry"));

    await waitFor(() =>
      expect(screen.getByTestId("digest")).toHaveTextContent("ready"),
    );
  });
});

function RefreshAndDigestProbe() {
  const { refresh } = useMembershipRefresh();
  const { digest: data } = useDigest();
  return (
    <div>
      <button onClick={refresh}>refresh</button>
      <div data-testid="digest">{data?.status ?? "-"}</div>
    </div>
  );
}

describe("invalidation", () => {
  it("a membership refresh actually refetches the digest", async () => {
    let gets = 0;
    server.use(
      http.get(`${API}/api/digest/:type/:id`, () => {
        gets += 1;
        return HttpResponse.json(
          digest({ status: gets === 1 ? "first-visit" : "ready" }),
        );
      }),
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ success: true, message: "ok" }),
      ),
    );

    renderWithProviders(<RefreshAndDigestProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("digest")).toHaveTextContent("first-visit"),
    );

    await userEvent.click(screen.getByText("refresh"));

    // Only the refetched value proves the root key matches the real entry.
    await waitFor(() =>
      expect(screen.getByTestId("digest")).toHaveTextContent("ready"),
    );
  });
});
