### Changed

- The CI agent doc's branch-protection section now matches the live ruleset:
  it covers `main` and `release/**`, and names strict up-to-date checks, merge
  commits as the only merge method, required review-thread resolution, Copilot
  code review on every push, the code-quality rule, and the deletion and
  force-push protections, linking the merging guide for what they mean when
  landing.
- The `postgres-specialist` agent calls its example a pinned version rather
  than a digest, and points at the `docker-containers` agent's "Pin
  maintenance" section, where the PostgreSQL pin guidance lives, instead of a
  "When updating PostgreSQL" section that does not exist.
