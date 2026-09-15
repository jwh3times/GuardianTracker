---
name: handoff
description: Hand the session to another machine — write a handoff doc to Proton Drive, register it in handoff_map.json, then close out with end-session.
argument-hint: "What will the next session be used for?"
disable-model-invocation: true
---

# Handoff

Write a handoff document a fresh agent on another machine can continue from,
publish it to the shared Proton Drive **Handoffs** folder, register it as this
repo's active handoff, then close out the session. `/lets-go` is the receiving
end: it reads the map, resumes from the doc, and clears the entry.

Reach the folder and map only through the helper. It resolves the folder the same
way on Windows and Fedora (`~/Proton Drive/<account>/My files/Documents/Handoffs`,
or `HANDOFF_DIR` when set) and keys the map by the `origin` repository name:

```bash
node .agents/skills/handoff/scripts/handoff-map.mjs dir          # the Handoffs folder
node .agents/skills/handoff/scripts/handoff-map.mjs get          # this repo's active entry
node .agents/skills/handoff/scripts/handoff-map.mjs set <file>   # make <file> the active entry
```

If the helper cannot resolve the folder, stop and tell the user: Proton Drive is
not running or not synced, and a doc written anywhere else never reaches the
other machine.

## Steps

### 1. Survey work not merged to main

The other machine starts from `origin/main`; anything not merged there is stranded
on this one. Run `git fetch origin --prune`, then run every check below in the
public checkout and again inside `private/` when it is a repository:

- **Uncommitted** — `git status --porcelain -uall`.
- **Unpushed** — `git for-each-ref --format="%(refname:short) %(upstream:track)" refs/heads`.
- **Unmerged branches** — every local branch other than `main`, and every branch
  checked out in `git worktree list`. A branch is merged only when a merged PR's
  `headRefOid` equals its tip:
  `gh pr list --head <branch> --state merged --json headRefOid`. Squash merges
  leave `git branch --merged` blind here.
- **Open PRs** — `gh pr list --author @me --state open`.

When any check finds something, **alert the user immediately** with a block headed
`⚠ Work not merged to main` that lists each item with its repository, branch,
worktree, or PR. Then continue; step 5 repeats the alert.

**Done when:** every check has run in both repositories, and each finding is in
the alert or the survey is confirmed clean.

### 2. Write the handoff document

Summarise the session so a fresh agent can continue it:

- the current state, what comes next, and where the work lives (branch, worktree,
  PR, issue)
- each finding from step 1 and what the next session must do about it
- a **Suggested skills** section naming the skills the next agent should invoke

Reference specs, plans, ADRs, issues, commits, and diffs by path or URL instead of
restating them. Redact API keys, passwords, other secrets, and personally
identifiable information — the doc lives outside the repository.

If the user passed arguments, they describe what the next session will focus on;
tailor the doc to that.

Name the file `<repo-slug>-handoff-<YYYY-MM-DD>[-<topic>].md`, with the repo name
in kebab case (`guardian-tracker-handoff-2026-09-15-current-activity.md`), and
write it directly into the Handoffs folder. When the name is taken, choose a
distinct topic so the earlier doc survives.

**Done when:** the file is in the Handoffs folder, and a reader holding only it and
the repository knows what to do first.

### 3. Register it in the map

Run `set <file>`. The helper sets this repo's entry, stamps `Last_Updated`, and
prints the entry it replaced. A non-null `previous` is an unconsumed handoff you
just superseded — name it to the user; its doc stays in the folder.

**Done when:** `get` returns the new filename with `exists: true`.

### 4. Close out with end-session

Invoke the `end-session` skill and complete it. Its harvest can open, comment on,
or close issues and clean the workspace; wherever it changes something the handoff
doc states, edit the doc in place to match.

**Done when:** end-session's report is delivered and the doc agrees with it.

### 5. Report

Give the user the doc's path and the map entry set, including any superseded one.
If step 1 found anything, end the response with the `⚠ Work not merged to main`
block, updated for whatever end-session changed.
