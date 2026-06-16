import React, { useEffect, useRef, useState } from "react";
import { Icon } from "./Icon";
import { Button } from "./primitives";
import {
  MIN_TIERS,
  ROLES,
  ROLE_LABEL,
  TIER_LABEL,
  roleColor,
  type Role,
  type Tier,
} from "../../lib/roles";
import type { APIAdminFlag, APIResolvedFlag } from "../../types/api";

type CSS = React.CSSProperties & Record<`--${string}`, string | number>;

/* ---------------- ROLE BADGE ---------------- */
export function RoleBadge({ role, lg }: { role: Role; lg?: boolean }) {
  return (
    <span
      className={`gt-badge${lg ? " gt-badge--lg" : ""}`}
      style={{ "--bdg": roleColor(role) } as CSS}
    >
      {role === "admin" ? (
        <Icon name="shield" size="0.85em" />
      ) : (
        <span className="gt-dot" />
      )}
      {ROLE_LABEL[role]}
    </span>
  );
}

/* ---------------- SWITCH ---------------- */
export function Switch({
  on,
  onChange,
  label,
  disabled,
}: {
  on: boolean;
  onChange: (v: boolean) => void;
  label: string;
  disabled?: boolean;
}) {
  return (
    <button
      className="gt-switch"
      data-on={on}
      role="switch"
      aria-checked={on}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!on)}
    >
      <span className="gt-switch-knob" />
    </button>
  );
}

/* ---------------- ROLE SELECT (dropdown) ---------------- */
export function RoleSelect({
  value,
  onPick,
  includeAdmin = true,
  compact,
}: {
  value: Role;
  onPick: (r: Role) => void;
  includeAdmin?: boolean;
  compact?: boolean;
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
  const roles = ROLES.filter((r) => includeAdmin || r !== "admin");
  return (
    <div className="gt-dd" ref={ref}>
      <button
        className={`gt-roleselect${compact ? " gt-roleselect--sm" : ""}`}
        onClick={() => setOpen((v) => !v)}
        style={{ "--bdg": roleColor(value) } as CSS}
      >
        {value === "admin" ? (
          <Icon name="shield" size="0.85em" />
        ) : (
          <span className="gt-dot" />
        )}
        {ROLE_LABEL[value]}
        <Icon name="chevronDown" size="0.75rem" />
      </button>
      {open && (
        <div
          className="gt-dd-menu"
          style={{ right: 0, left: "auto", minWidth: "9rem" }}
        >
          {roles.map((r) => (
            <button
              key={r}
              className="gt-dd-opt"
              data-on={value === r}
              onClick={() => {
                onPick(r);
                setOpen(false);
              }}
            >
              <span className="gt-dot" style={{ background: roleColor(r) }} />{" "}
              {ROLE_LABEL[r]}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

/* ---------------- TIER SEGMENT (flag gate) ---------------- */
export function TierSegment({
  value,
  onChange,
  disabled,
}: {
  value: Tier;
  onChange: (t: Tier) => void;
  disabled?: boolean;
}) {
  return (
    <div
      className="gt-tierseg"
      role="radiogroup"
      aria-label="Minimum access tier"
    >
      {MIN_TIERS.map((tier) => (
        <button
          key={tier}
          className="gt-tierseg-opt"
          role="radio"
          aria-checked={value === tier}
          data-on={value === tier}
          disabled={disabled}
          style={{ "--bdg": roleColor(tier) } as CSS}
          onClick={() => onChange(tier)}
        >
          {TIER_LABEL[tier]}
        </button>
      ))}
    </div>
  );
}

/* ---------------- FEATURE FLAG CARD ---------------- */
export function FlagCard({
  flag,
  reached,
  totalUsers,
  onToggle,
  onSetMinTier,
  pending,
}: {
  flag: APIAdminFlag;
  reached: number;
  totalUsers: number;
  onToggle: (enabled: boolean) => void;
  onSetMinTier: (tier: Tier) => void;
  pending?: boolean;
}) {
  const pct = totalUsers > 0 ? Math.round((reached / totalUsers) * 100) : 0;
  return (
    <div className="gt-flagcard" data-off={!flag.enabled}>
      <div className="gt-flagcard-head">
        <div className="gt-flagcard-title">
          <span className="gt-flag-name">{flag.name}</span>
          <span
            className="gt-badge"
            style={{ "--bdg": "var(--c-text-3)" } as CSS}
          >
            {flag.category}
          </span>
        </div>
        <Switch
          on={flag.enabled}
          onChange={onToggle}
          label={`Enable ${flag.name}`}
          disabled={pending}
        />
      </div>
      <p className="gt-flag-desc">{flag.description}</p>
      <div className="gt-flagcard-foot">
        <div className="gt-flag-gate">
          <span className="gt-flag-gate-label">Available to</span>
          <TierSegment
            value={flag.minTier as Tier}
            onChange={onSetMinTier}
            disabled={pending}
          />
        </div>
        <div className="gt-flag-reach">
          {flag.enabled ? (
            <>
              <span className="mono gt-flag-reach-num">
                {reached}/{totalUsers}
              </span>
              <span className="gt-flag-reach-lbl">users · {pct}%</span>
            </>
          ) : (
            <span
              className="gt-flag-reach-lbl"
              style={{ color: "var(--c-text-4)" }}
            >
              Disabled — hidden for everyone
            </span>
          )}
        </div>
      </div>
      {flag.enabled && (
        <div className="gt-flag-reachbar">
          <div
            className="gt-flag-reachbar-fill"
            style={
              { "--val": pct + "%", "--pc": roleColor(flag.minTier) } as CSS
            }
          />
        </div>
      )}
    </div>
  );
}

/* ---------------- USERS TABLE ROW ---------------- */
export function UserRow({
  name,
  handle,
  role,
  platform,
  active,
  me,
  onPick,
}: {
  name: string;
  handle: string;
  role: Role;
  platform: string;
  active: string;
  me?: boolean;
  onPick: (r: Role) => void;
}) {
  return (
    <div className="gt-userrow" data-me={me}>
      <span className="gt-avatar" data-cls={me ? "Warlock" : "Hunter"}>
        {(name || "?")[0]}
      </span>
      <div className="gt-userrow-id">
        <div className="gt-userrow-name">
          {name}
          {me && <span className="gt-you">You</span>}
        </div>
        <div className="gt-userrow-handle mono">{handle}</div>
      </div>
      <div className="gt-userrow-meta">
        <span className="gt-userrow-platform">{platform}</span>
        <span className="gt-userrow-active mono">{active}</span>
      </div>
      <RoleSelect value={role} onPick={onPick} />
    </div>
  );
}

/* ---------------- LOCKED FEATURE (upsell screen) ---------------- */
export function LockedFeature({
  flag,
  onChangeTier,
  onBack,
}: {
  flag: APIResolvedFlag;
  onChangeTier: () => void;
  onBack: () => void;
}) {
  const need = flag.minTier as Tier;
  return (
    <div className="gt-page">
      <div className="gt-card gt-locked">
        <div
          className="gt-locked-mark"
          style={{ "--bdg": roleColor(need) } as CSS}
        >
          <Icon name="lock" size="2.4rem" stroke={1.5} />
        </div>
        <div className="gt-locked-tier">
          <RoleBadge role={need} lg /> early access
        </div>
        <h2 className="gt-locked-title">{flag.name}</h2>
        <p className="gt-locked-body">{flag.desc}</p>
        <p className="gt-locked-note">
          This feature is rolling out to <strong>{TIER_LABEL[need]}</strong>{" "}
          members. Switch your access tier to unlock it — or ask an admin to
          widen the flag.
        </p>
        <div className="gt-locked-actions">
          <Button variant="primary" icon="bolt" onClick={onChangeTier}>
            Change access tier
          </Button>
          <Button variant="ghost" onClick={onBack}>
            Back to Dashboard
          </Button>
        </div>
      </div>
    </div>
  );
}
