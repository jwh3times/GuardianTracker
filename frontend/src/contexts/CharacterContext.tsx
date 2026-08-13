import React, {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useReducer,
} from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "./AuthContext";
import { apiFetch } from "../lib/api";
import { toCharacter } from "../lib/adapters";
import type { APICharacter } from "../types/api";
import type { Character } from "../types/design";

/**
 * Owns the characters query and the user's active-character pick.
 * The pick persists per Destiny membership in localStorage and survives reloads.
 * Collections/catalysts/seals remain membership-wide. Weekly authenticated vendor
 * inventory follows the active character because Bungie's component 402 is
 * character-scoped.
 */
interface CharacterContextValue {
  characters: Character[];
  activeCharacter: Character | null;
  setActiveCharacter: (id: string) => void;
  isLoading: boolean;
}

// Undefined rather than a default value, so a consumer rendered outside the
// provider fails loudly. A default of `characters: []` made that mistake
// indistinguishable from a real empty roster: a page hoisted above
// ProtectedLayout would quietly ship an empty character list forever. Matches
// AuthContext, FlagsContext and PreferencesContext.
const CharacterContext = createContext<CharacterContextValue | undefined>(
  undefined,
);

const storageKey = (membershipId: string) =>
  `guardian_active_character:${membershipId}`;

export function CharacterProvider({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  const membershipId = user?.membershipId;

  const { data, isLoading } = useQuery({
    queryKey: ["characters", user?.membershipType, membershipId],
    queryFn: () =>
      apiFetch<APICharacter[]>(
        `/api/characters/${user!.membershipType}/${user!.membershipId}`,
      ),
    enabled: !!membershipId && user?.membershipType != null,
  });

  const characters = useMemo(() => (data ?? []).map(toCharacter), [data]);

  // localStorage is the source of truth for the pick; `version` just forces a
  // re-read after writes. Switching memberships re-derives automatically.
  const [version, bump] = useReducer((x: number) => x + 1, 0);
  const pickedId = useMemo(
    () =>
      membershipId ? localStorage.getItem(storageKey(membershipId)) : null,
    // oxlint-disable-next-line react/exhaustive-deps -- version invalidates the localStorage read
    [membershipId, version],
  );

  const setActiveCharacter = useCallback(
    (id: string) => {
      if (membershipId) localStorage.setItem(storageKey(membershipId), id);
      bump();
    },
    [membershipId],
  );

  // A stale persisted pick (deleted character) falls back to the first character.
  const activeCharacter =
    characters.find((c) => c.id === pickedId) ?? characters[0] ?? null;

  const value = useMemo(
    () => ({ characters, activeCharacter, setActiveCharacter, isLoading }),
    [characters, activeCharacter, setActiveCharacter, isLoading],
  );

  return (
    <CharacterContext.Provider value={value}>
      {children}
    </CharacterContext.Provider>
  );
}

// oxlint-disable-next-line react/only-export-components
export function useCharacters(): CharacterContextValue {
  const ctx = useContext(CharacterContext);
  if (!ctx) {
    throw new Error("useCharacters must be used within a CharacterProvider");
  }
  return ctx;
}
