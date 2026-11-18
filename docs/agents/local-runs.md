# Local runs

Running the stack, the suites and reviews on a development machine without a false result. The
commands are in `AGENTS.md` ("Testing and Validation") and `frontend/README.md`; this file holds
their traps, most of them on the maintainer's Windows machine.

## Before handing a tree to a reviewer

Commit, or make a WIP commit, before dispatching any agent that reads the working tree
(`code-reviewer`, `docs-updater`, `penetration-tester`), and tell it never to run `git checkout --`,
`git restore` or `git reset`. A review agent once mutation-tested an uncommitted change, undid its
mutation with `git checkout --`, and reverted the file to `HEAD`, destroying the feature work; its
repair then corrupted the file. Read-only instructions didn't stop it. If an agent damages the tree,
restore the file from `HEAD`, re-apply your own edits, and re-run the whole verification: don't
patch from its account of what it changed.

## Ports and processes

- **Something else may own 5273 or 8081.** Outside CI, Playwright reuses whatever listens there:
  the Compose app stack, or another project's dev server (an Expo server once held 8081). The suite
  then fails at login (`Could not start Bungie authorization`, `Failed to fetch`) with nothing wrong
  in the code. Check the owner (`Get-NetTCPConnection -LocalPort 8081`), stop it, or for one run set
  `PORT=18081` and `E2E_API_URL=http://127.0.0.1:18081`.
- **The Compose network is named after the checkout directory**: `guardiantracker_default` in the
  main checkout, `<worktree>_default` in a worktree. Substitute it in the visual-baseline
  `docker run --network …` (`docker network ls`).
- **A worktree has no root `.env`**, so every `docker compose` command there needs `GO_ENV=development`
  exported just to interpolate the file.

## Files on Windows

- **The checkout is case-insensitive** (`core.ignorecase = true`), so two files in one directory
  that differ only by case can't both exist, though Linux CI allows it. Rename one to a
  case-distinct name.
- **Python's `pathlib.Path.write_text()` rewrites every line ending to CRLF**, and `open()` without
  `encoding="utf-8"` uses cp1252, which turned an em dash in a Go comment into invalid UTF-8. A
  policy test then fails on a file whose `git diff` shows one changed line: its regex expects
  `\n`. Write with `newline="\n"` and `encoding="utf-8"`, or use an editor that keeps the file's
  bytes. `.gitattributes` normalizes on commit, so the damage is local, but it misleads. A phantom
  `M` with an empty diff is line-ending churn; `file <path>` reports CRLF.
- **Symlinks don't work** (`core.symlinks=false`): `ln -s` makes a copy (`AGENTS.md`, "Agent
  Routing").

## The public directory is published

`frontend/public/` is served from the site root: Vite copies every file there into `dist/`, and
nginx serves it. A vendored asset pack's `README.md` would ship as `/README.md`. `frontend/.dockerignore`
doesn't help: its patterns are anchored to the context root, so `README.md` excludes only
`frontend/README.md`. Keep only assets the browser fetches in `public/`; source docs and the rest of
a pack go in `frontend/design/`, which both `.prettierignore` files exclude.

## Browser tests

- **MSW hides CORS failures.** Vitest intercepts `fetch` and runs no preflight, so a new HTTP verb or
  custom header can pass every unit test and be refused by the browser, with no server log. Check
  the allow lists in `api/middleware/security.go` and cover the call with a Playwright functional
  test (`PATCH` was first refused this way).
- **The destructive-auth spec stubs every request its fake identity makes.**
  `frontend/e2e/functional/identity-auth-destructive.spec.ts` swaps in a non-JWT second identity,
  and a request it doesn't stub reaches the real API, gets a 401 and logs both tabs out, failing far
  from the cause. When a change adds a request that fires on every authenticated page, add its stub
  to that handler in the same PR and run the spec locally.

## Minikube acceptance from an agent shell

- `startup.ps1`'s port-forwards are `Start-Job` jobs that live only in the PowerShell session that
  started them: run startup, every check and shutdown in one `pwsh -File` harness, in the
  background, since the builds take five to ten minutes.
- `shutdown.ps1` calls `Read-Host` twice and the agent's PowerShell is non-interactive. A harness
  function `global:Read-Host` that dequeues answers from a `$global:` queue shadows the cmdlet; it
  must be global, because it runs inside the script's scope.
- Ask before `minikube delete`.

## The local OAuth topology

Local development opens the app at `http://localhost:5273`, and only Bungie's HTTPS callback lands
on the ngrok tunnel. Debug sign-in assuming that shape. `VITE_*` build settings for the Docker
frontend go in the root `.env` (passed as build args); `frontend/.env.local` never reaches the Docker
build. Verify what the running stack contains (bundle strings, `docker compose config`) before
guessing again.
