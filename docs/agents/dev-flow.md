# Development flow

The binding process for changing code in this repository. `AGENTS.md`
(`## Development flow`) states the gate; this file holds the routing table, the
exemptions, and the issue-linking rules. For the full map of skills and phase
boundaries, read the router at
[`.agents/skills/ask-matt/SKILL.md`](../../.agents/skills/ask-matt/SKILL.md);
this file does not restate it.

## The gate

Every code change starts from a **ready issue**:

- an open issue labelled `ready-for-agent` whose body is a `/to-spec` spec, a
  `/to-tickets` ticket, or a `/triage` brief; or
- an open issue labelled `bug`, worked through `/diagnosing-bugs`. The issue
  number is required; a spec is not.

A project-board draft item is not an issue. Running `/to-spec` or `/to-tickets`
on it, or claiming it, makes it one
([ADR 0024](../adr/0024-committing-to-work-creates-the-issue.md)); it is ready
once its body meets the bar above.

Check before the first edit to a code path. Reading code, running tests, and
answering questions need no issue. When nothing qualifies, make no edit; route
the user instead.

## Routing

When no ready issue exists, name the matching command, say in one line why, and
wait for the user to run it:

| Situation                           | Command                                                                      |
| ----------------------------------- | ---------------------------------------------------------------------------- |
| An idea one session can hold        | `/grill-with-docs`, then `/to-spec` (add `/to-tickets` if it spans sessions) |
| A huge, foggy, multi-session effort | `/wayfinder`, then `/to-spec` once the map clears                            |
| A raw issue someone else filed      | `/triage`                                                                    |
| A ready issue nobody has started    | `/implement`, in a fresh context                                             |
| Something broken                    | File a `bug` issue, then `/diagnosing-bugs`                                  |

"Nobody has started" means the issue is on the ready frontier: open, no open
blocker, no assignee. Claim it by assigning it when work begins.

## Phases

`/wayfinder`, `/grill-with-docs`, `/to-spec`, `/to-tickets`, `/implement`, and
`/triage` are user-invoked and keep `disable-model-invocation: true`. The human
starts every phase. At a phase boundary, name the next command with a one-line
reason and wait.

Inside a phase, start these skills yourself whenever they fit: `/tdd`,
`/code-review`, `/diagnosing-bugs`, `/domain-modeling`, `/codebase-design`,
`/prototype`, `/research`.

## What counts as code

**Docs and agent config** are:

- every `*.md` file, and everything under `docs/`;
- `.claude/`, `.agents/`, `.codex/`, and `skills-lock.json`;
- `.github/ISSUE_TEMPLATE/`.

**Code** is every other tracked file — including `backend/`, `frontend/`,
`database/`, `k8s/`, `scripts/`, `.github/workflows/`, `.github/dependabot.yml`,
`docker-compose.yml`, `package.json`, `.nvmrc`, `.env.example`, and `setup.ps1`.
A change that touches both sets is a code change.

## Exemptions

These changes need no ready issue:

- **Dependency bumps** — Dependabot PRs, `chore(deps)` commits, the
  `npm run sync:image-pins` commit a Node or `@playwright/test` bump needs, and
  a toolchain or module pin moved to clear a `govulncheck` advisory.
- **Docs and agent config only** — the change touches nothing in the code set
  above.
- **Release prep** — `/ship`'s `docs: update docs and changelog for v<version>`
  commit, including its `VERSION` change, and changelog backfills.
- **User waiver** — the user explicitly waives the flow for this change. Record
  it as the first line of the PR body: `Flow waived by the user: <reason>`. Only
  the user grants a waiver.

An exempt branch keeps the plain `<type>/<slug>` name, e.g.
`docs/drop-setup-node-hold`.

## Linking work to its issue

- **Branch:** `<type>/<issue#>-<slug>`, using the repo's types (`feat`, `fix`,
  `chore`, `docs`, `test`, `refactor`) — e.g. `feat/512-wishlist-sort`,
  `fix/518-oauth-redirect`. Create it from the latest `main` before the first
  edit; `/implement` commits to the current branch.
- **PR body:** the first line is `Closes #N` when the PR finishes the ticket, or
  `Part of #N` when it is one slice of a spec that stays open. The PR template's
  `# Summary` follows. When `/ship` writes or refreshes the body, it keeps that
  line first.
- An issue number in the PR title alone does not link the issue.
