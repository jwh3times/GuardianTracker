# ADR 0022: GitHub Issues and Projects Own Task Status

- Status: Accepted — implemented in `v1.3.6`
- Date: 2026-09-04
- Supersedes in part: [ADR 0001](./0001-public-private-documentation-boundary.md); see
  [Supersession](#supersession)

## Context

[ADR 0001](./0001-public-private-documentation-boundary.md) put detailed
implementation handoffs under the gitignored `private/` directory. That worked
while a handoff was a single narrative document, but the architecture-deepening
program turned it into a status cache: `private/IMPLEMENTATION_PLAN.md` tracked
twenty-one remaining slices, their dependency order, their acceptance
constraints, and — implicitly — which ones had shipped.

A file cannot hold that last part without being rewritten after every merge. The
evidence is in the repository's own history:

- `IMPLEMENTATION_PLAN.md` carried a `Reviewed:` date and a `Verified baseline:`
  commit, both of which had to be hand-updated to stay true.
- A dated next-priority assessment ranked five work items; four of them shipped
  within two days, leaving a document that still read as current guidance. It has
  since been deleted — every item in it was shipped, resolved, or owned by
  another record.
- The plan's own `E4`–`E14` row listed twelve items for eleven slots, an
  ambiguity that survived because no mechanism forced it to reconcile against
  [ADR 0020](./0020-own-frontend-data-access.md)'s eleven named resources.

The plan already conceded the split — "GitHub Issues are the task tracker; this
file preserves only dependency order and acceptance constraints." Dependency
order is now expressible natively: GitHub issue dependencies (`blocked_by`) and
sub-issues are both available on this repository. That removes the file's last
unique job.

The countervailing force is that this repository is agent-driven, and
`AGENTS.md` states that tool-neutral agents cannot follow links out of that
file — which is why ports, the environment table, and test commands are
duplicated inline. Operating context must therefore stay in the working tree.

## Decision

**Task status lives on GitHub. Contracts and operating context stay in the
working tree.**

The dividing line is whether the content changes when a pull request merges:

| Content                                               | Owner                             |
| ----------------------------------------------------- | --------------------------------- |
| What remains, who has it, what blocks it              | GitHub Issues + Project           |
| Per-slice scope, boundary, and acceptance constraints | The issue or draft body           |
| Interface and migration contracts                     | `docs/adr/`                       |
| Domain vocabulary and invariants                      | `CONTEXT.md`                      |
| How to run, test, and validate                        | `AGENTS.md`, `SETUP.md`           |
| Released history                                      | `CHANGELOG.md`                    |
| Accepted and unresolved risk                          | `private/security-limitations.md` |
| Undecided production topology                         | `private/InfraTODO.md`            |

One user-level **Project** (`Guardian Tracker`, private) is the single planning
surface and is linked to both the public repository and the private companion.
Its fields are `Chain`, `Order`, `Blocked By`, `Gate`, `ADR`, and `Status`.

**Issues are filed in the public repository by default.** The private companion
receives an issue only when the body would need a credential, a real
provider/cost/account identifier, or exploitable security detail. Splitting the
tracker by default was rejected: it reintroduces two places to look, and a
private issue cannot be read from the public reference that points at it.

**Unclaimed slices are project draft items, not issues.** Closed issue
[#172](https://github.com/jwh3times/GuardianTracker/issues/172) established that
an issue is created when a slice is claimed, so that the open-issue list reads
as live status. Drafts preserve that convention while still putting the whole
backlog on GitHub: the board shows twenty-one items, the public issue list shows
only what is actually in flight, and a draft converts to a real issue in one
action.

Because a draft item has no repository or number, native dependencies cannot
attach to it. Draft ordering is carried by the `Blocked By` field, matching the
fallback already documented in
[`docs/agents/issue-tracker.md`](../agents/issue-tracker.md); native `blocked_by`
edges are wired at conversion time.

Point-in-time findings are recorded as comments on the issue they informed, not
as standing documents. Rerunnable verification evidence — the real-Manifest
verifier and its dated output — is exempt, because [ADR
0003](./0003-manifest-derived-data-and-verify-first-changes.md) requires it to
survive independently of any one task.

## Supersession

This record supersedes exactly one statement in ADR 0001: that _detailed
implementation handoffs_ belong under `private/`. Handoffs are now GitHub
project items.

Every other part of ADR 0001 stands unchanged — private operating notes,
deployment runbooks, private security review detail, cloud resource names, and
raw API research remain under `private/`, and public docs continue to describe
implemented behavior, local setup, durable decisions, security model, and gated
future work.

## Consequences

- The repository stops maintaining a hand-updated status cache, and the class of
  drift that produced a stale priority ranking and an unreconciled slice count
  is removed at the source.
- Dependency order becomes queryable rather than prose. The ready frontier is
  computed — items with no open blocker — instead of asserted.
- `private/IMPLEMENTATION_PLAN.md` is retired. Its durable content moved: the
  per-slice scope and acceptance constraints into the twenty-one project items,
  and its "Rules for every slice" into `AGENTS.md`, where tool-neutral agents can
  read them without leaving the file.
- Agents need an authenticated `gh` session to see task status. Operating
  context, which they must have offline, is unaffected.
- `ROADMAP.md` keeps its gated product items. They are public-facing product
  statements with human readers, they change on a scale of months rather than
  merges, and they are the one document an outside contributor reads to
  understand direction.
- The private companion repository retains issues as a capability but is
  expected to stay near-empty.
