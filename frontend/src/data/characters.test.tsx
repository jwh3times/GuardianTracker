import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API, sampleUser } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useCharacters } from "../contexts/CharacterContext";
import { useCharacterRoster } from "./characters";
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
