import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse, delay } from "msw";
import { API, server } from "../../test/testServer";
import { renderWithProviders } from "../../test/renderWithProviders";
import { RollTargetSection } from "./RollTargetSection";
import type { GTItem, PerkColumn } from "../../types/design";

beforeEach(() => {
  localStorage.clear();
});

const weapon: GTItem = {
  id: "500",
  name: "Fatebringer",
  type: "Hand Cannon",
  slot: "Kinetic",
  rarity: "legendary",
  acquisitionSources: [],
  availableNow: false,
  collected: true,
  desc: "",
};

const weaponPerkColumns: PerkColumn[] = [
  { role: "barrel", label: "Barrel", perks: ["Arrowhead Brake"] },
];

function flagPayload(overrides: Record<string, unknown> = {}) {
  return {
    role: "standard",
    flags: [
      {
        key: "god-roll",
        name: "God-roll insights",
        desc: "Per-weapon owned rolls and god-roll matching.",
        category: "Power",
        minTier: "alpha",
        enabled: true,
        accessible: true,
        locked: false,
        ...overrides,
      },
    ],
  };
}

function setFlag(overrides: Record<string, unknown>) {
  server.use(
    http.get(`${API}/api/flags`, () =>
      HttpResponse.json(flagPayload(overrides)),
    ),
  );
}

function rt(overrides: Record<string, unknown> = {}) {
  return {
    id: "1",
    itemHash: 500,
    anyWeapon: false,
    wanted: true,
    perks: ["Arrowhead Brake"],
    notes: "",
    dateAdded: "2026-09-01T00:00:00Z",
    ...overrides,
  };
}

function emptyMatches(overrides: Record<string, unknown> = {}) {
  return { wanted: [], unwanted: [], unmatchedTargets: [], ...overrides };
}

/** Counts requests to the two roll-target endpoints, so a test can assert the
 * live-Bungie matches query (and, incidentally, the list query) was never
 * issued when the section shouldn't render anything. */
function countRequests() {
  const counts = { list: 0, matches: 0 };
  server.use(
    http.get(`${API}/api/rolltargets`, () => {
      counts.list += 1;
      return HttpResponse.json([]);
    }),
    http.get(`${API}/api/rolltargets/matches`, () => {
      counts.matches += 1;
      return HttpResponse.json(emptyMatches());
    }),
  );
  return counts;
}

function renderSection(
  extra: Partial<React.ComponentProps<typeof RollTargetSection>> = {},
) {
  return renderWithProviders(
    <RollTargetSection
      item={weapon}
      perkColumns={weaponPerkColumns}
      perksLoading={false}
      {...extra}
    />,
  );
}

describe("flag gating", () => {
  it("renders nothing and fetches nothing when the flag is not enabled", async () => {
    setFlag({ enabled: false, accessible: false, locked: false });
    const counts = countRequests();

    renderSection();

    await waitFor(() =>
      expect(screen.queryByText("Your roll targets")).not.toBeInTheDocument(),
    );
    expect(counts.list).toBe(0);
    expect(counts.matches).toBe(0);
  });

  it("shows the locked upsell without promising farming, and fetches nothing", async () => {
    setFlag({ enabled: true, accessible: false, locked: true });
    const counts = countRequests();

    renderSection();

    await waitFor(() =>
      expect(
        screen.getByText(
          (_, node) =>
            node?.tagName === "DIV" &&
            (node.textContent ?? "").trim() ===
              "Roll targets is an Alpha feature.",
        ),
      ).toBeInTheDocument(),
    );
    expect(
      screen.getByText(/See which of your copies match the rolls/),
    ).toBeInTheDocument();
    expect(screen.queryByText(/farm/i)).not.toBeInTheDocument();
    expect(counts.list).toBe(0);
    expect(counts.matches).toBe(0);
  });

  it("does not fetch matches until the item resolves as a weapon with perk columns", async () => {
    setFlag({});
    const counts = countRequests();

    renderSection({ perkColumns: [], perksLoading: false });

    // Give any (incorrect) fetch a chance to fire.
    await new Promise((resolve) => setTimeout(resolve, 20));
    expect(counts.list).toBe(0);
    expect(counts.matches).toBe(0);
    expect(screen.queryByText("Your roll targets")).not.toBeInTheDocument();
  });

  it("does not fetch while perks are still loading, even if columns are empty so far", async () => {
    setFlag({});
    const counts = countRequests();

    renderSection({ perkColumns: [], perksLoading: true });

    await new Promise((resolve) => setTimeout(resolve, 20));
    expect(counts.list).toBe(0);
    expect(counts.matches).toBe(0);
  });

  it("fetches once accessible and resolved as a weapon", async () => {
    setFlag({});
    const counts = countRequests();

    renderSection();

    await waitFor(() => expect(counts.matches).toBe(1));
  });
});

describe("copies you own that match", () => {
  it("highlights the target's perks and badges an any-weapon target that matched this weapon", async () => {
    setFlag({});
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([
          rt({ id: "1", itemHash: 500, perks: ["Arrowhead Brake"] }),
          rt({
            id: "2",
            itemHash: null,
            anyWeapon: true,
            perks: ["Explosive Payload"],
          }),
        ]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          emptyMatches({
            wanted: [
              {
                targetId: "1",
                itemHash: 500,
                instanceId: "inst-a",
                perks: ["Arrowhead Brake", "Explosive Payload"],
                targetPerks: ["Arrowhead Brake"],
              },
              {
                targetId: "2",
                itemHash: 500,
                instanceId: "inst-a",
                perks: ["Arrowhead Brake", "Explosive Payload"],
                targetPerks: ["Explosive Payload"],
              },
            ],
          }),
        ),
      ),
    );

    renderSection();

    await waitFor(() => expect(screen.getByText("Copy 1")).toBeInTheDocument());
    const copy1 = screen
      .getByText("Copy 1")
      .closest(".gt-rt-copy") as HTMLElement;
    expect(
      within(copy1).getByText("Arrowhead Brake", { selector: ".gt-chip" }),
    ).toHaveAttribute("data-highlight", "true");
    expect(
      within(copy1).getByText("Explosive Payload", { selector: ".gt-chip" }),
    ).toHaveAttribute("data-highlight", "true");
    expect(
      within(copy1).getByText("Any weapon: Explosive Payload"),
    ).toBeInTheDocument();
  });
});

describe("still chasing", () => {
  it("shows this weapon's unmatched target but excludes an unmatched any-weapon target", async () => {
    setFlag({});
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([
          rt({ id: "1", itemHash: 500, perks: ["Arrowhead Brake"] }),
        ]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          emptyMatches({
            unmatchedTargets: [
              rt({ id: "1", itemHash: 500, perks: ["Arrowhead Brake"] }),
              rt({
                id: "2",
                itemHash: null,
                anyWeapon: true,
                perks: ["Rampage"],
              }),
            ],
          }),
        ),
      ),
    );

    renderSection();

    await waitFor(() =>
      expect(screen.getByText("Still chasing")).toBeInTheDocument(),
    );
    expect(screen.getByText("Arrowhead Brake")).toBeInTheDocument();
    expect(screen.queryByText("Rampage")).not.toBeInTheDocument();
  });
});

describe("marked unwanted", () => {
  it("lists this weapon's unwanted match informationally, never with dismantle wording", async () => {
    setFlag({});
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([
          rt({ id: "1", itemHash: 500, wanted: false, perks: ["Rampage"] }),
        ]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          emptyMatches({
            unwanted: [
              {
                targetId: "1",
                itemHash: 500,
                instanceId: "inst-u",
                perks: ["Rampage", "Firefly"],
                targetPerks: ["Rampage"],
              },
            ],
          }),
        ),
      ),
    );

    renderSection();

    await waitFor(() =>
      expect(
        screen.getByText("Matches a roll you marked unwanted (1)"),
      ).toBeInTheDocument(),
    );
    expect(screen.queryByText(/dismantle/i)).not.toBeInTheDocument();
    expect(
      screen.queryByText("Rampage", { selector: ".gt-rt-copy .gt-chip" }),
    ).not.toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("button", {
        name: /Matches a roll you marked unwanted/,
      }),
    );
    expect(screen.getByText("Copy 1")).toBeInTheDocument();
  });
});

describe("no targets", () => {
  it("shows a quiet line and always links to Manage roll targets", async () => {
    setFlag({});
    server.use(
      http.get(`${API}/api/rolltargets`, () => HttpResponse.json([])),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(emptyMatches()),
      ),
    );

    renderSection();

    await waitFor(() =>
      expect(
        screen.getByText("No roll targets for this weapon."),
      ).toBeInTheDocument(),
    );
    const link = screen.getByRole("link", { name: "Manage roll targets" });
    expect(link).toHaveAttribute("href", "/rolls");
  });
});

describe("loading", () => {
  it("shows a small checking-your-copies line while matches are in flight", async () => {
    setFlag({});
    server.use(
      http.get(`${API}/api/rolltargets`, () => HttpResponse.json([])),
      http.get(`${API}/api/rolltargets/matches`, async () => {
        await delay(40);
        return HttpResponse.json(emptyMatches());
      }),
    );

    renderSection();

    expect(screen.getByText("Checking your copies…")).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByText("No roll targets for this weapon."),
      ).toBeInTheDocument(),
    );
  });
});

describe("match failure", () => {
  it("shows this weapon's saved targets neutrally with Retry for a non-reauth failure", async () => {
    setFlag({});
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([
          rt({ id: "1", itemHash: 500, perks: ["Arrowhead Brake"] }),
        ]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          { error: "unavailable", code: "OWNED_ROLLS_UNAVAILABLE" },
          { status: 503 },
        ),
      ),
    );

    renderSection();

    await waitFor(() =>
      expect(screen.getByText("Match status unknown")).toBeInTheDocument(),
    );
    expect(screen.getByText("Arrowhead Brake")).toBeInTheDocument();
    expect(screen.queryByText("Still chasing")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Reconnect" }),
    ).not.toBeInTheDocument();
  });

  it("offers Reconnect for a BUNGIE_REAUTH_REQUIRED failure", async () => {
    setFlag({});
    server.use(
      http.get(`${API}/api/rolltargets`, () =>
        HttpResponse.json([rt({ id: "1", itemHash: 500 })]),
      ),
      http.get(`${API}/api/rolltargets/matches`, () =>
        HttpResponse.json(
          { error: "reauth", code: "BUNGIE_REAUTH_REQUIRED" },
          { status: 401 },
        ),
      ),
    );

    renderSection();

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
