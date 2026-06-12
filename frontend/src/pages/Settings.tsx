import React from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button, DataFreshnessChip, PageHead, Panel } from "../components/kit";
import { useAuth } from "../contexts/AuthContext";
import { usePreferences } from "../contexts/PreferencesContext";
import { apiFetch } from "../lib/api";
import { collectionsQuery } from "../lib/queries";
import { relTime, toCharacter } from "../lib/adapters";
import type {
  APICharacter,
  APICacheRefreshResponse,
} from "../types/api";

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
  const { user, logout: authLogout } = useAuth();
  const { cardStyle, personalize, setCardStyle, setPersonalize } = usePreferences();
  const navigate = useNavigate();

  const { data: charsData } = useQuery({
    queryKey: ["characters", user?.membershipType, user?.membershipId],
    queryFn: () =>
      apiFetch<APICharacter[]>(
        `/api/characters/${user!.membershipType}/${user!.membershipId}`
      ),
    enabled: !!(user?.membershipId) && user?.membershipType != null,
  });

  const characterList = (charsData ?? []).map(toCharacter);

  // Shared "missing" collections query — react-query dedupes with Dashboard and
  // the Collections page's missing view, so this is free once any of them has
  // loaded. Supplies real fetchedAt (B8).
  const { data: collections } = useQuery(
    collectionsQuery(user?.membershipType, user?.membershipId, false)
  );

  const queryClient = useQueryClient();

  const refreshMutation = useMutation({
    mutationFn: () =>
      apiFetch<APICacheRefreshResponse>(
        `/api/collections/${user!.membershipType}/${user!.membershipId}/refresh`,
        { method: "POST" }
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

  const platform =
    user?.platform ||
    (user?.membershipType != null ? PLATFORM_LABEL[user.membershipType] : undefined) ||
    "—";

  return (
    <div className="gt-page">
      <PageHead title="Settings" sub="Account, preferences & data" />

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
              Personalized "Missing" and "Available now" badges across your collection.
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
        <Panel title="Account" icon="bungie">
          <div className="gt-set-row">
            <span className="gt-set-k">Guardian</span>
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
            Guardian Tracker caches your collection and refreshes on demand — it never polls live,
            to respect Bungie's rate limits.
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
            Your collection must be set to public on Bungie.net for Guardian Tracker to read it. We
            never modify your account.
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
      </div>
    </div>
  );
}
