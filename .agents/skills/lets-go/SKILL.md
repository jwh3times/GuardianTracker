---
name: lets-go
description: Resume this repo's active handoff from Proton Drive and mark it consumed in handoff_map.json.
disable-model-invocation: true
---

# Let's go

Pick up where `/handoff` left off on another machine. The Proton Drive
**Handoffs** folder holds the doc, and `handoff_map.json` names this repo's active
one. Use the helper `/handoff` uses, so both machines read and write the map the
same way:

```bash
node .agents/skills/handoff/scripts/handoff-map.mjs get     # this repo's active entry
node .agents/skills/handoff/scripts/handoff-map.mjs clear   # mark it consumed (null)
```

## Steps

### 1. Find the active handoff

Run `get`, then take the first branch that applies:

- **The helper fails** — Proton Drive is not running or not synced on this machine.
  Tell the user and stop.
- **`file` is null** — tell the user this repo (`repo` in the output) has no active
  handoff, and stop.
- **`exists` is false** — the map names a doc that has not synced here yet. Tell
  the user, leave the map untouched, and stop.

**Done when:** you hold the `path` of a doc that exists, or you have stopped with
one of the messages above.

### 2. Read it, then claim it

Read the whole doc. Then run `clear`, so the handoff is resumed exactly once.
Clearing follows the read: an entry cleared before its doc is read is a lost
handoff.

**Done when:** `get` returns `file: null`.

### 3. Re-ground

The doc is a snapshot taken on another machine. Run `git fetch origin --prune` and
`git status`, follow its workspace instructions (`npm run sync:main`, a fresh
branch or worktree), and check every branch, PR, and issue it names against its
current state. Where the two disagree, current state wins.

Brief the user: what the doc says comes next, any work it flagged as not merged
to main (uncommitted work it lists stays on the other machine), and every
disagreement you found.

**Done when:** each branch, PR, and issue the doc names has been checked, and the
brief is delivered.

### 4. Proceed

Continue with the work the doc names as next, invoking its suggested skills where
they apply. When the doc hands over a decision only the user can make, ask it.
