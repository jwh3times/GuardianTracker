/* ============================================================
   GUARDIAN TRACKER — COMPONENT KIT (React)
   Exports to window. Styling via kit.css + tokens.css.
   ============================================================ */
const { useState, useEffect, useRef, useMemo } = React;

/* ---------------- ICONS (simple line set) ---------------- */
const ICONS = {
  dashboard: "M3 13h8V3H3v10zm0 8h8v-6H3v6zm10 0h8V11h-8v10zm0-18v6h8V3h-8z",
  week: "M7 2v3M17 2v3M3 9h18M5 5h14a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1z",
  collections: "M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z",
  catalyst: "M12 2l2.4 4.9 5.4.8-3.9 3.8.9 5.4-4.8-2.5-4.8 2.5.9-5.4L4.2 7.7l5.4-.8z",
  triumph: "M8 3h8v3a4 4 0 0 1-8 0V3zM6 4H3v2a4 4 0 0 0 4 4M18 4h3v2a4 4 0 0 1-4 4M9 14h6M10 14v4M14 14v4M8 21h8",
  wishlist: "M12 21l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.18z",
  settings: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-2.81 1.17V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 8 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 3.6 14H3.5a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 5 8a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 10 3.6h.09A2 2 0 0 1 14 3.6V3.5a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 2.81 1.17l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 20.4 10h.1a2 2 0 0 1 0 4h-.1z",
  search: "M11 19a8 8 0 1 0 0-16 8 8 0 0 0 0 16zM21 21l-4.3-4.3",
  refresh: "M21 12a9 9 0 1 1-2.64-6.36M21 3v6h-6",
  chevron: "M9 18l6-6-6-6",
  chevronDown: "M6 9l6 6 6-6",
  plus: "M12 5v14M5 12h14",
  check: "M20 6L9 17l-5-5",
  close: "M18 6L6 18M6 6l12 12",
  info: "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20zM12 16v-4M12 8h.01",
  clock: "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20zM12 6v6l4 2",
  bolt: "M13 2L3 14h7l-1 8 10-12h-7z",
  signout: "M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9",
  bungie: "M12 2l9 4.5v5c0 5-3.8 9.2-9 10.5-5.2-1.3-9-5.5-9-10.5v-5z",
  sparkle: "M12 3l1.6 5.4L19 10l-5.4 1.6L12 17l-1.6-5.4L5 10l5.4-1.6z",
  filter: "M3 5h18M7 12h10M10 19h4",
  external: "M14 3h7v7M21 3l-9 9M19 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h5",
  grid: "M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z",
  list: "M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01",
  menu: "M3 6h18M3 12h18M3 18h18",
  star: "M12 2l3 6.5 7 .9-5 4.8 1.3 7L12 18l-6.6 3.2L6.7 14l-5-4.8 7-.9z",
  shield: "M12 2l8 3v6c0 5-3.4 8.5-8 10-4.6-1.5-8-5-8-10V5z",
  flag: "M4 22V4M4 4h12l-2 4 2 4H4",
  users: "M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM22 21v-2a4 4 0 0 0-3-3.87M16 3.13A4 4 0 0 1 16 11",
  lock: "M5 11h14a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1v-8a1 1 0 0 1 1-1zM8 11V7a4 4 0 0 1 8 0v4",
};
function Icon({ name, size = "1.15rem", stroke = 2, fill = "none", style, ...p }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill={fill} stroke="currentColor"
      strokeWidth={stroke} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"
      style={{ flex: "none", ...style }} {...p}>
      <path d={ICONS[name]} />
    </svg>
  );
}

/* ---------------- ITEM TILE (placeholder icon) ---------------- */
function ItemTile({ rarity, type, style }) {
  const glyph = (window.GT.TYPE_GLYPH[type]) || type?.slice(0, 2).toUpperCase() || "??";
  return (
    <div className="gt-tile" data-rarity={rarity} style={style}>
      <span className="gt-tile-glyph">{glyph}</span>
    </div>
  );
}

/* ---------------- BADGE ---------------- */
const BADGE_COLOR = {
  // rarity
  exotic: "var(--c-exotic)", legendary: "var(--c-legendary)", rare: "var(--c-rare)", uncommon: "var(--c-uncommon)", common: "var(--c-common)",
  // difficulty
  easy: "var(--c-easy)", moderate: "var(--c-moderate)", challenging: "var(--c-challenging)",
  // availability / status
  "avail-now": "var(--c-avail)", expiring: "var(--c-expiring)", missing: "var(--c-signal)", "new": "var(--c-signal)",
  "completes-set": "var(--c-exotic)", complete: "var(--c-complete)", "in-progress": "var(--c-progress)", gilded: "var(--c-gild)",
  owned: "var(--c-text-3)",
  // priority
  urgent: "var(--c-danger)", high: "var(--c-expiring)", medium: "var(--c-rare)", low: "var(--c-text-3)",
};
const BADGE_LABEL = {
  "avail-now": "Avail now", "completes-set": "Completes set", "in-progress": "In progress",
  missing: "Missing", "new": "New", expiring: "Expiring", complete: "Complete", gilded: "Gilded", owned: "Owned",
};
function Badge({ kind, children, dot, solid, lg, icon, style }) {
  const color = BADGE_COLOR[kind] || "var(--c-text-2)";
  const label = children || BADGE_LABEL[kind] || (window.GT.label.rarity[kind]) || (window.GT.label.diff[kind]) || kind;
  return (
    <span className={`gt-badge${solid ? " gt-badge--solid" : ""}${lg ? " gt-badge--lg" : ""}`}
      style={{ "--bdg": color, ...style }}>
      {dot && <span className="gt-dot" />}
      {icon && <Icon name={icon} size="0.85em" />}
      {label}
    </span>
  );
}

/* ---------------- BUTTON ---------------- */
function Button({ variant = "outline", sm, icon, iconOnly, children, ...p }) {
  return (
    <button className={`gt-btn gt-btn--${variant}${sm ? " gt-btn--sm" : ""}${iconOnly ? " gt-btn--icon" : ""}`} {...p}>
      {icon && <Icon name={icon} size="1.05rem" />}
      {children}
    </button>
  );
}

/* ---------------- PROGRESS BAR ---------------- */
function ProgressBar({ value, max = 100, label, showVal, color, size, complete, valText }) {
  const pct = Math.max(0, Math.min(100, (value / max) * 100));
  const done = complete ?? pct >= 100;
  return (
    <div className={`gt-prog${size ? " gt-prog--" + size : ""}`}>
      {(label || showVal) && (
        <div className="gt-prog-head">
          {label && <span className="gt-prog-label">{label}</span>}
          {showVal && <span className="gt-prog-val mono">{valText || `${value}/${max}`}</span>}
        </div>
      )}
      <div className="gt-prog-track">
        <div className="gt-prog-fill" data-complete={done} style={{ "--val": pct + "%", "--pc": color || "var(--c-signal)" }} />
      </div>
    </div>
  );
}

/* ---------------- RADIAL (HEX) PROGRESS ---------------- */
const HEX_PTS = "50,6 88,28 88,72 50,94 12,72 12,28";
function RadialProgress({ value, size = "5rem", color, sub, pctSize = "1.6rem", label }) {
  const pct = Math.max(0, Math.min(100, value));
  return (
    <div className="gt-radial" style={{ width: size, height: size }}>
      <svg viewBox="0 0 100 100">
        <polygon className="gt-radial-track" points={HEX_PTS} pathLength="100" />
        <polygon className="gt-radial-val" points={HEX_PTS} pathLength="100"
          style={{ "--val": pct, "--rc": color || "var(--c-signal)" }} />
      </svg>
      <div className="gt-radial-center">
        <div>
          <div className="gt-radial-pct" style={{ "--rsize": pctSize }}>{label ?? `${pct}%`}</div>
          {sub && <div className="gt-radial-sub">{sub}</div>}
        </div>
      </div>
    </div>
  );
}

/* ---------------- STAT TILE ---------------- */
function StatTile({ num, label, mono, color }) {
  return (
    <div className="gt-stat">
      <span className={`gt-stat-num${mono ? " mono" : ""}`} style={color ? { color } : null}>{num}</span>
      <span className="gt-stat-label">{label}</span>
    </div>
  );
}

/* ---------------- COUNTDOWN CHIP (live) ---------------- */
function fmtDur({ d = 0, h = 0, m = 0 }) {
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}
function CountdownChip({ time, prefix, soon, icon = "clock" }) {
  return (
    <span className="gt-chip gt-chip--count" data-soon={!!soon}>
      <Icon name={icon} size="0.85rem" />
      {prefix && <span style={{ color: "var(--c-text-3)" }}>{prefix}</span>}
      <span className="gt-chip-time">{typeof time === "string" ? time : fmtDur(time)}</span>
    </span>
  );
}

/* ---------------- DATA FRESHNESS CHIP ---------------- */
function DataFreshnessChip({ ago, onRefresh }) {
  const [state, setState] = useState("fresh"); // fresh | refreshing
  const refresh = () => {
    if (state === "refreshing") return;
    setState("refreshing");
    setTimeout(() => { setState("fresh"); onRefresh && onRefresh(); }, 1100);
  };
  return (
    <button className="gt-chip gt-fresh" data-state={state} onClick={refresh} title="Refresh data">
      <Icon name="refresh" size="0.85rem" className="gt-fresh-icon" />
      {state === "refreshing" ? "Refreshing…" : `Updated ${ago} ago`}
    </button>
  );
}

/* ---------------- FILTER CHIP ---------------- */
function FilterChip({ on, onClick, children, removable }) {
  return (
    <button className="gt-fchip" data-on={!!on} onClick={onClick}>
      {children}
      {removable && on && <span className="gt-fchip-x">×</span>}
    </button>
  );
}

/* ---------------- SKELETON ---------------- */
function Skeleton({ w = "100%", h = "1rem", r, style }) {
  return <div className="gt-skel" style={{ width: w, height: h, borderRadius: r, ...style }} />;
}
function ItemCardSkeleton() {
  return (
    <div className="gt-card" style={{ display: "flex", flexDirection: "column", gap: "var(--s-3)" }}>
      <div style={{ display: "flex", gap: "var(--s-3)" }}>
        <Skeleton w="3rem" h="3rem" r="var(--r-sm)" />
        <div style={{ flex: 1, display: "flex", flexDirection: "column", gap: "var(--s-2)" }}>
          <Skeleton w="70%" h="0.9rem" />
          <Skeleton w="45%" h="0.7rem" />
        </div>
      </div>
      <Skeleton w="100%" h="0.7rem" />
      <Skeleton w="100%" h="2rem" r="var(--r-sm)" />
    </div>
  );
}

/* ---------------- EMPTY / ERROR STATE ---------------- */
function EmptyState({ icon = "sparkle", color, title, body, action, secondary }) {
  return (
    <div className="gt-empty">
      <div className="gt-empty-mark" style={{ "--em": color || "var(--c-text-3)" }}>
        <Icon name={icon} size="3rem" stroke={1.5} />
      </div>
      <div className="gt-empty-title">{title}</div>
      {body && <div className="gt-empty-body">{body}</div>}
      <div style={{ display: "flex", gap: "var(--s-2)", marginTop: "var(--s-2)" }}>
        {action}{secondary}
      </div>
    </div>
  );
}

/* ---------------- ITEM CARD ---------------- */
function ItemCard({ item, density = "grid", personalize = "normal", onOpen, onWish, wished, showCollected }) {
  const r = item.rarity;
  const availBadge = item.obtainable && !item.collected;
  const showFor = personalize !== "off";
  const aggressive = personalize === "aggressive";

  if (density === "list") {
    return (
      <div className="gt-item gt-item--list" data-rarity={r} data-collected={showCollected && item.collected} onClick={() => onOpen && onOpen(item)}>
        <ItemTile rarity={r} type={item.type} />
        <div className="gt-item-head">
          <div className="gt-il-main">
            <div className="gt-item-name">{item.name}</div>
            <div className="gt-item-type">{item.type} · {item.source}</div>
          </div>
          <div className="gt-item-badges">
            <Badge kind={r} dot />
            <Badge kind={item.diff}>{window.GT.label.diff[item.diff]}</Badge>
            {showFor && availBadge && <Badge kind="avail-now" dot />}
          </div>
        </div>
        <div className="gt-item-actions">
          <button className="gt-iconbtn" data-on={wished} title="Wishlist" onClick={(e) => { e.stopPropagation(); onWish && onWish(item); }}><Icon name="wishlist" size="1rem" fill={wished ? "currentColor" : "none"} /></button>
          <button className="gt-iconbtn" title="Details" onClick={(e) => { e.stopPropagation(); onOpen && onOpen(item); }}><Icon name="info" size="1rem" /></button>
        </div>
      </div>
    );
  }
  if (density === "compact") {
    return (
      <div className="gt-item gt-item--compact" data-rarity={r} data-collected={showCollected && item.collected} onClick={() => onOpen && onOpen(item)}>
        <ItemTile rarity={r} type={item.type} />
        <div className="gt-item-head" style={{ flex: 1 }}>
          <div className="gt-item-name">{item.name}</div>
          <div className="gt-item-type">{item.type}</div>
        </div>
        {showFor && availBadge && <Badge kind="avail-now" dot />}
        <button className="gt-iconbtn" data-on={wished} onClick={(e) => { e.stopPropagation(); onWish && onWish(item); }}><Icon name="wishlist" size="0.95rem" fill={wished ? "currentColor" : "none"} /></button>
      </div>
    );
  }
  // grid
  return (
    <div className="gt-item gt-item--grid" data-rarity={r} data-collected={showCollected && item.collected}
      style={aggressive && availBadge ? { boxShadow: "0 0 0 0.09rem var(--c-signal-line)" } : null}
      onClick={() => onOpen && onOpen(item)}>
      <div className="gt-item-top">
        <ItemTile rarity={r} type={item.type} />
        <div className="gt-item-head">
          <div className="gt-item-name">{item.name}</div>
          <div className="gt-item-type">{item.type} · {item.slot}</div>
          <div className="gt-item-badges" style={{ marginTop: "var(--s-1)" }}>
            <Badge kind={r} dot />
            <Badge kind={item.diff}>{window.GT.label.diff[item.diff]}</Badge>
          </div>
        </div>
      </div>
      <div className="gt-item-src"><Icon name="bolt" size="0.8rem" style={{ color: "var(--c-text-4)" }} />{item.source}</div>
      {showFor && (availBadge || item.collected) && (
        <div className="gt-item-badges">
          {availBadge && <Badge kind="avail-now" dot icon="bolt" />}
          {item.collected && <Badge kind="owned" dot>Collected</Badge>}
        </div>
      )}
      <div className="gt-item-foot">
        <button className="gt-iconbtn" data-on={wished} title="Add to wishlist" onClick={(e) => { e.stopPropagation(); onWish && onWish(item); }}>
          <Icon name="wishlist" size="1rem" fill={wished ? "currentColor" : "none"} />
        </button>
        <Button sm variant="ghost" icon="info" onClick={(e) => { e.stopPropagation(); onOpen && onOpen(item); }}>Details</Button>
      </div>
    </div>
  );
}

Object.assign(window, {
  Icon, ICONS, ItemTile, Badge, Button, ProgressBar, RadialProgress, StatTile,
  CountdownChip, DataFreshnessChip, FilterChip, Skeleton, ItemCardSkeleton, EmptyState, ItemCard,
  fmtDur, BADGE_COLOR,
});
