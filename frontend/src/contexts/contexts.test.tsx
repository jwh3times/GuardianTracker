import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { API, sampleUser, server } from "../test/testServer";
import { browserSessionClient } from "../lib/browserSessionBrowser";
import { AuthProvider, useAuth } from "./AuthContext";
import { CharacterProvider, useCharacters } from "./CharacterContext";

beforeEach(() => {
  localStorage.clear();
});

describe("AuthContext", () => {
  it("projects the hydrated client without exposing credentials or mutators", () => {
    localStorage.setItem("guardian_token", "tok");
    localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });
    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.user).toEqual(sampleUser);
    expect(result.current).not.toHaveProperty("token");
    expect(result.current).not.toHaveProperty("login");
    expect(result.current).not.toHaveProperty("isLoading");
  });

  it("projects anonymous state synchronously", () => {
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.user).toBeNull();
  });

  it("observes establishment and both logout scopes through the client", async () => {
    const scopes: string[] = [];
    server.use(
      http.post(`${API}/api/auth/bungie/callback`, () =>
        HttpResponse.json({ token: "new", user: sampleUser }),
      ),
      http.post(`${API}/api/auth/logout`, () => {
        scopes.push("current");
        return new HttpResponse(null, { status: 204 });
      }),
      http.post(`${API}/api/auth/logout/all`, () => {
        scopes.push("all");
        return new HttpResponse(null, { status: 204 });
      }),
    );
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });
    for (const scope of ["current", "all"] as const) {
      await act(() =>
        browserSessionClient.completeAuthorization({
          code: scope,
          state: "state",
        }),
      );
      expect(result.current.isAuthenticated).toBe(true);
      await act(() =>
        scope === "all" ? result.current.logoutAll() : result.current.logout(),
      );
      expect(result.current.isAuthenticated).toBe(false);
    }
    expect(scopes).toEqual(["current", "all"]);
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
