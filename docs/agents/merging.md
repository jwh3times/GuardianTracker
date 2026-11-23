# Merging

Everything from an open pull request to a finished merge. `/ship` stops when the PR opens; the
maintainer says when to merge. What CI runs and what the ruleset requires are in `AGENTS.md`
("CI/CD", "Branch protection"); this file holds what to do, and the ways CI status has given a
confident wrong answer.

## Is the result for your commit?

- **`gh pr checks <n> --watch` straight after a push watches the previous commit's run.** The new
  run hasn't registered yet, so it prints the old checks green and exits 0. Identical job ids and
  durations across two "runs" are the tell. Confirm the PR head is your commit, then count the
  non-passing runs on it:

  ```bash
  [ "$(git rev-parse HEAD)" = "$(gh pr view <n> --json headRefOid --jq .headRefOid)" ]
  gh api repos/jwh3times/GuardianTracker/commits/$(git rev-parse HEAD)/check-runs \
    --jq '[.check_runs[] | select(.conclusion != "success" and .conclusion != "neutral" and .conclusion != "skipped")] | length'
  ```

  Zero on a sha that equals the PR head is the safe answer.

- **`mergeStateStatus: CLEAN` can come before the checks register.** It means nothing is blocking
  yet, so it is necessary, not sufficient.
- **`--watch` can exit 1 while checks are still pending**, and `--fail-fast` leaves the rest
  `pending`. Read the table, not the exit code. An unattended loop polls
  `gh pr checks <n> --required` until no row is pending, then requires every row to pass.

## Reds that aren't the code

- **Cancelled duplicates.** Two `pull_request` events a few seconds apart cancel each workflow's
  first run (`cancel-in-progress`), and the PR page rolls the cancelled runs up as failures. If
  every red entry is `cancelled` with a `success` sibling on the same sha, re-run the cancelled ones
  (`gh run rerun <id>`) or ignore them.
- **`govulncheck` reads the live vulnerability database**, so a new advisory reddens `Test Go
Services` on `main` and every open PR at once. Grep the failing log for `GO-20`:
  - a module advisory: bump the module (`go get <module>@<fixed>`, `go mod tidy`); it is often
    indirect, so Dependabot won't;
  - a standard-library advisory ("Found in … go1.x.y"): bump the Go patch everywhere it is pinned —
    `backend/api-service/go.mod`'s `toolchain`, `GO_VERSION` in `ci-cd.yml` and `browser.yml`, and
    the `GOTOOLCHAIN=` Staticcheck line in `AGENTS.md`, `SETUP.md` and
    `.claude/agents/go-services.md` (`npm run test:go-toolchain-policy` checks they agree), then
    `npm run sync:agents`.

  Fix it on its own branch off `main`, then update the blocked PRs. Before blaming your branch for
  any red `Test Go Services`, check that `main` is green at your merge base
  (`gh run list --workflow ci-cd.yml --branch main`).

- **CodeQL's aggregate check can fail in two seconds** and read like a hiccup. Read the check's
  `.output.title` and the open alerts for the PR
  (`gh api "repos/jwh3times/GuardianTracker/code-scanning/alerts?pr=<n>&state=open"`): on one PR it
  was four new high-severity findings. Alerts here are fixed, never dismissed without the
  maintainer's agreement; when one points at a half-finished migration to a helper, finish the
  migration rather than patching the flagged line. An alert stays open until CodeQL's push
  analysis on `main` finishes.
- **`Browser Visual Regression`'s log opens with hundreds of `curl: (7) Failed to connect to
127.0.0.1 port 8090` lines**: the fake-Bungie readiness loop, which then succeeds. Grep for
  `toHaveScreenshot`, `Snapshot:` or `failed`. First rule out drifted image pins
  (`npm run sync:image-pins -- --check`), which fail the job at browser launch instead.

## Before merging

- **The ruleset requires up-to-date branches, and auto-merge is off.** A `BEHIND` PR needs
  `gh pr update-branch <n>` and a fresh green run, so merge several PRs one at a time: update,
  wait for the required checks, merge, then update the next.
- **No branch names a version**, so open PRs merge in any order: each carries only its own
  `changelog.d/` fragment, and catching up with `main` never touches `CHANGELOG.md`. A conflict in
  `CHANGELOG.md` means the branch edited it, which `Changelog Version` rejects; move the entries
  into the branch's fragment.
- **Check the branch's base before opening a PR.** `git diff --stat origin/main..HEAD` shows a
  wrong base as unrelated files; `git rebase --onto origin/main <old-base>` fixes it.

## Merging

`gh pr merge <n> --merge --delete-branch`. Merge commits are the only method the ruleset allows.
When the branch is checked out in another worktree, this also removes that worktree and the local
branch; a later `git worktree remove` failing with "is not a working tree" is harmless. Run
`git fetch --prune` afterwards.

## After the merge

- Wait for the version tag, then run `npm run board:blockers` and clear what it lists
  ([issue tracker](issue-tracker.md#project-board)).
- **`version.yml` releases the merge** (Actions → Version, about a minute): one
  `chore(release): v<x.y.z> [skip ci]` commit by the release App on `main` with the section, the
  tag on the merge commit, and a GitHub Release whose body is the section. A merge with no fragment,
  such as Dependabot's, gets a one-bullet section from its PR title. Fast-forward local `main`
  afterwards, since the stamp commit lands on top of the merge. A CodeQL "Push on main" run on the
  stamp commit is expected: `[skip ci]` does not suppress default-setup code scanning. A run that
  ends with "main moved during this run" is benign: the next run releases both merges. Any other
  failure leaves the merge untagged until a later run succeeds, and the next run releases it.

## Dependency pull requests

Half the merge traffic is Dependabot's. When a bot PR is red because the bump needs a companion
change that belongs to it, push the fix onto the bot's branch:

```bash
git fetch origin <dependabot-branch>
git checkout -B fixup FETCH_HEAD
# edit, commit
git push origin fixup:<dependabot-branch>
```

- **Never `@dependabot rebase` afterwards**: Dependabot force-pushes the branch it owns and drops
  your commit. Use `gh pr update-branch <n>`. A bot PR needs no fragment: `Changelog Version`
  passes it and `version.yml` writes its section from the PR title.
- **`/ship` works on a bot branch too**, when the fix deserves a better entry than the PR title: it
  writes the branch's fragment. Fetch with `git fetch origin pull/<n>/head`, then rename the
  local branch to the bot branch's name and set its upstream.
- **A fix that belongs on `main` regardless** (a govulncheck advisory reddening every PR) goes on
  its own branch off `main`, not onto a bot branch.
- **`@dependabot` ignore and unignore commands are scoped to the PR's ecosystem and directory**,
  and are dropped silently on a closed PR.
- **Close Dependabot's Go release-candidate proposals** for the builder image, and avoid
  `@dependabot ignore this major version`, which also suppresses the major once it is stable.

The bumps that arrive red, and their fixes:

| Bump               | Why it is red                                                                                                                                                                                                                                              | Fix                                                                                                                                      |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Node               | the Node policy wants one exact patch everywhere, and Dependabot edits only two Dockerfiles; its digest can also be single-platform, failing `Build Docker Images`                                                                                         | set `.nvmrc` to the new patch, `npm run sync:image-pins`, then `node --test scripts/node-version-policy.test.mjs`                        |
| Node, digest only  | lands green but can leave `Dockerfile.playwright` stale: the policy test compares versions, not digests                                                                                                                                                    | `npm run sync:image-pins -- --check` on every Node bump, green or not                                                                    |
| `@playwright/test` | the image pin is in the docker ecosystem, so the npm PR can't move it; the visual job then fails at browser launch                                                                                                                                         | `npm run sync:image-pins`; `Browser E2E + Axe` staying green says nothing, since it installs browsers on the runner                      |
| PostgreSQL version | the `docker-compose` ecosystem doesn't move the `Test Go Services` service container, and the pin policy fails on any retired `postgres:<version>` tag left in a tracked file outside the release history, such as `.claude/agents/postgres-specialist.md` | realign `ci-cd.yml`'s service image and edit the authored agent file, then `npm run sync:agents`; `sync:image-pins` doesn't own this pin |

`sync:image-pins` re-resolves every pin, and Microsoft republishes Playwright tags in place, so it
can move the Playwright digest with no version change. Say so in the PR: that image renders the
visual baselines.
