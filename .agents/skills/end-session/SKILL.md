---
name: end-session
description: Close out a work session — harvest what was learned into memory, private/ docs, and GitHub issues, then clean the local workspace. Use when the user says they are done for the day, wants to wrap up or end the session, or asks to clean up and record what this session found.
---

# End session

> clean up the local workspace, update any private/ docs and/or github issues that need it from this session.

**Announce at start:** "I'm using the end-session skill to close out this session."

## Why this exists

A session's durable value is the part that survives it. Discoveries live in the
transcript, scratch files, and a half-updated plan; the next session starts blind
unless each one is **routed** to the one place that owns it. This skill harvests
first and deletes last, so cleanup never destroys evidence that has not been
recorded yet.

It does not ship code. `/ship` owns `VERSION`, `CHANGELOG.md`, public docs, and
the PR. If the branch has undelivered work, name it in the report and let the user
call `/ship`.

## Steps

### 1. Harvest — inventory this session before touching anything

Re-read the session and write a flat list of everything it produced that a future
session would want: decisions made, facts verified against real data, dead ends
and why they were dead, work started, work discovered but not started, surprises
that contradicted a doc.

Include the scratch surfaces, because step 5 clears them:

- the OS temp scratchpad this session wrote to
- untracked files in the tree (`git status --porcelain -uall`)
- output from research or verification runs

**Done when:** every item is written down and routed in step 2. An item you cannot
route is one you do not yet understand — say so in the report rather than dropping it.

### 2. Route every item to exactly one home

One fact, one owner. Duplicating a fact across memory and `private/` means the next
session reads a stale copy of it somewhere.

| The item is…                                                                  | Home                                                                                                                                                                                                          |
| ----------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A durable cross-session fact — how the user works, a constraint, a gotcha     | Memory (step 4)                                                                                                                                                                                               |
| Execution sequencing, gates, slice order, "what's next"                       | `private/IMPLEMENTATION_PLAN.md` (step 3)                                                                                                                                                                     |
| Work that shipped, or an audit now retired                                    | `private/archive.md` (step 3)                                                                                                                                                                                 |
| A deferred production-infrastructure decision                                 | `private/InfraTODO.md` (step 3)                                                                                                                                                                               |
| A residual security risk accepted for now                                     | `private/security-limitations.md` (step 3)                                                                                                                                                                    |
| Raw Bungie API / manifest research, point-in-time evidence                    | A new `private/` file, indexed in `private/README.md` (step 3)                                                                                                                                                |
| A discrete piece of work someone should pick up                               | A GitHub issue (step 4)                                                                                                                                                                                       |
| Implemented behavior, setup, domain vocabulary, a durable architecture choice | Public docs — **not this skill.** `README.md`, `SETUP.md`, `CONTEXT.md`, `docs/architecture.md`, `docs/adr/`, `CHANGELOG.md` belong with the change, via `/ship` and `docs-updater`. Flag them in the report. |

Anything the repo already records — code structure, git history, an ADR, a
merged PR — is already recorded. Skip it.

### 3. Update `private/`

`private/` is gitignored, so nothing here reaches a commit; it is the working
memory the next session reads. `private/README.md` is its index and classifies
every file as **living**, **reference snapshot**, or **historical evidence**.

- **`IMPLEMENTATION_PLAN.md`** — the active queue and the precedence rule. Advance
  it: mark slices done, record which gate opened or closed, correct sequencing the
  session proved wrong. It holds precedence over the ADRs, which are accepted
  _plans_ rather than descriptions of current behavior.
- **`archive.md`** — retire completed work out of the plan into it. Its sections are
  Durable Decisions, Shipped Timeline, Shipped Baseline by Domain, Detailed Shipped
  Entries, Retired Audits; put the entry under the one that fits and keep the
  baseline version line honest.
- **New file** — add a row to `private/README.md` under the right heading, with the
  issue or PR it belongs to and its outcome. An unindexed private doc reads as a
  current task list to the next session.
- **`README.md`'s `**Updated:**` line** — set it to today whenever the index changes.

Absolute dates only. "Last week" is unreadable in a month.

**Done when:** the plan's "what's next" matches reality, and every private file the
session created or finished is classified in the index.

### 4. Update GitHub issues and memory

**Issues.** Command forms and the wayfinder map/child conventions live in
[`docs/agents/issue-tracker.md`](../../../docs/agents/issue-tracker.md); the label
vocabulary lives in [`docs/agents/triage-labels.md`](../../../docs/agents/triage-labels.md).
Confirm `gh auth status`, then for this session:

- **Resolved** → comment with what resolved it (PR number, version) and close.
- **Advanced but open** → comment with the current state and what unblocks it, so
  the issue stands alone without this transcript.
- **New work discovered** → open an issue. This repo creates an issue per
  implementation slice **as the slice becomes ready**, not up front — so open one
  for work that is ready, and record merely-possible work in the plan instead.
- **Underspecified** → apply the triage label rather than leaving it bare.

**Memory.** One fact per file in the per-project memory directory (its absolute path
is in the memory section of your system prompt), with `name`, `description`, and
`metadata.type` of `user`, `feedback`, `project`, or `reference`. Link related
memories with `[[slug]]`. Then add or update the one-line pointer in `MEMORY.md` —
the index, never the content.

Before writing, read the existing memory whose subject overlaps and **update it**;
a second file on the same subject is how the index rots. Delete memories this
session proved wrong. Convert relative dates to absolute. If the set has drifted
badly, `anthropic-skills:consolidate-memory` does a full reconciliation pass.

**Done when:** an issue reader with no access to this session knows the current
state, and `MEMORY.md` has exactly one live pointer per fact.

### 5. Clean the local workspace

Harvesting is done, so deleting is now safe.

**Uncommitted work first.** `git status --porcelain -uall`. For each tracked
modification, ask the user whether to commit or discard — never discard
user-authored work on your own judgment.

**Then the known artifact paths.** Delete only these; they all regenerate:

```powershell
frontend/playwright-report/  frontend/test-results/  frontend/playwright/.auth/
backend/api-service/.e2e/            # manifest + search-index the E2E API writes
backend/api-service/coverage         # extensionless profile from -coverprofile
coverage/  *.coverprofile  *.out  *.test  *.exe  *.tsbuildinfo
```

Keep `private/`, every `.env*`, and `k8s/*-secret.yaml` — all gitignored, none
disposable. That is why the paths above are enumerated rather than swept with
`git clean -X`: one blanket sweep takes the working docs and the credentials with
the junk.

**Throwaway containers.** Stop the test and E2E databases the local suites start:

```powershell
docker compose --profile test down          # gt-test-pg on 5533; or ./test-local.ps1 -Down
docker compose --profile e2e down           # e2e-postgres
```

Leave the main stack running unless the user asks for it down, and stop it with a
plain `docker compose down`. Use `-v` only on explicit request: it wipes the
`manifest-data` volume and costs a ~290MB Bungie re-download on next start.

**Branch state.** Report, do not act: unpushed commits (`git log --oneline @{u}..`),
stale worktrees (`git worktree list`), and merged local branches. Pushing is `/ship`'s.

**Done when:** `git status` shows only work the user chose to keep, and no throwaway
container is still bound to 5533 or the E2E port.

### 6. Report

Give the user, in this order:

1. What was recorded and where — memory files, `private/` files, issues opened,
   commented, or closed, each by name or number.
2. What was cleaned — paths removed, containers stopped.
3. What is still open — unpushed commits, undelivered branch work, public-doc
   updates owed to `/ship`, and any harvested item you could not route.

State plainly that nothing was pushed or merged.
