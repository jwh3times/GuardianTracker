# Releases

## Scheme

per-merge — `.github/workflows/version.yml` tags every merge to `main`
`v<major>.<minor>.<build>` with a GitHub Release, and each merge's dated changelog section names
the version it mints (ADR 0005, ADR 0009).

## Skipped tags

backfill — a merged Dependabot PR (exempt from **Changelog Version**) or a PR opened in the web UI
gets its section on the next ship.

- **Ignore:** `v0.*`, `v1.0.*` and `v1.1.*` — released sections archived in `docs/changelog/`
  (`0.x.md`, `1.0-1.1.md`), not missing; and the 4-part `v0.1.0.x` tags. Archiving another line
  adds its pattern here.

## Version file

`VERSION` — the bare `<major>.<minor>.<build>` target line

## Bump command

`printf '%s\n' <version> > VERSION`

## Ask before bumping

when ambiguous

## Next-version command

`bash scripts/next-version.sh`

## Changelog check

`test "$(bash scripts/next-version.sh)" = "$(sed -nE 's/^## \[([0-9]+\.[0-9]+\.[0-9]+)\].*/\1/p' CHANGELOG.md | head -1)" && node --test scripts/changelog-footer-policy.test.mjs`
— the required **Changelog Version** check's comparison in `.github/workflows/ci-cd.yml`, then the
footer policy **Format Check** enforces (every section linked, each comparing from the one below,
`[Unreleased]` from the newest).
