/* ============================================================
   GUARDIAN TRACKER — ADMIN PANEL + GATING UI
   AdminPanel (Users + Feature Flags), Switch, RoleBadge,
   RoleSelect, TierSegment, LockedFeature.
   Admin-only; gated in app.jsx by role === "admin".
   ============================================================ */
const { useState, useMemo } = React;

/* ---------------- ROLE BADGE ---------------- */
function RoleBadge({ role, lg }) {
  return (
    <span className={`gt-badge${lg ? " gt-badge--lg" : ""}`} style={{ "--bdg": window.GT.roleColor[role] }}>
      {role === "admin" ? <Icon name="shield" size="0.85em" /> : <span className="gt-dot" />}
      {window.GT.label.role[role]}
    </span>
  );
}

/* ---------------- SWITCH ---------------- */
function Switch({ on, onChange, label }) {
  return (
    <button className="gt-switch" data-on={on} role="switch" aria-checked={on} aria-label={label} onClick={() => onChange(!on)}>
      <span className="gt-switch-knob" />
    </button>
  );
}

/* ---------------- ROLE SELECT (dropdown) ---------------- */
function RoleSelect({ value, onPick, includeAdmin = true, compact }) {
  const [open, setOpen] = useState(false);
  const ref = React.useRef();
  React.useEffect(() => {
    const h = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    document.addEventListener("click", h); return () => document.removeEventListener("click", h);
  }, []);
  const roles = window.GT.ROLES.filter((r) => includeAdmin || r !== "admin");
  return (
    <div className="gt-dd" ref={ref}>
      <button className={`gt-roleselect${compact ? " gt-roleselect--sm" : ""}`} onClick={() => setOpen((v) => !v)} style={{ "--bdg": window.GT.roleColor[value] }}>
        {value === "admin" ? <Icon name="shield" size="0.85em" /> : <span className="gt-dot" />}
        {window.GT.label.role[value]}
        <Icon name="chevronDown" size="0.75rem" />
      </button>
      {open && (
        <div className="gt-dd-menu" style={{ right: 0, left: "auto", minWidth: "9rem" }}>
          {roles.map((r) => (
            <button key={r} className="gt-dd-opt" data-on={value === r} onClick={() => { onPick(r); setOpen(false); }}>
              <span className="gt-dot" style={{ background: window.GT.roleColor[r] }} /> {window.GT.label.role[r]}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

/* ---------------- TIER SEGMENT (flag gate) ---------------- */
function TierSegment({ value, onChange }) {
  return (
    <div className="gt-tierseg" role="radiogroup" aria-label="Minimum access tier">
      {window.GT.MIN_TIERS.map((tier) => (
        <button key={tier} className="gt-tierseg-opt" role="radio" aria-checked={value === tier}
          data-on={value === tier} style={{ "--bdg": window.GT.roleColor[tier] }} onClick={() => onChange(tier)}>
          {window.GT.label.tier[tier]}
        </button>
      ))}
    </div>
  );
}

/* ---------------- LOCKED FEATURE (upsell screen) ---------------- */
function LockedFeature({ flag, go }) {
  const need = flag.minTier;
  return (
    <div className="gt-page">
      <div className="gt-card gt-locked">
        <div className="gt-locked-mark" style={{ "--bdg": window.GT.roleColor[need] }}>
          <Icon name="lock" size="2.4rem" stroke={1.5} />
        </div>
        <div className="gt-locked-tier"><RoleBadge role={need} lg /> early access</div>
        <h2 className="gt-locked-title">{flag.name}</h2>
        <p className="gt-locked-body">{flag.desc}</p>
        <p className="gt-locked-note">
          This feature is rolling out to <strong>{window.GT.label.tier[need]}</strong> members. Switch your access tier
          to unlock it — or ask an admin to widen the flag.
        </p>
        <div className="gt-locked-actions">
          <Button variant="primary" icon="bolt" onClick={() => go("settings")}>Change access tier</Button>
          <Button variant="ghost" onClick={() => go("dashboard")}>Back to Dashboard</Button>
        </div>
      </div>
    </div>
  );
}

/* ---------------- FEATURE FLAG CARD ---------------- */
function FlagCard({ flag }) {
  const { setFlag, reach, totalUsers } = useGT();
  const reached = reach(flag);
  const pct = Math.round((reached / totalUsers) * 100);
  return (
    <div className="gt-flagcard" data-off={!flag.enabled}>
      <div className="gt-flagcard-head">
        <div className="gt-flagcard-title">
          <span className="gt-flag-name">{flag.name}</span>
          <span className="gt-badge" style={{ "--bdg": "var(--c-text-3)" }}>{flag.category}</span>
        </div>
        <Switch on={flag.enabled} onChange={(v) => setFlag(flag.id, { enabled: v })} label={`Enable ${flag.name}`} />
      </div>
      <p className="gt-flag-desc">{flag.desc}</p>
      <div className="gt-flagcard-foot">
        <div className="gt-flag-gate">
          <span className="gt-flag-gate-label">Available to</span>
          <TierSegment value={flag.minTier} onChange={(v) => setFlag(flag.id, { minTier: v })} />
        </div>
        <div className="gt-flag-reach">
          {flag.enabled
            ? <React.Fragment><span className="mono gt-flag-reach-num">{reached}/{totalUsers}</span><span className="gt-flag-reach-lbl">users · {pct}%</span></React.Fragment>
            : <span className="gt-flag-reach-lbl" style={{ color: "var(--c-text-4)" }}>Disabled — hidden for everyone</span>}
        </div>
      </div>
      {flag.enabled && <div className="gt-flag-reachbar"><div className="gt-flag-reachbar-fill" style={{ "--val": pct + "%", "--pc": window.GT.roleColor[flag.minTier] }} /></div>}
    </div>
  );
}

/* ---------------- USERS TABLE ---------------- */
function UserRow({ id, name, handle, role, platform, active, me }) {
  const { setUserRole } = useGT();
  return (
    <div className="gt-userrow" data-me={me}>
      <span className="gt-avatar" data-cls={me ? "Warlock" : "Hunter"}>{name[0]}</span>
      <div className="gt-userrow-id">
        <div className="gt-userrow-name">{name}{me && <span className="gt-you">You</span>}</div>
        <div className="gt-userrow-handle mono">{handle}</div>
      </div>
      <div className="gt-userrow-meta">
        <span className="gt-userrow-platform">{platform}</span>
        <span className="gt-userrow-active mono">{active}</span>
      </div>
      <RoleSelect value={role} onPick={(r) => setUserRole(id, r)} />
    </div>
  );
}

function AdminPanel({ go }) {
  const { users, flags, role, roster, totalUsers } = useGT();
  const [tab, setTab] = useState("users");
  const [q, setQ] = useState("");
  const [roleFilter, setRoleFilter] = useState("all");

  const me = { id: "me", ...window.GT.currentUser, role, active: "this session", me: true };
  const everyone = [me, ...users];
  const filtered = everyone.filter((u) =>
    (roleFilter === "all" || u.role === roleFilter) &&
    (q.length < 1 || (u.name + u.handle).toLowerCase().includes(q.toLowerCase())));

  const roleCounts = useMemo(() => {
    const c = { all: everyone.length };
    window.GT.ROLES.forEach((r) => c[r] = everyone.filter((u) => u.role === r).length);
    return c;
  }, [users, role]);

  const flagOn = flags.filter((f) => f.enabled).length;

  return (
    <div className="gt-page">
      <PageHead title={<span className="gt-admin-title"><Icon name="shield" size="1.4rem" style={{ color: "var(--c-admin)" }} /> Admin Console</span>}
        sub="Manage member roles and feature-flag rollout"
        right={<div className="gt-subtabs">
          <button className="gt-subtab" data-on={tab === "users"} onClick={() => setTab("users")}><Icon name="users" size="0.95rem" /> Users</button>
          <button className="gt-subtab" data-on={tab === "flags"} onClick={() => setTab("flags")}><Icon name="flag" size="0.95rem" /> Feature Flags</button>
        </div>} />

      {tab === "users" ? (
        <React.Fragment>
          <div className="gt-coll-stats">
            <StatTile num={roleCounts.all} label="Total members" mono />
            <StatTile num={roleCounts.beta} label="Beta" mono color="var(--c-rare)" />
            <StatTile num={roleCounts.alpha} label="Alpha" mono color="var(--c-exotic)" />
            <StatTile num={roleCounts.admin} label="Admins" mono color="var(--c-admin)" />
          </div>

          <div className="gt-coll-toolbar">
            <div className="gt-search gt-admin-search">
              <Icon name="search" size="1rem" style={{ color: "var(--c-text-3)" }} />
              <input className="gt-search-input" placeholder="Search members…" value={q} onChange={(e) => setQ(e.target.value)} />
            </div>
            <div className="gt-filterbar">
              {["all", ...window.GT.ROLES].map((r) => (
                <FilterChip key={r} on={roleFilter === r} onClick={() => setRoleFilter(r)}>
                  {r === "all" ? "All" : window.GT.label.role[r]} <span className="mono" style={{ opacity: 0.6 }}>{roleCounts[r]}</span>
                </FilterChip>
              ))}
            </div>
          </div>

          <div className="gt-card gt-pad0">
            <div className="gt-userrow gt-userrow--head">
              <span></span><span className="gt-userrow-h">Member</span>
              <span className="gt-userrow-h gt-userrow-meta">Platform · last active</span>
              <span className="gt-userrow-h" style={{ textAlign: "right" }}>Role</span>
            </div>
            {filtered.map((u) => <UserRow key={u.id} {...u} />)}
            {filtered.length === 0 && <div style={{ padding: "var(--s-6)" }}><EmptyState icon="users" title="No members match" body="Try a different search or role filter." /></div>}
          </div>
        </React.Fragment>
      ) : (
        <React.Fragment>
          <div className="gt-coll-stats">
            <StatTile num={`${flagOn}/${flags.length}`} label="Flags enabled" mono />
            <StatTile num={flags.filter((f) => f.minTier === "beta" && f.enabled).length} label="Beta-gated" mono color="var(--c-rare)" />
            <StatTile num={flags.filter((f) => f.minTier === "alpha" && f.enabled).length} label="Alpha-gated" mono color="var(--c-exotic)" />
            <div className="gt-coll-resultcount mono">Changes apply live across the app</div>
          </div>
          <div className="gt-flag-grid">
            {flags.map((f) => <FlagCard key={f.id} flag={f} />)}
          </div>
        </React.Fragment>
      )}
    </div>
  );
}

Object.assign(window, { RoleBadge, Switch, RoleSelect, TierSegment, LockedFeature, FlagCard, UserRow, AdminPanel });
