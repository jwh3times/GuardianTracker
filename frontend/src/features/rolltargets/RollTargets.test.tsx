import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { API, server } from "../../test/testServer";
import { renderWithProviders } from "../../test/renderWithProviders";
import { RollTargets } from "./RollTargets";

beforeEach(() => {
  localStorage.clear();
});

function renderPage() {
  return renderWithProviders(<RollTargets />, { route: "/rolls" });
}

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

function emptyMatches(overrides: Record<string, unknown> = {}) {
  return { wanted: [], unwanted: [], unmatchedTargets: [], ...overrides };
}

/** A `?include=all`-shaped collections payload naming one item by hash. */
function collectionsWith(
  entries: Record<string, { name: string; collected: boolean }>,
) {
  const items: Record<string, unknown> = {};
  const collectedHashes: string[] = [];
  for (const [hash, { name, collected }] of Object.entries(entries)) {
    items[hash] = {
      itemHash: hash,
      name,
      description: "",
      icon: "",
      itemType: "Hand Cannon",
      tierType: 5,
      rarity: "Legendary",
      acquisitionSources: [],
      isExotic: false,
    };
    if (collected) collectedHashes.push(hash);
  }
  return {
    tree: [],
    summary: {
      weapons: { total: 1, collected: collectedHashes.length },
      armor: { total: 0, collected: 0 },
      exotics: { total: 0, collected: 0 },
      cosmetics: { total: 0, collected: 0 },
    },
    fetchedAt: "2026-09-01T00:00:00Z",
    items,
    collectedHashes,
    availableNow: {},
  };
}

function useCollectionsFixture(
  entries: Record<string, { name: string; collected: boolean }>,
) {
  server.use(
    http.get(`${API}/api/collections/:type/:id`, () =>
      HttpResponse.json(collectionsWith(entries)),
    ),
  );
}

describe("empty state", () => {
  it("explains roll targets and points to import when there are none", async () => {
    server.use(
      http.get(`${API}/api/rolltargets`, () => HttpResponse.json([])),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(emptyMatches()),
      ),
    );

    renderPage();

    await waitFor(() =>
      expect(
        screen.getByText("Save the rolls you're chasing"),
      ).toBeInTheDocument(),
    );
    expect(
      screen.getByText(/import a DIM-format wish list above/i),
    ).toBeInTheDocument();
  });
});

describe("sections, in order", () => {
  it("renders Still chasing, You have it, then the collapsed unwanted disclosure", async () => {
    useCollectionsFixture({ "100": { name: "Fatebringer", collected: true } });
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([
          target("1"),
          target("2", {
            itemHash: null,
            anyWeapon: true,
            perks: ["Rampage"],
            wanted: false,
          }),
        ]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          emptyMatches({
            unmatchedTargets: [target("1")],
            unwanted: [
              {
                targetId: "2",
                itemHash: 200,
                instanceId: "inst-unwanted",
                perks: ["Rampage", "Firefly"],
                targetPerks: ["Rampage"],
              },
            ],
          }),
        ),
      ),
    );

    renderPage();

    await waitFor(() =>
      expect(
        screen.getByRole("heading", { name: "Still chasing" }),
      ).toBeInTheDocument(),
    );
    const headings = screen
      .getAllByRole("heading", { level: 2 })
      .map((h) => h.textContent);
    const chasingIdx = headings.findIndex((h) => h === "Still chasing");
    const wantedIdx = headings.findIndex((h) => h === "You have it");
    const unwantedIdx = headings.findIndex((h) =>
      h?.startsWith("Matches a roll you marked unwanted"),
    );
    expect(chasingIdx).toBeGreaterThanOrEqual(0);
    expect(wantedIdx).toBeGreaterThan(chasingIdx);
    expect(unwantedIdx).toBeGreaterThan(wantedIdx);

    // Collapsed by default: the unwanted card's copy is not in the DOM yet.
    expect(
      screen.queryByText("Rampage", { selector: ".gt-rt-copy .gt-chip" }),
    ).not.toBeInTheDocument();
    const disclosure = screen.getByRole("button", {
      name: /Matches a roll you marked unwanted/,
    });
    expect(disclosure).toHaveAttribute("aria-expanded", "false");
    await userEvent.click(disclosure);
    expect(disclosure).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("Copy 1")).toBeInTheDocument();
  });

  it("frames an any-weapon unmatched target as 'Any weapon with X + Y', never absence", async () => {
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([
          target("1", {
            itemHash: null,
            anyWeapon: true,
            perks: ["Rampage", "Firefly"],
          }),
        ]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          emptyMatches({
            unmatchedTargets: [
              target("1", {
                itemHash: null,
                anyWeapon: true,
                perks: ["Rampage", "Firefly"],
              }),
            ],
          }),
        ),
      ),
    );

    renderPage();

    await waitFor(() =>
      expect(
        screen.getByText("Any weapon with Rampage + Firefly"),
      ).toBeInTheDocument(),
    );
  });
});

describe("default filter", () => {
  it("hides an unacquired, non-wishlisted weapon and reports how many, until Show all", async () => {
    useCollectionsFixture({
      "100": { name: "Fatebringer", collected: true },
      "200": { name: "Midnight Coup", collected: false },
    });
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([
          target("1", { itemHash: 100 }),
          target("2", { itemHash: 200 }),
        ]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          emptyMatches({
            unmatchedTargets: [
              target("1", { itemHash: 100 }),
              target("2", { itemHash: 200 }),
            ],
          }),
        ),
      ),
    );

    renderPage();

    await waitFor(() =>
      expect(screen.getAllByText("Fatebringer").length).toBeGreaterThan(0),
    );
    expect(screen.queryByText("Midnight Coup")).not.toBeInTheDocument();
    expect(screen.getByText(/1 roll target hidden/)).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Show all" }));
    await waitFor(() =>
      expect(screen.getAllByText("Midnight Coup").length).toBeGreaterThan(0),
    );
  });

  it("always shows an any-weapon target regardless of the filter", async () => {
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([target("1", { itemHash: null, anyWeapon: true })]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          emptyMatches({
            unmatchedTargets: [
              target("1", { itemHash: null, anyWeapon: true }),
            ],
          }),
        ),
      ),
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Any weapon")).toBeInTheDocument(),
    );
  });

  it("wish-listed weapons pass the default filter too", async () => {
    useCollectionsFixture({ "100": { name: "Fatebringer", collected: false } });
    server.use(
      http.get(`${API}/api/wishlist`, () =>
        HttpResponse.json([
          {
            id: "w1",
            itemHash: 100,
            name: "Fatebringer",
            itemType: "Hand Cannon",
            rarity: "Legendary",
            icon: "",
            priority: "HIGH",
            notes: "",
            acquisitionSources: [],
            availableNow: false,
            dateAdded: "2026-09-01T00:00:00Z",
          },
        ]),
      ),
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([target("1", { itemHash: 100 })]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          emptyMatches({ unmatchedTargets: [target("1", { itemHash: 100 })] }),
        ),
      ),
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getAllByText("Fatebringer").length).toBeGreaterThan(0),
    );
    expect(screen.queryByText(/roll target.*hidden/)).not.toBeInTheDocument();
  });
});

describe("search", () => {
  it("filters still-chasing groups by weapon or perk name", async () => {
    useCollectionsFixture({
      "100": { name: "Fatebringer", collected: true },
      "200": { name: "Midnight Coup", collected: true },
    });
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([
          target("1", { itemHash: 100, perks: ["Arrowhead Brake"] }),
          target("2", { itemHash: 200, perks: ["Rampage"] }),
        ]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          emptyMatches({
            unmatchedTargets: [
              target("1", { itemHash: 100, perks: ["Arrowhead Brake"] }),
              target("2", { itemHash: 200, perks: ["Rampage"] }),
            ],
          }),
        ),
      ),
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getAllByText("Fatebringer").length).toBeGreaterThan(0),
    );
    expect(screen.getAllByText("Midnight Coup").length).toBeGreaterThan(0);

    await userEvent.type(
      screen.getByRole("searchbox", {
        name: "Search still-chasing roll targets",
      }),
      "rampage",
    );
    await waitFor(() =>
      expect(screen.queryByText("Fatebringer")).not.toBeInTheDocument(),
    );
    expect(screen.getAllByText("Midnight Coup").length).toBeGreaterThan(0);
  });
});

describe("You have it", () => {
  it("labels several owned copies Copy 1, Copy 2 and highlights the target's perks", async () => {
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([target("1", { perks: ["Arrowhead Brake"] })]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          emptyMatches({
            wanted: [
              {
                targetId: "1",
                itemHash: 100,
                instanceId: "inst-a",
                perks: ["Arrowhead Brake", "Explosive Payload"],
                targetPerks: ["Arrowhead Brake"],
              },
              {
                targetId: "1",
                itemHash: 100,
                instanceId: "inst-b",
                perks: ["Corkscrew Rifling", "Firefly"],
                targetPerks: ["Arrowhead Brake"],
              },
            ],
          }),
        ),
      ),
    );

    renderPage();
    await waitFor(() => expect(screen.getByText("Copy 1")).toBeInTheDocument());
    expect(screen.getByText("Copy 2")).toBeInTheDocument();

    const copy1 = screen
      .getByText("Copy 1")
      .closest(".gt-rt-copy") as HTMLElement;
    const highlighted = within(copy1).getByText("Arrowhead Brake");
    expect(highlighted).toHaveAttribute("data-highlight", "true");
    const notHighlighted = within(copy1).getByText("Explosive Payload");
    expect(notHighlighted).toHaveAttribute("data-highlight", "false");
  });
});

describe("match failure", () => {
  it("lists every target neutrally, still resolving its weapon name, and offers Retry for a non-reauth failure", async () => {
    useCollectionsFixture({ "100": { name: "Fatebringer", collected: true } });
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([target("1")]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          { error: "unavailable", code: "OWNED_ROLLS_UNAVAILABLE" },
          { status: 503 },
        ),
      ),
    );

    renderPage();

    await waitFor(() =>
      expect(
        screen.getAllByText("Match status unknown").length,
      ).toBeGreaterThan(0),
    );
    // Collections is a separate, independently-successful query — a match
    // report failure must not fall back to a bare hash label.
    expect(screen.getAllByText("Fatebringer").length).toBeGreaterThan(0);
    expect(
      screen.queryByRole("heading", { name: "Still chasing" }),
    ).not.toBeInTheDocument();
    const retry = screen.getByRole("button", { name: "Retry" });
    expect(retry).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Reconnect" }),
    ).not.toBeInTheDocument();
  });

  it("offers Reconnect for a BUNGIE_REAUTH_REQUIRED failure", async () => {
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([target("1")]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          { error: "reauth", code: "BUNGIE_REAUTH_REQUIRED" },
          { status: 401 },
        ),
      ),
    );

    renderPage();

    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: "Reconnect" }),
      ).toBeInTheDocument(),
    );
    expect(
      screen.queryByRole("button", { name: "Retry" }),
    ).not.toBeInTheDocument();
  });
});

describe("management", () => {
  it("edits notes inline", async () => {
    useCollectionsFixture({ "100": { name: "Fatebringer", collected: true } });
    // Stateful so the settle-time invalidation's refetch agrees with the
    // optimistic write instead of restoring the blank notes.
    let notes = "";
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([target("1", { notes })]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          emptyMatches({ unmatchedTargets: [target("1", { notes })] }),
        ),
      ),
      http.patch(`${API}/api/rolltargets/:id`, () => {
        notes = "chasing this for PvE";
        return HttpResponse.json(target("1", { notes }));
      }),
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Add notes")).toBeInTheDocument(),
    );
    await userEvent.click(screen.getByText("Add notes"));
    await userEvent.type(
      screen.getByRole("textbox", { name: /Notes for/ }),
      "chasing this for PvE",
    );
    await userEvent.click(screen.getByText("Save"));
    await waitFor(() =>
      expect(screen.getByText('"chasing this for PvE"')).toBeInTheDocument(),
    );
  });

  it("bulk-deletes selected targets", async () => {
    useCollectionsFixture({ "100": { name: "Fatebringer", collected: true } });
    let store = [target("1"), target("2")];
    server.use(
      http.get(`${API}/api/rolltargets`, () => HttpResponse.json(store)),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(emptyMatches({ unmatchedTargets: store })),
      ),
      http.post(`${API}/api/rolltargets/bulk`, async ({ request }) => {
        const body = (await request.json()) as { ids: string[] };
        const ids = new Set(body.ids);
        store = store.filter((r) => !ids.has(r.id));
        return HttpResponse.json({ deleted: ids.size, skipped: 0 });
      }),
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getAllByText("Delete").length).toBeGreaterThan(0),
    );

    await userEvent.click(screen.getByRole("button", { name: "Select" }));
    const checkboxes = screen.getAllByRole("checkbox");
    await userEvent.click(checkboxes[0]);
    await userEvent.click(screen.getByText("Delete selected"));

    await waitFor(() =>
      expect(screen.getByText("1 deleted")).toBeInTheDocument(),
    );
  });

  it("requires explicit confirmation before deleting all", async () => {
    let store = [target("1"), target("2")];
    server.use(
      http.get(`${API}/api/rolltargets`, () => HttpResponse.json(store)),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(emptyMatches({ unmatchedTargets: store })),
      ),
      http.post(`${API}/api/rolltargets/bulk`, () => {
        const deleted = store.length;
        store = [];
        return HttpResponse.json({ deleted, skipped: 0 });
      }),
    );

    renderPage();
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: "Delete all…" }),
      ).toBeInTheDocument(),
    );

    await userEvent.click(screen.getByRole("button", { name: "Delete all…" }));
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    // Not deleted yet — only the confirmation is showing.
    expect(
      screen.queryByText("All roll targets deleted"),
    ).not.toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("button", { name: "Yes, delete all" }),
    );
    await waitFor(() =>
      expect(screen.getByText("All roll targets deleted")).toBeInTheDocument(),
    );
  });
});

describe("DIM import", () => {
  it("shows counts first, then non-imported lines grouped by outcome; skipped lines are not listed", async () => {
    server.use(
      http.get(`${API}/api/rolltargets`, () => HttpResponse.json([])),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(emptyMatches()),
      ),
      http.post(`${API}/api/rolltargets/import`, () =>
        HttpResponse.json({
          imported: 1,
          counts: { imported: 1, skipped: 1, "unresolved perk": 1 },
          lines: [
            { line: 1, outcome: "skipped", wanted: true },
            {
              line: 2,
              outcome: "imported",
              wanted: true,
              itemHash: 100,
              perks: ["Arrowhead Brake"],
            },
            {
              line: 3,
              outcome: "unresolved perk",
              wanted: true,
              unresolved: {
                perkHash: 999,
                perkName: "Fake Perk",
                reason: "not-in-pool",
              },
            },
          ],
        }),
      ),
    );

    renderPage();
    await waitFor(() =>
      expect(
        screen.getByLabelText("Paste a DIM-format wish list"),
      ).toBeInTheDocument(),
    );

    await userEvent.type(
      screen.getByLabelText("Paste a DIM-format wish list"),
      "dim text",
    );
    await userEvent.click(screen.getByRole("button", { name: "Import" }));

    await waitFor(() =>
      expect(screen.getByText("1 imported · 1 unresolved")).toBeInTheDocument(),
    );
    expect(
      screen.getByText(/This weapon can't roll perk "Fake Perk"/),
    ).toBeInTheDocument();
    // The skipped comment/header line is never listed.
    expect(screen.getByText("Unresolved perk (1)")).toBeInTheDocument();
    expect(screen.queryByText(/Line 1:/)).not.toBeInTheDocument();
  });

  it("also imports a file chosen through the file picker", async () => {
    server.use(
      http.get(`${API}/api/rolltargets`, () => HttpResponse.json([])),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(emptyMatches()),
      ),
      http.post(`${API}/api/rolltargets/import`, async ({ request }) => {
        const body = await request.text();
        return HttpResponse.json({
          imported: body === "file contents" ? 1 : 0,
          counts: { imported: body === "file contents" ? 1 : 0 },
          lines: [],
        });
      }),
    );

    renderPage();
    await waitFor(() =>
      expect(
        screen.getByLabelText("Import a DIM-format file"),
      ).toBeInTheDocument(),
    );

    const file = new File(["file contents"], "rolls.txt", {
      type: "text/plain",
    });
    await userEvent.upload(
      screen.getByLabelText("Import a DIM-format file"),
      file,
    );

    await waitFor(() =>
      expect(screen.getByText("1 imported")).toBeInTheDocument(),
    );
  });
});
