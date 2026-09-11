# Historical Design Prototype

This directory contains the self-contained prototype that informed Guardian
Tracker's current visual language. It is retained as design history, not as an
implementation specification or a second frontend.

The production application lives in [`frontend/src/`](../src/). Treat its
components, styles, tests, and [frontend guide](../README.md) as current. Do not
copy prototype behavior or component inventories into production without first
verifying them against the application and current product direction.

Contents:

- `Guardian Tracker.html` and `Design System.html` — rendered prototype pages.
- `src/`, `tokens.css`, and `tweaks-panel.jsx` — prototype source and styling.
- `scratch/` — retained visual exploration images.

## Where the shipped tokens intentionally diverge

`tokens.css` here records the prototype as it was authored and is deliberately
not kept in step with `frontend/src/styles/tokens.css`. Known divergences:

- `--c-challenging` (and `--c-challenging-dim`) are `oklch(0.67 0.180 32)` in the
  prototype. The application raised them to `oklch(0.72 0.17 32)` in v1.3.35,
  because at the prototype's lightness the difficulty badge scored 4.37:1 against
  `--c-surface-2` and failed WCAG AA. The shipped value is the correct one for
  production; the prototype keeps its original as design history.

Treat a mismatch between the two files as expected, not as drift to repair.

The obsolete wireframe handoff documents were removed because they mixed
implemented screens, planned work, and outdated component structure. Durable
product principles now live in [`docs/product.md`](../../docs/product.md).
