# Test proof

How `test-prover` runs one test on its own here, and how it tells a run that tested nothing from a
pass.

## Test commands

- `frontend/`: Vitest, `src/**/*.test.{ts,tsx}`.
  - One file: `npx vitest run <file>`; one test by name: `npx vitest run <file> -t "<name>"`.
  - The count is Vitest's `Tests  N passed (N)` line, which `mutate.mjs` reads already.
  - The scratch worktree links `frontend/node_modules`.
- `backend/api-service/`: Go, `**/*_test.go`. Go runs a package, not a file.
  - One file's package: `go test -v -count=1 "./$(dirname <file>)"`; one test by name:
    `go test -v -count=1 -run '^<name>$' "./$(dirname <file>)"`.
  - The count is the `--- PASS` and `--- FAIL` lines of `-v`, which `mutate.mjs` reads already.
  - The scratch worktree needs nothing: modules come from the shared module cache
    (`go mod download` when it is empty).
  - The `db` package's integration tests skip without a Postgres at `TEST_DATABASE_URL`
    (`./test-local.ps1` starts one), and the SQLite-backed manifest and search tests skip without
    cgo (`CGO_ENABLED=1` and a C compiler). A test that skips itself reads as `zero-match`, not
    proven.
- repo root: `scripts/*.test.mjs`, node's built-in test runner.
  - One file: `node --test <file>`; one test by name:
    `node --test --test-name-pattern="<name>" <file>`, with `<name>` regex-escaped.
  - The count is node's `ℹ tests N` line, which `mutate.mjs` reads already.
  - The scratch worktree needs nothing: the root `package.json` has no dependencies.
  - `scripts/docker-context-isolation.test.mjs` and `scripts/nginx-headers.test.mjs` need a running
    Docker engine (and Buildx for the first).
- `frontend/e2e/`: Playwright, `*.spec.ts`. Not provable here: it needs the `e2e-postgres`
  Compose service and the hermetic servers on 5273 and 8081 (`AGENTS.md`, "Testing and
  Validation").
