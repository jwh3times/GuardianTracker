import React, { useState } from "react";
import { Icon } from "./Icon";
import {
  DIFF_LABEL,
  PRIORITY_LABEL,
  RARITY_LABEL,
  TYPE_GLYPH,
} from "../../lib/constants";

/** Allow CSS custom properties in inline styles without fighting the type. */
type CSS = React.CSSProperties & Record<`--${string}`, string | number>;

/* ---------------- ITEM TILE (placeholder icon) ---------------- */
export function ItemTile({
  rarity,
  type,
  style,
}: {
  rarity: string;
  type?: string;
  style?: React.CSSProperties;
}) {
  const glyph =
    (type && TYPE_GLYPH[type]) || type?.slice(0, 2).toUpperCase() || "??";
  return (
    <div className="gt-tile" data-rarity={rarity} style={style}>
      <span className="gt-tile-glyph">{glyph}</span>
    </div>
  );
}

/* ---------------- BADGE ---------------- */
export const BADGE_COLOR: Record<string, string> = {
  exotic: "var(--c-exotic)", legendary: "var(--c-legendary)", rare: "var(--c-rare)", uncommon: "var(--c-uncommon)", common: "var(--c-common)",
  easy: "var(--c-easy)", moderate: "var(--c-moderate)", challenging: "var(--c-challenging)",
  "avail-now": "var(--c-avail)", expiring: "var(--c-expiring)", missing: "var(--c-signal)", new: "var(--c-signal)",
  "completes-set": "var(--c-exotic)", complete: "var(--c-complete)", "in-progress": "var(--c-progress)", gilded: "var(--c-gild)",
  owned: "var(--c-text-3)",
  urgent: "var(--c-danger)", high: "var(--c-expiring)", medium: "var(--c-rare)", low: "var(--c-text-3)",
};
const BADGE_LABEL: Record<string, string> = {
  "avail-now": "Avail now", "completes-set": "Completes set", "in-progress": "In progress",
  missing: "Missing", new: "New", expiring: "Expiring", complete: "Complete", gilded: "Gilded", owned: "Owned",
};

export function Badge({
  kind,
  children,
  dot,
  solid,
  lg,
  icon,
  style,
}: {
  kind: string;
  children?: React.ReactNode;
  dot?: boolean;
  solid?: boolean;
  lg?: boolean;
  icon?: string;
  style?: React.CSSProperties;
}) {
  const color = BADGE_COLOR[kind] || "var(--c-text-2)";
  const label =
    children ??
    BADGE_LABEL[kind] ??
    RARITY_LABEL[kind as keyof typeof RARITY_LABEL] ??
    DIFF_LABEL[kind as keyof typeof DIFF_LABEL] ??
    PRIORITY_LABEL[kind as keyof typeof PRIORITY_LABEL] ??
    kind;
  return (
    <span
      className={`gt-badge${solid ? " gt-badge--solid" : ""}${lg ? " gt-badge--lg" : ""}`}
      style={{ "--bdg": color, ...style } as CSS}
    >
      {dot && <span className="gt-dot" />}
      {icon && <Icon name={icon} size="0.85em" />}
      {label}
    </span>
  );
}

/* ---------------- BUTTON ---------------- */
type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "outline" | "ghost";
  sm?: boolean;
  icon?: string;
  iconOnly?: boolean;
};
export function Button({
  variant = "outline",
  sm,
  icon,
  iconOnly,
  children,
  className = "",
  ...rest
}: ButtonProps) {
  return (
    <button
      className={`gt-btn gt-btn--${variant}${sm ? " gt-btn--sm" : ""}${iconOnly ? " gt-btn--icon" : ""}${className ? " " + className : ""}`}
      {...rest}
    >
      {icon && <Icon name={icon} size="1.05rem" />}
      {children}
    </button>
  );
}

/* ---------------- PROGRESS BAR ---------------- */
export function ProgressBar({
  value,
  max = 100,
  label,
  showVal,
  color,
  size,
  complete,
  valText,
}: {
  value: number;
  max?: number;
  label?: string;
  showVal?: boolean;
  color?: string;
  size?: "thin" | "tall";
  complete?: boolean;
  valText?: string;
}) {
  const pct = Math.max(0, Math.min(100, (value / max) * 100));
  const done = complete ?? pct >= 100;
  return (
    <div className={`gt-prog${size ? " gt-prog--" + size : ""}`}>
      {(label || showVal) && (
        <div className="gt-prog-head">
          {label && <span className="gt-prog-label">{label}</span>}
          {showVal && (
            <span className="gt-prog-val mono">{valText || `${value}/${max}`}</span>
          )}
        </div>
      )}
      <div className="gt-prog-track">
        <div
          className="gt-prog-fill"
          data-complete={done}
          style={{ "--val": pct + "%", "--pc": color || "var(--c-signal)" } as CSS}
        />
      </div>
    </div>
  );
}

/* ---------------- RADIAL (HEX) PROGRESS ---------------- */
const HEX_PTS = "50,6 88,28 88,72 50,94 12,72 12,28";
export function RadialProgress({
  value,
  size = "5rem",
  color,
  sub,
  pctSize = "1.6rem",
  label,
}: {
  value: number;
  size?: string;
  color?: string;
  sub?: React.ReactNode;
  pctSize?: string;
  label?: React.ReactNode;
}) {
  const pct = Math.max(0, Math.min(100, value));
  return (
    <div className="gt-radial" style={{ width: size, height: size }}>
      <svg viewBox="0 0 100 100">
        <polygon className="gt-radial-track" points={HEX_PTS} pathLength={100} />
        <polygon
          className="gt-radial-val"
          points={HEX_PTS}
          pathLength={100}
          style={{ "--val": pct, "--rc": color || "var(--c-signal)" } as CSS}
        />
      </svg>
      <div className="gt-radial-center">
        <div>
          <div className="gt-radial-pct" style={{ "--rsize": pctSize } as CSS}>
            {label ?? `${pct}%`}
          </div>
          {sub && <div className="gt-radial-sub">{sub}</div>}
        </div>
      </div>
    </div>
  );
}

/* ---------------- STAT TILE ---------------- */
export function StatTile({
  num,
  label,
  mono,
  color,
}: {
  num: React.ReactNode;
  label: string;
  mono?: boolean;
  color?: string;
}) {
  return (
    <div className="gt-stat">
      <span className={`gt-stat-num${mono ? " mono" : ""}`} style={color ? { color } : undefined}>
        {num}
      </span>
      <span className="gt-stat-label">{label}</span>
    </div>
  );
}

/* ---------------- COUNTDOWN CHIP ---------------- */
export function fmtDur({ d = 0, h = 0, m = 0 }: { d?: number; h?: number; m?: number }) {
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}
export function CountdownChip({
  time,
  prefix,
  soon,
  icon = "clock",
}: {
  time: string | { d?: number; h?: number; m?: number };
  prefix?: string;
  soon?: boolean;
  icon?: string;
}) {
  return (
    <span className="gt-chip gt-chip--count" data-soon={!!soon}>
      <Icon name={icon} size="0.85rem" />
      {prefix && <span style={{ color: "var(--c-text-3)" }}>{prefix}</span>}
      <span className="gt-chip-time">
        {typeof time === "string" ? time : fmtDur(time)}
      </span>
    </span>
  );
}

/* ---------------- DATA FRESHNESS CHIP ---------------- */
export function DataFreshnessChip({
  ago,
  onRefresh,
}: {
  ago: string;
  onRefresh?: () => void;
}) {
  const [state, setState] = useState<"fresh" | "refreshing">("fresh");
  const refresh = () => {
    if (state === "refreshing") return;
    setState("refreshing");
    setTimeout(() => {
      setState("fresh");
      onRefresh?.();
    }, 1100);
  };
  return (
    <button
      className="gt-chip gt-fresh"
      data-state={state}
      onClick={refresh}
      title="Refresh data"
    >
      <Icon name="refresh" size="0.85rem" className="gt-fresh-icon" />
      {state === "refreshing" ? "Refreshing…" : `Updated ${ago} ago`}
    </button>
  );
}

/* ---------------- FILTER CHIP ---------------- */
export function FilterChip({
  on,
  onClick,
  children,
  removable,
}: {
  on?: boolean;
  onClick?: () => void;
  children: React.ReactNode;
  removable?: boolean;
}) {
  return (
    <button className="gt-fchip" data-on={!!on} onClick={onClick}>
      {children}
      {removable && on && <span className="gt-fchip-x">×</span>}
    </button>
  );
}

/* ---------------- SKELETON ---------------- */
export function Skeleton({
  w = "100%",
  h = "1rem",
  r,
  style,
}: {
  w?: string;
  h?: string;
  r?: string;
  style?: React.CSSProperties;
}) {
  return (
    <div className="gt-skel" style={{ width: w, height: h, borderRadius: r, ...style }} />
  );
}
export function ItemCardSkeleton() {
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
export function EmptyState({
  icon = "sparkle",
  color,
  title,
  body,
  action,
  secondary,
}: {
  icon?: string;
  color?: string;
  title: React.ReactNode;
  body?: React.ReactNode;
  action?: React.ReactNode;
  secondary?: React.ReactNode;
}) {
  return (
    <div className="gt-empty">
      <div className="gt-empty-mark" style={{ "--em": color || "var(--c-text-3)" } as CSS}>
        <Icon name={icon} size="3rem" stroke={1.5} />
      </div>
      <div className="gt-empty-title">{title}</div>
      {body && <div className="gt-empty-body">{body}</div>}
      {(action || secondary) && (
        <div style={{ display: "flex", gap: "var(--s-2)", marginTop: "var(--s-2)" }}>
          {action}
          {secondary}
        </div>
      )}
    </div>
  );
}
