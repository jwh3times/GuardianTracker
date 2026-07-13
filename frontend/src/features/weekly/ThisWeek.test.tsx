import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse, delay } from "msw";
import { API, sampleUser, sampleWeekly, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import {
  CharacterProvider,
  useCharacters,
} from "../../contexts/CharacterContext";
import { ToastProvider } from "../../components/Toast";
import { ThisWeek } from "./ThisWeek";

beforeEach(() => {
  localStorage.clear();
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
});

function renderPage(ui: React.ReactNode, route = "/") {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <AuthProvider>
        <CharacterProvider>
          <ToastProvider>
            <MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter>
          </ToastProvider>
        </CharacterProvider>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

describe("ThisWeek page", () => {
  function LocationProbe() {
    const loc = useLocation();
    return <div data-testid="loc">{loc.pathname + loc.search}</div>;
  }

  it("renders reset label and the Xûr-away module by default", async () => {
    renderPage(<ThisWeek />);
    expect(
      await screen.findByText("Resets Tuesday 17:00 UTC"),
    ).toBeInTheDocument();
    expect(screen.getByText("Xûr returns Friday")).toBeInTheDocument();
    expect(screen.getByText("0/1 done")).toBeInTheDocument();
  });

  it("toggles a recommended action done and persists across the reset key", async () => {
    server.use(
      http.get(`${API}/api/weekly/recommendations`, () =>
        HttpResponse.json({
          ...sampleWeekly,
          recommended: [
            {
              id: "rec-1",
              text: "Run a Nightfall",
              detail: "Strikes",
              badge: "missing",
              done: false,
              diff: "moderate",
              time: "30m",
            },
          ],
        }),
      ),
    );
    renderPage(<ThisWeek />);
    expect(await screen.findByText("Run a Nightfall")).toBeInTheDocument();
    expect(screen.getByText("0/1 done")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Mark done" }));
    expect(await screen.findByText("1/1 done")).toBeInTheDocument();
    // Persisted under the reset-keyed localStorage entry
    expect(localStorage.getItem("gt_done:" + sampleWeekly.resetAt)).toContain(
      "rec-1",
    );

    // Toggle back off
    fireEvent.click(screen.getByRole("button", { name: "Mark done" }));
    expect(await screen.findByText("0/1 done")).toBeInTheDocument();
  });

  it("renders the degraded banner, Xûr inventory and milestones", async () => {
    server.use(
      http.get(`${API}/api/weekly/recommendations`, () =>
        HttpResponse.json({
          ...sampleWeekly,
          degraded: true,
          xur: {
            present: true,
            leavesIn: { d: 1, h: 2, m: 0 },
            location: "The Tower",
            items: [
              {
                hash: "9001",
                name: "Gjallarhorn",
                type: "Rocket Launcher",
                rarity: "exotic",
                missing: true,
                icon: "/icons/gjallarhorn.png",
                cost: "29 Strange Coins",
              },
              {
                hash: "9002",
                name: "Ace of Spades",
                type: "Armor",
                rarity: "exotic",
                missing: false,
                cost: "29 Strange Coins",
                className: "Warlock",
              },
            ],
          },
          milestones: [
            {
              id: "m1",
              label: "Weekly",
              name: "Vanguard Ops",
              reward: "Pinnacle",
              missing: 2,
              note: "",
            },
            {
              id: "m2",
              label: "Weekly",
              name: "Crucible",
              reward: "Pinnacle",
              note: "",
            },
          ],
        }),
      ),
    );
    const { container } = renderPage(<ThisWeek />);
    expect(
      await screen.findByText(/Item names are still loading/),
    ).toBeInTheDocument();
    expect(screen.getByText("The Tower")).toBeInTheDocument();
    expect(screen.getByText("Gjallarhorn")).toBeInTheDocument();
    expect(screen.getByText("Warlock armor")).toBeInTheDocument();
    expect(
      container.querySelector(
        'img[src="https://www.bungie.net/icons/gjallarhorn.png"]',
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Vanguard Ops")).toBeInTheDocument();
  });

  it("omits the Xûr location row when the backend cannot resolve it", async () => {
    server.use(
      http.get(`${API}/api/weekly/recommendations`, () =>
        HttpResponse.json({
          ...sampleWeekly,
          xur: {
            present: true,
            leavesIn: { d: 1, h: 2 },
            items: [
              {
                hash: "9001",
                name: "Gjallarhorn",
                type: "Rocket Launcher",
                rarity: "exotic",
                missing: true,
                icon: "/icons/gjallarhorn.png",
                cost: "29 Strange Coins",
              },
            ],
          },
        }),
      ),
    );

    const { container } = renderPage(<ThisWeek />);
    expect(await screen.findByText("Gjallarhorn")).toBeInTheDocument();
    expect(container.querySelector(".gt-xur-loc")).toBeNull();
    expect(screen.queryByText("Unknown")).not.toBeInTheDocument();
  });

  it("purges checkmarks from previous reset weeks on load", async () => {
    localStorage.setItem(
      "gt_done:2020-01-01T00:00:00Z",
      JSON.stringify(["old"]),
    );
    renderPage(<ThisWeek />);
    await screen.findByText("Resets Tuesday 17:00 UTC");
    await waitFor(() =>
      expect(localStorage.getItem("gt_done:2020-01-01T00:00:00Z")).toBeNull(),
    );
  });

  it("renders the loading spinner while the query is in flight", async () => {
    server.use(
      http.get(`${API}/api/weekly/recommendations`, async () => {
        await delay(40);
        return HttpResponse.json(sampleWeekly);
      }),
    );
    const { container } = renderPage(<ThisWeek />);
    expect(container.querySelector(".gt-page")).not.toBeNull();
    // The spinner-only page has no PageHead while loading
    expect(
      screen.queryByText("Resets Tuesday 17:00 UTC"),
    ).not.toBeInTheDocument();
  });

  it("deep-links a Xûr item to the collections drawer", async () => {
    server.use(
      http.get(`${API}/api/weekly/recommendations`, () =>
        HttpResponse.json({
          ...sampleWeekly,
          xur: {
            present: true,
            leavesIn: { d: 1, h: 2, m: 0 },
            location: "The Tower",
            items: [
              {
                hash: "100",
                name: "Fatebringer",
                type: "Hand Cannon",
                rarity: "exotic",
                missing: true,
                cost: "29 Strange Coins",
              },
            ],
          },
        }),
      ),
    );
    renderPage(
      <>
        <ThisWeek />
        <LocationProbe />
      </>,
    );
    fireEvent.click(await screen.findByText("Fatebringer"));
    expect(screen.getByTestId("loc").textContent).toBe("/collections?item=100");
  });

  it("refetches weekly vendor context when the active Guardian changes", async () => {
    const requestedCharacters: string[] = [];
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json([
          {
            characterId: "char-hunter",
            classType: 1,
            className: "Hunter",
            raceName: "Awoken",
            light: 2010,
            emblemPath: "",
            emblemBackgroundPath: "",
            dateLastPlayed: "2026-07-12T12:00:00Z",
          },
          {
            characterId: "char-warlock",
            classType: 2,
            className: "Warlock",
            raceName: "Human",
            light: 2005,
            emblemPath: "",
            emblemBackgroundPath: "",
            dateLastPlayed: "2026-07-11T12:00:00Z",
          },
        ]),
      ),
      http.get(`${API}/api/weekly/recommendations`, ({ request }) => {
        requestedCharacters.push(
          new URL(request.url).searchParams.get("characterId") ?? "",
        );
        return HttpResponse.json(sampleWeekly);
      }),
    );

    function SwitchProbe() {
      const { setActiveCharacter } = useCharacters();
      return (
        <>
          <button onClick={() => setActiveCharacter("char-warlock")}>
            Switch
          </button>
          <ThisWeek />
        </>
      );
    }

    renderPage(<SwitchProbe />);
    await waitFor(() => expect(requestedCharacters).toContain("char-hunter"));
    fireEvent.click(screen.getByRole("button", { name: "Switch" }));
    await waitFor(() => expect(requestedCharacters).toContain("char-warlock"));
  });
});
