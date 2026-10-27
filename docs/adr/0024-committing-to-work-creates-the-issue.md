# ADR 0024: Committing to Work Creates the Issue

- Status: Accepted
- Date: 2026-10-26
- Supersedes in part: [ADR 0022](./0022-github-owns-task-status.md); see
  [Supersession](#supersession)

## Context

[ADR 0022](./0022-github-owns-task-status.md) kept unclaimed slices as project
draft items and created an issue only when a slice was claimed, so that the
open-issue list read as live status.

The repository has since made its skill flow binding
([`docs/agents/dev-flow.md`](../agents/dev-flow.md)). That flow's gate requires
every code change to start from a ready issue: a `ready-for-agent` label, a body
that is a spec, ticket, or triage brief, and a number for the branch name and the
PR's `Closes #N` or `Part of #N` line. A draft item carries none of these.

The flow's planning skills publish issues, not drafts:

- `/to-spec` publishes the spec as an issue labelled `ready-for-agent`.
- `/to-tickets` publishes one issue per ticket and wires native blocking links
  between them. Those links are its main output, and a draft cannot hold one.
- `/wayfinder` already creates its map and every decision ticket as issues, an
  exception ADR 0022 never recorded.

Overriding the skills to publish drafts would cost the flow its native blocking
links and still leave every spec needing a number. Dropping the draft rule
entirely would turn the open-issue list into the whole backlog.

## Decision

**An issue is created when work is committed to. Running `/to-spec`,
`/to-tickets`, or `/wayfinder` is that commitment.**

- Work nobody has committed to — an idea, or a roadmap slice nobody has grilled
  or specified yet — stays a project draft item, as ADR 0022 decided.
- The issues those skills publish are added to the project board. When the work
  started as a draft, the draft is converted into, or replaced by, the issue that
  now owns it, so each piece of work has one item.
- Claiming a draft directly still converts it to an issue, as before.
- Issues someone else files, and `bug` issues filed to drive `/diagnosing-bugs`,
  are outside this rule; they already exist as issues when the flow meets them.
- Assignment, not existence, marks work in flight. The ready frontier stays open
  items with no open blocker and no assignee.

## Supersession

This record supersedes exactly one statement in ADR 0022: that _unclaimed slices
are project draft items, not issues_. Work that the skill flow has committed to
is an issue even before anyone claims it.

Every other part of ADR 0022 stands unchanged — GitHub owns task status, issues
are public by default, the board's fields carry draft ordering, and native
`blocked_by` edges are wired when a draft converts.

## Consequences

- The skills run unchanged, and tickets get native blocking links from the moment
  they are published.
- The open-issue list reads as committed work: in-flight issues plus the
  not-yet-started tickets of specs the flow has committed to. Filtering by
  assignee or the board's `Status` field shows what is in flight right now.
- The draft-only stale-blocker problem, which `npm run board:blockers` detects,
  shrinks to work that is still a draft.
