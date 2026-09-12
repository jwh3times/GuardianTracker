import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API, sampleWeekly } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useCharacters } from "../contexts/CharacterContext";
import { useWeekly } from "./weekly";

/**
 * Contract tests for the Weekly data-access module (ADR 0020) — key identity,
 * the character-scoped transport, and the gate the module absorbed from its
 * two consumers.
 *
 * Projection is not retested here. `toWeekly` keeps its own tests in
 * `lib/weeklyView.ts`, including the tolerant difficulty vocabulary from C3.
 */

function WeeklyProbe() {
  const { week, isLoading, isError, retry } = useWeekly();
  return (
    <div>
      <button onClick={retry}>retry</button>
      <div data-testid="weekly">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : (week?.resetLabel ?? "none")}
      </div>
    </div>
  );
}

describe("weekly query identity", () => {
  it("serves both consumers from one request per character", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/weekly/recommendations`, () => {
        requests += 1;
        return HttpResponse.json(sampleWeekly);
      }),
    );

    renderWithProviders(
      <>
        <WeeklyProbe />
        <WeeklyProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getAllByTestId("weekly")[0]).toHaveTextContent(
        sampleWeekly.resetLabel,
      ),
    );
    // The Dashboard and This Week declared this key separately before; private
    // key ownership is what keeps them on one cache entry.
    expect(requests).toBe(1);
  });

  it("scopes the request to the selected character", async () => {
    const sent: (string | null)[] = [];
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([apiCharacter("2305843009300000001")]),
      ),
      http.get(`${API}/api/weekly/recommendations`, ({ request }) => {
        sent.push(new URL(request.url).searchParams.get("characterId"));
        return HttpResponse.json(sampleWeekly);
      }),
    );

    renderWithProviders(<WeeklyProbe />);

    await waitFor(() => expect(sent).toContain("2305843009300000001"));
    // The id comes from the module's own read of the selected character; a
    // consumer cannot pass a different one, because the hook takes no argument.
  });

  it("omits the parameter when the roster is empty", async () => {
    const sent: (string | null)[] = [];
    server.use(
      http.get(`${API}/api/weekly/recommendations`, ({ request }) => {
        sent.push(new URL(request.url).searchParams.get("characterId"));
        return HttpResponse.json(sampleWeekly);
      }),
    );

    // The default fixture is an empty roster. "No character" is a real state
    // the endpoint answers, so this must still fetch — not hang on the gate.
    renderWithProviders(<WeeklyProbe />);

    await waitFor(() => expect(sent).toEqual([null]));
  });
});

/** Renders the week and can switch the selected character. */
function SwitchProbe() {
  const { characters, setActiveCharacter } = useCharacters();
  const { week } = useWeekly();
  return (
    <div>
      {characters.map((c) => (
        <button key={c.id} onClick={() => setActiveCharacter(c.id)}>
          pick {c.id}
        </button>
      ))}
      <div data-testid="weekly">{week?.resetLabel ?? "none"}</div>
    </div>
  );
}

describe("character scoping", () => {
  it("fetches again when the selected character changes", async () => {
    const sent: (string | null)[] = [];
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([apiCharacter("char-a"), apiCharacter("char-b")]),
      ),
      http.get(`${API}/api/weekly/recommendations`, ({ request }) => {
        sent.push(new URL(request.url).searchParams.get("characterId"));
        return HttpResponse.json(sampleWeekly);
      }),
    );

    renderWithProviders(<SwitchProbe />);
    await waitFor(() => expect(sent).toEqual(["char-a"]));

    await userEvent.click(screen.getByText("pick char-b"));

    // The character id is part of the query identity. Without it in the key,
    // the second character silently reads the first one's cached week.
    await waitFor(() => expect(sent).toEqual(["char-a", "char-b"]));
  });
});

describe("retry", () => {
  it("refetches after a failure", async () => {
    let attempt = 0;
    server.use(
      http.get(`${API}/api/weekly/recommendations`, () => {
        attempt += 1;
        return attempt === 1
          ? HttpResponse.json({ message: "nope" }, { status: 500 })
          : HttpResponse.json(sampleWeekly);
      }),
    );

    renderWithProviders(<WeeklyProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("weekly")).toHaveTextContent("failed"),
    );

    await userEvent.click(screen.getByText("retry"));

    await waitFor(() => expect(attempt).toBe(2));
  });
});

describe("the absorbed gate", () => {
  it("waits for the character roster before fetching", async () => {
    let requests = 0;
    let rosterAsked = false;
    server.use(
      // Never resolves: the roster is still in flight.
      http.get(`${API}/api/characters/:type/:id`, () => {
        rosterAsked = true;
        return new Promise(() => {});
      }),
      http.get(`${API}/api/weekly/recommendations`, () => {
        requests += 1;
        return HttpResponse.json(sampleWeekly);
      }),
    );

    renderWithProviders(<WeeklyProbe />);

    // A gated query is pending but not *loading*, so the rendered text cannot
    // carry this assertion — the request count is what discriminates. Waiting
    // on the roster request proves the tower mounted and both queries had
    // their chance to fire in the same flush.
    await waitFor(() => expect(rosterAsked).toBe(true));
    // Both pages carried this half of the gate themselves before. Dropping it
    // would fire a request keyed on a character that has not arrived, then
    // refire on a second key the moment it does.
    expect(requests).toBe(0);
  });
});

function apiCharacter(characterId: string) {
  return {
    characterId,
    className: "Hunter",
    raceName: "Awoken",
    light: 2010,
    emblemPath: "/emblem.png",
  };
}
