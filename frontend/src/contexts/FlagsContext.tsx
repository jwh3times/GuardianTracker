import React, {
  createContext,
  useCallback,
  useContext,
  useMemo,
  ReactNode,
} from "react";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "../lib/api";
import type { APIFlagsResponse, APIResolvedFlag } from "../types/api";
import type { Role } from "../lib/roles";

/**
 * Resolved gating state for one feature flag — the server-evaluated port of the
 * design's store.jsx flagState():
 *   enabled === false            -> hidden everywhere
 *   enabled && tier < minTier    -> locked (upsell)
 *   enabled && tier >= minTier    -> accessible
 * An unknown key (degraded mode, or a flag the server didn't send) fails open:
 * enabled + accessible, so shipped features are never hidden by a missing flag.
 */
export interface FlagState {
  flag: APIResolvedFlag | null;
  enabled: boolean;
  accessible: boolean;
  locked: boolean;
}

interface FlagsContextType {
  role: Role;
  isAdmin: boolean;
  isLoading: boolean;
  flagState: (key: string) => FlagState;
  accessible: (key: string) => boolean;
  /** Re-fetch resolved flags + role (e.g. after a self opt-in). */
  refresh: () => void;
}

const UNKNOWN: FlagState = {
  flag: null,
  enabled: true,
  accessible: true,
  locked: false,
};

const FlagsContext = createContext<FlagsContextType | undefined>(undefined);

export const FlagsProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const { data, isLoading, refetch } = useQuery({
    queryKey: ["flags"],
    queryFn: () => apiFetch<APIFlagsResponse>("/api/flags"),
    staleTime: 60_000,
  });

  const role: Role = data?.role ?? "standard";

  const byKey = useMemo(() => {
    const m = new Map<string, APIResolvedFlag>();
    (data?.flags ?? []).forEach((f) => m.set(f.key, f));
    return m;
  }, [data]);

  const flagState = useCallback(
    (key: string): FlagState => {
      const f = byKey.get(key);
      if (!f) return UNKNOWN;
      return {
        flag: f,
        enabled: f.enabled,
        accessible: f.accessible,
        locked: f.locked,
      };
    },
    [byKey],
  );

  const accessible = useCallback(
    (key: string) => flagState(key).accessible,
    [flagState],
  );

  const value: FlagsContextType = {
    role,
    isAdmin: role === "admin",
    isLoading,
    flagState,
    accessible,
    refresh: () => void refetch(),
  };

  return (
    <FlagsContext.Provider value={value}>{children}</FlagsContext.Provider>
  );
};

export function useFlags(): FlagsContextType {
  const ctx = useContext(FlagsContext);
  if (!ctx) {
    throw new Error("useFlags must be used within a FlagsProvider");
  }
  return ctx;
}

/** Resolved gating state for a single flag key. */
export function useFlag(key: string): FlagState {
  return useFlags().flagState(key);
}
