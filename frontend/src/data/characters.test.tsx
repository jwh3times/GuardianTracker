import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API, sampleUser } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useCharacters } from "../contexts/CharacterContext";
import {
  useCharacterRoster,
  useGuardianActivityHistory,
  useGuardianEquipment,
} from "./characters";
import { useMembershipRefresh } from "./membershipRefresh";

/**
 * Contract tests for the Characters data-access module (ADR 0020) — key
 * identity, the membership-scoped transport, projection to the domain
 * `Character`, and its invalidation entry point.
 */

function RosterProbe() {
  const { characters, isLoading, isError, retry } = useCharacterRoster();
  return (
    <div>
      <button onClick={retry}>retry</button>
      <div data-testid="roster">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : characters
                .map(
                  (c) =>
                    `${c.id}|${c.cls}|${c.race}|${c.power}|${c.emblemUrl ?? "no-emblem"}`,
                )
                .join(",") || "empty"}
      </div>
    </div>
  );
}

/** Reads the roster through the selection context rather than directly. */
function PickProbe() {
  const { activeCharacter } = useCharacters();
  return <div data-testid="pick">{activeCharacter?.id ?? "none"}</div>;
}

function EquipmentProbe({ characterId = "char-a" }: { characterId?: string }) {
  const { equipment, isLoading, isError, retry } =
    useGuardianEquipment(characterId);
  return (
    <div>
      <button onClick={retry}>retry equipment</button>
      <div data-testid="equipment">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : equipment
              ? `${equipment.characterId}|${equipment.state}|${equipment.fetchedAt}|${equipment.items
                  .map(
                    (item) =>
                      `${item.id}:${item.slot}:${item.group}:${item.name}:${item.type}:${item.rarity}:${item.power ?? "no-power"}:${item.icon ?? "no-icon"}`,
                  )
                  .join(",")}`
              : "none"}
      </div>
    </div>
  );
}

function ActivityProbe({ characterId = "char-a" }: { characterId?: string }) {
  const { history, isLoading, isError, retry } =
    useGuardianActivityHistory(characterId);
  return (
    <div>
      <button onClick={retry}>retry activity</button>
      <div data-testid="activity-history">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : history
              ? `${history.characterId}|${history.state}|${history.fetchedAt}|${history.activities
                  .map(
                    (activity) =>
                      `${activity.activityHash}:${activity.name}:${activity.occurredAt ?? "no-time"}:${activity.duration ?? "no-duration"}:${activity.privateMatch}:${activity.resolved}`,
                  )
                  .join(",")}`
              : "none"}
      </div>
    </div>
  );
}

describe("characters query identity", () => {
  it("serves the roster hook and the selection context from one request", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () => {
        requests += 1;
        return HttpResponse.json([apiCharacter("char-a")]);
      }),
    );

    renderWithProviders(
      <>
        <RosterProbe />
        <PickProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getByTestId("pick")).toHaveTextContent("char-a"),
    );
    await waitFor(() =>
      expect(screen.getByTestId("roster")).toHaveTextContent("char-a"),
    );
    // Settings and CharacterContext declared this query separately before; one
    // private key is what keeps them on one cache entry.
    expect(requests).toBe(1);
  });

  it("requests the signed-in membership", async () => {
    const sent: string[] = [];
    server.use(
      http.get(`${API}/api/characters/:type/:id`, ({ params }) => {
        sent.push(`${String(params.type)}/${String(params.id)}`);
        return HttpResponse.json([]);
      }),
    );

    renderWithProviders(<RosterProbe />);

    await waitFor(() =>
      expect(sent).toEqual([
        `${sampleUser.membershipType}/${sampleUser.membershipId}`,
      ]),
    );
  });
});

describe("projection", () => {
  it("projects wire characters to domain characters", async () => {
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([
          apiCharacter("char-a"),
          { ...apiCharacter("char-b"), className: "Warlock", emblemPath: "" },
        ]),
      ),
    );

    renderWithProviders(<RosterProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("roster")).toHaveTextContent(
        "char-a|Hunter|Awoken|2010|/emblem.png,char-b|Warlock|Awoken|2010|no-emblem",
      ),
    );
  });

  it("projects equipment and requests the signed-in membership plus Guardian", async () => {
    const sent: string[] = [];
    server.use(
      http.get(
        `${API}/api/characters/:type/:id/:characterId/equipment`,
        ({ params }) => {
          sent.push(
            `${String(params.type)}/${String(params.id)}/${String(params.characterId)}`,
          );
          return HttpResponse.json({
            characterId: "char-a",
            state: "ready",
            fetchedAt: "2026-09-14T00:00:00Z",
            items: [
              {
                itemHash: "10",
                slot: "Kinetic",
                group: "Weapons",
                name: "Fatebringer",
                itemType: "Hand Cannon",
                rarity: "Exotic",
                icon: "/fatebringer.png",
                resolved: true,
                power: 550,
              },
              {
                itemHash: "11",
                slot: "Energy",
                group: "Weapons",
                name: "Unknown item",
                itemType: "Energy",
                icon: "",
                resolved: false,
              },
            ],
          });
        },
      ),
    );

    renderWithProviders(<EquipmentProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("equipment")).toHaveTextContent(
        "char-a|ready|2026-09-14T00:00:00Z|10:Kinetic:Weapons:Fatebringer:Hand Cannon:exotic:550:/fatebringer.png,11:Energy:Weapons:Unknown item:Energy:undefined:no-power:no-icon",
      ),
    );
    expect(sent).toEqual([
      `${sampleUser.membershipType}/${sampleUser.membershipId}/char-a`,
    ]);
  });

  it("projects bounded activity history and requests the signed-in membership plus Guardian", async () => {
    const sent: string[] = [];
    server.use(
      http.get(
        `${API}/api/characters/:type/:id/:characterId/activity-history`,
        ({ params }) => {
          sent.push(
            `${String(params.type)}/${String(params.id)}/${String(params.characterId)}`,
          );
          return HttpResponse.json({
            characterId: "char-a",
            state: "ready",
            fetchedAt: "2026-09-14T00:00:00Z",
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

    renderWithProviders(<ActivityProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("activity-history")).toHaveTextContent(
        "char-a|ready|2026-09-14T00:00:00Z|3637500864:The Insight Terminus:2026-09-13T22:15:00Z:11m 12s:true:true",
      ),
    );
    expect(sent).toEqual([
      `${sampleUser.membershipType}/${sampleUser.membershipId}/char-a`,
    ]);
  });
});

describe("retry", () => {
  it("surfaces a failure and refetches on retry", async () => {
    let attempt = 0;
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () => {
        attempt += 1;
        return attempt === 1
          ? HttpResponse.json({ message: "nope" }, { status: 500 })
          : HttpResponse.json([apiCharacter("char-a")]);
      }),
    );

    renderWithProviders(<RosterProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("roster")).toHaveTextContent("failed"),
    );

    await userEvent.click(screen.getByText("retry"));

    await waitFor(() =>
      expect(screen.getByTestId("roster")).toHaveTextContent("char-a"),
    );
  });
});

function RefreshAndRosterProbe() {
  const { refresh } = useMembershipRefresh();
  const { characters } = useCharacterRoster();
  return (
    <div>
      <button onClick={refresh}>refresh</button>
      <div data-testid="roster">{characters.map((c) => c.id).join(",")}</div>
    </div>
  );
}

function RefreshAndEquipmentProbe() {
  const { refresh } = useMembershipRefresh();
  const { equipment } = useGuardianEquipment("char-a");
  return (
    <div>
      <button onClick={refresh}>refresh equipment</button>
      <div data-testid="equipment-state">
        {equipment?.items.map((item) => item.name).join(",") ?? "loading"}
      </div>
    </div>
  );
}

function RefreshAndActivityProbe() {
  const { refresh } = useMembershipRefresh();
  const { history } = useGuardianActivityHistory("char-a");
  return (
    <div>
      <button onClick={refresh}>refresh activity</button>
      <div data-testid="activity-name">
        {history?.activities.map((activity) => activity.name).join(",") ??
          "loading"}
      </div>
    </div>
  );
}

describe("invalidation", () => {
  it("a membership refresh actually refetches the roster", async () => {
    let gets = 0;
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () => {
        gets += 1;
        return HttpResponse.json([apiCharacter(`char-${gets}`)]);
      }),
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ success: true, message: "ok" }),
      ),
    );

    renderWithProviders(<RefreshAndRosterProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("roster")).toHaveTextContent("char-1"),
    );

    await userEvent.click(screen.getByText("refresh"));

    // Asserting an invalidation call or the key head does not discriminate: a
    // reshaped root key still reports "characters" while matching no real
    // entry. Only the refetched roster reaching the screen proves the key.
    await waitFor(() =>
      expect(screen.getByTestId("roster")).toHaveTextContent("char-2"),
    );
  });

  it("a membership refresh refetches Guardian equipment", async () => {
    let gets = 0;
    server.use(
      http.get(`${API}/api/characters/:type/:id/:characterId/equipment`, () => {
        gets += 1;
        return HttpResponse.json({
          characterId: "char-a",
          state: "ready",
          fetchedAt: "2026-09-14T00:00:00Z",
          items: [
            {
              itemHash: String(gets),
              slot: "Kinetic",
              group: "Weapons",
              name: `Weapon ${gets}`,
              itemType: "Hand Cannon",
              rarity: "Legendary",
              icon: "",
              resolved: true,
            },
          ],
        });
      }),
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ success: true, message: "ok" }),
      ),
    );

    renderWithProviders(<RefreshAndEquipmentProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("equipment-state")).toHaveTextContent(
        "Weapon 1",
      ),
    );

    await userEvent.click(screen.getByText("refresh equipment"));

    await waitFor(() =>
      expect(screen.getByTestId("equipment-state")).toHaveTextContent(
        "Weapon 2",
      ),
    );
  });

  it("a membership refresh refetches Guardian activity history", async () => {
    let gets = 0;
    server.use(
      http.get(
        `${API}/api/characters/:type/:id/:characterId/activity-history`,
        () => {
          gets += 1;
          return HttpResponse.json({
            characterId: "char-a",
            state: "ready",
            fetchedAt: "2026-09-14T00:00:00Z",
            activities: [
              {
                activityHash: String(gets),
                name: `Activity ${gets}`,
                privateMatch: false,
                resolved: true,
              },
            ],
          });
        },
      ),
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ success: true, message: "ok" }),
      ),
    );

    renderWithProviders(<RefreshAndActivityProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("activity-name")).toHaveTextContent(
        "Activity 1",
      ),
    );

    await userEvent.click(screen.getByText("refresh activity"));

    await waitFor(() =>
      expect(screen.getByTestId("activity-name")).toHaveTextContent(
        "Activity 2",
      ),
    );
  });
});

function apiCharacter(characterId: string) {
  return {
    characterId,
    classType: 1,
    className: "Hunter",
    raceName: "Awoken",
    light: 2010,
    emblemPath: "/emblem.png",
    emblemBackgroundPath: "/emblem-bg.png",
    dateLastPlayed: "2026-09-01T00:00:00Z",
  };
}
