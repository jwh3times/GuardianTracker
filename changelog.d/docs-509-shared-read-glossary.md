### Changed

- `CONTEXT.md` now defines **Shared read** (`sharedread.Group`), the seam
  through which concurrent reads for one key join a single detached load, a
  **flight**, and says what the Group owns and what its owners keep. The
  architecture doc's Collections, Records and request-logging passages use
  "flight" where they said "in-flight registry" or "shared load", and link the
  new term.
