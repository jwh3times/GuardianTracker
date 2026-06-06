/* ============================================================
   GUARDIAN TRACKER — PAGES (part 2)
   Catalysts & Crafting, Triumphs & Seals
   ============================================================ */
const { useState, useMemo } = React;

/* =================================================================
   CATALYSTS & CRAFTING
   ================================================================= */
function CatalystCard({ c }) {
  const color = c.status === "complete" ? "var(--c-complete)" : c.status === "in-progress" ? "var(--c-progress)" : "var(--c-text-3)";
  return (
    <div className="gt-item gt-item--prog" data-rarity="exotic">
      <div className="gt-item-top">
        <ItemTile rarity="exotic" type="Auto Rifle" />
        <div className="gt-item-head">
          <div className="gt-item-name">{c.name}</div>
          <div className="gt-item-type">Catalyst</div>
          <div className="gt-item-badges" style={{ marginTop: "var(--s-1)" }}><Badge kind={c.status} dot /></div>
        </div>
      </div>
      {c.obj
        ? <ProgressBar value={c.obj.cur} max={c.obj.max} label={c.obj.label} showVal color={color} size="tall" />
        : <div className="gt-prog-empty mono">Not yet acquired</div>}
      <div className="gt-item-src"><Icon name="bolt" size="0.8rem" style={{ color: "var(--c-text-4)" }} />{c.source}</div>
    </div>
  );
}
function CraftCard({ c }) {
  const done = c.patterns.cur >= c.patterns.max;
  return (
    <div className="gt-item gt-item--prog" data-rarity="legendary">
      <div className="gt-item-top">
        <ItemTile rarity="legendary" type={c.type} />
        <div className="gt-item-head">
          <div className="gt-item-name">{c.name}</div>
          <div className="gt-item-type">{c.type}</div>
          <div className="gt-item-badges" style={{ marginTop: "var(--s-1)" }}>
            {done ? <Badge kind="complete" dot>Craftable</Badge> : <Badge kind="in-progress" dot>{c.note}</Badge>}
          </div>
        </div>
      </div>
      <ProgressBar value={c.patterns.cur} max={c.patterns.max} label="Patterns extracted" showVal complete={done} color={done ? "var(--c-complete)" : "var(--c-exotic)"} size="tall" />
      <div className="gt-item-src"><Icon name="bolt" size="0.8rem" style={{ color: "var(--c-text-4)" }} />{c.source}</div>
    </div>
  );
}
function Catalysts({ tweaks }) {
  const [tab, setTab] = useState("catalysts");
  const [filter, setFilter] = useState("all");
  const cats = window.GT.catalysts;
  const craft = window.GT.crafting;

  const catFiltered = filter === "all" ? cats : cats.filter((c) => c.status === filter);
  const craftFiltered = filter === "all" ? craft : craft.filter((c) => {
    const done = c.patterns.cur >= c.patterns.max;
    return filter === "complete" ? done : filter === "missing" ? c.patterns.cur === 0 : (!done && c.patterns.cur > 0);
  });
  const catDone = cats.filter((c) => c.status === "complete").length;

  return (
    <div className="gt-page">
      <PageHead title="Catalysts & Crafting" sub="Long-grind progress, all in one place"
        right={<div className="gt-subtabs">
          <button className="gt-subtab" data-on={tab === "catalysts"} onClick={() => setTab("catalysts")}>Catalysts</button>
          <button className="gt-subtab" data-on={tab === "crafting"} onClick={() => setTab("crafting")}>Crafting Patterns</button>
        </div>} />

      <div className="gt-coll-toolbar">
        <div className="gt-filterbar">
          {["all", "missing", "in-progress", "complete"].map((f) => (
            <FilterChip key={f} on={filter === f} onClick={() => setFilter(f)}>
              {f === "all" ? "All" : f === "in-progress" ? "In progress" : f[0].toUpperCase() + f.slice(1)}
            </FilterChip>
          ))}
        </div>
        <span className="gt-action-meta mono">{tab === "catalysts" ? `${catDone}/${cats.length} complete` : `${craft.filter(c=>c.patterns.cur>=c.patterns.max).length}/${craft.length} craftable`}</span>
      </div>

      {tab === "catalysts" ? (
        catFiltered.length ? <div className="gt-itemgrid gt-itemgrid--prog">{catFiltered.map((c) => <CatalystCard key={c.id} c={c} />)}</div>
          : <div className="gt-card"><EmptyState icon="filter" title="Nothing here" body="No catalysts match this filter." action={<Button variant="outline" sm onClick={() => setFilter("all")}>Clear</Button>} /></div>
      ) : (
        craftFiltered.length ? <div className="gt-itemgrid gt-itemgrid--prog">{craftFiltered.map((c) => <CraftCard key={c.id} c={c} />)}</div>
          : <div className="gt-card"><EmptyState icon="filter" title="Nothing here" body="No patterns match this filter." action={<Button variant="outline" sm onClick={() => setFilter("all")}>Clear</Button>} /></div>
      )}
    </div>
  );
}

/* =================================================================
   TRIUMPHS & SEALS
   ================================================================= */
function Triumphs({ tweaks }) {
  const [sort, setSort] = useState("closest");
  const [openId, setOpenId] = useState(window.GT.seals[0].id);
  const seals = useMemo(() => {
    const l = window.GT.seals.slice();
    if (sort === "closest") l.sort((a, b) => (b.pct >= 100 ? -1 : b.pct) - (a.pct >= 100 ? -1 : a.pct));
    else if (sort === "name") l.sort((a, b) => a.name.localeCompare(b.name));
    return l;
  }, [sort]);
  const gilded = window.GT.seals.filter((s) => s.gilded > 0).length;

  return (
    <div className="gt-page">
      <PageHead title="Triumphs & Seals" sub={<span className="mono">{window.GT.seals.length} seals · {gilded} gilded</span>}
        right={<Dropdown label="Sort: Closest to done" value={{ closest: "Sort: Closest to done", name: "Sort: Name" }[sort]} noClear
          options={[{ v: "closest", l: "Sort: Closest to done" }, { v: "name", l: "Sort: Name" }]} onPick={setSort} />} />

      <div className="gt-seal-grid">
        {seals.map((s) => <SealCard key={s.id} seal={s} expanded={openId === s.id} onToggle={() => setOpenId((id) => id === s.id ? null : s.id)} />)}
      </div>
    </div>
  );
}

/* =================================================================
   SETTINGS
   ================================================================= */
function Settings({ tweaks, go }) {
  const gt = useGT();
  const tierOpts = window.GT.MIN_TIERS; // standard / beta / alpha
  return (
    <div className="gt-page">
      <PageHead title="Settings" sub="Account, access & data" />

      {/* MEMBERSHIP & ACCESS */}
      <Panel title="Membership & access" icon="shield" accent={window.GT.roleColor[gt.role]}
        right={<RoleBadge role={gt.role} lg />}>
        <p className="gt-set-note">
          Guardian Tracker rolls features out in waves. Opt into an early-access tier to try features before they
          reach everyone — <strong>Beta</strong> for upcoming features, <strong>Alpha</strong> for the bleeding edge.
        </p>
        <div className="gt-access-switch">
          <span className="gt-set-k">Access tier</span>
          <div className="gt-tierpick" role="radiogroup" aria-label="Access tier" data-disabled={gt.isAdmin}>
            {tierOpts.map((r) => (
              <button key={r} className="gt-tierpick-opt" role="radio" aria-checked={gt.role === r}
                data-on={gt.role === r} disabled={gt.isAdmin}
                style={{ "--bdg": window.GT.roleColor[r] }}
                onClick={() => { gt.setRole(r); gt.setPrevTier(r); }}>
                {r === "alpha" ? <Icon name="bolt" size="0.9em" /> : <span className="gt-dot" />}
                {window.GT.label.role[r]}
              </button>
            ))}
          </div>
        </div>
        {gt.isAdmin && <p className="gt-set-note" style={{ color: "var(--c-admin)" }}>You currently have <strong>Admin</strong> access — full visibility of every feature. Turn off the simulation below to browse as a member.</p>}

        <div className="gt-hr"></div>
        <div className="gt-set-row" style={{ borderBottom: "none" }}>
          <div>
            <span className="gt-set-v">Simulate admin access</span>
            <div className="gt-set-note" style={{ marginTop: "0.15rem" }}>Prototype control — grants the Admin Console to preview gating.</div>
          </div>
          <Switch on={gt.isAdmin} onChange={(v) => gt.simulateAdmin(v)} label="Simulate admin access" />
        </div>
        {gt.isAdmin && <Button variant="primary" icon="shield" onClick={() => go("admin")}>Open Admin Console</Button>}
      </Panel>

      <div className="gt-set-grid">
        <Panel title="Account" icon="bungie">
          <div className="gt-set-row"><span className="gt-set-k">Guardian</span><span className="gt-set-v">Maya#4271</span></div>
          <div className="gt-set-row"><span className="gt-set-k">Platform</span><span className="gt-set-v">Steam · cross-save active</span></div>
          <div className="gt-set-row"><span className="gt-set-k">Access</span><span className="gt-set-v mono" style={{ color: "var(--c-avail)" }}>Read-only collection data</span></div>
        </Panel>
        <Panel title="Characters" icon="dashboard">
          {window.GT.characters.map((c) => (
            <div key={c.id} className="gt-set-row">
              <span className="gt-set-k">{c.name} · {c.cls}</span>
              <span className="gt-set-v mono">{c.power}</span>
            </div>
          ))}
        </Panel>
        <Panel title="Data freshness" icon="refresh">
          <div className="gt-set-row"><span className="gt-set-k">Last updated</span><span className="gt-set-v mono">4m ago</span></div>
          <p className="gt-set-note">Guardian Tracker caches your collection and refreshes on demand — it never polls live, to respect Bungie's rate limits.</p>
          <DataFreshnessChip ago="4m" />
        </Panel>
        <Panel title="Privacy" icon="settings">
          <p className="gt-set-note">Your collection must be set to public on Bungie.net for Guardian Tracker to read it. We never modify your account.</p>
          <Button variant="outline" sm icon="external">Bungie privacy settings</Button>
        </Panel>
      </div>
      <div className="gt-dash-actions">
        <Button variant="outline" icon="signout">Sign out</Button>
        <button className="gt-link gt-link--danger" onClick={() => gt.resetDemo()}><Icon name="refresh" size="0.85rem" /> Reset demo data</button>
      </div>
    </div>
  );
}

Object.assign(window, { Catalysts, Triumphs, Settings });
