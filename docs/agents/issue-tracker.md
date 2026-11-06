# Issue tracker: GitHub

Issues and specs for this repo live as GitHub issues. Use the `gh` CLI for all operations.

Use the host's configured GitHub CLI session. When an agent sandbox on Windows
cannot reach the host Credential Manager, run authenticated `gh` operations in
the approved host context. Never print `gh auth token`, copy it into the
workspace, or place it in an environment variable.

## Conventions

- **Create an issue**: `gh issue create --title "..." --body "..."`. For
  multi-line content, prefer `--body-file <path>`; it works across PowerShell and
  POSIX shells without shell-specific heredoc syntax.
- **Read an issue**: `gh issue view <number> --comments --json number,title,body,labels,comments` and use `--jq` only when a smaller projection is needed.
- **List issues**: `gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'` with appropriate `--label` and `--state` filters.
- **Make an issue a sub-issue of a parent**: `gh issue create --parent <parent> ...`, or `gh issue edit <parent> --add-sub-issue <child>` afterwards (`gh` 2.94+). Older `gh`: `gh api --method POST repos/<owner>/<repo>/issues/<parent>/sub_issues -F sub_issue_id=<child-db-id>` (database id, as in **Blocking** below). Without sub-issues, put `Part of #<parent>` at the top of the child body.
- **Blocking**: GitHub's **native issue dependencies**, the canonical, UI-visible representation. Add an edge with `gh issue create --blocked-by <n>,<n> ...` or `gh issue edit <child> --add-blocked-by <blocker>` (`gh` 2.94+). Older `gh`: `gh api --method POST repos/<owner>/<repo>/issues/<child>/dependencies/blocked_by -F issue_id=<blocker-db-id>`, where `<blocker-db-id>` is the blocker's numeric **database id** (`gh api repos/<owner>/<repo>/issues/<n> --jq .id`, _not_ the `#number` or `node_id`). Where dependencies aren't available, fall back to a `Blocked by: #<n>, #<n>` line at the top of the child body. A ticket is unblocked when every blocker is closed.
- **Count open blockers**: `gh api repos/<owner>/<repo>/issues/<n> --jq .issue_dependencies_summary.blocked_by` (open blockers only; 0 means startable). `gh issue view <n> --json blockedBy` also lists closed blockers, so filter it: `--jq '[.blockedBy.nodes[] | select(.state == "OPEN")] | length'`.
- **Comment on an issue**: `gh issue comment <number> --body "..."`
- **Apply / remove labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: post the explanation with `gh issue comment <number> --body "..."`, then `gh issue close <number>`. On an already-closed issue, `gh issue close --comment` drops the comment and still exits 0.

Infer the repo from `git remote -v`; `gh` does this automatically when run inside a clone.

File issues only into the user's own repo. Never file into `mattpocock/skills` (the repo these skills come from) unless the user owns it. In a fork, `gh` can default to the upstream parent, so before the first `gh issue create` of a session, check the target with `gh repo view --json nameWithOwner` and confirm it with the user if it is not their repo.

## Project board

Task status lives on the private user-level Project `Guardian Tracker`
(project number **4**, owner `jwh3times`), linked to both this repository and the
private companion. See [ADR 0022](../adr/0022-github-owns-task-status.md).

- **List items**: `gh project item-list 4 --owner jwh3times --format json`
- **Read fields**: `gh project field-list 4 --owner jwh3times --format json --jq '.fields[] | "\(.name) | \(.id)"'`
- **Add an issue**: `gh project item-add 4 --owner jwh3times --url <issue-url>`. Every new issue
  goes on the board, then gets `Status`, `Priority` and `Triage` set.
- **Set a field**: `gh project item-edit --id <item-id> --project-id <project-id> --field-id <field-id> --text "..."` (or `--number`, or `--single-select-option-id`)

Fields: `Chain` (single-select), `Order` (number), `Blocked By` (text), `Gate`
(text), `ADR` (text), `Status` (single-select), `Triage` (single-select),
`Priority` (single-select).

**Every board item is an issue.** The board holds no draft items
([ADR 0026](../adr/0026-every-board-item-is-an-issue.md)). New work is filed as an
issue and added to the board, whether or not anyone has committed to it.
Readiness is the triage label. `/to-spec`, `/to-tickets` and `/wayfinder` still
make work `ready-for-agent`
([ADR 0024](../adr/0024-committing-to-work-creates-the-issue.md)). An issue waiting
on something outside itself also carries `blocked`. A blocker that is another
issue is a native `blocked_by` edge. Anything outside the project goes in
`Blocked By`.

**Stale blockers.** Nothing clears `Blocked By` text when its blocker closes, and
an open item still naming a finished blocker drops silently out of the computed
ready frontier. After a merge closes a slice, run `npm run board:blockers` from
the repository root. It lists every open item whose `Blocked By` names a slice
label (a title's `(E3)` suffix) or an issue number that is already Done, or a
slice label no item carries, and exits non-zero when it finds any. Clear each to
`Ready — <blocker> completed <date> (#<n>)`: text starting with `Ready`,
`Unblocked`, or `Shipped` is treated as cleared. It reads the private board
through your `gh` session, so it runs locally; CI tests only its parsing.

**Which repository.** File public by default. Use the private companion
repository only when the body would need a
credential, a real provider/cost/account identifier, or exploitable security
detail.

## Required human follow-ups

When completed agent work leaves any required human action, follow
[Human follow-ups](human-follow-up.md) before reporting completion. These are
private-companion issues on the shared project with `ready-for-human` labels and
linked private-wiki procedures, even if not yet claimed. This exception takes
precedence over the default repository rule above.

## Pull requests as a triage surface

**PRs as a request surface: no.** _(Set to `yes` if this repo treats external PRs as feature requests; `/triage` reads this flag.)_

When set to `yes`, PRs run through the same labels and states as issues, using the `gh pr` equivalents:

- **Read a PR**: `gh pr view <number>`, then `gh pr view <number> --comments` (off a TTY, `--comments` prints only the comments, not the body), and `gh pr diff <number>` for the diff.
- **List external PRs for triage**: `gh api --paginate 'repos/{owner}/{repo}/pulls?state=open' --jq '.[] | select(.author_association | IN("OWNER","MEMBER","COLLABORATOR") | not) | {number, title, author: .user.login, author_association, labels: [.labels[].name]}'`.
- **Comment / label / close**: `gh pr comment`, `gh pr edit --add-label`/`--remove-label`, `gh pr close`.

GitHub shares one number space across issues and PRs, so a bare `#42` may be either: resolve with `gh pr view 42` and fall back to `gh issue view 42`.

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

Read it as in **Read an issue** above.
Run `gh issue view <number> --comments`.

## Wayfinding operations

Used by `/wayfinder`. The **map** is a single issue with **child** issues as tickets.

- **Map**: a single issue labelled `wayfinder:map`, holding the Notes / Decisions-so-far / Fog body. `gh issue create --label wayfinder:map`.
- **Child ticket**: an issue linked to the map as a GitHub sub-issue (see **Make an issue a sub-issue of a parent**). Where sub-issues aren't enabled, add the child to a task list in the map body and put `Part of #<map>` at the top of the child body. Labels: `wayfinder:<type>` (`research`/`prototype`/`grilling`/`task`). Once claimed, the ticket is assigned to the driving dev.
- **Blocking**: see **Blocking** under Conventions.
- **Frontier query**: list the map's open children (`gh issue list --state open`, scoped to the map's sub-issues / task list), drop any with an open blocker (see **Count open blockers**, or an open issue in the `Blocked by` line) or an assignee; first in map order wins.
- **Claim**: `gh issue edit <n> --add-assignee @me`, the session's first write.
- **Resolve**: `gh issue comment <n> --body "<answer>"`, then `gh issue close <n>`, then append a context pointer (gist + link) to the map's Decisions-so-far.

## Public repository

jwh3times/GuardianTracker

## Board

jwh3times 4

## Board fields

- `Status`: `Todo` / `In Progress` / `Done`. There is no blocked option: blocking is carried by
  the `blocked` label, native `blocked_by` edges and `Blocked By`, and assignment marks work in
  flight (ADR 0024).
- `Blocked By` (text): the blocker, or a cleared value starting with `Ready`, `Unblocked` or
  `Shipped` (`Ready — <blocker> completed <date> (#<n>)`). Nothing clears it when a blocker closes:
  after a merge, run `npm run board:blockers` and clear every item it lists. A blocker that is
  another issue is only a native edge; this field holds what blocks an issue from outside the
  project, and is empty when only native edges block it.
- `Gate` (text): what must be true before work can start, `none` when nothing.
- `Chain`: `B — Collections & Wish list`, `C — Acquisition recommendations`,
  `E — Frontend ownership`, `Ops`, `Product`, `Maintenance`; `Order` (number): position within the
  chain.
- `ADR` (text): the controlling ADR number (`0020`), when there is one.
- `Triage`: mirrors the triage label — `ready-for-agent`, `ready-for-human`, `needs-info`,
  `needs-triage`, `wontfix`.
- `Priority`: `P1 — Next`, `P2 — Soon`, `P3 — Later`, `P4 — Parked`. Set on every open item,
  blocked or not; the board's manual order follows it (ADR 0026).

## Security findings

The default. An accepted residual risk is recorded in the companion's `security-limitations.md`
(ADR 0022).
