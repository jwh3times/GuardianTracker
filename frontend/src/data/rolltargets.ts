import { useCallback } from "react";
import {
  useQuery,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";
import { useAuth } from "../contexts/AuthContext";
import { useIdentityMutation } from "../contexts/IdentityMutation";
import { apiFetch, type ApiError } from "../lib/api";
import type {
  APIImportReport,
  APIRollTarget,
  APIRollTargetBulkResult,
  APIRollTargetMatchReport,
  APIUnresolvedPerk,
} from "../types/api";
import type {
  RollTarget,
  RollTargetImportLine,
  RollTargetImportReport,
  RollTargetMatch,
  RollTargetMatchReport,
  RollTargetNearMiss,
  UnmatchedRollTarget,
  UnresolvedPerk,
} from "../types/design";

/**
 * The Roll targets data-access module (ADR 0020, slice 5b). It owns this
 * resource's query identity, endpoint paths, projection to the domain types,
 * every mutation and its optimistic coordination, and its own invalidation.
 *
 * Two queries, not one: the saved-target list (`GET /api/rolltargets`, this
 * membership's authored rows — the source of truth for management: notes,
 * bulk delete, delete-all) and the match report (`GET /api/rolltargets/matches`,
 * which of them anything owned currently satisfies). They fail independently —
 * a Bungie-backed match failure must never hide the player's own saved rows —
 * so each reader decides its own degraded rendering.
 *
 * Read by the Roll targets page and, behind the `god-roll` flag's accessible
 * state and only while open on a weapon, Collections' item detail drawer
 * (`features/collections/RollTargetSection.tsx`) — both share this module's
 * one query identity per resource, so they share one cache entry.
 */

/** Private, and deliberately not exported: no other module may name this key. */
const ROLL_TARGETS_LIST_KEY = ["rollTargets", "list"] as const;
/** Private, and deliberately not exported: no other module may name this key. */
const ROLL_TARGETS_MATCHES_KEY = ["rollTargets", "matches"] as const;
/** Matches both queries above. Used only by the refresh seam. */
const ROLL_TARGETS_ROOT_KEY = ["rollTargets"] as const;

/**
 * This module's invalidation entry point, called by `membershipRefresh.ts` so
 * the refresh owner never names this key.
 *
 * A match report is derived from owned inventory (profile component 305), so a
 * membership refresh that re-fetches Bungie data can change which targets are
 * satisfied. The saved-target list carries no Bungie data and would not
 * legitimately change from a refresh, but invalidating it alongside the match
 * report is one call, not two, and costs nothing beyond a redundant refetch of
 * a small, already-cached list.
 */
export function invalidateRollTargets(client: QueryClient) {
  void client.invalidateQueries({ queryKey: ROLL_TARGETS_ROOT_KEY });
}

/** Every failure surfaced by this module. `apiFetch` throws only `ApiError`. */
export type RollTargetError = ApiError;

function toRollTarget(t: APIRollTarget): RollTarget {
  return {
    id: t.id,
    itemHash: t.itemHash != null ? String(t.itemHash) : null,
    anyWeapon: t.anyWeapon,
    wanted: t.wanted,
    perks: t.perks ?? [],
    notes: t.notes ?? "",
    dateAdded: t.dateAdded,
    importId: t.importId,
    importTitle: t.importTitle,
  };
}

/**
 * The list query's `select`. Stable module-level reference so React Query's
 * per-observer `select` memoisation holds.
 */
function toRollTargets(rows: APIRollTarget[]): RollTarget[] {
  return (rows ?? []).map(toRollTarget);
}

function toUnresolvedPerk(
  u: APIUnresolvedPerk | undefined,
): UnresolvedPerk | undefined {
  if (!u) return undefined;
  return { hash: u.perkHash, name: u.perkName, reason: u.reason };
}

function toImportLine(
  l: APIImportReport["lines"][number],
): RollTargetImportLine {
  return {
    line: l.line,
    outcome: l.outcome,
    detail: l.detail,
    unresolved: toUnresolvedPerk(l.unresolved),
    itemHash: l.itemHash != null ? String(l.itemHash) : undefined,
    wanted: l.wanted,
    perks: l.perks,
  };
}

/** Adapt an import report onto the domain shape. Not a query `select` — this
 * is a mutation result, projected once when the mutation resolves. */
function toImportReport(r: APIImportReport): RollTargetImportReport {
  return {
    importId: r.importId,
    title: r.title,
    description: r.description,
    imported: r.imported,
    counts: r.counts ?? {},
    lines: (r.lines ?? []).map(toImportLine),
  };
}

function toMatch(
  m: APIRollTargetMatchReport["wanted"][number],
): RollTargetMatch {
  return {
    targetId: m.targetId,
    itemHash: String(m.itemHash),
    instanceId: m.instanceId,
    perks: m.perks ?? [],
    targetPerks: m.targetPerks ?? [],
    notes: m.notes,
  };
}

function toNearMiss(
  copy: APIRollTargetMatchReport["unmatchedTargets"][number]["bestCopy"],
): RollTargetNearMiss | undefined {
  if (!copy) return undefined;
  return {
    itemHash: String(copy.itemHash),
    instanceId: copy.instanceId,
    perks: copy.perks ?? [],
    matchedPerks: copy.matchedPerks ?? [],
    matchedColumns: copy.matchedColumns,
    targetColumns: copy.targetColumns,
  };
}

function toUnmatchedTarget(
  target: APIRollTargetMatchReport["unmatchedTargets"][number],
): UnmatchedRollTarget {
  return {
    ...toRollTarget(target),
    bestCopy: toNearMiss(target.bestCopy),
  };
}

/**
 * The matches query's `select`. Stable module-level reference so React
 * Query's per-observer `select` memoisation holds.
 */
function toMatchReport(r: APIRollTargetMatchReport): RollTargetMatchReport {
  return {
    wanted: (r.wanted ?? []).map(toMatch),
    unwanted: (r.unwanted ?? []).map(toMatch),
    unmatchedTargets: (r.unmatchedTargets ?? []).map(toUnmatchedTarget),
  };
}

/** Stable empty list, so `targets` keeps referential identity across renders. */
const NO_TARGETS: RollTarget[] = [];

interface RollTargetQueryOptions {
  /**
   * Extra gate ANDed with this module's own precondition. Only the item
   * detail drawer needs it — it waits until the drawer is open on a weapon
   * with a resolved perk pool before fetching anything roll-target-shaped —
   * and it takes a plain boolean so React Query's option shape stays inside
   * this module (mirrors `data/collections.ts`'s `CollectionsQueryOptions`).
   */
  enabled?: boolean;
}

/**
 * Every saved roll target for the signed-in membership, most recent first.
 *
 * Takes no membership argument: the endpoint derives it from the session, so a
 * page cannot ask for another membership's targets. This is the source of
 * truth for target management (notes, bulk delete, delete-all) — the match
 * report below only says which of these are currently satisfied.
 */
export function useRollTargets({
  enabled = true,
}: RollTargetQueryOptions = {}) {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ROLL_TARGETS_LIST_KEY,
    queryFn: () => apiFetch<APIRollTarget[]>("/api/rolltargets"),
    enabled,
    select: toRollTargets,
  });

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return {
    targets: data ?? NO_TARGETS,
    isLoading,
    isError,
    error,
    retry,
  };
}

/**
 * Which saved targets anything currently owned satisfies, split into wanted
 * matches, unwanted matches, and unmatched targets.
 *
 * `enabled: !!user` is belt-and-braces, matching `weekly.ts`'s guard:
 * ProtectedLayout unmounts this whole subtree the instant the session goes
 * anonymous, so there is no render in which this hook runs signed out. It is
 * kept because the module, not the route, should own its own precondition.
 * The caller's own `enabled` is ANDed with it — the item detail drawer uses
 * this to avoid issuing a live Bungie profile read until the flag is
 * accessible and the drawer has resolved the open item as a weapon.
 *
 * This is the one query in this module that reads owned Bungie inventory
 * (profile component 305), so a membership refresh that re-fetches Bungie
 * data can change the answer — see {@link invalidateRollTargets}.
 */
export function useRollTargetMatches({
  enabled = true,
}: RollTargetQueryOptions = {}) {
  const { user } = useAuth();

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ROLL_TARGETS_MATCHES_KEY,
    queryFn: () =>
      apiFetch<APIRollTargetMatchReport>("/api/rolltargets/matches"),
    enabled: enabled && !!user,
    select: toMatchReport,
  });

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return {
    matches: data,
    isLoading,
    isError,
    error,
    retry,
  };
}

/**
 * Feature-owned side effects for a mutation. The module owns the cache half —
 * cancel, snapshot, optimistic write, rollback, settle — and the feature owns
 * the user-visible copy (ADR 0020, matching `data/wishlist.ts`'s shape).
 */
export interface RollTargetMutationCallbacks<TVars, TResult = void> {
  onSuccess?: (result: TResult, vars: TVars) => void;
  onError?: (error: RollTargetError, vars: TVars) => void;
  onSettled?: (vars: TVars) => void;
}

/** Snapshot taken before an optimistic write, replayed on failure. */
interface Rollback {
  previous?: APIRollTarget[];
}

/**
 * The one optimistic recipe, shared by every mutation below that can predict
 * the server's answer (mirrors `data/wishlist.ts`'s
 * `useOptimisticWishlistMutation`). `apply` is expressed in wire terms because
 * the cache stores the wire shape.
 */
function useOptimisticRollTargetMutation<TVars, TResult>(
  mutationFn: (vars: TVars) => Promise<TResult>,
  apply: (rows: APIRollTarget[], vars: TVars) => APIRollTarget[],
  callbacks?: RollTargetMutationCallbacks<TVars, TResult>,
) {
  const client = useQueryClient();
  return useIdentityMutation<TResult, RollTargetError, TVars, Rollback>({
    mutationFn,
    onMutate: async (vars) => {
      await client.cancelQueries({ queryKey: ROLL_TARGETS_LIST_KEY });
      const previous = client.getQueryData<APIRollTarget[]>(
        ROLL_TARGETS_LIST_KEY,
      );
      client.setQueryData<APIRollTarget[]>(ROLL_TARGETS_LIST_KEY, (old) =>
        apply(old ?? [], vars),
      );
      return { previous };
    },
    onSuccess: (result, vars) => callbacks?.onSuccess?.(result, vars),
    onError: (error, vars, context) => {
      if (context?.previous) {
        client.setQueryData(ROLL_TARGETS_LIST_KEY, context.previous);
      }
      callbacks?.onError?.(error, vars);
    },
    onSettled: (_result, _error, vars) => {
      invalidateRollTargets(client);
      callbacks?.onSettled?.(vars);
    },
  });
}

export interface UpdateRollTargetNotesVars {
  id: string;
  notes: string;
}

/** Change a target's notes, optimistically. The server only accepts `notes`
 * on this endpoint — perks are read-only after a target is saved. */
export function useUpdateRollTargetNotes(
  callbacks?: RollTargetMutationCallbacks<UpdateRollTargetNotesVars>,
) {
  const mutation = useOptimisticRollTargetMutation<
    UpdateRollTargetNotesVars,
    void
  >(
    async (vars) => {
      await apiFetch<APIRollTarget>(`/api/rolltargets/${vars.id}`, {
        method: "PATCH",
        body: JSON.stringify({ notes: vars.notes }),
      });
    },
    (rows, vars) =>
      rows.map((r) => (r.id === vars.id ? { ...r, notes: vars.notes } : r)),
    callbacks,
  );

  return {
    setNotes: (vars: UpdateRollTargetNotesVars) => mutation.mutate(vars),
    isPending: mutation.isPending,
  };
}

export interface RemoveRollTargetVars {
  id: string;
}

/** Remove one saved target, optimistically. */
export function useRemoveRollTarget(
  callbacks?: RollTargetMutationCallbacks<RemoveRollTargetVars>,
) {
  const mutation = useOptimisticRollTargetMutation<RemoveRollTargetVars, void>(
    (vars) =>
      apiFetch<void>(`/api/rolltargets/${vars.id}`, { method: "DELETE" }),
    (rows, vars) => rows.filter((r) => r.id !== vars.id),
    callbacks,
  );

  return {
    remove: (vars: RemoveRollTargetVars) => mutation.mutate(vars),
    isPending: mutation.isPending,
  };
}

/**
 * The server's own per-request cap (`rolltargets.MaxBulkTargets`). The page
 * caps selection at this many rows rather than chunking a larger selection
 * into several bulk calls — simpler, and a selection this size is already an
 * edge case a real player is unlikely to hit deliberately.
 */
export const MAX_BULK_DELETE = 100;

export interface BulkDeleteResult {
  deleted: number;
  skipped: number;
}

/** Delete all surviving rows from one import, regardless of list filters or size. */
export function useDeleteRollTargetImport(
  callbacks?: RollTargetMutationCallbacks<string, BulkDeleteResult>,
) {
  const mutation = useOptimisticRollTargetMutation<string, BulkDeleteResult>(
    (importId) =>
      apiFetch<APIRollTargetBulkResult>("/api/rolltargets/bulk", {
        method: "POST",
        body: JSON.stringify({ action: "delete_import", importId }),
      }),
    (rows, importId) => rows.filter((row) => row.importId !== importId),
    callbacks,
  );
  return {
    deleteImport: (importId: string) => mutation.mutate(importId),
    isPending: mutation.isPending,
  };
}

/** Delete many saved targets by id in one call, optimistically. */
export function useBulkDeleteRollTargets(
  callbacks?: RollTargetMutationCallbacks<string[], BulkDeleteResult>,
) {
  const mutation = useOptimisticRollTargetMutation<string[], BulkDeleteResult>(
    (ids) =>
      apiFetch<APIRollTargetBulkResult>("/api/rolltargets/bulk", {
        method: "POST",
        body: JSON.stringify({ action: "delete", ids }),
      }),
    (rows, ids) => {
      const set = new Set(ids);
      return rows.filter((r) => !set.has(r.id));
    },
    callbacks,
  );

  return {
    bulkDelete: (ids: string[]) => mutation.mutate(ids),
    isPending: mutation.isPending,
  };
}

/** Delete every saved target for this membership, optimistically. The page
 * gates this behind its own explicit confirmation before calling it. */
export function useDeleteAllRollTargets(
  callbacks?: RollTargetMutationCallbacks<void, BulkDeleteResult>,
) {
  const mutation = useOptimisticRollTargetMutation<void, BulkDeleteResult>(
    () =>
      apiFetch<APIRollTargetBulkResult>("/api/rolltargets/bulk", {
        method: "POST",
        body: JSON.stringify({ action: "delete_all" }),
      }),
    () => [],
    callbacks,
  );

  return {
    deleteAll: () => mutation.mutate(),
    isPending: mutation.isPending,
  };
}

/**
 * Import a DIM-format file's text, add-only.
 *
 * Deliberately not optimistic: the server decides, per line, what imported —
 * there is no honest row to write into the cache ahead of the response. The
 * request body is the raw file text (not JSON), matching the endpoint's own
 * contract. Invalidates the target list and match report on success, since
 * import can add new rows.
 */
export function useImportRollTargets(
  callbacks?: RollTargetMutationCallbacks<string, RollTargetImportReport>,
) {
  const client = useQueryClient();
  const mutation = useIdentityMutation<
    RollTargetImportReport,
    RollTargetError,
    string
  >({
    mutationFn: async (text) => {
      const report = await apiFetch<APIImportReport>(
        "/api/rolltargets/import",
        {
          method: "POST",
          headers: { "Content-Type": "text/plain" },
          body: text,
        },
      );
      return toImportReport(report);
    },
    onSuccess: (result, vars) => {
      invalidateRollTargets(client);
      callbacks?.onSuccess?.(result, vars);
    },
    onError: (error, vars) => callbacks?.onError?.(error, vars),
    onSettled: (_result, _error, vars) => callbacks?.onSettled?.(vars),
  });

  return {
    importDIM: (text: string) => mutation.mutate(text),
    isPending: mutation.isPending,
  };
}
