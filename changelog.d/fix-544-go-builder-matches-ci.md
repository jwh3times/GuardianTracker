### Fixed

- The API service is now tested on the same Go release that builds the image that ships. The
  Docker builder had moved to Go 1.27.4 through Dependabot while every CI test still ran on Go
  1.26.11. `go.mod`'s `toolchain`, `GO_VERSION` in both workflows and the documented
  `GOTOOLCHAIN=` lines now name 1.27.4, and `scripts/go-toolchain-policy.test.mjs` fails
  **Format Check** when a `golang` builder tag in `backend/api-service/Dockerfile` names any other
  release, or a floating tag. A Dependabot `golang` bump now needs the other pins moved on its
  branch (`docs/agents/dependabot.md`). The shipped binary is unchanged (#544).

### Changed

- Staticcheck moved from 2026.1 to 2026.2.2 in CI and the documented local command. 2026.1 cannot
  read Go 1.27's standard library and analyzes nothing on it; 2026.2 added Go 1.27 support.
