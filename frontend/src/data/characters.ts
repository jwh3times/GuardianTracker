import { useCallback } from "react";
import { useQuery, type QueryClient } from "@tanstack/react-query";
import { useAuth } from "../contexts/AuthContext";
import { apiFetch } from "../lib/api";
import type { APICharacter } from "../types/api";
import type { Character } from "../types/design";

/**
 * The Characters data-access module (ADR 0020). It owns this resource's query
 * identity, endpoint path, projection to {@link Character}, and its own
 * invalidation.
 *
 * Before this module the query was declared twice — by `CharacterContext` and
 * by Settings — each mapping the wire rows through `toCharacter` itself.
 *
 * `CharacterContext` survives as a thin selection-state wrapper over
 * {@link useCharacterRoster}. The persisted active-character pick is UI state,
 * not query state, and it must be shared: switching character in the AppShell
 * has to re-render the Dashboard and `data/weekly.ts`, which a context already
 * provides. Read the roster here when the pick is irrelevant (Settings); read
 * `useCharacters()` when it is.
 */

/**
 * Private, and deliberately not exported: no other module may name this key.
 * Membership-qualified because the endpoint path carries the membership.
 */
function charactersKey(
  membershipType: number | undefined,
  membershipId: string | undefined,
) {
  return ["characters", membershipType, membershipId] as const;
}

/** Matches every membership's entry. Used only by the refresh seam. */
const CHARACTERS_ROOT_KEY = ["characters"] as const;

/**
 * This module's invalidation entry point, called by `membershipRefresh.ts` so
 * the refresh owner never names this key.
 */
export function invalidateCharacters(client: QueryClient) {
  void client.invalidateQueries({ queryKey: CHARACTERS_ROOT_KEY });
}

/** Adapt a REST API character into the design system's Character shape. */
function toCharacter(c: APICharacter): Character {
  return {
    id: c.characterId,
    name: c.className,
    cls: c.className,
    race: c.raceName,
    power: c.light,
    emblem: 0,
    emblemUrl: c.emblemPath || undefined,
  };
}

/**
 * The query's `select`. Must stay a stable module-level reference: React Query
 * memoises `select` per observer on function identity, and `CharacterContext`
 * relies on the result keeping its identity across renders.
 */
function toCharacters(rows: APICharacter[]): Character[] {
  return rows.map(toCharacter);
}

/** Stable empty roster, so `characters` keeps referential identity. */
const NO_CHARACTERS: Character[] = [];

/**
 * The signed-in membership's characters, projected to domain characters.
 *
 * Takes no membership argument: it reads the current one from the session, so
 * a page cannot list another membership's characters.
 */
export function useCharacterRoster() {
  const { user } = useAuth();
  const membershipType = user?.membershipType;
  const membershipId = user?.membershipId;

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: charactersKey(membershipType, membershipId),
    queryFn: () =>
      apiFetch<APICharacter[]>(
        `/api/characters/${membershipType}/${membershipId}`,
      ),
    enabled: membershipType != null && !!membershipId,
    select: toCharacters,
  });

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return {
    characters: data ?? NO_CHARACTERS,
    isLoading,
    isError,
    error,
    retry,
  };
}
