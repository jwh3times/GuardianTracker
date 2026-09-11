import React, { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { Icon } from "./Icon";

/* ---------------- PAGE HEADER ---------------- */
export function PageHead({
  title,
  sub,
  right,
}: {
  title: React.ReactNode;
  sub?: React.ReactNode;
  right?: React.ReactNode;
}) {
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
/* ---------------- PANEL (titled section card) ---------------- */
export function Panel({
  title,
  icon,
  right,
  children,
  pad,
  style,
  accent,
}: {
  title?: React.ReactNode;
  icon?: string;
  right?: React.ReactNode;
  children: React.ReactNode;
  pad?: boolean;
  style?: React.CSSProperties;
  accent?: string;
}) {
  return (
    <section
      className="gt-card gt-panel"
      style={{
        borderColor: accent
          ? "color-mix(in oklch, " + accent + " 35%, var(--c-line-soft))"
          : undefined,
        ...style,
      }}
    >
      {(title || right) && (
        <header className="gt-panel-head">
          <div className="gt-section-title">
            {icon && (
              <Icon
                name={icon}
                size="0.95rem"
                style={{ color: accent || "var(--c-text-3)" }}
              />
            )}
            {title}
          </div>
          {right}
        </header>
      )}
      <div
        className="gt-panel-body"
        style={pad === false ? { padding: 0 } : undefined}
      >
        {children}
      </div>
    </section>
  );
}
/* ---------------- DROPDOWN (filter / sort) ---------------- */
export interface DropdownOption {
  v: string;
  l: string;
}
export function Dropdown({
  label,
  value,
  options,
  onPick,
  note,
  noClear,
  disabled,
}: {
  label: string;
  value?: string | null;
  options: DropdownOption[];
  onPick: (v: string | null) => void;
  note?: string;
  noClear?: boolean;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const h = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node))
        setOpen(false);
    };
    document.addEventListener("click", h);
    return () => document.removeEventListener("click", h);
  }, []);
  return (
    <div className="gt-dd" ref={ref}>
      <button
        className="gt-fchip"
        data-on={!!value}
        disabled={disabled}
        onClick={() => setOpen((v) => !v)}
      >
        {value || label}
        {note && !value && <span className="gt-dd-note">({note})</span>}
        <Icon name="chevronDown" size="0.75rem" />
      </button>
      {open && (
        <div className="gt-dd-menu">
          {!noClear && (
            <button
              className="gt-dd-opt"
              onClick={() => {
                onPick(null);
                setOpen(false);
              }}
            >
              All {label.toLowerCase()}
            </button>
          )}
          {options.map((o) => (
            <button
              key={o.v}
              className="gt-dd-opt"
              data-on={value === o.l}
              onClick={() => {
                onPick(o.v);
                setOpen(false);
              }}
            >
              {o.l}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
