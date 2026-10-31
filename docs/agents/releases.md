# Releases

## Scheme

per-merge — `.github/workflows/version.yml` tags every merge to `main`
`v<major>.<minor>.<build>` with a GitHub Release, and each merge's dated changelog section names
the version it mints (ADR 0005, ADR 0009).

## Skipped tags

backfill — a merged Dependabot PR (exempt from **Changelog Version**) or a PR opened in the web UI
gets its section on the next ship.

- **Ignore:** `v0.*`, `v1.0.*` and `v1.1.*` — released sections archived in `docs/changelog/`
  (`0.x.md`, `1.0-1.1.md`), not missing. Archiving another line adds its pattern here.

## Version file

`VERSION` — the bare `<major>.<minor>.<build>` target line

## Ask before bumping

when ambiguous

## Changelog check

`node scripts/check-changelog.mjs && node --test scripts/changelog-footer-policy.test.mjs`
— the required **Changelog Version** check's comparison in `.github/workflows/ci-cd.yml`, then the
footer policy **Format Check** enforces (every section linked, each comparing from the one below,
`[Unreleased]` from the newest).
