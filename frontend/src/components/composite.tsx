import React, { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { Icon } from "./Icon";
import {
  Badge,
  Button,
  CountdownChip,
  EmptyState,
  ItemTile,
  ProgressBar,
  RadialProgress,
} from "./primitives";
import { DIFF_LABEL } from "../lib/constants";
import type {
  GTItem,
  ItemCatalyst,
  Milestone,
  PerkColumn,
  RecommendedAction,
  Seal,
  TreeNode,
  Xur,
} from "../types/design";

type CSS = React.CSSProperties & Record<`--${string}`, string | number>;

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

/* ---------------- CATEGORY TREE ---------------- */
export function CategoryTree({
  nodes,
  activeId,
  onSelect,
  expand,
}: {
  nodes: TreeNode[];
  activeId: string;
  onSelect: (id: string) => void;
  expand?: string[];
}) {
  const [open, setOpen] = useState<Set<string>>(() => new Set());
  const toggle = (id: string) =>
    setOpen((s) => {
      const n = new Set(s);
      if (n.has(id)) n.delete(id);
      else n.add(id);
      return n;
    });

  // Controlled-seed reveal: when a deep-link resolves an ancestor path, merge
  // those ids into the open set so the selected node is visible. Existing
  // manually-opened ids are preserved. The `expand` prop is the external system
  // being synchronized into local open state here, so the in-effect setState is
  // intentional.
  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    if (!expand || expand.length === 0) return;
    setOpen((s) => {
      const n = new Set(s);
      for (const id of expand) n.add(id);
      return n;
    });
  }, [expand]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const renderNode = (node: TreeNode, depth: number): ReactNode => {
    const hasChildren = !!node.children && node.children.length > 0;
    const isOpen = open.has(node.id);
    return (
      <div
        key={node.id}
        className="gt-tree-group"
        role="treeitem"
        aria-expanded={hasChildren ? isOpen : undefined}
      >
        <div
          className="gt-tree-row gt-tree-row--parent"
          data-active={activeId === node.id}
          style={{ paddingLeft: `calc(${depth} * var(--s-3))` }}
        >
          {hasChildren ? (
            <button
              className="gt-tree-caret"
              onClick={() => toggle(node.id)}
              aria-label={`${isOpen ? "Collapse" : "Expand"} ${node.label}`}
              data-open={isOpen}
            >
              <Icon name="chevron" size="0.8rem" />
            </button>
          ) : (
            <span
              className="gt-tree-caret gt-tree-caret--leaf"
              aria-hidden="true"
            />
          )}
          <button className="gt-tree-main" onClick={() => onSelect(node.id)}>
            <span className="gt-tree-label">{node.label}</span>
            <span className="gt-tree-pct mono">
              {node.count[0]}/{node.count[1]}
            </span>
          </button>
        </div>
        <div
          className="gt-tree-bar"
          style={{ paddingLeft: `calc(${depth} * var(--s-3))` }}
        >
          <div
            className="gt-tree-bar-fill"
            style={{ "--val": node.pct + "%" } as CSS}
          />
        </div>
        {hasChildren && isOpen && (
          <div className="gt-tree-children" role="group">
            {node.children!.map((c) => renderNode(c, depth + 1))}
          </div>
        )}
      </div>
    );
  };

  return (
    <nav className="gt-tree" role="tree">
      {nodes.map((n) => renderNode(n, 0))}
    </nav>
  );
}

/* ---------------- ACTION LIST ---------------- */
export function ActionList({
  items,
  onToggle,
}: {
  items: RecommendedAction[];
  onToggle: (id: string) => void;
}) {
  return (
    <ul className="gt-actions">
      {items.map((a) => (
        <li key={a.id} className="gt-action" data-done={a.done}>
          <button
            className="gt-check"
            data-on={a.done}
            onClick={() => onToggle(a.id)}
            aria-label="Mark done"
          >
            {a.done && <Icon name="check" size="0.85rem" stroke={3} />}
          </button>
          <div className="gt-action-body">
            <div className="gt-action-top">
              <span className="gt-action-text">{a.text}</span>
              {a.badge && <Badge kind={a.badge} dot />}
            </div>
            <div className="gt-action-meta mono">
              {a.detail} ·{" "}
              <span style={{ color: "var(--c-text-2)" }}>{a.time}</span>
            </div>
          </div>
          <Badge kind={a.diff} style={{ flex: "none" }}>
            {DIFF_LABEL[a.diff]}
          </Badge>
        </li>
      ))}
    </ul>
  );
}

/* ---------------- XÛR MODULE ---------------- */
export function XurModule({
  xur,
  onSelect,
}: {
  xur: Xur | null | undefined;
  onSelect?: (hash: string) => void;
}) {
  if (!xur || !xur.present) {
    return (
      <Panel title="Xûr — Agent of Nine" icon="bungie">
        <EmptyState
          icon="bungie"
          title="Xûr returns Friday"
          body="The weekend exotic vendor is away. Check back in 3d 2h."
        />
      </Panel>
    );
  }
  return (
    <Panel
      title="Xûr — weekend exotics"
      icon="bungie"
      accent="var(--c-exotic)"
      right={
        <CountdownChip prefix="Leaves" time={xur.leavesIn} soon icon="clock" />
      }
    >
      <div className="gt-xur-loc mono">{xur.location}</div>
      <ul className="gt-vendor-list">
        {xur.items.map((it) => (
          <li
            key={it.hash}
            className="gt-vendor-item"
            data-rarity={it.rarity}
            role={onSelect ? "button" : undefined}
            tabIndex={onSelect ? 0 : undefined}
            style={onSelect ? { cursor: "pointer" } : undefined}
            onClick={() => onSelect?.(it.hash)}
            onKeyDown={(e) => {
              if (onSelect && (e.key === "Enter" || e.key === " ")) {
                e.preventDefault();
                onSelect(it.hash);
              }
            }}
          >
            <ItemTile
              rarity={it.rarity}
              type={it.type}
              style={{ width: "2.4rem" }}
            />
            <div className="gt-vendor-main">
              <div className="gt-item-name">{it.name}</div>
              <div className="gt-item-type">
                {it.type} · <span className="mono">{it.cost}</span>
              </div>
            </div>
            {it.missing ? (
              <Badge kind="missing" dot icon="bolt" />
            ) : (
              <Badge kind="owned" dot>
                Owned
              </Badge>
            )}
          </li>
        ))}
      </ul>
    </Panel>
  );
}

/* ---------------- MILESTONE MODULE ---------------- */
export function MilestoneModule({ milestones }: { milestones: Milestone[] }) {
  return (
    <Panel title="Milestones & Activities" icon="week">
      <ul className="gt-vendor-list">
        {milestones.map((m) => (
          <li key={m.id} className="gt-milestone">
            <div className="gt-milestone-l">
              <div className="gt-action-meta mono">{m.label}</div>
              <div className="gt-item-name">{m.name}</div>
              <div className="gt-item-type">Reward: {m.reward}</div>
            </div>
            {m.missing != null && m.missing > 0 && (
              <Badge kind="missing" dot>
                {m.missing} missing
              </Badge>
            )}
          </li>
        ))}
      </ul>
    </Panel>
  );
}

/* ---------------- SEAL CARD ---------------- */
export function SealCard({
  seal,
  expanded,
  onToggle,
}: {
  seal: Seal;
  expanded: boolean;
  onToggle: () => void;
}) {
  const isGild = seal.gilded > 0;
  const color =
    seal.pct >= 100
      ? "var(--c-gild)"
      : seal.pct >= 75
        ? "var(--c-complete)"
        : "var(--c-progress)";
  return (
    <div className="gt-seal" data-expanded={expanded}>
      <button className="gt-seal-head" onClick={onToggle}>
        <RadialProgress
          value={seal.pct}
          size="4.2rem"
          color={color}
          pctSize="1.15rem"
          label={isGild ? "✦" : `${seal.pct}%`}
          sub={isGild ? "gilded" : null}
        />
        <div className="gt-seal-main">
          <div className="gt-seal-name">{seal.name}</div>
          <div className="gt-seal-meta">
            {isGild ? (
              <Badge kind="gilded" dot icon="star">
                Gilded ×{seal.gilded}
              </Badge>
            ) : (
              <span className="gt-action-meta">{seal.left}</span>
            )}
          </div>
        </div>
        <Icon
          name="chevronDown"
          size="1.1rem"
          style={{
            color: "var(--c-text-3)",
            transform: expanded ? "rotate(180deg)" : "none",
            transition: "transform var(--dur)",
          }}
        />
      </button>
      {expanded && (
        <ul className="gt-seal-triumphs">
          {seal.triumphs.map((t, i) => (
            <li key={i} className="gt-triumph">
              <button
                className="gt-check gt-check--sm"
                data-on={t.done}
                aria-hidden="true"
                tabIndex={-1}
              >
                {t.done && <Icon name="check" size="0.7rem" stroke={3} />}
              </button>
              <span
                className="gt-triumph-label"
                style={{ color: t.done ? "var(--c-text-3)" : "var(--c-text)" }}
              >
                {t.label}
              </span>
              <div style={{ width: "6rem", flex: "none" }}>
                <ProgressBar
                  value={t.cur}
                  max={t.max}
                  size="thin"
                  complete={t.done}
                  color={t.done ? "var(--c-complete)" : "var(--c-progress)"}
                />
              </div>
              <span className="mono gt-triumph-val">
                {t.cur}/{t.max}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

/* ---------------- ITEM DETAIL DRAWER ---------------- */
export function ItemDetailDrawer({
  item,
  onClose,
  onWish,
  wished,
  perkColumns,
  perksLoading,
  catalysts,
}: {
  item: GTItem | null;
  onClose: () => void;
  onWish: (item: GTItem) => void;
  wished: boolean;
  perkColumns?: PerkColumn[];
  perksLoading?: boolean;
  catalysts?: ItemCatalyst[];
}) {
  const [showWhy, setShowWhy] = useState(false);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);
  if (!item) return null;
  const viewOnly = !!item.viewOnly;
  return (
    <div className="gt-drawer-scrim" onClick={onClose}>
      <aside
        className="gt-drawer"
        data-rarity={item.rarity}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-label={item.name}
      >
        <button
          className="gt-drawer-close gt-iconbtn"
          onClick={onClose}
          aria-label="Close"
        >
          <Icon name="close" size="1.1rem" />
        </button>
        <div className="gt-drawer-hero">
          <ItemTile
            rarity={item.rarity}
            type={item.type}
            icon={item.icon}
            style={{ width: "4.6rem" }}
          />
          <div>
            <div
              className="gt-item-badges"
              style={{ marginBottom: "var(--s-1)" } as CSS}
            >
              <Badge kind={item.rarity} dot lg />
            </div>
            <h2 className="gt-drawer-name">{item.name}</h2>
            <div className="gt-item-type">
              {item.type}
              {item.slot ? ` · ${item.slot}` : ""}
            </div>
          </div>
        </div>

        {viewOnly && (
          <p className="gt-drawer-desc" style={{ color: "var(--c-text-3)" }}>
            Not in your trackable collections — view only.
          </p>
        )}

        {!viewOnly && item.obtainable && !item.collected && (
          <div className="gt-drawer-avail">
            <Badge kind="avail-now" dot icon="bolt" lg />
            <span>
              {item.availFrom
                ? `Available now — ${item.availFrom}`
                : "Obtainable right now — see source below."}
            </span>
          </div>
        )}

        <p className="gt-drawer-desc">{item.desc}</p>

        {!viewOnly && (
          <div className="gt-drawer-block">
            <div className="gt-section-title">
              Acquisition{" "}
              <Badge kind={item.diff}>{DIFF_LABEL[item.diff]}</Badge>
              {item.farmOnly && <Badge kind="farmonly">Farm only</Badge>}
              <button className="gt-why" onClick={() => setShowWhy((v) => !v)}>
                why?{" "}
                <Icon
                  name="chevronDown"
                  size="0.7rem"
                  style={{ transform: showWhy ? "rotate(180deg)" : "none" }}
                />
              </button>
            </div>
            <div className="gt-drawer-src">
              <Icon
                name="bolt"
                size="0.9rem"
                style={{ color: "var(--rarity)" }}
              />
              <div>
                <div style={{ color: "var(--c-text)" }}>{item.source}</div>
                <div className="gt-item-type">{item.sourceDetail}</div>
              </div>
            </div>
            {item.farmOnly && (
              <p className="gt-why-text">
                Random perks — earn a fresh drop from its source; this item
                can't be pulled from Collections.
              </p>
            )}
            {showWhy &&
              (item.diff === "unrated" ? (
                <p className="gt-why-text">
                  We couldn't determine difficulty from this item's source.
                </p>
              ) : (
                <p className="gt-why-text">
                  Difficulty is <strong>estimated</strong> from this item's
                  source — drop sources like raids and Grandmasters score higher
                  than vendor or world drops. Treat it as a guide, not a
                  guarantee.
                </p>
              ))}
          </div>
        )}

        {(perksLoading || (perkColumns && perkColumns.length > 0)) && (
          <div className="gt-drawer-block">
            <div className="gt-section-title">Possible perks / rolls</div>
            {perksLoading ? (
              <div className="gt-perks-loading">Loading perks…</div>
            ) : (
              <div className="gt-perk-cols">
                {perkColumns!.map((col) => (
                  <div key={col.label} className="gt-perk-col">
                    <div className="gt-perk-col-label">{col.label}</div>
                    <div className="gt-perks">
                      {col.perks.map((p) => (
                        <span key={p} className="gt-perk">
                          {p}
                        </span>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {catalysts && catalysts.length > 0 && (
          <div className="gt-drawer-block">
            <div className="gt-section-title">Catalyst</div>
            <ul className="gt-catalyst-list">
              {catalysts.map((c) => (
                <li key={c.name} className="gt-catalyst">
                  <div className="gt-catalyst-name">{c.name}</div>
                  {c.description && (
                    <p className="gt-catalyst-desc">{c.description}</p>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}

        <div className="gt-drawer-actions">
          {!viewOnly && (
            <Button
              variant="primary"
              icon="wishlist"
              onClick={() => onWish(item)}
              style={{ flex: 1 }}
            >
              {wished ? "On wishlist" : "Add to Wishlist"}
            </Button>
          )}
          <Button
            variant="outline"
            icon="external"
            onClick={() =>
              window.open(
                `https://www.light.gg/db/items/${item.id}`,
                "_blank",
                "noopener,noreferrer",
              )
            }
          >
            View on light.gg ↗
          </Button>
        </div>
      </aside>
    </div>
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
}: {
  label: string;
  value?: string | null;
  options: DropdownOption[];
  onPick: (v: string | null) => void;
  note?: string;
  noClear?: boolean;
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
