# Releases

## Scheme

stamp-on-tag — `.github/workflows/version.yml` tags every merge to `main`
`v<major>.<minor>.<build>` with a GitHub Release. Each branch's entries live in its own
`changelog.d/<branch-slug>.md` fragment; the workflow stamps them into the merge's dated section
and tags the merge commit (ADR 0027).

## Version file

`VERSION` — the bare `<major>.<minor>.<build>` target line

## Ask before bumping

when ambiguous

## Changelog check

`node scripts/check-changelog.mjs && node --test scripts/changelog-footer-policy.test.mjs`
— the required **Changelog Version** check in `.github/workflows/ci-cd.yml`, then the footer
policy **Format Check** enforces (every section linked, each comparing from the one below,
`[Unreleased]` from the newest).
