#!/usr/bin/env bash
# Prints the version the next merge to main will mint (bare SemVer, no "v" prefix).
#
# Single source of truth for the build-number rule. Three consumers:
#   - .github/workflows/version.yml  (creates the tag on merge)
#   - .github/workflows/ci-cd.yml    (Changelog Version guard)
#   - .claude/skills/ship/SKILL.md   (/ship writes this version into CHANGELOG.md)
#
# The rule, unchanged from the original inline implementation:
#   no tags on this major.minor line -> use VERSION's build component
#   VERSION's build > highest tag     -> use VERSION's build component (major/minor bump)
#   otherwise                         -> highest tag's build + 1
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
target=$(tr -d '[:space:]' < "${repo_root}/VERSION")

if ! echo "$target" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'; then
  echo "next-version: VERSION '$target' is not a plain SemVer x.y.z version." >&2
  exit 1
fi

IFS=. read -r major minor configured_build <<< "$target"

escaped_prefix="v${major}\.${minor}"
last=$(git tag --list "v${major}.${minor}.*" \
  | grep -E "^${escaped_prefix}\.[0-9]+$" \
  | sed -E "s/^${escaped_prefix}\.//" \
  | sort -n \
  | tail -1 || true)

if [ -z "$last" ]; then
  build="$configured_build"
elif [ "$configured_build" -gt "$last" ]; then
  build="$configured_build"
else
  build=$(( last + 1 ))
fi

echo "${major}.${minor}.${build}"
