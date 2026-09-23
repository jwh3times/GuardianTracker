import { useToast } from "./Toast";
import React, { useEffect, useState } from "react";
import { NavLink, useLocation, useNavigate } from "react-router";
import { Brand } from "./Brand";
import { Icon } from "./Icon";
import { ItemTile } from "./primitives";
import { useAuth } from "../contexts/AuthContext";
import { useCharacters } from "../contexts/CharacterContext";
import { useFlags } from "../contexts/FlagsContext";
import { MIN_SEARCH_LENGTH, useItemSearch } from "../data/search";
import { useOutsideClick } from "../lib/useOutsideClick";
import { emblemStyle } from "../lib/emblem";

interface NavItem {
  id: string;
  label: string;
  icon: string;
  path: string;
}

const NAV: NavItem[] = [
  {
    id: "dashboard",
    label: "Dashboard",
    icon: "dashboard",
    path: "/dashboard",
  },
  { id: "guardian", label: "Guardian", icon: "guardian", path: "/guardian" },
  { id: "week", label: "This Week", icon: "week", path: "/this-week" },
  {
    id: "collections",
    label: "Collections",
    icon: "collections",
    path: "/collections",
  },
  { id: "cosmetics", label: "Cosmetics", icon: "sparkle", path: "/cosmetics" },
  {
    id: "catalysts",
    label: "Catalysts & Crafting",
    icon: "catalyst",
    path: "/catalysts",
  },
  {
    id: "triumphs",
    label: "Triumphs & Seals",
    icon: "triumph",
    path: "/triumphs",
  },
  { id: "wishlist", label: "Wishlist", icon: "wishlist", path: "/wishlist" },
  { id: "rolls", label: "Roll targets", icon: "star", path: "/rolls" },
];

// Nav items gated by a feature flag: hidden when the flag is disabled, marked
// with a lock when the user's tier is below the flag's minimum (the route then
// renders the upsell). Mirrors NAV_FLAG in the design's app.jsx.
const NAV_FLAG: Record<string, string> = {
  week: "weekly-planner",
  catalysts: "catalysts-crafting",
  triumphs: "triumphs-seals",
  cosmetics: "cosmetics",
  rolls: "god-roll",
};
const MOBILE_NAV: NavItem[] = [
  { id: "dashboard", label: "Home", icon: "dashboard", path: "/dashboard" },
  { id: "week", label: "Week", icon: "week", path: "/this-week" },
  {
    id: "collections",
    label: "Collect",
    icon: "collections",
    path: "/collections",
  },
  { id: "wishlist", label: "Wishlist", icon: "wishlist", path: "/wishlist" },
];

/* ---------------- CHARACTER SWITCHER ---------------- */

function CharacterSwitcher({ displayName }: { displayName?: string }) {
  const {
    characters: list,
    activeCharacter,
    setActiveCharacter,
  } = useCharacters();

  const [open, setOpen] = useState(false);
  const ref = useOutsideClick<HTMLDivElement>(() => setOpen(false));

  if (!list.length || !activeCharacter) {
    return (
      <div className="gt-charsw">
        <div className="gt-charsw-btn">
          <span className="gt-avatar">{(displayName || "G")[0]}</span>
          <span className="gt-charsw-name">{displayName}</span>
        </div>
      </div>
    );
  }

  const cur = activeCharacter;
  return (
    <div className="gt-charsw" ref={ref}>
      <button className="gt-charsw-btn" onClick={() => setOpen((v) => !v)}>
        <span
          className="gt-avatar"
          data-cls={cur.cls}
          style={emblemStyle(cur.emblemUrl)}
        >
          {cur.emblemUrl ? "" : (displayName || cur.name)[0]}
        </span>
        <span className="gt-charsw-name">{displayName || cur.name}</span>
        <Icon
          name="chevronDown"
          size="0.8rem"
          style={{ color: "var(--c-text-3)" }}
        />
      </button>
      {open && (
        <div className="gt-charsw-menu">
          <div className="gt-charsw-head mono">Switch Guardian</div>
          {list.map((c) => (
            <button
              key={c.id}
              className="gt-charsw-opt"
              data-on={c.id === cur.id}
              onClick={() => {
                setActiveCharacter(c.id);
                setOpen(false);
              }}
            >
              <span
                className="gt-avatar"
                data-cls={c.cls}
                style={emblemStyle(c.emblemUrl)}
              >
                {c.emblemUrl ? "" : c.name[0]}
              </span>
              <div className="gt-charsw-opt-main">
                <span className="gt-charsw-opt-name">
                  {c.name === c.cls ? c.cls : `${c.name} — ${c.cls}`}
                </span>
                <span className="gt-charsw-opt-sub mono">
                  {c.race} · {c.power}
                </span>
              </div>
              {c.id === cur.id && (
                <Icon
                  name="check"
                  size="0.9rem"
                  style={{ color: "var(--c-signal)" }}
                />
              )}
            </button>
          ))}
          <div className="gt-charsw-note mono">
            Vendors shown for this Guardian · collections are membership-wide
          </div>
        </div>
      )}
    </div>
  );
}

/* ---------------- GLOBAL SEARCH ---------------- */
function SearchBar() {
  const [q, setQ] = useState("");
  const [debouncedQ, setDebouncedQ] = useState("");
  const [open, setOpen] = useState(false);
  const ref = useOutsideClick<HTMLDivElement>(() => setOpen(false));
  const navigate = useNavigate();

  // 250ms debounce
  useEffect(() => {
    const id = setTimeout(() => setDebouncedQ(q), 250);
    return () => clearTimeout(id);
  }, [q]);

  const {
    results,
    isLoading: searching,
    isError: searchFailed,
  } = useItemSearch(debouncedQ);

  return (
    <div className="gt-search" ref={ref}>
      <Icon name="search" size="1rem" style={{ color: "var(--c-text-3)" }} />
      <input
        className="gt-search-input"
        type="search"
        aria-label="Search items"
        placeholder="Search items…"
        value={q}
        onChange={(e) => {
          setQ(e.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
      />
      {open && q.length >= MIN_SEARCH_LENGTH && (
        <div className="gt-search-menu">
          {searching ? (
            <div className="gt-search-empty mono">Searching…</div>
          ) : searchFailed ? (
            // A one-line message rather than the shared QueryErrorPanel: a
            // gt-card with an icon, body and two buttons would break this
            // dropdown's layout.
            <div className="gt-search-empty mono">Search unavailable</div>
          ) : results.length ? (
            results.slice(0, 6).map((i) => (
              <button
                key={i.id}
                className="gt-search-opt"
                data-rarity={i.rarity}
                onClick={() => {
                  setOpen(false);
                  setQ("");
                  navigate(`/collections?item=${i.id}`);
                }}
              >
                <ItemTile
                  rarity={i.rarity}
                  type={i.type}
                  icon={i.icon}
                  style={{ width: "1.8rem" }}
                />
                <span className="gt-search-opt-name">{i.name}</span>
                <span className="gt-item-type">{i.type}</span>
              </button>
            ))
          ) : (
            <div className="gt-search-empty mono">No items match "{q}"</div>
          )}
        </div>
      )}
    </div>
  );
}

/* ---------------- APP SHELL ---------------- */
export function AppShell({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout: authLogout } = useAuth();
  const { showToast } = useToast();
  const { flagState, accessible, isAdmin } = useFlags();
  const [mobileNav, setMobileNav] = useState(false);
  const closeMobileNav = () => setMobileNav(false);

  const isActive = (path: string) => location.pathname.startsWith(path);

  // Drop nav items whose flag is disabled; flag locked items so they render a lock.
  const visibleNav = NAV.filter((n) => {
    const fk = NAV_FLAG[n.id];
    return !fk || flagState(fk).enabled;
  }).map((n) => {
    const fk = NAV_FLAG[n.id];
    return { ...n, locked: fk ? flagState(fk).locked : false };
  });

  // Drop bottom-tab items whose flag is disabled; mirrors the sidebar's drop rule.
  // Locked items stay tappable (no lock affordance on the compact tab) and route to
  // the upsell via FlaggedRoute — consistent with decision 3 of the design.
  const visibleMobileNav = MOBILE_NAV.filter((n) => {
    const fk = NAV_FLAG[n.id];
    return !fk || flagState(fk).enabled;
  });

  const handleSignOut = () => {
    void authLogout()
      .then(() => navigate("/login"))
      .catch(() =>
        showToast("Could not save sign-out. Please try again.", "error"),
      );
  };

  return (
    <div className="gt-app" data-nav="sidebar">
      {/* SIDEBAR (desktop) */}
      <aside className="gt-sidebar">
        <div className="gt-sidebar-brand">
          <Brand />
        </div>
        <nav className="gt-nav">
          {visibleNav.map((n) => (
            <NavLink
              key={n.id}
              to={n.path}
              className="gt-navitem"
              data-active={isActive(n.path)}
              data-locked={n.locked}
            >
              <Icon name={n.icon} size="1.15rem" />
              <span>{n.label}</span>
              {n.locked && (
                <Icon
                  name="lock"
                  size="0.85rem"
                  style={{ marginLeft: "auto", color: "var(--c-text-4)" }}
                />
              )}
            </NavLink>
          ))}
        </nav>
        <div className="gt-sidebar-foot">
          {isAdmin && (
            <NavLink
              to="/admin"
              className="gt-navitem gt-navitem--admin"
              data-active={isActive("/admin")}
            >
              <Icon name="shield" size="1.15rem" />
              <span>Admin Console</span>
            </NavLink>
          )}
          <NavLink
            to="/settings"
            className="gt-navitem"
            data-active={isActive("/settings")}
          >
            <Icon name="settings" size="1.15rem" />
            <span>Settings</span>
          </NavLink>
          <button className="gt-navitem" onClick={handleSignOut}>
            <Icon name="signout" size="1.15rem" />
            <span>Sign out</span>
          </button>
        </div>
      </aside>

      <div className="gt-shell-main">
        {/* TOP BAR */}
        <header className="gt-topbar">
          <button
            className="gt-burger gt-iconbtn"
            onClick={() => setMobileNav(true)}
            aria-label="Menu"
          >
            <Icon name="menu" size="1.2rem" />
          </button>
          {accessible("global-search") && (
            <div className="gt-topbar-search">
              <SearchBar />
            </div>
          )}
          <div className="gt-topbar-right">
            <CharacterSwitcher displayName={user?.displayName} />
          </div>
        </header>

        {/* CONTENT */}
        <main className="gt-content">
          <div className="gt-content-inner">{children}</div>
        </main>
      </div>

      {/* MOBILE BOTTOM TABS */}
      <nav className="gt-bottomnav">
        {visibleMobileNav.map((n) => (
          <NavLink
            key={n.id}
            to={n.path}
            className="gt-bottomtab"
            data-active={isActive(n.path)}
          >
            <Icon name={n.icon} size="1.3rem" />
            <span>{n.label}</span>
          </NavLink>
        ))}
        <button
          className="gt-bottomtab"
          data-active={[
            "/guardian",
            "/catalysts",
            "/triumphs",
            "/settings",
            "/admin",
          ].some((p) => isActive(p))}
          onClick={() => setMobileNav(true)}
        >
          <Icon name="menu" size="1.3rem" />
          <span>More</span>
        </button>
      </nav>

      {/* MOBILE NAV DRAWER */}
      {mobileNav && (
        <div className="gt-mobnav-scrim" onClick={() => setMobileNav(false)}>
          <div className="gt-mobnav" onClick={(e) => e.stopPropagation()}>
            <div className="gt-mobnav-head">
              <Brand />
              <button
                className="gt-iconbtn"
                onClick={() => setMobileNav(false)}
                aria-label="Close menu"
              >
                <Icon name="close" size="1.2rem" />
              </button>
            </div>
            {visibleNav.map((n) => (
              <NavLink
                key={n.id}
                to={n.path}
                className="gt-navitem"
                data-active={isActive(n.path)}
                data-locked={n.locked}
                onClick={closeMobileNav}
              >
                <Icon name={n.icon} size="1.15rem" />
                <span>{n.label}</span>
                {n.locked && (
                  <Icon
                    name="lock"
                    size="0.85rem"
                    style={{ marginLeft: "auto", color: "var(--c-text-4)" }}
                  />
                )}
              </NavLink>
            ))}
            {isAdmin && (
              <NavLink
                to="/admin"
                className="gt-navitem gt-navitem--admin"
                data-active={isActive("/admin")}
                onClick={closeMobileNav}
              >
                <Icon name="shield" size="1.15rem" />
                <span>Admin Console</span>
              </NavLink>
            )}
            <NavLink
              to="/settings"
              className="gt-navitem"
              data-active={isActive("/settings")}
              onClick={closeMobileNav}
            >
              <Icon name="settings" size="1.15rem" />
              <span>Settings</span>
            </NavLink>
            <button className="gt-navitem" onClick={handleSignOut}>
              <Icon name="signout" size="1.15rem" />
              <span>Sign out</span>
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
