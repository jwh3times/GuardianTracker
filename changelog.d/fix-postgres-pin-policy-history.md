### Fixed

- A PostgreSQL tag bump can go green again: the PostgreSQL pin policy
  (`scripts/postgres-pin-policy.test.mjs`) no longer reports the tags quoted in the release history
  (`CHANGELOG.md`, `changelog.d/`, `docs/changelog/`) as retired references. Released sections
  record the image each release ran and no branch may edit them, so every tag bump failed
  `Format Check` with no way to fix it (#515). The sweep still covers every other tracked text
  file. No application behavior changes.
