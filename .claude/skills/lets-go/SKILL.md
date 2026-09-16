---
# GENERATED FILE — DO NOT EDIT.
# Source: .agents/skills/lets-go/SKILL.md
# Regenerate: node scripts/sync-agent-configs.mjs
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

Run the shell commands through the POSIX shell (Git Bash on Windows). The map is
read and written only through `node .agents/skills/handoff/scripts/handoff-map.mjs`,
never with `jq` or by hand.

## How the Handoffs folder reaches this machine

Two transports, decided by what is installed:

- **Desktop client** (Windows) — the Proton Drive client keeps
  `~/Proton Drive/<account>/My files/Documents/Handoffs` in sync on its own.
  Writing into that folder is the whole sync; the CLI steps below are skipped.
- **CLI mirror** (Fedora, where no client exists) — `proton-drive` (the Proton
  Drive CLI) is on `PATH` and `HANDOFFS_DIR` names a local mirror folder. Nothing
  syncs by itself: each **Pull** and **Push** block below is run explicitly, and
  the cloud folder is always `/my-files/Documents/Handoffs`. A CLI reply of
  `You need to login first` means `proton-drive auth login` first — that is an
  interactive step for the user.

Decide once at the start: `command -v proton-drive` succeeds **and** the desktop
client's folder is absent means CLI mirror; otherwise desktop client.

## Steps

### 1. Find the active handoff

**Pull** (CLI mirror only) — fetch the current map before reading it:

```bash
mkdir -p "$HANDOFFS_DIR"
proton-drive filesystem download -f remove /my-files/Documents/Handoffs/handoff_map.json "$HANDOFFS_DIR"
```

Run `get`, then take the first branch that applies:

- **The helper fails** — Proton Drive is not running or not synced on this machine.
  Tell the user and stop.
- **`file` is null** — tell the user this repo (`repo` in the output) has no active
  handoff, and stop.
- **`exists` is false**, CLI mirror — the doc is fetched by name, then `get` is
  rerun:

  ```bash
  proton-drive filesystem download -f remove "/my-files/Documents/Handoffs/<file>" "$HANDOFFS_DIR"
  ```

  A `Node not found` reply means the other machine has not synced the doc to the
  cloud yet (the CLI still exits 0, so read the message, not the exit code). Tell
  the user the file name and stop without clearing, so a retry after sync still
  works.

- **`exists` is false**, desktop client — the map names a doc the client has not
  synced here yet. Tell the user the file name and stop without clearing, so a
  retry after sync still works.

**Done when:** you hold the `path` of a doc that exists, or you have stopped with
one of the messages above.

### 2. Read it, then claim it

Read the whole doc. Then run `clear`, so the handoff is resumed exactly once.
Clearing follows the read: an entry cleared before its doc is read is a lost
handoff.

**Done when:** `get` returns `file: null`.

**Push** (CLI mirror only) — the cleared map goes back to the cloud, so the other
machine cannot resume the same handoff a second time:

```bash
proton-drive filesystem upload -f create-new-revision -t "$HANDOFFS_DIR/handoff_map.json" /my-files/Documents/Handoffs
```

Complete when the transfer summary lists the map as uploaded.

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
