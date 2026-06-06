import React from "react";

export function Brand({ compact }: { compact?: boolean }) {
  return (
    <div className="gt-brand" data-compact={compact}>
      <svg
        viewBox="0 0 32 32"
        width="1.7rem"
        height="1.7rem"
        aria-hidden="true"
        className="gt-brand-mark"
      >
        <polygon
          points="16,2 28,9 28,23 16,30 4,23 4,9"
          fill="none"
          stroke="var(--c-signal)"
          strokeWidth="1.6"
        />
        <polygon
          points="16,9 22,12.5 22,19.5 16,23 10,19.5 10,12.5"
          fill="var(--c-signal-dim)"
          stroke="var(--c-signal)"
          strokeWidth="1"
        />
      </svg>
      {!compact && (
        <div className="gt-brand-text">
          <span className="gt-brand-1">GUARDIAN</span>
          <span className="gt-brand-2">TRACKER</span>
        </div>
      )}
    </div>
  );
}
