import React from "react";
import { useNavigate } from "react-router-dom";
import { useMutation, useQuery } from "@apollo/client/react";
import { Button, DataFreshnessChip, PageHead, Panel } from "../components/kit";
import { useAuth } from "../contexts/AuthContext";
import { usePreferences } from "../contexts/PreferencesContext";
import { LOGOUT } from "../graphql/mutations";
import { GET_CHARACTERS } from "../graphql/queries";
import { toCharacter, type GraphQLCharacter } from "../lib/adapters";
import { characters } from "../lib/mockData";

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
  const [logout] = useMutation(LOGOUT);
  const navigate = useNavigate();

  // Real characters when available; otherwise fall back to mock data.
  const { data: charData } = useQuery(GET_CHARACTERS, {
    variables: {
      membershipType: user?.membershipType ?? 0,
      membershipId: user?.membershipId ?? "",
    },
    skip: !user?.membershipId || user?.membershipType == null,
  });
  const realChars = ((charData?.characters ?? []) as GraphQLCharacter[]).map(toCharacter);
  const characterList = realChars.length ? realChars : characters;

  const handleSignOut = async () => {
    try {
      await logout();
    } catch (err) {
      console.error("Logout error:", err);
    } finally {
      authLogout();
      navigate("/login");
    }
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
            <span className="gt-set-v">“For you” badges</span>
            <div className="gt-set-note" style={{ marginTop: "0.15rem" }}>
              Personalized “Missing” and “Available now” badges across your collection.
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
          {characterList.map((c) => (
            <div key={c.id} className="gt-set-row">
              <span className="gt-set-k">
                {c.name === c.cls ? c.cls : `${c.name} · ${c.cls}`}
                {c.race ? ` · ${c.race}` : ""}
              </span>
              <span className="gt-set-v mono">{c.power}</span>
            </div>
          ))}
        </Panel>

        <Panel title="Data freshness" icon="refresh">
          <div className="gt-set-row">
            <span className="gt-set-k">Last updated</span>
            <span className="gt-set-v mono">4m ago</span>
          </div>
          <p className="gt-set-note">
            Guardian Tracker caches your collection and refreshes on demand — it never polls live,
            to respect Bungie's rate limits.
          </p>
          <DataFreshnessChip ago="4m" />
        </Panel>

        <Panel title="Privacy" icon="lock">
          <p className="gt-set-note">
            Your collection must be set to public on Bungie.net for Guardian Tracker to read it. We
            never modify your account.
          </p>
          <Button variant="outline" sm icon="external">
            Bungie privacy settings
          </Button>
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
