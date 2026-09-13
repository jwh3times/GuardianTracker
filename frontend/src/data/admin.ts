import { useCallback } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useIdentityMutation } from "../contexts/IdentityMutation";
import { apiFetch } from "../lib/api";
import { toTier, type Role, type Tier } from "../lib/roles";
import type {
  APIAdminFlag,
  APIAdminUser,
  APIAuditEntry,
  APIAuditPage,
  APIAuditParty,
} from "../types/api";

/**
 * The Admin data-access module (ADR 0020). It owns the admin console's query
 * identities, endpoint paths, projection to domain types, both mutations, and
 * their invalidation.
 *
 * Three resources behind one console — the member roster, the flag
 * configuration, and the audit log — share this module because they share one
 * consumer and one authorization boundary (the server's RequireAdmin).
 *
 * The mutations are deliberately not optimistic, matching the console before
 * this module: a role or rollout change should be seen to have been accepted
 * by the server. Each invalidates only its own resource; a caller that also
 * needs the signed-in user's resolved flags refreshed (the console does) asks
 * `useFlags().refresh` through its callback, so this module never names the
 * Flags module's key.
 *
 * Admin data is not membership-scoped, so it is not part of the
 * membership-refresh fan-out.
 */

/** Private, and deliberately not exported: no other module may name these keys. */
const USERS_KEY = ["admin", "users"] as const;
const FLAGS_KEY = ["admin", "flags"] as const;

/** Private: the audit event type is a genuine query parameter, so it is part of the key. */
function auditKey(type: string) {
  return ["admin", "audit", type] as const;
}

/** One member, as the roster shows them. */
export interface AdminUser {
  id: string;
  displayName: string;
  membershipId: string;
  platform: string;
  role: Role;
  /** RFC3339. */
  lastActive: string;
}

/** One feature flag's configuration, as an admin edits it. */
export interface AdminFlag {
  key: string;
  name: string;
  description: string;
  category: string;
  /** The lowest tier the flag is gated to. */
  minTier: Tier;
  enabled: boolean;
  updatedAt: string;
}

/** The member who acted, or was acted on, in an audit event. */
export interface AuditParty {
  membershipId: string;
  displayName: string;
}

/** One audit log event. */
export interface AuditEntry {
  id: string;
  eventType: string;
  outcome: "success" | "failure";
  actor: AuditParty;
  target?: AuditParty;
  ip?: string;
  details: Record<string, unknown>;
  createdAt: string;
}

function toAdminUser(u: APIAdminUser): AdminUser {
  return {
    id: u.id,
    displayName: u.displayName,
    membershipId: u.membershipId,
    platform: u.platform,
    role: u.role,
    lastActive: u.lastActive,
  };
}

function toAdminFlag(f: APIAdminFlag): AdminFlag {
  return {
    key: f.key,
    name: f.name,
    description: f.description,
    category: f.category,
    minTier: toTier(f.minTier),
    enabled: f.enabled,
    updatedAt: f.updatedAt,
  };
}

function toAuditParty(p: APIAuditParty): AuditParty {
  return { membershipId: p.membershipId, displayName: p.displayName };
}

function toAuditEntry(e: APIAuditEntry): AuditEntry {
  return {
    id: e.id,
    eventType: e.eventType,
    outcome: e.outcome,
    actor: toAuditParty(e.actor),
    target: e.target ? toAuditParty(e.target) : undefined,
    ip: e.ip,
    details: e.details ?? {},
    createdAt: e.createdAt,
  };
}

/*
 * The queries' `select` functions. Stable module-level references so React
 * Query's per-observer `select` memoisation holds.
 */
function toAdminUsers(rows: APIAdminUser[]): AdminUser[] {
  return rows.map(toAdminUser);
}

function toAdminFlags(rows: APIAdminFlag[]): AdminFlag[] {
  return rows.map(toAdminFlag);
}

function toAuditEntries(page: APIAuditPage): AuditEntry[] {
  return (page.entries ?? []).map(toAuditEntry);
}

/* Stable empty results, so each list keeps referential identity. */
const NO_USERS: AdminUser[] = [];
const NO_FLAGS: AdminFlag[] = [];
const NO_ENTRIES: AuditEntry[] = [];

/** Every member and their role. */
export function useAdminUsers() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: USERS_KEY,
    queryFn: () => apiFetch<APIAdminUser[]>("/api/admin/users"),
    select: toAdminUsers,
  });

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return { users: data ?? NO_USERS, isLoading, isError, error, retry };
}

/** Every feature flag's configuration. */
export function useAdminFlags() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: FLAGS_KEY,
    queryFn: () => apiFetch<APIAdminFlag[]>("/api/admin/flags"),
    select: toAdminFlags,
  });

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  return { flags: data ?? NO_FLAGS, isLoading, isError, error, retry };
}

interface AuditLogOptions {
  /**
   * Extra gate, a plain boolean so React Query's option shape stays inside
   * this module. The console reads the log only while its tab is open.
   */
  enabled?: boolean;
}

/**
 * The newest hundred audit events, optionally narrowed to an event type or a
 * type-family prefix such as `logout.`. An empty type means every event.
 */
export function useAuditLog(
  type: string,
  { enabled = true }: AuditLogOptions = {},
) {
  const { data, isLoading } = useQuery({
    queryKey: auditKey(type),
    queryFn: () =>
      apiFetch<APIAuditPage>(
        `/api/admin/audit?limit=100${type ? `&type=${encodeURIComponent(type)}` : ""}`,
      ),
    enabled,
    select: toAuditEntries,
  });

  return { entries: data ?? NO_ENTRIES, isLoading };
}

/**
 * Feature-owned side effects for an admin mutation. The module owns the cache
 * half; the feature owns user-visible copy and any cross-resource follow-up.
 * Framework-neutral: domain values only.
 */
export interface AdminMutationCallbacks<TVars> {
  onSuccess?: (vars: TVars) => void;
  onError?: (error: Error, vars: TVars) => void;
}

export interface SetMemberRoleVars {
  /** The member's Guardian Tracker user id. */
  id: string;
  role: Role;
}

/** Change a member's role, then refresh the roster. */
export function useSetMemberRole(
  callbacks?: AdminMutationCallbacks<SetMemberRoleVars>,
) {
  const client = useQueryClient();
  const mutation = useIdentityMutation<unknown, Error, SetMemberRoleVars>({
    mutationFn: ({ id, role }) =>
      apiFetch(`/api/admin/users/${id}/role`, {
        method: "PUT",
        body: JSON.stringify({ role }),
      }),
    onSuccess: (_result, vars) => {
      void client.invalidateQueries({ queryKey: USERS_KEY });
      callbacks?.onSuccess?.(vars);
    },
    onError: (error, vars) => callbacks?.onError?.(error, vars),
  });

  return {
    setRole: (vars: SetMemberRoleVars) => mutation.mutate(vars),
    isPending: mutation.isPending,
  };
}

export interface UpdateFlagVars {
  key: string;
  patch: { enabled?: boolean; minTier?: Tier };
}

/** Enable, disable or re-gate a feature flag, then refresh the flag list. */
export function useUpdateFlag(
  callbacks?: AdminMutationCallbacks<UpdateFlagVars>,
) {
  const client = useQueryClient();
  const mutation = useIdentityMutation<unknown, Error, UpdateFlagVars>({
    mutationFn: ({ key, patch }) =>
      apiFetch<APIAdminFlag>(`/api/admin/flags/${key}`, {
        method: "PUT",
        body: JSON.stringify(patch),
      }),
    onSuccess: (_result, vars) => {
      void client.invalidateQueries({ queryKey: FLAGS_KEY });
      callbacks?.onSuccess?.(vars);
    },
    onError: (error, vars) => callbacks?.onError?.(error, vars),
  });

  return {
    updateFlag: (vars: UpdateFlagVars) => mutation.mutate(vars),
    isPending: mutation.isPending,
  };
}
