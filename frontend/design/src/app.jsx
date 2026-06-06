/* ============================================================
   GUARDIAN TRACKER — APP SHELL + ROUTING + TWEAKS
   ============================================================ */
const { useState, useEffect, useRef } = React;
const NAV = [
  { id: "dashboard", label: "Dashboard", icon: "dashboard" },
  { id: "week", label: "This Week", icon: "week" },
  { id: "collections", label: "Collections", icon: "collections" },
  { id: "catalysts", label: "Catalysts & Crafting", icon: "catalyst" },
  { id: "triumphs", label: "Triumphs & Seals", icon: "triumph" },
  { id: "wishlist", label: "Wishlist", icon: "wishlist" },
];
const MOBILE_NAV = [
  { id: "dashboard", label: "Home", icon: "dashboard" },
  { id: "week", label: "Week", icon: "week" },
  { id: "collections", label: "Collect", icon: "collections" },
  { id: "wishlist", label: "Wishlist", icon: "wishlist" },
];

const ACCENTS = {
  aqua: "oklch(0.80 0.13 200)", gold: "oklch(0.83 0.155 88)",
  violet: "oklch(0.71 0.15 300)", green: "oklch(0.76 0.14 158)",
};

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "navLayout": "sidebar",
  "density": "comfortable",
  "cardStyle": "framed",
  "personalize": "normal",
  "accent": "oklch(0.80 0.13 200)"
}/*EDITMODE-END*/;

/* ---------------- BRAND MARK ---------------- */
function Brand({ compact }) {
  return (
    <div className="gt-brand" data-compact={compact}>
      <svg viewBox="0 0 32 32" width="1.7rem" height="1.7rem" aria-hidden="true" className="gt-brand-mark">
        <polygon points="16,2 28,9 28,23 16,30 4,23 4,9" fill="none" stroke="var(--c-signal)" strokeWidth="1.6" />
        <polygon points="16,9 22,12.5 22,19.5 16,23 10,19.5 10,12.5" fill="var(--c-signal-dim)" stroke="var(--c-signal)" strokeWidth="1" />
      </svg>
      {!compact && <div className="gt-brand-text"><span className="gt-brand-1">GUARDIAN</span><span className="gt-brand-2">TRACKER</span></div>}
    </div>
  );
}

/* ---------------- CHARACTER SWITCHER ---------------- */
function CharacterSwitcher() {
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState("maya");
  const ref = useRef();
  useEffect(() => {
    const h = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    document.addEventListener("click", h); return () => document.removeEventListener("click", h);
  }, []);
  const cur = window.GT.characters.find((c) => c.id === active);
  return (
    <div className="gt-charsw" ref={ref}>
      <button className="gt-charsw-btn" onClick={() => setOpen((v) => !v)}>
        <span className="gt-avatar" data-cls={cur.cls}>{cur.name[0]}</span>
        <span className="gt-charsw-name">{cur.name}</span>
        <Icon name="chevronDown" size="0.8rem" style={{ color: "var(--c-text-3)" }} />
      </button>
      {open && (
        <div className="gt-charsw-menu">
          <div className="gt-charsw-head mono">Switch Guardian</div>
          {window.GT.characters.map((c) => (
            <button key={c.id} className="gt-charsw-opt" data-on={c.id === active} onClick={() => { setActive(c.id); setOpen(false); }}>
              <span className="gt-avatar" data-cls={c.cls}>{c.name[0]}</span>
              <div className="gt-charsw-opt-main">
                <span className="gt-charsw-opt-name">{c.name} — {c.cls}</span>
                <span className="gt-charsw-opt-sub mono">{c.race} · {c.power}</span>
              </div>
              {c.id === active && <Icon name="check" size="0.9rem" style={{ color: "var(--c-signal)" }} />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

/* ---------------- GLOBAL SEARCH ---------------- */
function SearchBar() {
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  const ref = useRef();
  useEffect(() => {
    const h = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    document.addEventListener("click", h); return () => document.removeEventListener("click", h);
  }, []);
  const results = q.length > 1 ? window.GT.items.filter((i) => i.name.toLowerCase().includes(q.toLowerCase())).slice(0, 6) : [];
  return (
    <div className="gt-search" ref={ref}>
      <Icon name="search" size="1rem" style={{ color: "var(--c-text-3)" }} />
      <input className="gt-search-input" placeholder="Search items…" value={q}
        onChange={(e) => { setQ(e.target.value); setOpen(true); }} onFocus={() => setOpen(true)} />
      {open && q.length > 1 && (
        <div className="gt-search-menu">
          {results.length ? results.map((i) => (
            <button key={i.id} className="gt-search-opt" data-rarity={i.rarity}>
              <ItemTile rarity={i.rarity} type={i.type} style={{ width: "1.8rem" }} />
              <span className="gt-search-opt-name">{i.name}</span>
              <span className="gt-item-type">{i.type}</span>
            </button>
          )) : <div className="gt-search-empty mono">No items match “{q}”</div>}
        </div>
      )}
    </div>
  );
}

/* ---------------- LOCKED SEARCH (beta gate) ---------------- */
function LockedSearch() {
  return (
    <div className="gt-search gt-search--locked" title="Global search is a Beta feature">
      <Icon name="search" size="1rem" style={{ color: "var(--c-text-4)" }} />
      <span className="gt-search-lockedtxt">Search items…</span>
      <span className="gt-badge" style={{ "--bdg": "var(--c-rare)", marginLeft: "auto" }}><Icon name="lock" size="0.8em" />Beta</span>
    </div>
  );
}

/* ---------------- LOGIN ---------------- */
function Login({ onEnter }) {
  const [state, setState] = useState("idle");
  const signin = () => { setState("signing"); setTimeout(() => { setState("callback"); setTimeout(onEnter, 900); }, 900); };
  return (
    <div className="gt-login">
      <div className="gt-login-bg" />
      <div className="gt-login-card">
        <Brand />
        <h1 className="gt-login-title">See what you're missing.<br />Chase what matters this week.</h1>
        <p className="gt-login-sub">A Destiny 2 companion that turns your sprawling collection into a focused plan of action.</p>
        {state === "idle" && <Button variant="primary" icon="bungie" onClick={signin} style={{ width: "100%" }}>Sign in with Bungie</Button>}
        {state === "signing" && <Button variant="primary" disabled style={{ width: "100%" }}>Redirecting to Bungie.net…</Button>}
        {state === "callback" && <Button variant="primary" disabled style={{ width: "100%" }}><Icon name="refresh" size="1rem" className="gt-fresh-icon" style={{ animation: "gt-spin 0.9s linear infinite" }} /> Completing sign-in…</Button>}
        <div className="gt-login-note mono">Read-only · we never modify your account</div>
        <ul className="gt-login-feats">
          <li><Icon name="collections" size="1rem" style={{ color: "var(--c-signal)" }} /> What's missing across every category</li>
          <li><Icon name="week" size="1rem" style={{ color: "var(--c-exotic)" }} /> The best things to do each week</li>
          <li><Icon name="catalyst" size="1rem" style={{ color: "var(--c-legendary)" }} /> Track catalysts, patterns & seals</li>
        </ul>
      </div>
    </div>
  );
}

/* nav items that are gated behind a feature flag */
const NAV_FLAG = { catalysts: "catalysts-crafting", triumphs: "triumphs-seals" };

/* ---------------- APP ---------------- */
function App() {
  const gt = useGT();
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);
  const [authed, setAuthed] = useState(false);
  const [page, setPage] = useState("dashboard");
  const [wished, setWished] = useState(() => new Set(window.GT.wishlist.map((w) => {
    const m = window.GT.items.find((i) => i.name === w.name); return m ? m.id : w.id;
  })));
  const [detail, setDetail] = useState(null);
  const [mobileNav, setMobileNav] = useState(false);
  const [toast, setToast] = useState(null);

  // apply tweaks to :root
  useEffect(() => {
    const r = document.documentElement;
    r.setAttribute("data-density", t.density);
    r.style.setProperty("--c-signal", t.accent);
    r.style.setProperty("--c-signal-dim", `color-mix(in oklch, ${t.accent} 16%, transparent)`);
    r.style.setProperty("--c-signal-line", `color-mix(in oklch, ${t.accent} 40%, transparent)`);
  }, [t.density, t.accent]);

  const go = (p) => { setPage(p); setMobileNav(false); window.scrollTo({ top: 0 }); };
  const onWish = (item) => {
    setWished((s) => { const n = new Set(s); n.has(item.id) ? n.delete(item.id) : n.add(item.id); return n; });
    setToast({ msg: wished.has(item.id) ? `Removed ${item.name}` : `${item.name} added to wishlist`, kind: wished.has(item.id) ? "info" : "success" });
    setTimeout(() => setToast(null), 2200);
  };

  if (!authed) return (<React.Fragment><Login onEnter={() => setAuthed(true)} /><TweaksUI t={t} setTweak={setTweak} /></React.Fragment>);

  // visible nav: drop items whose flag is globally disabled; mark tier-locked ones
  const visibleNav = NAV.filter((n) => !NAV_FLAG[n.id] || gt.flagState(NAV_FLAG[n.id]).enabled)
    .map((n) => NAV_FLAG[n.id] ? { ...n, locked: gt.flagState(NAV_FLAG[n.id]).locked } : n);

  // resolve current page, honoring flag gates + admin guard
  const resolvePage = () => {
    if (page === "admin") return gt.isAdmin ? <AdminPanel go={go} /> : <Dashboard go={go} tweaks={t} />;
    const fid = NAV_FLAG[page];
    if (fid) {
      const st = gt.flagState(fid);
      if (!st.enabled) return <Dashboard go={go} tweaks={t} />;
      if (st.locked) return <LockedFeature flag={st.flag} go={go} />;
    }
    return {
      dashboard: <Dashboard go={go} tweaks={t} />,
      week: <ThisWeek tweaks={t} />,
      collections: <Collections wished={wished} onWish={onWish} onOpen={setDetail} tweaks={t} />,
      catalysts: <Catalysts tweaks={t} />,
      triumphs: <Triumphs tweaks={t} />,
      wishlist: <Wishlist go={go} />,
      settings: <Settings tweaks={t} go={go} />,
    }[page] || <Dashboard go={go} tweaks={t} />;
  };
  const PageEl = resolvePage();

  return (
    <div className="gt-app" data-nav={t.navLayout}>
      {/* SIDEBAR (desktop) */}
      {t.navLayout === "sidebar" && (
        <aside className="gt-sidebar">
          <div className="gt-sidebar-brand"><Brand /></div>
          <nav className="gt-nav">
            {visibleNav.map((n) => (
              <button key={n.id} className="gt-navitem" data-active={page === n.id} data-locked={n.locked} onClick={() => go(n.id)}>
                <Icon name={n.icon} size="1.15rem" /><span>{n.label}</span>
                {n.locked && <Icon name="lock" size="0.85rem" style={{ marginLeft: "auto", color: "var(--c-text-4)" }} />}
              </button>
            ))}
          </nav>
          <div className="gt-sidebar-foot">
            {gt.isAdmin && <button className="gt-navitem gt-navitem--admin" data-active={page === "admin"} onClick={() => go("admin")}><Icon name="shield" size="1.15rem" /><span>Admin Console</span></button>}
            <button className="gt-navitem" data-active={page === "settings"} onClick={() => go("settings")}><Icon name="settings" size="1.15rem" /><span>Settings</span></button>
            <button className="gt-navitem" onClick={() => setAuthed(false)}><Icon name="signout" size="1.15rem" /><span>Sign out</span></button>
          </div>
        </aside>
      )}

      <div className="gt-shell-main">
        {/* TOP BAR */}
        <header className="gt-topbar">
          {t.navLayout === "topnav" && <div className="gt-topbar-brand"><Brand /></div>}
          <button className="gt-burger gt-iconbtn" onClick={() => setMobileNav(true)} aria-label="Menu"><Icon name="menu" size="1.2rem" /></button>
          {t.navLayout === "topnav" && (
            <nav className="gt-topnav">
              {visibleNav.map((n) => (
                <button key={n.id} className="gt-topnav-item" data-active={page === n.id} onClick={() => go(n.id)}>
                  <Icon name={n.icon} size="1rem" /><span>{n.label}</span>
                  {n.locked && <Icon name="lock" size="0.8rem" style={{ color: "var(--c-text-4)" }} />}
                </button>
              ))}
              {gt.isAdmin && <button className="gt-topnav-item gt-topnav-item--admin" data-active={page === "admin"} onClick={() => go("admin")}><Icon name="shield" size="1rem" /><span>Admin</span></button>}
            </nav>
          )}
          <div className="gt-topbar-search">{gt.accessible("global-search") ? <SearchBar /> : <LockedSearch />}</div>
          <div className="gt-topbar-right">
            <CharacterSwitcher />
          </div>
        </header>

        {/* CONTENT */}
        <main className="gt-content">
          <div className="gt-content-inner">{PageEl}</div>
        </main>
      </div>

      {/* MOBILE BOTTOM TABS */}
      <nav className="gt-bottomnav">
        {MOBILE_NAV.map((n) => (
          <button key={n.id} className="gt-bottomtab" data-active={page === n.id} onClick={() => go(n.id)}>
            <Icon name={n.icon} size="1.3rem" /><span>{n.label}</span>
          </button>
        ))}
        <button className="gt-bottomtab" data-active={["catalysts", "triumphs", "settings"].includes(page)} onClick={() => setMobileNav(true)}>
          <Icon name="menu" size="1.3rem" /><span>More</span>
        </button>
      </nav>

      {/* MOBILE NAV DRAWER */}
      {mobileNav && (
        <div className="gt-mobnav-scrim" onClick={() => setMobileNav(false)}>
          <div className="gt-mobnav" onClick={(e) => e.stopPropagation()}>
            <div className="gt-mobnav-head"><Brand /><button className="gt-iconbtn" onClick={() => setMobileNav(false)}><Icon name="close" size="1.2rem" /></button></div>
            {[...visibleNav, ...(gt.isAdmin ? [{ id: "admin", label: "Admin Console", icon: "shield" }] : []), { id: "settings", label: "Settings", icon: "settings" }].map((n) => (
              <button key={n.id} className="gt-navitem" data-active={page === n.id} onClick={() => go(n.id)}><Icon name={n.icon} size="1.15rem" /><span>{n.label}</span>{n.locked && <Icon name="lock" size="0.85rem" style={{ marginLeft: "auto", color: "var(--c-text-4)" }} />}</button>
            ))}
            <button className="gt-navitem" onClick={() => { setAuthed(false); setMobileNav(false); }}><Icon name="signout" size="1.15rem" /><span>Sign out</span></button>
          </div>
        </div>
      )}

      {/* DETAIL DRAWER */}
      {detail && <ItemDetailDrawer item={detail} onClose={() => setDetail(null)} onWish={onWish} wished={wished.has(detail.id)} />}

      {/* TOAST */}
      {toast && <div className="gt-toast" data-kind={toast.kind}><Icon name={toast.kind === "success" ? "check" : "info"} size="1rem" />{toast.msg}</div>}

      <TweaksUI t={t} setTweak={setTweak} />
    </div>
  );
}

/* ---------------- TWEAKS UI ---------------- */
function TweaksUI({ t, setTweak }) {
  return (
    <TweaksPanel title="Tweaks">
      <TweakSection label="Layout & IA" />
      <TweakRadio label="Navigation" value={t.navLayout} options={["sidebar", "topnav"]} onChange={(v) => setTweak("navLayout", v)} />
      <TweakRadio label="Density" value={t.density} options={["comfortable", "compact"]} onChange={(v) => setTweak("density", v)} />
      <TweakSection label="Item cards" />
      <TweakRadio label="Card style" value={t.cardStyle} options={["framed", "compact"]} onChange={(v) => setTweak("cardStyle", v)} />
      <TweakSection label="Personalization" />
      <TweakRadio label="“For you” badges" value={t.personalize} options={["off", "normal", "aggressive"]} onChange={(v) => setTweak("personalize", v)} />
      <TweakSection label="Accent" />
      <TweakColor label="Signal color" value={t.accent} options={Object.values(ACCENTS)} onChange={(v) => setTweak("accent", v)} />
    </TweaksPanel>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<GTProvider><App /></GTProvider>);
