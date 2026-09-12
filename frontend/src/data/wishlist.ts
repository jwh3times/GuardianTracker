import { useCallback } from "react";
import {
  useQuery,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";
import { useIdentityMutation } from "../contexts/IdentityMutation";
import { apiFetch, type ApiError } from "../lib/api";
import { RARITY_MAP } from "../lib/adapters";
import { toAcquisitionSource } from "../lib/acquisitionSources";
import { relTime } from "../lib/format";
import type { WishListItem } from "../types/api";
import type { Priority, WishlistEntry } from "../types/design";

/**
 * The Wish list data-access module (ADR 0020). It owns this resource's query
 * identity, endpoint paths, projection to {@link WishlistEntry}, every mutation
 * and its optimistic coordination, and its own invalidation.
 *
 * Three features read this resource — Wish list, Collections and Dashboard —
 * which is why ownership follows the resource rather than the page. Before this
 * module they declared the query three times and the mutations six times
 * between them, and Collections reached into the cache to resolve a row id.
 *
 * Features get hooks and domain types only. The wire type, the query key and
 * React Query itself stay inside this file.
 */

/**
 * Private, and deliberately not exported: no other module may name this key.
 *
 * Flat rather than membership-qualified because `/api/wishlist` carries no
 * membership — the server derives it from the session JWT — so membership is
 * not an input this request varies on. ADR 0017's identity cleanup clears the
 * whole QueryClient when the membership changes, which is what keeps one
 * membership's rows from surviving into another's session.
 */
const WISHLIST_KEY = ["wishlist"] as const;

const PRIORITY_MAP: Record<string, Priority> = {
  URGENT: "urgent",
  HIGH: "high",
  MEDIUM: "medium",
  LOW: "low",
};

/** Stable empty result, so `entries` keeps referential identity across renders. */
const NO_ENTRIES: WishlistEntry[] = [];

/** Adapt a REST API WishListItem into the design system's WishlistEntry shape. */
function toWishlistEntry(w: WishListItem): WishlistEntry {
  return {
    id: w.id,
    itemId: String(w.itemHash),
    name: w.name,
    type: w.itemType,
    rarity: RARITY_MAP[w.rarity] ?? "legendary",
    icon: w.icon || undefined,
    priority: PRIORITY_MAP[w.priority] ?? "medium",
    avail: {
      now: w.availableNow ?? false,
      where: w.availableNow ? (w.availableFrom ?? "Xûr") : "",
    },
    acquisitionSources: (w.acquisitionSources ?? []).map(toAcquisitionSource),
    notes: w.notes ?? "",
    added: relTime(w.dateAdded),
  };
}

/**
 * The query's `select`. Must stay a stable module-level reference: React Query
 * memoises `select` per observer keyed on function identity, so an inline arrow
 * would re-project every row on every render of all three consumers.
 */
function toWishlistEntries(rows: WishListItem[]): WishlistEntry[] {
  return rows.map(toWishlistEntry);
}

/** Every failure surfaced by this module. `apiFetch` throws only `ApiError`. */
export type WishlistError = ApiError;

/**
 * Feature-owned side effects for a mutation. The module owns the cache half —
 * cancel, snapshot, optimistic write, rollback, settle — and the feature owns
 * the user-visible copy, which legitimately differs by context: the two
 * delete-failure messages are deliberately not unified (ADR 0020).
 *
 * Framework-neutral by design. React Query's own option types must not reach a
 * feature module, so these carry domain values only.
 */
export interface WishlistMutationCallbacks<TVars, TResult = void> {
  onSuccess?: (result: TResult, vars: TVars) => void;
  onError?: (error: WishlistError, vars: TVars) => void;
  onSettled?: (vars: TVars) => void;
}

/** Snapshot taken before an optimistic write, replayed on failure. */
interface Rollback {
  previous?: WishListItem[];
}

function invalidateWishlist(client: QueryClient) {
  void client.invalidateQueries({ queryKey: WISHLIST_KEY });
}

/**
 * The one optimistic recipe, shared by every mutation that can predict the
 * server's answer. Consolidating it is the point of this slice: the four
 * hand-rolled copies on the Wish list page each repeated cancel, snapshot,
 * write, rollback and settle, so a fix to any of those five steps had to be
 * found by grep and applied four times.
 *
 * `apply` is the only part that differs per mutation, and it is expressed in
 * wire terms because the cache stores the wire shape (ADR 0020) — that cost
 * lands inside the module that already owns the wire type.
 */
function useOptimisticWishlistMutation<TVars, TResult>(
  mutationFn: (vars: TVars) => Promise<TResult>,
  apply: (rows: WishListItem[], vars: TVars) => WishListItem[],
  callbacks?: WishlistMutationCallbacks<TVars, TResult>,
) {
  const client = useQueryClient();
  return useIdentityMutation<TResult, WishlistError, TVars, Rollback>({
    mutationFn,
    onMutate: async (vars) => {
      await client.cancelQueries({ queryKey: WISHLIST_KEY });
      const previous = client.getQueryData<WishListItem[]>(WISHLIST_KEY);
      client.setQueryData<WishListItem[]>(WISHLIST_KEY, (old) =>
        apply(old ?? [], vars),
      );
      return { previous };
    },
    onSuccess: (result, vars) => callbacks?.onSuccess?.(result, vars),
    onError: (error, vars, context) => {
      // Restores the whole snapshot rather than reverting just this mutation's
      // delta. With two mutations in flight, a failure here can momentarily
      // stomp a sibling that already settled; the unconditional invalidation
      // below closes that window on the next frame. Accepted, and inherited
      // from the four hand-rolled copies this replaces — but it is the shape
      // every later resource will copy, so it is stated rather than implied.
      if (context?.previous) {
        client.setQueryData(WISHLIST_KEY, context.previous);
      }
      callbacks?.onError?.(error, vars);
    },
    onSettled: (_result, _error, vars) => {
      invalidateWishlist(client);
      callbacks?.onSettled?.(vars);
    },
  });
}

/**
 * The membership's wish list, projected to domain entries.
 *
 * Takes no membership argument: the endpoint derives it from the session, so a
 * page cannot ask for a wish list other than the current one.
 */
export function useWishlist() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: WISHLIST_KEY,
    queryFn: () => apiFetch<WishListItem[]>("/api/wishlist"),
    select: toWishlistEntries,
  });

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return {
    entries: data ?? NO_ENTRIES,
    isLoading,
    isError,
    error,
    retry,
  };
}

export interface AddWishlistItemVars {
  /** The item's hash, as carried by the design-system item id. */
  itemId: string;
  /** The item's name, for caller-owned copy. */
  name: string;
}

/**
 * Add an item to the wish list.
 *
 * Deliberately not optimistic, unlike the mutations below: the server assigns
 * the row id and resolves the name, rarity, icon and acquisition sources, so
 * there is no honest row to write into the cache ahead of the response. It
 * invalidates on success instead.
 */
export function useAddWishlistItem(
  callbacks?: WishlistMutationCallbacks<AddWishlistItemVars>,
) {
  const client = useQueryClient();
  const mutation = useIdentityMutation<
    void,
    WishlistError,
    AddWishlistItemVars
  >({
    mutationFn: (vars) =>
      apiFetch<void>("/api/wishlist", {
        method: "POST",
        body: JSON.stringify({ itemHash: Number(vars.itemId) }),
      }),
    onSuccess: (result, vars) => {
      invalidateWishlist(client);
      callbacks?.onSuccess?.(result, vars);
    },
    onError: (error, vars) => callbacks?.onError?.(error, vars),
    onSettled: (_result, _error, vars) => callbacks?.onSettled?.(vars),
  });

  return {
    add: (vars: AddWishlistItemVars) => mutation.mutate(vars),
    isPending: mutation.isPending,
  };
}

export interface RemoveWishlistItemVars {
  /** The wish list row id — the `DELETE` path segment. */
  rowId: string;
  /**
   * The item's hash. Required, though only Collections reads it back, because
   * both call sites hold it and an optional field here forced a non-null
   * assertion at the one that needs it.
   */
  itemId: string;
  /** The item's name, for caller-owned copy. */
  name: string;
}

/**
 * Remove a row from the wish list, optimistically.
 *
 * One hook for both call sites. Collections previously ran a non-optimistic
 * variant of this same `DELETE`, so its wish-list star now clears on click
 * rather than on the refetch that follows. Two cache policies for one endpoint
 * is the incoherence this module exists to remove.
 */
export function useRemoveWishlistItem(
  callbacks?: WishlistMutationCallbacks<RemoveWishlistItemVars>,
) {
  const mutation = useOptimisticWishlistMutation<RemoveWishlistItemVars, void>(
    (vars) =>
      apiFetch<void>(`/api/wishlist/${vars.rowId}`, { method: "DELETE" }),
    (rows, vars) => rows.filter((r) => r.id !== vars.rowId),
    callbacks,
  );

  return {
    remove: (vars: RemoveWishlistItemVars) => mutation.mutate(vars),
    isPending: mutation.isPending,
  };
}

export interface SetWishlistPriorityVars {
  /** The wish list row id. */
  rowId: string;
  priority: Priority;
}

/** Change a row's priority, optimistically. */
export function useSetWishlistPriority(
  callbacks?: WishlistMutationCallbacks<SetWishlistPriorityVars>,
) {
  const mutation = useOptimisticWishlistMutation<SetWishlistPriorityVars, void>(
    async (vars) => {
      await apiFetch<WishListItem>(`/api/wishlist/${vars.rowId}`, {
        method: "PUT",
        body: JSON.stringify({ priority: wirePriority(vars.priority) }),
      });
    },
    (rows, vars) =>
      rows.map((r) =>
        r.id === vars.rowId
          ? { ...r, priority: wirePriority(vars.priority) }
          : r,
      ),
    callbacks,
  );

  return {
    setPriority: (vars: SetWishlistPriorityVars) => mutation.mutate(vars),
    isPending: mutation.isPending,
  };
}

export interface SetWishlistNotesVars {
  /** The wish list row id. */
  rowId: string;
  notes: string;
}

/** Change a row's notes, optimistically. */
export function useSetWishlistNotes(
  callbacks?: WishlistMutationCallbacks<SetWishlistNotesVars>,
) {
  const mutation = useOptimisticWishlistMutation<SetWishlistNotesVars, void>(
    async (vars) => {
      await apiFetch<WishListItem>(`/api/wishlist/${vars.rowId}`, {
        method: "PUT",
        body: JSON.stringify({ notes: vars.notes }),
      });
    },
    (rows, vars) =>
      rows.map((r) => (r.id === vars.rowId ? { ...r, notes: vars.notes } : r)),
    callbacks,
  );

  return {
    setNotes: (vars: SetWishlistNotesVars) => mutation.mutate(vars),
    isPending: mutation.isPending,
  };
}

export type BulkWishlistAction =
  | { action: "delete"; rowIds: string[] }
  | { action: "set_priority"; rowIds: string[]; priority: Priority };

/** How many rows a bulk action changed, and how many it declined to touch. */
export interface BulkWishlistResult {
  updated: number;
  skipped: number;
}

/** Apply one action to many rows, optimistically. */
export function useBulkWishlistAction(
  callbacks?: WishlistMutationCallbacks<BulkWishlistAction, BulkWishlistResult>,
) {
  const mutation = useOptimisticWishlistMutation<
    BulkWishlistAction,
    BulkWishlistResult
  >(
    (vars) =>
      apiFetch<BulkWishlistResult>("/api/wishlist/bulk", {
        method: "POST",
        body: JSON.stringify({
          action: vars.action,
          ids: vars.rowIds.map(Number),
          ...(vars.action === "set_priority"
            ? { priority: wirePriority(vars.priority) }
            : {}),
        }),
      }),
    (rows, vars) => {
      const ids = new Set(vars.rowIds);
      if (vars.action === "delete") return rows.filter((r) => !ids.has(r.id));
      return rows.map((r) =>
        ids.has(r.id) ? { ...r, priority: wirePriority(vars.priority) } : r,
      );
    },
    callbacks,
  );

  return {
    runBulkAction: (vars: BulkWishlistAction) => mutation.mutate(vars),
    isPending: mutation.isPending,
  };
}

/**
 * Domain priority to the wire's upper-case vocabulary. Private: the wire
 * spelling is this module's business, and before this slice each caller
 * up-cased it inline at the point of mutation.
 */
function wirePriority(p: Priority): string {
  return p.toUpperCase();
}
