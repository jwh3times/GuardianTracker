import React, { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AuditTable } from "./AuditTable";
import { EmptyState, FilterChip, StatTile } from "../../components/primitives";
import { PageHead } from "../../components/composite";
import { Icon } from "../../components/Icon";
import { FlagCard, UserRow } from "./AdminKit";
import { useToast } from "../../components/Toast";
import { useAuth } from "../../contexts/AuthContext";
import { apiFetch, ApiError } from "../../lib/api";
import { relTime } from "../../lib/adapters";
import {
  ROLES,
  ROLE_LABEL,
  tierOf,
  type Role,
  type Tier,
} from "../../lib/roles";
import type { APIAdminFlag, APIAdminUser, APIAuditPage } from "../../types/api";

type Tab = "users" | "flags" | "audit";

export function Admin() {
  const qc = useQueryClient();
  const { showToast } = useToast();
  const { user } = useAuth();
  const [tab, setTab] = useState<Tab>("users");
  const [q, setQ] = useState("");
  const [roleFilter, setRoleFilter] = useState<"all" | Role>("all");

  const usersQuery = useQuery({
    queryKey: ["admin", "users"],
    queryFn: () => apiFetch<APIAdminUser[]>("/api/admin/users"),
  });
  const flagsQuery = useQuery({
    queryKey: ["admin", "flags"],
    queryFn: () => apiFetch<APIAdminFlag[]>("/api/admin/flags"),
  });

  const [auditType, setAuditType] = useState<string>("");
  const auditQuery = useQuery({
    queryKey: ["admin", "audit", auditType],
    queryFn: () =>
      apiFetch<APIAuditPage>(
        `/api/admin/audit?limit=100${auditType ? `&type=${encodeURIComponent(auditType)}` : ""}`,
      ),
    enabled: tab === "audit",
  });

  const users = useMemo(() => usersQuery.data ?? [], [usersQuery.data]);
  const flags = flagsQuery.data ?? [];

  const roleMutation = useMutation({
    mutationFn: ({ id, role }: { id: string; role: Role }) =>
      apiFetch(`/api/admin/users/${id}/role`, {
        method: "PUT",
        body: JSON.stringify({ role }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["admin", "users"] });
      void qc.invalidateQueries({ queryKey: ["flags"] });
    },
    onError: (e) =>
      showToast(
        e instanceof ApiError ? e.message : "Couldn't change role",
        "error",
      ),
  });

  const flagMutation = useMutation({
    mutationFn: ({
      key,
      patch,
    }: {
      key: string;
      patch: { enabled?: boolean; minTier?: Tier };
    }) =>
      apiFetch<APIAdminFlag>(`/api/admin/flags/${key}`, {
        method: "PUT",
        body: JSON.stringify(patch),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["admin", "flags"] });
      void qc.invalidateQueries({ queryKey: ["flags"] });
    },
    onError: (e) =>
      showToast(
        e instanceof ApiError ? e.message : "Couldn't update flag",
        "error",
      ),
  });

  const roleCounts = useMemo(() => {
    const c: Record<string, number> = { all: users.length };
    ROLES.forEach((r) => (c[r] = users.filter((u) => u.role === r).length));
    return c;
  }, [users]);

  const filtered = users.filter(
    (u) =>
      (roleFilter === "all" || u.role === roleFilter) &&
      (q.length < 1 ||
        (u.displayName + u.membershipId)
          .toLowerCase()
          .includes(q.toLowerCase())),
  );

  const flagOn = flags.filter((f) => f.enabled).length;
  const totalUsers = users.length;
  const reach = (f: APIAdminFlag) =>
    f.enabled
      ? users.filter((u) => tierOf(u.role) >= tierOf(f.minTier)).length
      : 0;

  return (
    <div className="gt-page">
      <PageHead
        title={
          <span className="gt-admin-title">
            <Icon
              name="shield"
              size="1.4rem"
              style={{ color: "var(--c-admin)" }}
            />{" "}
            Admin Console
          </span>
        }
        sub="Manage member roles and feature-flag rollout"
        right={
          <div className="gt-subtabs">
            <button
              className="gt-subtab"
              data-on={tab === "users"}
              onClick={() => setTab("users")}
            >
              <Icon name="users" size="0.95rem" /> Users
            </button>
            <button
              className="gt-subtab"
              data-on={tab === "flags"}
              onClick={() => setTab("flags")}
            >
              <Icon name="flag" size="0.95rem" /> Feature Flags
            </button>
            <button
              className="gt-subtab"
              data-on={tab === "audit"}
              onClick={() => setTab("audit")}
            >
              <Icon name="shield" size="0.95rem" /> Audit Log
            </button>
          </div>
        }
      />

      {tab === "users" ? (
        <>
          <div className="gt-coll-stats">
            <StatTile num={roleCounts.all} label="Total members" mono />
            <StatTile
              num={roleCounts.beta}
              label="Beta"
              mono
              color="var(--c-rare)"
            />
            <StatTile
              num={roleCounts.alpha}
              label="Alpha"
              mono
              color="var(--c-exotic)"
            />
            <StatTile
              num={roleCounts.admin}
              label="Admins"
              mono
              color="var(--c-admin)"
            />
          </div>

          <div className="gt-coll-toolbar">
            <div className="gt-search gt-admin-search">
              <Icon
                name="search"
                size="1rem"
                style={{ color: "var(--c-text-3)" }}
              />
              <input
                className="gt-search-input"
                placeholder="Search members…"
                value={q}
                onChange={(e) => setQ(e.target.value)}
              />
            </div>
            <div className="gt-filterbar">
              {(["all", ...ROLES] as const).map((r) => (
                <FilterChip
                  key={r}
                  on={roleFilter === r}
                  onClick={() => setRoleFilter(r)}
                >
                  {r === "all" ? "All" : ROLE_LABEL[r]}{" "}
                  <span className="mono" style={{ opacity: 0.6 }}>
                    {roleCounts[r]}
                  </span>
                </FilterChip>
              ))}
            </div>
          </div>

          <div className="gt-card gt-pad0">
            <div className="gt-userrow gt-userrow--head">
              <span></span>
              <span className="gt-userrow-h">Member</span>
              <span className="gt-userrow-h gt-userrow-meta">
                Platform · last active
              </span>
              <span className="gt-userrow-h" style={{ textAlign: "right" }}>
                Role
              </span>
            </div>
            {usersQuery.isLoading ? (
              <div style={{ padding: "var(--s-6)" }}>
                <EmptyState icon="users" title="Loading members…" />
              </div>
            ) : filtered.length === 0 ? (
              <div style={{ padding: "var(--s-6)" }}>
                <EmptyState
                  icon="users"
                  title="No members match"
                  body="Try a different search or role filter."
                />
              </div>
            ) : (
              filtered.map((u) => (
                <UserRow
                  key={u.id}
                  name={u.displayName}
                  handle={u.membershipId}
                  role={u.role}
                  platform={u.platform}
                  active={relTime(u.lastActive)}
                  me={u.membershipId === user?.membershipId}
                  onPick={(r) => roleMutation.mutate({ id: u.id, role: r })}
                />
              ))
            )}
          </div>
        </>
      ) : tab === "flags" ? (
        <>
          <div className="gt-coll-stats">
            <StatTile
              num={`${flagOn}/${flags.length}`}
              label="Flags enabled"
              mono
            />
            <StatTile
              num={
                flags.filter((f) => f.minTier === "beta" && f.enabled).length
              }
              label="Beta-gated"
              mono
              color="var(--c-rare)"
            />
            <StatTile
              num={
                flags.filter((f) => f.minTier === "alpha" && f.enabled).length
              }
              label="Alpha-gated"
              mono
              color="var(--c-exotic)"
            />
            <div className="gt-coll-resultcount mono">
              Changes apply live across the app
            </div>
          </div>
          {flagsQuery.isLoading ? (
            <EmptyState icon="flag" title="Loading feature flags…" />
          ) : (
            <div className="gt-flag-grid">
              {flags.map((f) => (
                <FlagCard
                  key={f.key}
                  flag={f}
                  reached={reach(f)}
                  totalUsers={totalUsers}
                  pending={flagMutation.isPending}
                  onToggle={(enabled) =>
                    flagMutation.mutate({ key: f.key, patch: { enabled } })
                  }
                  onSetMinTier={(minTier) =>
                    flagMutation.mutate({ key: f.key, patch: { minTier } })
                  }
                />
              ))}
            </div>
          )}
        </>
      ) : (
        <>
          <div className="gt-coll-toolbar">
            <div className="gt-filterbar">
              {[
                ["", "All"],
                ["login.", "Logins"],
                ["logout.", "Logouts"],
                ["refresh.", "Sessions"],
                ["role.", "Roles"],
                ["flag.update", "Flags"],
              ].map(([val, lbl]) => (
                <FilterChip
                  key={val || "all"}
                  on={auditType === val}
                  onClick={() => setAuditType(val)}
                >
                  {lbl}
                </FilterChip>
              ))}
            </div>
          </div>
          <AuditTable
            entries={auditQuery.data?.entries ?? []}
            loading={auditQuery.isLoading}
          />
        </>
      )}
    </div>
  );
}
