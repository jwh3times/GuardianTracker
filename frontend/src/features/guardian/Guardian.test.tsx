import { describe, expect, it } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { Route, Routes } from "react-router";
import { API, sampleUser, server } from "../../test/testServer";
import { renderWithProviders } from "../../test/renderWithProviders";
import { useCharacters } from "../../contexts/CharacterContext";
import { Guardian } from "./Guardian";

const character = {
  characterId: "2305843009263456789",
  classType: 1,
  className: "Hunter",
  raceName: "Awoken",
  light: 550,
  emblemPath: "https://www.bungie.net/emblem.png",
  emblemBackgroundPath: "https://www.bungie.net/emblem-bg.jpg",
  dateLastPlayed: "2026-09-01T00:00:00Z",
};

function renderGuardian() {
  return renderWithProviders(
    <Routes>
      <Route path="/guardian" element={<Guardian />} />
    </Routes>,
    { route: "/guardian" },
  );
}

describe("Guardian equipment", () => {
  it("renders the selected Guardian identity and grouped equipment", async () => {
    const requested: string[] = [];
    const activityRequested: string[] = [];
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([character]),
      ),
      http.get(
        `${API}/api/characters/:type/:id/:characterId/equipment`,
        ({ params }) => {
          requested.push(
            `${String(params.type)}/${String(params.id)}/${String(params.characterId)}`,
          );
          return HttpResponse.json({
            characterId: character.characterId,
            state: "ready",
            fetchedAt: new Date().toISOString(),
            items: [
              {
                itemHash: "10",
                slot: "Kinetic",
                group: "Weapons",
                name: "Fatebringer",
                itemType: "Hand Cannon",
                rarity: "Legendary",
                icon: "/fatebringer.png",
                resolved: true,
                power: 550,
              },
              {
                itemHash: "20",
                slot: "Helmet",
                group: "Armor",
                name: "Celestial Nighthawk",
                itemType: "Helmet",
                rarity: "Exotic",
                icon: "/nighthawk.png",
                resolved: true,
                power: 551,
              },
              {
                itemHash: "30",
                slot: "Ghost",
                group: "Equipment",
                name: "Generalist Shell",
                itemType: "Ghost",
                rarity: "Common",
                icon: "",
                resolved: true,
              },
              {
                itemHash: "40",
                slot: "Artifact",
                group: "Equipment",
                name: "Unknown item",
                itemType: "Artifact",
                icon: "",
                resolved: false,
              },
            ],
          });
        },
      ),
      http.get(
        `${API}/api/characters/:type/:id/:characterId/activity-history`,
        ({ params }) => {
          activityRequested.push(
            `${String(params.type)}/${String(params.id)}/${String(params.characterId)}`,
          );
          return HttpResponse.json({
            characterId: character.characterId,
            state: "ready",
            fetchedAt: new Date().toISOString(),
            activities: [
              {
                activityHash: "3637500864",
                name: "The Insight Terminus",
                occurredAt: "2026-09-13T22:15:00Z",
                duration: "11m 12s",
                privateMatch: true,
                resolved: true,
              },
            ],
          });
        },
      ),
    );

    const { container } = renderGuardian();

    expect(
      await screen.findByRole("heading", { name: "Hunter" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Awoken Guardian")).toBeInTheDocument();
    expect(
      screen.getByText("Collections remain shared", { exact: false }),
    ).toBeInTheDocument();
    expect(
      await screen.findByRole("heading", { name: "Weapons" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Armor" })).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Equipment" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Fatebringer")).toBeInTheDocument();
    expect(screen.getByText("Celestial Nighthawk")).toBeInTheDocument();
    expect(screen.getByText("Unknown item").closest("article")).toHaveAttribute(
      "data-rarity",
      "unresolved",
    );
    expect(screen.getByLabelText("551 Power")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Recent activity" }),
    ).toBeInTheDocument();
    expect(screen.getByText("The Insight Terminus")).toBeInTheDocument();
    expect(screen.getByText("11m 12s played")).toBeInTheDocument();
    expect(screen.getByText("Private match")).toBeInTheDocument();
    expect(
      container.querySelector('time[datetime="2026-09-13T22:15:00Z"]'),
    ).not.toBeNull();
    expect(screen.getAllByText(/^Updated /)).toHaveLength(3);
    expect(container.querySelector(".gt-guardian-hero-art")).toHaveAttribute(
      "src",
      character.emblemBackgroundPath,
    );
    expect(container.querySelector(".gt-tile-img")).toHaveAttribute(
      "src",
      "https://www.bungie.net/fatebringer.png",
    );
    expect(requested).toEqual([
      `${sampleUser.membershipType}/${sampleUser.membershipId}/${character.characterId}`,
    ]);
    expect(activityRequested).toEqual([
      `${sampleUser.membershipType}/${sampleUser.membershipId}/${character.characterId}`,
    ]);
  });

  it("distinguishes an unavailable component from an empty loadout", async () => {
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([character]),
      ),
      http.get(`${API}/api/characters/:type/:id/:characterId/equipment`, () =>
        HttpResponse.json({
          characterId: character.characterId,
          state: "unavailable",
          items: [],
          fetchedAt: "2026-09-14T00:00:00Z",
        }),
      ),
    );

    renderGuardian();

    expect(
      await screen.findByText("Equipment is unavailable"),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("No equipped items returned"),
    ).not.toBeInTheDocument();
  });

  it("falls back cleanly when emblem artwork cannot load", async () => {
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([character]),
      ),
    );

    const { container } = renderGuardian();
    await screen.findByRole("heading", { name: "Hunter" });

    const background = container.querySelector(".gt-guardian-hero-art");
    const emblem = container.querySelector(".gt-guardian-mark img");
    expect(background).not.toBeNull();
    expect(emblem).not.toBeNull();
    fireEvent.error(background as HTMLImageElement);
    fireEvent.error(emblem as HTMLImageElement);

    expect(container.querySelector(".gt-guardian-hero-art")).toBeNull();
    expect(container.querySelector(".gt-guardian-mark img")).toBeNull();
    expect(container.querySelector(".gt-guardian-mark svg")).not.toBeNull();
  });

  it("renders a useful empty roster state", async () => {
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () => HttpResponse.json([])),
    );

    renderGuardian();

    expect(await screen.findByText("No Guardians found")).toBeInTheDocument();
  });

  it("distinguishes a roster failure from an empty roster", async () => {
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json({ error: "down" }, { status: 500 }),
      ),
    );

    renderGuardian();

    expect(await screen.findByText("Couldn't load data")).toBeInTheDocument();
    expect(screen.queryByText("No Guardians found")).not.toBeInTheDocument();
  });

  it("surfaces an equipment failure and retries", async () => {
    let attempts = 0;
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([character]),
      ),
      http.get(`${API}/api/characters/:type/:id/:characterId/equipment`, () => {
        attempts += 1;
        return attempts === 1
          ? HttpResponse.json({ error: "down" }, { status: 500 })
          : HttpResponse.json({
              characterId: character.characterId,
              state: "ready",
              items: [],
              fetchedAt: "2026-09-14T00:00:00Z",
            });
      }),
    );

    renderGuardian();

    expect(await screen.findByText("Couldn't load data")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() =>
      expect(
        screen.getByText("No equipped items returned"),
      ).toBeInTheDocument(),
    );
  });

  it("distinguishes unavailable activity history from an empty history", async () => {
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([character]),
      ),
      http.get(
        `${API}/api/characters/:type/:id/:characterId/activity-history`,
        () =>
          HttpResponse.json({
            characterId: character.characterId,
            state: "unavailable",
            activities: [],
            fetchedAt: "2026-09-14T00:00:00Z",
          }),
      ),
    );

    const first = renderGuardian();
    expect(
      await screen.findByText("Recent activity is unavailable"),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("No recent completed activity returned"),
    ).not.toBeInTheDocument();
    first.unmount();

    server.use(
      http.get(
        `${API}/api/characters/:type/:id/:characterId/activity-history`,
        () =>
          HttpResponse.json({
            characterId: character.characterId,
            state: "ready",
            activities: [],
            fetchedAt: "2026-09-14T00:00:00Z",
          }),
      ),
    );
    renderGuardian();
    expect(
      await screen.findByText("No recent completed activity returned"),
    ).toBeInTheDocument();
  });

  it("renders partially populated activity rows honestly", async () => {
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([character]),
      ),
      http.get(
        `${API}/api/characters/:type/:id/:characterId/activity-history`,
        () =>
          HttpResponse.json({
            characterId: character.characterId,
            state: "ready",
            activities: [
              {
                activityHash: "99",
                name: "Unknown activity",
                privateMatch: false,
                resolved: false,
              },
            ],
            fetchedAt: "2026-09-14T00:00:00Z",
          }),
      ),
    );

    const { container } = renderGuardian();
    expect(await screen.findByText("Unknown activity")).toBeInTheDocument();
    expect(screen.getByText("Time unavailable")).toBeInTheDocument();
    expect(screen.getByText("Unknown activity").closest("li")).toHaveAttribute(
      "data-resolved",
      "false",
    );
    expect(container.querySelector("time")).toBeNull();
  });

  it("surfaces an activity-history failure and retries", async () => {
    let attempts = 0;
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([character]),
      ),
      http.get(
        `${API}/api/characters/:type/:id/:characterId/activity-history`,
        () => {
          attempts += 1;
          return attempts === 1
            ? HttpResponse.json({ error: "down" }, { status: 500 })
            : HttpResponse.json({
                characterId: character.characterId,
                state: "ready",
                activities: [],
                fetchedAt: "2026-09-14T00:00:00Z",
              });
        },
      ),
    );

    renderGuardian();

    expect(await screen.findByText("Couldn't load data")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() =>
      expect(
        screen.getByText("No recent completed activity returned"),
      ).toBeInTheDocument(),
    );
  });
});

describe("Guardian current activity", () => {
  const CURRENT = `${API}/api/characters/:type/:id/:characterId/current-activity`;
  const HISTORY = `${API}/api/characters/:type/:id/:characterId/activity-history`;

  function withRosterAndHistory() {
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([character]),
      ),
      http.get(HISTORY, () =>
        HttpResponse.json({
          characterId: character.characterId,
          state: "ready",
          fetchedAt: "2026-09-15T00:00:00Z",
          activities: [
            {
              activityHash: "3637500864",
              name: "The Insight Terminus",
              privateMatch: false,
              resolved: true,
            },
          ],
        }),
      ),
    );
  }

  function currentPanel() {
    return screen.getByRole("region", { name: "Current activity" });
  }

  it("shows a resolved activity with its mode, playlist, and best-effort label", async () => {
    withRosterAndHistory();
    server.use(
      http.get(CURRENT, () =>
        HttpResponse.json({
          state: "ready",
          activityName: "Lake of Shadows",
          modeName: "Strike",
          playlistName: "Quickplay: Master",
          fetchedAt: new Date().toISOString(),
        }),
      ),
    );

    renderGuardian();

    expect(
      await screen.findByRole("heading", { name: "Lake of Shadows" }),
    ).toBeInTheDocument();
    const panel = currentPanel();
    expect(panel).toHaveTextContent("Strike");
    expect(panel).toHaveTextContent("Quickplay: Master");
    expect(panel).toHaveTextContent("Best effort");
    expect(panel).toHaveTextContent(/^.*Updated /);
  });

  it.each([
    ["idle", { state: "idle" }, "Not in an activity"],
    [
      "unavailable",
      { state: "unavailable" },
      "Current activity is unavailable",
    ],
    ["unknown", { state: "unknown", modeName: "Strike" }, "Unknown activity"],
  ])("renders the %s state distinctly", async (_name, body, title) => {
    withRosterAndHistory();
    server.use(
      http.get(CURRENT, () =>
        HttpResponse.json({ ...body, fetchedAt: "2026-09-15T00:00:00Z" }),
      ),
    );

    renderGuardian();

    expect(await screen.findByText(title)).toBeInTheDocument();
    for (const other of [
      "Not in an activity",
      "Current activity is unavailable",
      "Unknown activity",
    ].filter((t) => t !== title)) {
      expect(screen.queryByText(other)).not.toBeInTheDocument();
    }
    expect(screen.getByText("The Insight Terminus")).toBeInTheDocument();
  });

  it("presents a request failure as unavailable, keeps recent history, and retries", async () => {
    withRosterAndHistory();
    let attempts = 0;
    server.use(
      http.get(CURRENT, () => {
        attempts += 1;
        return attempts === 1
          ? HttpResponse.json({ error: "down" }, { status: 502 })
          : HttpResponse.json({
              state: "idle",
              fetchedAt: "2026-09-15T00:00:00Z",
            });
      }),
    );

    renderGuardian();

    expect(
      await screen.findByText("Current activity is unavailable"),
    ).toBeInTheDocument();
    expect(await screen.findByText("The Insight Terminus")).toBeInTheDocument();
    expect(screen.queryByText("Couldn't load data")).not.toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("button", { name: "Retry current activity" }),
    );
    expect(await screen.findByText("Not in an activity")).toBeInTheDocument();
  });

  it("follows the selected Guardian without showing the previous one's activity", async () => {
    const warlock = {
      ...character,
      characterId: "2305843009263456790",
      className: "Warlock",
    };
    const requested: string[] = [];
    let releaseWarlock: () => void = () => {};
    const warlockGate = new Promise<void>((resolve) => {
      releaseWarlock = resolve;
    });
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([character, warlock]),
      ),
      http.get(CURRENT, async ({ params }) => {
        const id = String(params.characterId);
        requested.push(id);
        if (id === warlock.characterId) await warlockGate;
        return HttpResponse.json({
          state: "ready",
          activityName:
            id === warlock.characterId ? "The Pale Heart" : "Lake of Shadows",
          fetchedAt: "2026-09-15T00:00:00Z",
        });
      }),
    );

    function Switcher() {
      const { setActiveCharacter } = useCharacters();
      return (
        <button onClick={() => setActiveCharacter(warlock.characterId)}>
          switch
        </button>
      );
    }

    renderWithProviders(
      <Routes>
        <Route
          path="/guardian"
          element={
            <>
              <Switcher />
              <Guardian />
            </>
          }
        />
      </Routes>,
      { route: "/guardian" },
    );

    expect(await screen.findByText("Lake of Shadows")).toBeInTheDocument();
    await userEvent.click(screen.getByText("switch"));
    await waitFor(() =>
      expect(screen.queryByText("Lake of Shadows")).not.toBeInTheDocument(),
    );
    releaseWarlock();
    expect(await screen.findByText("The Pale Heart")).toBeInTheDocument();
    expect(requested).toEqual([character.characterId, warlock.characterId]);
  });
});
