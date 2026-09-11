import { useState } from "react";
import { Icon } from "../../components/Icon";
import {
  Badge,
  ProgressBar,
  RadialProgress,
} from "../../components/primitives";
import type { Seal, Triumph } from "../../types/design";

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
            <TriumphRow key={i} t={t} />
          ))}
        </ul>
      )}
    </div>
  );
}

/**
 * One triumph row within a SealCard. Triumphs with no `objectives` data
 * render as a flat, non-interactive row (unchanged behavior). Triumphs that
 * carry objective data get a collapsed-by-default, keyboard-accessible
 * disclosure revealing each objective's own label and exact progress.
 * Expansion state is local to each row so opening one triumph never affects
 * another.
 */
function TriumphRow({ t }: { t: Triumph }) {
  const [expanded, setExpanded] = useState(false);
  const objectives = t.objectives;
  const hasObjectives = !!objectives && objectives.length > 0;

  const check = (
    <button
      className="gt-check gt-check--sm"
      data-on={t.done}
      aria-hidden="true"
      tabIndex={-1}
    >
      {t.done && <Icon name="check" size="0.7rem" stroke={3} />}
    </button>
  );

  const label = (
    <span
      className="gt-triumph-label"
      style={{ color: t.done ? "var(--c-text-3)" : "var(--c-text)" }}
    >
      {t.label}
    </span>
  );

  if (!hasObjectives) {
    return (
      <li className="gt-triumph">
        {check}
        {label}
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
    );
  }

  // Multi-objective: the collapsed summary is how many objectives are done
  // out of the total. Single-objective: a "1 of 1" count isn't meaningful,
  // so the summary is that one objective's own exact progress instead.
  const single = objectives.length === 1;
  const completedCount = objectives.filter((o) => o.done).length;
  const summaryValue = single ? objectives[0].cur : completedCount;
  const summaryMax = single ? objectives[0].max : objectives.length;
  const summaryDone = single
    ? objectives[0].done
    : completedCount === objectives.length;

  return (
    <li className="gt-triumph gt-triumph--disclosure" data-expanded={expanded}>
      <div className="gt-triumph-row">
        {check}
        <button
          className="gt-triumph-head"
          onClick={() => setExpanded((e) => !e)}
          aria-expanded={expanded}
        >
          {label}
          <div style={{ width: "6rem", flex: "none" }}>
            <ProgressBar
              value={summaryValue}
              max={summaryMax}
              size="thin"
              complete={summaryDone}
              color={summaryDone ? "var(--c-complete)" : "var(--c-progress)"}
            />
          </div>
          <span className="mono gt-triumph-val">
            {summaryValue}/{summaryMax}
          </span>
          <Icon
            name="chevronDown"
            size="0.85rem"
            style={{
              color: "var(--c-text-3)",
              transform: expanded ? "rotate(180deg)" : "none",
              transition: "transform var(--dur)",
            }}
          />
        </button>
      </div>
      {expanded && (
        <ul className="gt-triumph-objectives">
          {objectives.map((o, i) => (
            <li key={i} className="gt-objective">
              <span
                className="gt-triumph-label"
                style={{
                  color: o.done ? "var(--c-text-3)" : "var(--c-text)",
                }}
              >
                {o.label}
              </span>
              <div style={{ width: "6rem", flex: "none" }}>
                <ProgressBar
                  value={o.cur}
                  max={o.max}
                  size="thin"
                  complete={o.done}
                  color={o.done ? "var(--c-complete)" : "var(--c-progress)"}
                />
              </div>
              <span className="mono gt-triumph-val">
                {o.cur}/{o.max}
              </span>
            </li>
          ))}
        </ul>
      )}
    </li>
  );
}
