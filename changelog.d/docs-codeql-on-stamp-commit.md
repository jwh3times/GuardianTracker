### Changed

- The merging guide notes that GitHub's default-setup CodeQL scan still runs on the
  `chore(release): v<x.y.z> [skip ci]` stamp commit, because `[skip ci]` does not suppress it, so
  that run is expected rather than a sign the skip failed.
