import React from "react";
import {
  describe,
  it,
  expect,
  beforeAll,
  afterAll,
  afterEach,
  beforeEach,
} from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "./testServer";
import { AuthProvider, useAuth } from "../contexts/AuthContext";
import { CharacterProvider, useCharacters } from "../contexts/CharacterContext";

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  localStorage.clear();
});

describe("AuthContext", () => {
  it("hydrates synchronously from localStorage", () => {
    localStorage.setItem("guardian_token", "tok");
    localStorage.setItem("guardian_refresh_token", "ref");
    localStorage.setItem("guardian_user", JSON.stringify(sampleUser));

    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });

    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.user?.displayName).toBe("TestGuardian");
    expect(result.current.isLoading).toBe(false);
  });

  it("is unauthenticated with no stored tokens", () => {
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });
    expect(result.current.isAuthenticated).toBe(false);
  });

  it("login persists to localStorage; logout clears it", () => {
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });

    act(() => result.current.login("t1", "r1", sampleUser));
    expect(localStorage.getItem("guardian_token")).toBe("t1");
    expect(result.current.isAuthenticated).toBe(true);

    act(() => result.current.logout());
    expect(localStorage.getItem("guardian_token")).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });
});

describe("CharacterContext", () => {
  const characters = [
    {
      characterId: "char-hunter",
      classType: 1,
      className: "Hunter",
      raceName: "Awoken",
      light: 2010,
      emblemPath: "/h.png",
      emblemBackgroundPath: "/hb.png",
      dateLastPlayed: new Date().toISOString(),
    },
    {
      characterId: "char-warlock",
      classType: 2,
      className: "Warlock",
      raceName: "Human",
      light: 2005,
      emblemPath: "/w.png",
      emblemBackgroundPath: "/wb.png",
      dateLastPlayed: new Date().toISOString(),
    },
  ];

  function wrapper({ children }: { children: React.ReactNode }) {
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    return (
      <QueryClientProvider client={qc}>
        <AuthProvider>
          <CharacterProvider>{children}</CharacterProvider>
        </AuthProvider>
      </QueryClientProvider>
    );
  }

  beforeEach(() => {
    localStorage.setItem("guardian_token", "tok");
    localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
    server.use(
      http.get(`${API}/api/characters/:type/:id`, () =>
        HttpResponse.json(characters),
      ),
    );
  });

  it("defaults to the first character and persists an explicit pick", async () => {
    const { result } = renderHook(() => useCharacters(), { wrapper });

    await waitFor(() => expect(result.current.characters).toHaveLength(2));
    expect(result.current.activeCharacter?.id).toBe("char-hunter");

    act(() => result.current.setActiveCharacter("char-warlock"));
    expect(result.current.activeCharacter?.id).toBe("char-warlock");
    expect(
      localStorage.getItem(
        `guardian_active_character:${sampleUser.membershipId}`,
      ),
    ).toBe("char-warlock");
  });

  it("restores a persisted pick on a fresh mount", async () => {
    localStorage.setItem(
      `guardian_active_character:${sampleUser.membershipId}`,
      "char-warlock",
    );
    const { result } = renderHook(() => useCharacters(), { wrapper });

    await waitFor(() => expect(result.current.characters).toHaveLength(2));
    expect(result.current.activeCharacter?.id).toBe("char-warlock");
  });

  it("falls back to the first character when the persisted pick is stale", async () => {
    localStorage.setItem(
      `guardian_active_character:${sampleUser.membershipId}`,
      "char-deleted",
    );
    const { result } = renderHook(() => useCharacters(), { wrapper });

    await waitFor(() => expect(result.current.characters).toHaveLength(2));
    expect(result.current.activeCharacter?.id).toBe("char-hunter");
  });
});
