import { useState } from "react";
import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { server, API } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useItemPerks, useItemView } from "./items";

/**
 * Contract tests for the Items data-access module (ADR 0020) — key identity
 * per item hash, the lazy gate, projection to domain types, and the
 * never-stale caching both lookups rely on.
 */

function PerksProbe({ hash }: { hash: string | undefined }) {
  const { perkColumns, catalysts, isLoading } = useItemPerks(hash);
  return (
    <div data-testid={`perks-${hash ?? "none"}`}>
      {isLoading
        ? "loading"
        : `${(perkColumns ?? [])
            .map((c) => `${c.role}:${c.label}:${c.perks.join("+")}`)
            .join(",")}|${(catalysts ?? [])
            .map((c) => `${c.name}:${c.description}`)
            .join(",")}`}
    </div>
  );
}

function perksPayload(hash: string) {
  return {
    itemHash: hash,
    perkColumns: [{ role: "barrel", label: "Barrel", perks: ["Full Bore"] }],
    catalysts: [{ name: `Catalyst ${hash}`, description: "Faster" }],
  };
}

describe("item perks", () => {
  it("projects columns and catalysts, keyed per hash", async () => {
    const asked: string[] = [];
    server.use(
      http.get(`${API}/api/items/:hash/perks`, ({ params }) => {
        asked.push(String(params.hash));
        return HttpResponse.json(perksPayload(String(params.hash)));
      }),
    );

    renderWithProviders(
      <>
        <PerksProbe hash="200" />
        <PerksProbe hash="200" />
        <PerksProbe hash="300" />
      </>,
    );

    await waitFor(() =>
      expect(screen.getByTestId("perks-300")).toHaveTextContent(
        "barrel:Barrel:Full Bore|Catalyst 300:Faster",
      ),
    );
    await waitFor(() =>
      expect(screen.getAllByTestId("perks-200")[0]).toHaveTextContent(
        "barrel:Barrel:Full Bore|Catalyst 200:Faster",
      ),
    );
    // Two readers of one hash share an entry; a different hash gets its own.
    expect([...asked].sort()).toEqual(["200", "300"]);
  });

  it("tolerates a payload with no catalysts", async () => {
    server.use(
      http.get(`${API}/api/items/:hash/perks`, () =>
        HttpResponse.json({
          itemHash: "200",
          perkColumns: [{ role: "trait", label: "Trait 1", perks: ["Frenzy"] }],
        }),
      ),
    );

    renderWithProviders(<PerksProbe hash="200" />);

    await waitFor(() =>
      expect(screen.getByTestId("perks-200")).toHaveTextContent(
        "trait:Trait 1:Frenzy|",
      ),
    );
  });

  it("requests nothing while no item is selected", async () => {
    const asked: string[] = [];
    server.use(
      http.get(`${API}/api/items/:hash/perks`, ({ params }) => {
        asked.push(String(params.hash));
        return HttpResponse.json(perksPayload(String(params.hash)));
      }),
    );

    renderWithProviders(
      <>
        <PerksProbe hash={undefined} />
        <PerksProbe hash="200" />
      </>,
    );

    // A gated query is pending but not loading, so the rendered text cannot
    // carry this — the request log is what discriminates. Waiting on the
    // selected probe proves both had their chance to fire.
    await waitFor(() =>
      expect(screen.getByTestId("perks-200")).toHaveTextContent("Full Bore"),
    );
    expect(asked).toEqual(["200"]);
  });

  it("does not refetch when the drawer reopens the same item", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/items/:hash/perks`, ({ params }) => {
        requests += 1;
        return HttpResponse.json(perksPayload(String(params.hash)));
      }),
    );

    function Toggle() {
      const [open, setOpen] = useState(true);
      return (
        <>
          <button onClick={() => setOpen((v) => !v)}>toggle</button>
          {open && <PerksProbe hash="200" />}
        </>
      );
    }

    renderWithProviders(<Toggle />);
    await waitFor(() =>
      expect(screen.getByTestId("perks-200")).toHaveTextContent("Full Bore"),
    );

    await userEvent.click(screen.getByText("toggle"));
    await userEvent.click(screen.getByText("toggle"));

    expect(screen.getByTestId("perks-200")).toHaveTextContent("Full Bore");
    // Perk pools are static per manifest version; a remount must be served
    // from cache rather than asking the server again.
    expect(requests).toBe(1);
  });
});

function ViewProbe({ hash }: { hash: string | null }) {
  const { item, isError } = useItemView(hash);
  return (
    <div data-testid="view">
      {isError
        ? "failed"
        : item
          ? `${item.id}|${item.name}|${item.type}|${item.rarity}|${item.desc}|${item.icon}|${String(item.viewOnly)}|${String(item.collected)}`
          : "none"}
    </div>
  );
}

function viewPayload(overrides: Record<string, unknown> = {}) {
  return {
    itemHash: "999",
    name: "Vendor Mod",
    icon: "/mod.png",
    itemType: "Mod",
    tierType: 5,
    rarity: "Exotic",
    description: "A mod.",
    ...overrides,
  };
}

describe("item view", () => {
  it("projects to a view-only item for the requested hash", async () => {
    const asked: string[] = [];
    server.use(
      http.get(`${API}/api/items/:hash`, ({ params }) => {
        asked.push(String(params.hash));
        return HttpResponse.json(viewPayload());
      }),
    );

    renderWithProviders(<ViewProbe hash="999" />);

    await waitFor(() =>
      expect(screen.getByTestId("view")).toHaveTextContent(
        "999|Vendor Mod|Mod|exotic|A mod.|/mod.png|true|false",
      ),
    );
    expect(asked).toEqual(["999"]);
  });

  it("falls back to legendary for an unrecognised rarity", async () => {
    server.use(
      http.get(`${API}/api/items/:hash`, () =>
        HttpResponse.json(viewPayload({ rarity: "Mythic" })),
      ),
    );

    renderWithProviders(<ViewProbe hash="999" />);

    await waitFor(() =>
      expect(screen.getByTestId("view")).toHaveTextContent("|legendary|"),
    );
  });

  it("surfaces a missing item as an error", async () => {
    server.use(
      http.get(`${API}/api/items/:hash`, () =>
        HttpResponse.json({ error: "not found" }, { status: 404 }),
      ),
    );

    renderWithProviders(<ViewProbe hash="999" />);

    await waitFor(() =>
      expect(screen.getByTestId("view")).toHaveTextContent("failed"),
    );
  });

  it("keys each hash separately", async () => {
    server.use(
      http.get(`${API}/api/items/:hash`, ({ params }) =>
        HttpResponse.json(
          viewPayload({
            itemHash: String(params.hash),
            name: `Item ${String(params.hash)}`,
          }),
        ),
      ),
    );

    renderWithProviders(
      <>
        <div data-testid="a">
          <ViewProbe hash="111" />
        </div>
        <div data-testid="b">
          <ViewProbe hash="222" />
        </div>
      </>,
    );

    await waitFor(() =>
      expect(screen.getByTestId("b")).toHaveTextContent("222|Item 222"),
    );
    // Without the hash in the key, the second deep link would read the first
    // one's cached item.
    expect(screen.getByTestId("a")).toHaveTextContent("111|Item 111");
  });

  it("requests nothing when there is no hash to resolve", async () => {
    const asked: string[] = [];
    server.use(
      http.get(`${API}/api/items/:hash`, ({ params }) => {
        asked.push(String(params.hash));
        return HttpResponse.json(viewPayload());
      }),
    );

    renderWithProviders(
      <>
        <ViewProbe hash={null} />
        <div data-testid="resolved">
          <ViewProbe hash="999" />
        </div>
      </>,
    );

    await waitFor(() =>
      expect(screen.getByTestId("resolved")).toHaveTextContent("Vendor Mod"),
    );
    expect(asked).toEqual(["999"]);
  });

  it("does not refetch a resolved item on remount", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/items/:hash`, () => {
        requests += 1;
        return HttpResponse.json(viewPayload());
      }),
    );

    function Toggle() {
      const [open, setOpen] = useState(true);
      return (
        <>
          <button onClick={() => setOpen((v) => !v)}>toggle</button>
          {open && <ViewProbe hash="999" />}
        </>
      );
    }

    renderWithProviders(<Toggle />);
    await waitFor(() =>
      expect(screen.getByTestId("view")).toHaveTextContent("Vendor Mod"),
    );

    await userEvent.click(screen.getByText("toggle"));
    await userEvent.click(screen.getByText("toggle"));

    expect(screen.getByTestId("view")).toHaveTextContent("Vendor Mod");
    expect(requests).toBe(1);
  });

  it("does not retry a missing item even when the client would", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/items/:hash`, () => {
        requests += 1;
        return HttpResponse.json({ error: "not found" }, { status: 404 });
      }),
    );
    // The shared test client disables retries globally, which would hide the
    // module's own `retry: false`. This client retries by default.
    const client = new QueryClient({
      defaultOptions: { queries: { retry: 1, retryDelay: 0 } },
    });

    renderWithProviders(<ViewProbe hash="999" />, { client });

    await waitFor(() =>
      expect(screen.getByTestId("view")).toHaveTextContent("failed"),
    );
    expect(requests).toBe(1);
  });
});
