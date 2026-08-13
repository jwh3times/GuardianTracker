import React from "react";
import { useNavigate } from "react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button, DataFreshnessChip } from "../../components/primitives";
import { PageHead, Panel } from "../../components/composite";
import { Icon } from "../../components/Icon";
import { RoleBadge } from "../admin/AdminKit";
import { useAuth } from "../../contexts/AuthContext";
import { useFlags } from "../../contexts/FlagsContext";
import { usePreferences } from "../../contexts/PreferencesContext";
import { useToast } from "../../components/Toast";
import { apiFetch, ApiError } from "../../lib/api";
import { collectionsQuery } from "../../lib/queries";
import { toCharacter } from "../../lib/adapters";
import { relTime } from "../../lib/format";
import { MIN_TIERS, ROLE_LABEL, roleColor, type Tier } from "../../lib/roles";
import type {
  APICharacter,
  APICacheRefreshResponse,
  APIRoleResponse,
} from "../../types/api";

type CSS = React.CSSProperties & Record<`--${string}`, string | number>;

const PLATFORM_LABEL: Record<number, string> = {
  1: "Xbox",
  2: "PlayStation",
  3: "Steam",
  4: "Battle.net",
  5: "Stadia",
  6: "Epic Games",
};

function Segmented<T extends string>({
  value,
  options,
  onChange,
}: {
  value: T;
  options: { v: T; l: string }[];
  onChange: (v: T) => void;
}) {
  return (
    <div className="gt-subtabs" role="radiogroup">
      {options.map((o) => (
        <button
          key={o.v}
          className="gt-subtab"
          role="radio"
          aria-checked={value === o.v}
          data-on={value === o.v}
          onClick={() => onChange(o.v)}
        >
          {o.l}
        </button>
      ))}
    </div>
  );
}

export function Settings() {
  const { user, logout: authLogout, logoutAll: authLogoutAll } = useAuth();
  const { role, isAdmin, refresh: refreshFlags } = useFlags();
  const { cardStyle, personalize, setCardStyle, setPersonalize } =
    usePreferences();
  const { showToast } = useToast();
  const navigate = useNavigate();

  const { data: charsData } = useQuery({
    queryKey: ["characters", user?.membershipType, user?.membershipId],
    queryFn: () =>
      apiFetch<APICharacter[]>(
        `/api/characters/${user!.membershipType}/${user!.membershipId}`,
      ),
    enabled: !!user?.membershipId && user?.membershipType != null,
  });

  const characterList = (charsData ?? []).map(toCharacter);

  // Shared "missing" collections query — react-query dedupes with Dashboard and
  // the Collections page's missing view, so this is free once any of them has
  // loaded. Supplies real fetchedAt (B8).
  const { data: collections } = useQuery(
    collectionsQuery(user?.membershipType, user?.membershipId),
  );

  const queryClient = useQueryClient();

  // Self-service early-access opt-in (standard / beta / alpha). The server keeps
  // the session (no token churn) and the new tier propagates on the next request;
  // we refresh resolved flags so gated nav/pages update immediately.
  const roleMutation = useMutation({
    mutationFn: (tier: Tier) =>
      apiFetch<APIRoleResponse>("/api/account/role", {
        method: "PUT",
        body: JSON.stringify({ role: tier }),
      }),
    onSuccess: () => {
      refreshFlags();
      void queryClient.invalidateQueries({ queryKey: ["flags"] });
    },
    onError: (e) =>
      showToast(
        e instanceof ApiError ? e.message : "Couldn't change access tier",
        "error",
      ),
  });

  const refreshMutation = useMutation({
    mutationFn: () =>
      apiFetch<APICacheRefreshResponse>(
        `/api/collections/${user!.membershipType}/${user!.membershipId}/refresh`,
        { method: "POST" },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["collections"] });
      void queryClient.invalidateQueries({ queryKey: ["characters"] });
      void queryClient.invalidateQueries({ queryKey: ["weekly"] });
      void queryClient.invalidateQueries({ queryKey: ["catalysts"] });
      void queryClient.invalidateQueries({ queryKey: ["crafting"] });
      void queryClient.invalidateQueries({ queryKey: ["seals"] });
    },
  });

  const handleSignOut = () => {
    authLogout();
    navigate("/login");
  };

  const handleSignOutAll = () => {
    authLogoutAll();
    navigate("/login");
  };

  const platform =
    user?.platform ||
    (user?.membershipType != null
      ? PLATFORM_LABEL[user.membershipType]
      : undefined) ||
    "—";

  return (
    <div className="gt-page">
      <PageHead title="Settings" sub="Identity, access & data" />

      {/* MEMBERSHIP & ACCESS */}
      <Panel
        title="Membership & access"
        icon="shield"
        accent={roleColor(role)}
        right={<RoleBadge role={role} lg />}
      >
        <p className="gt-set-note">
          Guardian Tracker rolls features out in waves. Opt into an early-access
          tier to try features before they reach everyone —{" "}
          <strong>Beta</strong> for upcoming features, <strong>Alpha</strong>{" "}
          for the bleeding edge.
        </p>
        <div className="gt-access-switch">
          <span className="gt-set-k">Access tier</span>
          <div
            className="gt-tierpick"
            role="radiogroup"
            aria-label="Access tier"
            data-disabled={isAdmin}
          >
            {MIN_TIERS.map((r) => (
              <button
                key={r}
                className="gt-tierpick-opt"
                role="radio"
                aria-checked={role === r}
                data-on={role === r}
                disabled={isAdmin || roleMutation.isPending}
                style={{ "--bdg": roleColor(r) } as CSS}
                onClick={() => roleMutation.mutate(r)}
              >
                {r === "alpha" ? (
                  <Icon name="bolt" size="0.9em" />
                ) : (
                  <span className="gt-dot" />
                )}
                {ROLE_LABEL[r]}
              </button>
            ))}
          </div>
        </div>
        {isAdmin && (
          <>
            <p className="gt-set-note" style={{ color: "var(--c-admin)" }}>
              You have <strong>Admin</strong> access — full visibility of every
              feature. Manage member roles and feature flags in the Admin
              Console.
            </p>
            <Button
              variant="primary"
              icon="shield"
              onClick={() => navigate("/admin")}
            >
              Open Admin Console
            </Button>
          </>
        )}
      </Panel>

      {/* PREFERENCES */}
      <Panel title="Appearance" icon="settings">
        <div className="gt-set-row">
          <div>
            <span className="gt-set-v">Item card style</span>
            <div className="gt-set-note" style={{ marginTop: "0.15rem" }}>
              Full framed cards, or condensed rows that fit more on screen.
            </div>
          </div>
          <Segmented
            value={cardStyle}
            onChange={setCardStyle}
            options={[
              { v: "framed", l: "Framed" },
              { v: "compact", l: "Compact" },
            ]}
          />
        </div>
        <div className="gt-set-row" style={{ borderBottom: "none" }}>
          <div>
            <span className="gt-set-v">"For you" badges</span>
            <div className="gt-set-note" style={{ marginTop: "0.15rem" }}>
              Personalized "Missing" and "Available now" badges across your
              collection.
            </div>
          </div>
          <Segmented
            value={personalize}
            onChange={setPersonalize}
            options={[
              { v: "normal", l: "On" },
              { v: "off", l: "Off" },
            ]}
          />
        </div>
      </Panel>

      <div className="gt-set-grid">
        <Panel title="Destiny membership" icon="bungie">
          <div className="gt-set-row">
            <span className="gt-set-k">Display name</span>
            <span className="gt-set-v">{user?.displayName || "—"}</span>
          </div>
          <div className="gt-set-row">
            <span className="gt-set-k">Platform</span>
            <span className="gt-set-v">{platform}</span>
          </div>
          <div className="gt-set-row" style={{ borderBottom: "none" }}>
            <span className="gt-set-k">Access</span>
            <span className="gt-set-v mono" style={{ color: "var(--c-avail)" }}>
              Read-only collection data
            </span>
          </div>
        </Panel>

        <Panel title="Characters" icon="dashboard">
          {characterList.length > 0 ? (
            characterList.map((c) => (
              <div key={c.id} className="gt-set-row">
                <span className="gt-set-k">
                  {c.name === c.cls ? c.cls : `${c.name} · ${c.cls}`}
                  {c.race ? ` · ${c.race}` : ""}
                </span>
                <span className="gt-set-v mono">{c.power}</span>
              </div>
            ))
          ) : (
            <div className="gt-set-row">
              <span className="gt-set-note">No characters loaded yet.</span>
            </div>
          )}
        </Panel>

        <Panel title="Data freshness" icon="refresh">
          <div className="gt-set-row">
            <span className="gt-set-k">Last updated</span>
            <span className="gt-set-v mono">
              {collections?.fetchedAt ? relTime(collections.fetchedAt) : "—"}
            </span>
          </div>
          <p className="gt-set-note">
            Guardian Tracker caches your collection and refreshes on demand — it
            never polls live, to respect Bungie's rate limits.
          </p>
          <DataFreshnessChip
            updatedAt={collections?.fetchedAt}
            refreshing={refreshMutation.isPending}
            onRefresh={() => {
              if (user?.membershipType != null && !!user?.membershipId) {
                refreshMutation.mutate();
              }
            }}
          />
        </Panel>

        <Panel title="Privacy" icon="lock">
          <p className="gt-set-note">
            Your collection must be set to public on Bungie.net for Guardian
            Tracker to read it. We never modify your Destiny data.
          </p>
          <a
            href="https://www.bungie.net/7/en/User/Account/Privacy"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Button variant="outline" sm icon="external">
              Bungie privacy settings
            </Button>
          </a>
        </Panel>
      </div>

      <div className="gt-dash-actions">
        <Button variant="outline" icon="signout" onClick={handleSignOut}>
          Sign out
        </Button>
        <Button variant="ghost" sm icon="signout" onClick={handleSignOutAll}>
          Sign out all devices
        </Button>
      </div>
    </div>
  );
}
