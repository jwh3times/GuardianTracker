### Changed

- The API module's `go.mod` is now the one place the Go release is written: its `toolchain` line,
  or its `go` line once Go drops the `toolchain` line, always an exact `X.Y.Z`. CI's `setup-go`
  steps read it with `go-version-file`, and `GO_VERSION` is gone from both workflows.
  `npm run sync:image-pins` now also writes the `golang` builder image's tag and digest and every
  documented `GOTOOLCHAIN=` Staticcheck pin from it, so moving Go is
  `go mod edit -toolchain=go<X.Y.Z>` in `backend/api-service/`, then `npm run sync:image-pins` and
  `npm run sync:agents`. `scripts/go-toolchain-policy.test.mjs` fails **Format Check** when any
  copy drifts and names that fix (#564).
- Dependabot now proposes only patch releases of the `golang` builder image. It proposes only the
  highest tag, so a new Go minor would otherwise hide the next patch of the current one. Moving to
  a new Go minor, which can also need a newer Staticcheck, is a deliberate change.
- The Playwright image's `node:26.13.1-bookworm-slim` base was republished under the same tag; its
  digest in `frontend/Dockerfile.playwright` is refreshed. The visual baselines render in this
  image.
