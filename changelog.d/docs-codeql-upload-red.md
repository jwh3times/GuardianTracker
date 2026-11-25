### Changed

- `docs/agents/merging.md` now covers a CodeQL job that fails at "Uploading results" with no error
  line and no alerts. That is GitHub's upload failing, not the code. A default-setup CodeQL run
  cannot be re-run ("This workflow run cannot be retried"), so an empty commit on the PR branch
  starts a fresh analysis. Until one registers, the code-scanning rule keeps the PR from merging.
