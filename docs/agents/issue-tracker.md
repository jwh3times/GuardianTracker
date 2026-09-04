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
- **Comment on an issue**: `gh issue comment <number> --body "..."`
- **Apply / remove labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`

Infer the repo from `git remote -v` — `gh` does this automatically when run inside a clone.

## Project board

Task status lives on the private user-level Project `Guardian Tracker`
(project number **4**, owner `jwh3times`), linked to both this repository and the
private companion. See [ADR 0022](../adr/0022-github-owns-task-status.md).

- **List items**: `gh project item-list 4 --owner jwh3times --format json`
- **Read fields**: `gh project field-list 4 --owner jwh3times --format json --jq '.fields[] | "\(.name) | \(.id)"'`
- **Add a draft**: `gh project item-create 4 --owner jwh3times --title "..." --body-file <path>`
- **Add an existing issue**: `gh project item-add 4 --owner jwh3times --url <issue-url>`
- **Set a field**: `gh project item-edit --id <item-id> --project-id <project-id> --field-id <field-id> --text "..."` (or `--number`, or `--single-select-option-id`)

Fields: `Chain` (single-select), `Order` (number), `Blocked By` (text), `Gate`
(text), `ADR` (text), `Status` (single-select).

**Drafts versus issues.** Unclaimed work is a draft item; an issue is created
when the work is claimed, which keeps the open-issue list readable as live
status. A draft has no repository or number, so **native dependencies cannot
attach to it** — carry the order in the `Blocked By` field and wire real
`blocked_by` edges after converting the draft to an issue.

**Which repository.** File public by default. Use the private companion
(`jwh3times/GuardianTracker-private`) only when the body would need a
credential, a real provider/cost/account identifier, or exploitable security
detail.

## Pull requests as a triage surface

**PRs as a request surface: no.** _(Set to `yes` if this repo treats external PRs as feature requests; `/triage` reads this flag.)_

When set to `yes`, PRs run through the same labels and states as issues, using the `gh pr` equivalents:

- **Read a PR**: `gh pr view <number> --comments` and `gh pr diff <number>` for the diff.
- **List external PRs for triage**: `gh pr list --state open --json number,title,body,labels,author,authorAssociation,comments` then keep only `authorAssociation` of `CONTRIBUTOR`, `FIRST_TIME_CONTRIBUTOR`, or `NONE` (drop `OWNER`/`MEMBER`/`COLLABORATOR`).
- **Comment / label / close**: `gh pr comment`, `gh pr edit --add-label`/`--remove-label`, `gh pr close`.

GitHub shares one number space across issues and PRs, so a bare `#42` may be either — resolve with `gh pr view 42` and fall back to `gh issue view 42`.

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --comments`.

## Wayfinding operations

Used by `/wayfinder`. The **map** is a single issue with **child** issues as tickets.

- **Map**: a single issue labelled `wayfinder:map`, holding the Notes / Decisions-so-far / Fog body. `gh issue create --label wayfinder:map`.
- **Child ticket**: an issue linked to the map as a GitHub sub-issue (`gh api` on the sub-issues endpoint). Where sub-issues aren't enabled, add the child to a task list in the map body and put `Part of #<map>` at the top of the child body. Labels: `wayfinder:<type>` (`research`/`prototype`/`grilling`/`task`). Once claimed, the ticket is assigned to the driving dev.
- **Blocking**: GitHub's **native issue dependencies** — the canonical, UI-visible representation. Add an edge with `gh api --method POST repos/<owner>/<repo>/issues/<child>/dependencies/blocked_by -F issue_id=<blocker-db-id>`, where `<blocker-db-id>` is the blocker's numeric **database id** (`gh api repos/<owner>/<repo>/issues/<n> --jq .id`, _not_ the `#number` or `node_id`). GitHub reports `issue_dependencies_summary.blocked_by` (open blockers only — the live gate). Where dependencies aren't available, fall back to a `Blocked by: #<n>, #<n>` line at the top of the child body. A ticket is unblocked when every blocker is closed.
- **Frontier query**: list the map's open children (`gh issue list --state open`, scoped to the map's sub-issues / task list), drop any with an open blocker (`issue_dependencies_summary.blocked_by > 0`, or an open issue in the `Blocked by` line) or an assignee; first in map order wins.
- **Claim**: `gh issue edit <n> --add-assignee @me` — the session's first write.
- **Resolve**: `gh issue comment <n> --body "<answer>"`, then `gh issue close <n>`, then append a context pointer (gist + link) to the map's Decisions-so-far.
