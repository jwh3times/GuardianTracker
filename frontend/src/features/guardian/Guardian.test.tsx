import { describe, expect, it } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { Route, Routes } from "react-router";
import { API, sampleUser, server } from "../../test/testServer";
import { renderWithProviders } from "../../test/renderWithProviders";
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
            items: [
              {
                itemHash: "10",
                slot: "Kinetic",
                group: "Weapons",
                name: "Fatebringer",
                itemType: "Hand Cannon",
                rarity: "Legendary",
                icon: "/fatebringer.png",
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
    expect(screen.getByLabelText("551 Power")).toBeInTheDocument();
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
});
