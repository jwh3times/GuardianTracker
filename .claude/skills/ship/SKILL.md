---
name: ship
description: Ship the current branch — refresh docs, write the CHANGELOG entry for the version this merge will mint, run fast checks, push, and open or update the PR. Use when a feature branch is ready for review, or when the user says "ship it", "open a PR", or "push this".
---

# Ship

Take the current branch from "code is done" to "PR is open and green-able", and
make sure the changelog names the version this merge will actually create.

**Announce at start:** "I'm using the ship skill to open a PR for this branch."

## Why this exists

Every merge to `main` is auto-tagged `v<major>.<minor>.<build>` by
`.github/workflows/version.yml`, where `build` auto-increments. So a branch's
changelog entry must be written for **the version its merge will mint** — an
`[Unreleased]` section is always wrong the moment it lands. `/ship` computes that
version and writes the entry, and the `Changelog Version` CI job verifies the
prediction still holds at merge time.

## Steps

### 1. Preconditions — stop if any fail

- **Not on `main`.** `main` is protected; work must be on a branch. If on `main`,
  stop and offer to create one (`git checkout -b agent/<topic>`).
- **Clean working tree.** Run `git status --porcelain`. If anything is uncommitted,
  stop and ask the user whether to commit it — do not commit silently.
- **`gh` authenticated.** `gh auth status` must succeed.

### 2. Backfill any undocumented released versions

Compare git tags against `CHANGELOG.md`:

```bash
git fetch --tags -q origin
git tag -l "v*" --sort=v:refname | tail -5
sed -nE 's/^## \[([0-9]+\.[0-9]+\.[0-9]+)\].*/\1/p' CHANGELOG.md | head -5
```

Any tag with **no** matching `## [x.y.z]` section is a released version with no
entry — in practice a merged dependabot PR, which the CI guard exempts. Backfill it
now: read what that tag actually changed (`git show --stat <tag>`, and the
`package.json` / `go.mod` diff for dependency bumps) and add a dated section for it
in the right position. Keep it factual — name the packages and versions.

This is what makes the bot exemption safe. Skipping it lets the changelog silently
lose versions.

### 3. Compute the target version

```bash
bash scripts/next-version.sh
```

This prints a bare SemVer (e.g. `0.3.24`) — no `v` prefix. It is the single source of
truth; the tag workflow and the CI guard call the same script. Do not compute this
yourself.

### 4. Refresh the docs

Invoke the `docs-updater` subagent, scoped to **this branch's diff only** — not a
full audit:

```bash
git diff $(git merge-base main HEAD)..HEAD --stat
```

Tell it exactly what changed and let it update the docs it owns (README.md, SETUP.md,
docs/architecture.md, ROADMAP.md, SECURITY.md, CLAUDE.md, docs/adr/, .claude/agents/).
It owns CHANGELOG.md too, but **you** write the changelog section in step 5 — tell it
to leave CHANGELOG.md alone so you don't fight over the file.

### 5. Write the CHANGELOG entry

Insert a section for the target version immediately below `## [Unreleased]`:

```markdown
## [Unreleased]

No unreleased changes.

## [0.3.24] - 2026-07-14

### Added

- ...
```

Rules:

- `## [Unreleased]` **stays**, with the `No unreleased changes.` placeholder. This is
  the convention the file already uses.
- Date is today, `YYYY-MM-DD`.
- Group under Keep a Changelog headings — `Added`, `Changed`, `Fixed`, `Removed`,
  `Security`. Use **one** heading of each kind per section.
- Describe user-visible behavior and its consequences, derived from the branch diff.
  Not a commit log.
- **Idempotent:** if you already wrote a section for this version on a previous
  `/ship` of this branch, **rewrite it in place** — never stack a second one. If the
  target version changed since last time (someone else merged first), renumber the
  existing section rather than adding a new one.

### 6. Fast checks — refuse to push if any fail

Tests, Staticcheck, `govulncheck`, and the Docker build are **not** run here; CI owns
them. These are the cheap gates that catch most mistakes in seconds:

```bash
# frontend/
npm run format:check
npm run lint
npx tsc --noEmit

# backend/api-service/
gofmt -l .
go vet ./...
```

`format:check` also covers markdown, so run it **after** the doc edits. If it fails,
fix with `npm run format` and re-run. If any check is red, stop and report — do not
push.

### 7. Commit the docs and changelog

```bash
git add -A
git commit -m "docs: update docs and changelog for v<version>"
```

### 8. Push and open or update the PR

```bash
git push -u origin "$(git branch --show-current)"
```

Then check whether a PR already exists for this branch:

```bash
gh pr list --head "$(git branch --show-current)" --state open --json number -q '.[0].number'
```

- **No PR** → `gh pr create --base main` with a title and a body derived from the
  changelog section you just wrote.
- **PR exists** → `gh pr edit <number>` to refresh the body. Do not open a second PR.

### 9. Report

Give the user: the PR URL, the version this merge will mint, and anything the fast
checks or backfill surfaced. State plainly that tests run in CI, not locally — do not
imply the branch is verified beyond the fast checks.

## Do not

- Merge the PR. The repo self-merges once green; `/ship` stops at "PR open".
- Push to `main`.
- Run the full test suites — that is CI's job and it makes this skill slow.
- Invent the version number. Always call `scripts/next-version.sh`.
