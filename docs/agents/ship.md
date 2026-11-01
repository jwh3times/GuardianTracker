# Ship

## Fast checks

Together they cover what the required checks run that can run locally. Root Markdown Prettier and
the frontend checks need `frontend/node_modules` (`npm ci` in `frontend/`).

- repo root: `./frontend/node_modules/.bin/prettier --check "**/*.md"` (**Format Check**), fixed
  by the same command with `--write`
- `frontend/`, only when `frontend/` changed: `npm run format:check` (**Format Check**), fixed by
  `npm run format`
- `frontend/`, only when `frontend/` changed: `npm run type-check` and `npm run lint`
  (**Test Frontend**)
- `frontend/`, only when `frontend/` changed: `npm test -- --coverage` (**Test Frontend**'s
  Vitest suite, about 10 seconds)
- `backend/api-service/`, only when `backend/` changed: `test -z "$(gofmt -l .)"`
  (**Format Check**), fixed by `gofmt -w .`; then `go vet ./...` (**Test Go Services**). Without
  a Go toolchain both are left to CI.
- repo root: `npm run sync:agents:check` (**Format Check**), fixed by the root Prettier `--write`
  then `npm run sync:agents`, committing the regenerated mirrors
- repo root: `npm run lint:agents` (**Format Check**)
- repo root: `npm run test:policy` (**Format Check**'s policy tests, about 20 seconds)
- repo root: `node scripts/check-docs.mjs` (**Format Check**)
- repo root, only when `scripts/next-version.mjs` or `scripts/check-changelog.mjs` changed:
  `node --test scripts/next-version.test.mjs scripts/check-changelog.test.mjs` (**Changelog Version**)
- repo root, only when a `Dockerfile*`, `.dockerignore` or `frontend/nginx.conf` changed, with
  Docker running: `npm run test:docker-context` and `npm run test:nginx-headers`
  (**Build Docker Images**)

The production build (**Test Frontend**), Staticcheck, `govulncheck` and
`go test -race` against Postgres (**Test Go Services**), the image builds (**Build Docker
Images**), **Browser E2E + Axe**, the non-required Windows portability job and CodeQL run only in
CI.
