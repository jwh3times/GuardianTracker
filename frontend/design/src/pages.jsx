/* ============================================================
   GUARDIAN TRACKER — PAGES
   Dashboard, ThisWeek, Collections, Wishlist, Catalysts, Triumphs
   Depend on kit.jsx + detail.jsx globals.
   ============================================================ */
const { useState, useEffect, useRef, useMemo } = React;

/* ---------------- PAGE HEADER ---------------- */
function PageHead({ title, sub, right }) {
  return (
    <header className="gt-pagehead">
      <div>
        <h1 className="gt-page-title">{title}</h1>
        {sub && <div className="gt-page-sub">{sub}</div>}
      </div>
      {right && <div className="gt-pagehead-right">{right}</div>}
    </header>
  );
}

/* =================================================================
   DASHBOARD
   ================================================================= */
function Dashboard({ go, tweaks }) {
  const s = window.GT.summary;
  const w = window.GT.weekly;
  const wl = window.GT.wishlist.filter((i) => i.avail.now);
  return (
    <div className="gt-page gt-dash">
      <PageHead title="Welcome back, Maya" sub="Warlock · 1810"
        right={<CountdownChip prefix="Weekly reset" time={w.resetIn} icon="clock" />} />

      {/* HERO COMPLETION */}
      <Panel pad={false} style={{ padding: "var(--s-5)" }}>
        <div className="gt-hero">
          <div className="gt-hero-radial">
            <RadialProgress value={s.overall} size="clamp(8rem,9vw,10rem)" color="var(--c-exotic)" sub="overall" pctSize="clamp(2rem,3vw,2.6rem)" />
          </div>
          <div className="gt-hero-bars">
            {s.categories.map((c) => (
              <button key={c.id} className="gt-hero-bar" onClick={() => go("collections")}>
                <div className="gt-hero-bar-top">
                  <span className="gt-hero-bar-label">{c.label}</span>
                  <span className="mono gt-hero-bar-count">{c.count[0].toLocaleString()}/{c.count[1].toLocaleString()}</span>
                </div>
                <ProgressBar value={c.pct} showVal valText={`${c.pct}%`} color={c.id === "exotics" ? "var(--c-exotic)" : "var(--c-signal)"} />
              </button>
            ))}
          </div>
        </div>
      </Panel>

      {/* DO THIS TODAY */}
      <Panel title="Do this today" icon="bolt" accent="var(--c-signal)"
        right={<button className="gt-link" onClick={() => go("week")}>Full week <Icon name="chevron" size="0.8rem" /></button>}>
        <div className="gt-today">
          <button className="gt-today-row" onClick={() => go("week")}>
            <Icon name="bungie" size="1.2rem" style={{ color: "var(--c-exotic)" }} />
            <div className="gt-today-main">
              <div className="gt-today-text"><strong>Xûr is selling Hawkmoon</strong> — a Hand Cannon exotic you're missing.</div>
              <div className="gt-action-meta mono">The Tower · leaves in 1d 6h</div>
            </div>
            <Badge kind="missing" dot icon="bolt" />
          </button>
          <button className="gt-today-row" onClick={() => go("collections")}>
            <Icon name="collections" size="1.2rem" style={{ color: "var(--c-rare)" }} />
            <div className="gt-today-main">
              <div className="gt-today-text"><strong>Featured raid: Vault of Glass</strong> — 2 weapons you don't have yet.</div>
              <div className="gt-action-meta mono">Pinnacle gear · resets in 2d 14h</div>
            </div>
            <Badge kind="completes-set" dot />
          </button>
        </div>
      </Panel>

      <div className="gt-dash-cols">
        <Panel title="This week — preview" icon="week"
          right={<button className="gt-link" onClick={() => go("week")}>See all <Icon name="chevron" size="0.8rem" /></button>}>
          <ul className="gt-vendor-list">
            {w.milestones.slice(0, 2).map((m) => (
              <li key={m.id} className="gt-milestone">
                <div className="gt-milestone-l">
                  <div className="gt-action-meta mono">{m.label}</div>
                  <div className="gt-item-name">{m.name}</div>
                  <div className="gt-item-type">Reward: {m.reward}</div>
                </div>
                {m.missing > 0 ? <Badge kind="missing" dot>{m.missing} missing</Badge> : <Badge kind="complete" dot />}
              </li>
            ))}
          </ul>
        </Panel>

        <Panel title="Wishlist available now" icon="wishlist" accent="var(--c-avail)"
          right={<button className="gt-link" onClick={() => go("wishlist")}>Manage <Icon name="chevron" size="0.8rem" /></button>}>
          <div className="gt-avail-head">
            <span className="gt-avail-num mono">{wl.length}</span>
            <span className="gt-avail-text">of your wishlisted items are obtainable right now</span>
          </div>
          <div className="gt-avail-list">
            {wl.slice(0, 3).map((i) => (
              <button key={i.id} className="gt-item gt-item--compact" data-rarity={i.rarity} onClick={() => go("wishlist")}>
                <ItemTile rarity={i.rarity} type={i.type} style={{ width: "1.9rem" }} />
                <div className="gt-item-head" style={{ flex: 1 }}>
                  <div className="gt-item-name">{i.name}</div>
                  <div className="gt-item-type">{i.avail.where}</div>
                </div>
                <Badge kind="avail-now" dot />
              </button>
            ))}
          </div>
        </Panel>
      </div>

      <div className="gt-dash-actions">
        <Button variant="outline" icon="collections" onClick={() => go("collections")}>View Collections</Button>
        <Button variant="outline" icon="wishlist" onClick={() => go("wishlist")}>Manage Wishlist</Button>
        <DataFreshnessChip ago={s.updatedAgo} />
      </div>
    </div>
  );
}

/* =================================================================
   THIS WEEK
   ================================================================= */
function ThisWeek({ tweaks }) {
  const w = window.GT.weekly;
  const [acts, setActs] = useState(w.recommended);
  const toggle = (id) => setActs((a) => a.map((x) => x.id === id ? { ...x, done: !x.done } : x));
  const doneCount = acts.filter((a) => a.done).length;
  return (
    <div className="gt-page">
      <PageHead title="This Week" sub={`Resets ${w.resetLabel}`}
        right={<CountdownChip prefix="Resets in" time={w.resetIn} icon="clock" />} />

      <Panel title="Recommended for you" icon="sparkle" accent="var(--c-signal)"
        right={<span className="gt-action-meta mono">{doneCount}/{acts.length} done</span>}>
        <ActionList items={acts} onToggle={toggle} />
      </Panel>

      <div className="gt-week-cols">
        <XurModule xur={w.xur} />
        <MilestoneModule milestones={w.milestones} />
      </div>

      <VendorModule vendors={w.vendors} />
    </div>
  );
}

/* =================================================================
   COLLECTIONS
   ================================================================= */
const SOURCE_MAP = {
  weapons: null, "w-raids": "Vault of Glass|Deep Stone Crypt|Last Wish|King's Fall",
  "w-trials": "Trials", "w-nightfall": "Nightfall", "w-world": "World", "w-vendor": "Banshee|vendor",
  exotics: "exotic", cosmetics: "Shader|Ghost|Sparrow|Ship|Emblem",
};
function Collections({ wished, onWish, onOpen, tweaks }) {
  const [active, setActive] = useState("weapons");
  const [missingOnly, setMissingOnly] = useState(true);
  const [rarity, setRarity] = useState(null);
  const [diff, setDiff] = useState(null);
  const [sort, setSort] = useState("rarity");
  const [view, setView] = useState("grid");
  const [loading, setLoading] = useState(false);

  const node = useMemo(() => {
    const flat = [];
    window.GT.tree.forEach((n) => { flat.push(n); (n.children || []).forEach((c) => flat.push(c)); });
    return flat.find((n) => n.id === active) || window.GT.tree[0];
  }, [active]);

  const items = useMemo(() => {
    let list = window.GT.items.slice();
    // crude category filter by id keyword
    if (active === "exotics" || active === "e-weapons" || active === "e-armor") list = list.filter((i) => i.rarity === "exotic");
    else if (active === "cosmetics" || active.startsWith("c-")) list = list.filter((i) => i.slot === "Cosmetic");
    else if (active === "armor" || active.startsWith("a-")) list = list.filter((i) => ["Helmet", "Gauntlets", "Chest Armor", "Leg Armor", "Class Item"].includes(i.type) || i.slot === "Energy");
    if (missingOnly) list = list.filter((i) => !i.collected);
    if (rarity) list = list.filter((i) => i.rarity === rarity);
    if (diff) list = list.filter((i) => i.diff === diff);
    const rank = { exotic: 0, legendary: 1, rare: 2, uncommon: 3, common: 4 };
    const drank = { challenging: 0, moderate: 1, easy: 2 };
    if (sort === "rarity") list.sort((a, b) => rank[a.rarity] - rank[b.rarity]);
    else if (sort === "name") list.sort((a, b) => a.name.localeCompare(b.name));
    else if (sort === "difficulty") list.sort((a, b) => drank[a.diff] - drank[b.diff]);
    else if (sort === "avail") list.sort((a, b) => (b.obtainable ? 1 : 0) - (a.obtainable ? 1 : 0));
    return list;
  }, [active, missingOnly, rarity, diff, sort]);

  const reload = () => { setLoading(true); setTimeout(() => setLoading(false), 650); };
  useEffect(() => { reload(); }, [active]);

  const clearFilters = () => { setRarity(null); setDiff(null); setMissingOnly(false); };
  const hasFilters = rarity || diff || missingOnly;

  return (
    <div className="gt-page gt-collections">
      <PageHead title="Collections"
        sub={<span className="mono">74% complete · 1,204/1,628 collected</span>}
        right={<DataFreshnessChip ago="4m" />} />

      <div className="gt-coll-layout">
        {/* CATEGORY TREE */}
        <aside className="gt-coll-aside">
          <div className="gt-section-title" style={{ marginBottom: "var(--s-3)" }}>Categories</div>
          <CategoryTree nodes={window.GT.tree} activeId={active} onSelect={setActive} />
        </aside>

        {/* MAIN */}
        <div className="gt-coll-main">
          {/* FILTER BAR */}
          <div className="gt-coll-toolbar">
            <div className="gt-filterbar">
              <FilterChip on={missingOnly} onClick={() => setMissingOnly((v) => !v)}>
                <Icon name="check" size="0.85rem" /> Missing only
              </FilterChip>
              <Dropdown label="Rarity" value={rarity && window.GT.label.rarity[rarity]} options={window.GT.RARITIES.map((r) => ({ v: r, l: window.GT.label.rarity[r] }))} onPick={setRarity} />
              <Dropdown label="Difficulty" value={diff && window.GT.label.diff[diff]} options={window.GT.DIFFS.map((d) => ({ v: d, l: window.GT.label.diff[d] }))} onPick={setDiff} note="estimate" />
              <Dropdown label="Sort" value={{ rarity: "Rarity", name: "Name", difficulty: "Difficulty", avail: "Availability" }[sort]} options={[{ v: "rarity", l: "Rarity" }, { v: "name", l: "Name" }, { v: "difficulty", l: "Difficulty" }, { v: "avail", l: "Availability" }]} onPick={setSort} noClear />
            </div>
            <div className="gt-viewtoggle">
              <button className="gt-iconbtn" data-on={view === "grid"} onClick={() => setView("grid")} aria-label="Grid"><Icon name="grid" size="1rem" /></button>
              <button className="gt-iconbtn" data-on={view === "list"} onClick={() => setView("list")} aria-label="List"><Icon name="list" size="1rem" /></button>
            </div>
          </div>

          <div className="gt-coll-stats">
            <StatTile num={node.count ? node.count[1].toLocaleString() : "—"} label="Total" mono />
            <StatTile num={node.count ? node.count[0].toLocaleString() : "—"} label="Collected" mono color="var(--c-complete)" />
            <StatTile num={node.count ? (node.count[1] - node.count[0]).toLocaleString() : "—"} label="Missing" mono color="var(--c-signal)" />
            <div className="gt-coll-resultcount mono">{items.length} shown</div>
          </div>

          {/* GRID */}
          {loading ? (
            <div className={`gt-itemgrid`}>{Array.from({ length: 8 }).map((_, i) => <ItemCardSkeleton key={i} />)}</div>
          ) : items.length === 0 ? (
            <div className="gt-card">
              <EmptyState icon={hasFilters ? "filter" : "check"} color={hasFilters ? "var(--c-text-3)" : "var(--c-complete)"}
                title={hasFilters ? "No items match these filters" : "All caught up!"}
                body={hasFilters ? "Try loosening a filter to see more of this category." : "You've collected everything in this category. Nice work, Guardian."}
                action={hasFilters ? <Button variant="outline" sm onClick={clearFilters}>Clear filters</Button> : null} />
            </div>
          ) : view === "grid" ? (
            <div className="gt-itemgrid">
              {items.map((it) => <ItemCard key={it.id} item={it} density={tweaks.cardStyle === "compact" ? "compact" : "grid"} personalize={tweaks.personalize} showCollected={!missingOnly}
                wished={wished.has(it.id)} onWish={onWish} onOpen={onOpen} />)}
            </div>
          ) : (
            <div className="gt-itemlist">
              {items.map((it) => <ItemCard key={it.id} item={it} density="list" personalize={tweaks.personalize} showCollected={!missingOnly}
                wished={wished.has(it.id)} onWish={onWish} onOpen={onOpen} />)}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/* small dropdown filter */
function Dropdown({ label, value, options, onPick, note, noClear }) {
  const [open, setOpen] = useState(false);
  const ref = useRef();
  useEffect(() => {
    const h = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    document.addEventListener("click", h); return () => document.removeEventListener("click", h);
  }, []);
  return (
    <div className="gt-dd" ref={ref}>
      <button className="gt-fchip" data-on={!!value} onClick={() => setOpen((v) => !v)}>
        {value || label}{note && !value && <span className="gt-dd-note">({note})</span>}
        <Icon name="chevronDown" size="0.75rem" />
      </button>
      {open && (
        <div className="gt-dd-menu">
          {!noClear && <button className="gt-dd-opt" onClick={() => { onPick(null); setOpen(false); }}>All {label.toLowerCase()}</button>}
          {options.map((o) => <button key={o.v} className="gt-dd-opt" data-on={value === o.l} onClick={() => { onPick(o.v); setOpen(false); }}>{o.l}</button>)}
        </div>
      )}
    </div>
  );
}

/* =================================================================
   WISHLIST
   ================================================================= */
const PRIORITY_ORDER = ["urgent", "high", "medium", "low"];
function Wishlist({ go }) {
  const [list, setList] = useState(window.GT.wishlist);
  const [filter, setFilter] = useState("all");
  const [sort, setSort] = useState("availability");

  const counts = useMemo(() => {
    const c = { all: list.length };
    PRIORITY_ORDER.forEach((p) => c[p] = list.filter((i) => i.priority === p).length);
    return c;
  }, [list]);

  const shown = useMemo(() => {
    let l = filter === "all" ? list.slice() : list.filter((i) => i.priority === filter);
    if (sort === "availability") l.sort((a, b) => (b.avail.now ? 1 : 0) - (a.avail.now ? 1 : 0));
    else if (sort === "priority") l.sort((a, b) => PRIORITY_ORDER.indexOf(a.priority) - PRIORITY_ORDER.indexOf(b.priority));
    return l;
  }, [list, filter, sort]);

  const setPriority = (id, p) => setList((l) => l.map((i) => i.id === id ? { ...i, priority: p } : i));
  const remove = (id) => setList((l) => l.filter((i) => i.id !== id));

  if (list.length === 0) {
    return (
      <div className="gt-page">
        <PageHead title="Wishlist" />
        <div className="gt-card"><EmptyState icon="wishlist" color="var(--c-signal)" title="Your wishlist is empty"
          body="Track the items you're chasing. We'll tell you the moment they're available from a vendor or this week's activities."
          action={<Button variant="primary" icon="collections" onClick={() => go("collections")}>Browse Collections</Button>} /></div>
      </div>
    );
  }

  return (
    <div className="gt-page">
      <PageHead title="Wishlist" sub={<span className="mono">{list.length} items · {list.filter((i) => i.avail.now).length} available now</span>}
        right={<Dropdown label="Sort: Availability" value={{ availability: "Sort: Availability", priority: "Sort: Priority" }[sort]} noClear
          options={[{ v: "availability", l: "Sort: Availability" }, { v: "priority", l: "Sort: Priority" }]} onPick={setSort} />} />

      <div className="gt-filterbar gt-wl-filters">
        {["all", ...PRIORITY_ORDER].map((p) => (
          <FilterChip key={p} on={filter === p} onClick={() => setFilter(p)}>
            {p === "all" ? "All" : window.GT.label.priority[p]} <span className="mono" style={{ opacity: 0.6 }}>{counts[p]}</span>
          </FilterChip>
        ))}
      </div>

      <div className="gt-wl-list">
        {shown.map((i) => (
          <div key={i.id} className="gt-wl-item gt-card" data-rarity={i.rarity}>
            <ItemTile rarity={i.rarity} type={i.type} style={{ width: "3rem" }} />
            <div className="gt-wl-body">
              <div className="gt-wl-top">
                <div>
                  <div className="gt-item-name">{i.name}</div>
                  <div className="gt-item-type">{i.type} · <Badge kind={i.rarity} dot /></div>
                </div>
                <Badge kind={i.priority} solid>{window.GT.label.priority[i.priority]}</Badge>
              </div>
              {i.avail.now
                ? <div className="gt-wl-avail"><Badge kind="avail-now" dot icon="bolt" /><span className="gt-wl-where">{i.avail.where}</span></div>
                : <div className="gt-action-meta mono">Source: {i.avail.where}</div>}
              {i.notes && <div className="gt-wl-notes">“{i.notes}”</div>}
              <div className="gt-wl-foot">
                <Dropdown label="Priority" value={window.GT.label.priority[i.priority]} noClear
                  options={PRIORITY_ORDER.map((p) => ({ v: p, l: window.GT.label.priority[p] }))} onPick={(p) => setPriority(i.id, p)} />
                <span className="gt-action-meta mono">Added {i.added}</span>
                <button className="gt-link gt-link--danger" onClick={() => remove(i.id)}><Icon name="close" size="0.8rem" /> Remove</button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

Object.assign(window, { PageHead, Dashboard, ThisWeek, Collections, Wishlist, Dropdown });
