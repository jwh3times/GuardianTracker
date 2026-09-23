import { useMemo, useState } from "react";
import { useCollections } from "../../data/collections";
import { useRollTargetMatches, useRollTargets } from "../../data/rolltargets";
import { useWishlist } from "../../data/wishlist";
import type { RollTarget } from "../../types/design";
import {
  applyDefaultFilter,
  applySearch,
  groupChasingTargets,
  groupMatchesByTarget,
  type ChasingGroup,
  type MatchGroup,
  type WeaponLookup,
} from "./rollTargetsView";

/**
 * Wires the pure grouping/filtering logic in `rollTargetsView.ts` to live
 * data: the saved-target list and match report (`data/rolltargets.ts`), plus
 * the item lookup the rest of the app already uses for name/icon/collected
 * status (`data/collections.ts`'s full item join) and the wish list
 * (`data/wishlist.ts`) for the default filter's second condition.
 *
 * Deliberately does not resolve a hash the collections join has no entry for
 * through a second, per-hash lookup: `data/items.ts`'s single-item queries
 * are hook calls, and this page can have an arbitrary number of distinct
 * weapon hashes across its targets and matches. `WeaponLookup` falls back to
 * a hash-derived name for that (rare) case instead.
 */
export function useRollTargetsBrowser() {
  const {
    targets,
    isLoading: targetsLoading,
    isError: targetsError,
    error: targetsErrorObj,
    retry: retryTargets,
  } = useRollTargets();
  const {
    matches,
    isLoading: matchesLoading,
    isError: matchesError,
    error: matchesErrorObj,
    retry: retryMatches,
  } = useRollTargetMatches();
  const { view: collectionsView, isError: collectionsError } = useCollections();
  const { entries: wishlistEntries } = useWishlist();

  const [showAll, setShowAll] = useState(false);
  const [search, setSearch] = useState("");

  const wishlistedHashes = useMemo(
    () => new Set(wishlistEntries.map((w) => w.itemId)),
    [wishlistEntries],
  );

  const lookup: WeaponLookup = useMemo(
    () => ({
      name: (hash) => collectionsView?.itemByHash(hash)?.name ?? "",
      icon: (hash) => collectionsView?.itemByHash(hash)?.icon,
      type: (hash) => collectionsView?.itemByHash(hash)?.type,
      acquiredOrWishlisted: (hash) =>
        !!collectionsView?.itemByHash(hash)?.collected ||
        wishlistedHashes.has(hash),
    }),
    [collectionsView, wishlistedHashes],
  );

  const targetsById = useMemo(
    () => new Map<string, RollTarget>(targets.map((t) => [t.id, t])),
    [targets],
  );

  // Ready means the match report itself resolved — independent of whether
  // the target list is still loading, since the two queries fail separately.
  const matchesReady = !matchesLoading && !matchesError && !!matches;

  const chasingGroups = useMemo<ChasingGroup[]>(
    () =>
      matchesReady ? groupChasingTargets(matches.unmatchedTargets, lookup) : [],
    [matchesReady, matches, lookup],
  );

  const { groups: defaultFiltered, hiddenCount } = useMemo(
    () => applyDefaultFilter(chasingGroups, { showAll, lookup }),
    [chasingGroups, showAll, lookup],
  );

  const visibleChasing = useMemo(
    () => applySearch(defaultFiltered, search),
    [defaultFiltered, search],
  );

  const wantedGroups = useMemo<MatchGroup[]>(
    () =>
      matchesReady
        ? groupMatchesByTarget(matches.wanted, targetsById, lookup, true)
        : [],
    [matchesReady, matches, targetsById, lookup],
  );

  const unwantedGroups = useMemo<MatchGroup[]>(
    () =>
      matchesReady
        ? groupMatchesByTarget(matches.unwanted, targetsById, lookup, false)
        : [],
    [matchesReady, matches, targetsById, lookup],
  );

  return {
    targets,
    targetsLoading,
    targetsError,
    targetsErrorObj,
    retryTargets,

    matchesLoading,
    matchesError,
    matchesErrorObj,
    retryMatches,
    matchesReady,

    collectionsError,
    lookup,

    showAll,
    setShowAll,
    search,
    setSearch,

    stillChasing: matchesReady ? { groups: visibleChasing, hiddenCount } : null,
    wantedGroups,
    unwantedGroups,
  };
}
