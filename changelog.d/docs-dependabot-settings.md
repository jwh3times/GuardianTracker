### Added

- `docs/agents/dependabot.md` gives the shared `dependabot` review skill this
  repository's facts: the bumps reviewed in full even below a major (a `postgres`
  tag change, `@playwright/test`, the frontend `node` image, `@testing-library/jest-dom`),
  how CI installs each manifest, where each dependency is used, the stack facts a
  verdict must not get wrong, and the known fix for each bump that arrives red
  (`npm run sync:image-pins` for Playwright and Node, the CI service-container
  realignment for PostgreSQL). The docs map lists it. No application behavior
  changes.
