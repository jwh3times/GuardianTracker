import { useCallback } from "react";
import { useQuery, type QueryClient } from "@tanstack/react-query";
import { useAuth } from "../contexts/AuthContext";
import { useCharacters } from "../contexts/CharacterContext";
import { apiFetch } from "../lib/api";
import { toWeekly } from "../lib/weeklyView";
import type { APIWeekly } from "../types/api";

/**
 * The Weekly data-access module (ADR 0020). It owns this resource's query
 * identity, endpoint path, projection to {@link Weekly}, and its own
 * invalidation.
 *
 * Two features read it — the Dashboard and This Week — and they sit on one
 * cache entry per selected character rather than firing two requests.
 *
 * As with Collections, the projection is not moved in: `toWeekly` stays in
 * `lib/weeklyView.ts` with the tests that call it directly, which already own
 * the tolerant difficulty vocabulary shipped by C3.
 */

/**
 * Private, and deliberately not exported. The character id is part of the
 * identity because the payload is character-scoped; `null` stands for "no
 * character selected", which is a real state the endpoint answers.
 */
function weeklyKey(characterId: string | null) {
  return ["weekly", characterId] as const;
}

/** Matches every character's entry. Used by the membership-refresh seam. */
const WEEKLY_ROOT_KEY = ["weekly"] as const;

/**
 * This module's invalidation entry point, called by `membershipRefresh.ts` so
 * the refresh owner never names this key.
 */
export function invalidateWeekly(client: QueryClient) {
  void client.invalidateQueries({ queryKey: WEEKLY_ROOT_KEY });
}

/**
 * This week's reset timing, milestones, Xûr inventory and ranked actions for
 * the selected character.
 *
 * Takes no arguments. Both consumers previously passed `activeCharacter?.id`
 * and composed their own `enabled` onto the shared options object — the leak
 * that would make E16's lint zone unlandable — and the two gates were the same
 * fact written two ways: `!!user && !charactersLoading` on one side, and
 * `membershipType != null && !!membershipId && !charactersLoading` on the
 * other. `membershipId` and `membershipType` are required fields on the user,
 * so the first implies the second.
 *
 * `lib/queries.ts` claimed the two "gate on different identity facts" and that
 * "one shared guard would be wrong for both". That was not true when this
 * module was written; the guard lives here now, and neither page can reach a
 * character other than the selected one.
 */
export function useWeekly() {
  const { user } = useAuth();
  const { activeCharacter, isLoading: charactersLoading } = useCharacters();
  const characterId = activeCharacter?.id ?? null;

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: weeklyKey(characterId),
    queryFn: () =>
      apiFetch<APIWeekly>(
        `/api/weekly/recommendations${
          characterId ? `?characterId=${encodeURIComponent(characterId)}` : ""
        }`,
      ),
    enabled: !!user && !charactersLoading,
    select: toWeekly,
  });

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return {
    week: data,
    isLoading,
    isError,
    error,
    retry,
  };
}
