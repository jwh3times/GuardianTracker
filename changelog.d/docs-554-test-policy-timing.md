### Fixed

- The open-pr settings' Fast checks list now says the repository policy
  tests (`npm run test:policy`) take about 40 seconds on CI and up to two
  minutes on Windows, instead of about 20 seconds. They stay in the local
  list, which keeps a suite that finishes in about two minutes.
