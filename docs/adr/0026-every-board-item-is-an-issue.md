# ADR 0026: Every Board Item Is an Issue

- Status: Accepted
- Date: 2026-11-06
- Supersedes in part: [ADR 0024](./0024-committing-to-work-creates-the-issue.md)
  and [ADR 0022](./0022-github-owns-task-status.md); see
  [Supersession](#supersession)

## Context

[ADR 0022](./0022-github-owns-task-status.md) kept unclaimed work as project
draft items, and [ADR 0024](./0024-committing-to-work-creates-the-issue.md)
narrowed that to work nobody had committed to. The aim was that the open-issue
list read as committed work.

A full board audit on 2026-11-06 found nine open items. Six of them were drafts:
three waited on an owner decision, two on upstream libraries, and one had become
actionable again when new upstream releases shipped. A draft carries no labels,
no number and no native dependency, so the board could not say which work was
blocked, which was available, or in what order to take it. The owner asked for
explicit `blocked` labels and a priority order across all open work.

At this repository's scale, with one maintainer and a backlog of about ten
items, keeping the open-issue list down to committed work costs more than it
saves. Labels and a priority field already say what is ready, and they work only
on issues.

## Decision

**Every open board item is an issue.** New work, whether an idea, a roadmap
pointer or a follow-up, is filed as an issue (in the public repository by
default, as before) and added to the board. The board holds no draft items.

- **Readiness is a label, not an item type.** Each open issue carries one triage
  label (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`,
  `wontfix`). The development-flow gate is unchanged: code still starts only from
  a `ready-for-agent` issue whose body is a spec, a ticket or a triage brief.
  `/to-spec`, `/to-tickets` and `/wayfinder` remain the commitment that makes
  work ready. What changes is when the issue exists, not when it is ready.
- **`blocked` sits beside the triage label.** It marks an issue that waits on
  something outside itself and cannot progress until that clears. A blocked
  issue keeps its triage label, so its shape is known once it is unblocked.
  Remove `blocked` when the blocker clears. A blocker that is another issue is
  a native `blocked_by` edge; anything outside the project goes in the board's
  `Blocked By` field.
- **Priority is a board field.** `Priority` is a single-select field with four
  values: `P1 — Next`, `P2 — Soon`, `P3 — Later` and `P4 — Parked`. Every open
  item gets one, whether or not it is blocked, and the board's manual order
  follows it.
- **The ready frontier** for an agent is the open issues labelled
  `ready-for-agent`, without `blocked`, with no open native blocker and no
  assignee, taken in `Priority` order.

## Supersession

This record supersedes ADR 0024's statement that _work nobody has committed to
stays a project draft item_, and the parts of ADR 0022 about draft items:
_unclaimed slices are project draft items, not issues_, and draft ordering
carried in the `Blocked By` field. Every other decision in both records stands,
including GitHub owning task status, issues being public by default, and
assignment marking work in flight.

## Consequences

- The open-issue list is the whole backlog. Filter it by label, or read the
  board by `Priority`, to see what is available now.
- Blocked state is visible on the issue itself, not only in board text.
- `npm run board:blockers` keeps working for the `Blocked By` text on issues,
  which now holds only blockers outside the project.
- The six open drafts were converted on 2026-11-06 (#464–#469). Draft-specific
  instructions are removed from the tracker settings.
