### Fixed

- `scripts/postgres-pin-policy.test.mjs` now catches a broken release-history exemption in its
  retired PostgreSQL tag sweep (#546). Dropping the trailing `/` from `changelog.d/`, or matching
  `changelog.d/` anywhere in a path, used to pass, because no negative case shared the prefix
  without being inside the folder. Deleting the exemption from the sweep was caught only while
  `CHANGELOG.md` happened to quote a retired tag. The sweep is now `retiredReferences`, and a
  fixture test checks that release history is skipped and that the same tag anywhere else is
  reported.
