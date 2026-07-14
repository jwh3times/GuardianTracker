#!/usr/bin/env bash
# Tests scripts/next-version.sh against throwaway git repos with synthetic tags.
# Run: bash scripts/next-version.test.sh
set -uo pipefail

script_under_test="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/next-version.sh"
failures=0

# run_case <name> <VERSION contents> <expected output> [tags...]
run_case() {
  name="$1"; version="$2"; expected="$3"; shift 3
  tmp=$(mktemp -d)
  (
    cd "$tmp" || exit 1
    git init -q .
    git config user.email "test@example.com"
    git config user.name "Test"
    echo "$version" > VERSION
    mkdir -p scripts
    cp "$script_under_test" scripts/next-version.sh
    git add -A
    git commit -qm "init"
    for tag in "$@"; do git tag "$tag"; done

    actual=$(bash scripts/next-version.sh)
    if [ "$actual" = "$expected" ]; then
      echo "PASS  $name (got $actual)"
    else
      echo "FAIL  $name: expected '$expected', got '$actual'"
      exit 1
    fi
  ) || failures=$((failures + 1))
  rm -rf "$tmp"
}

run_case "no tags on this line uses VERSION build"  "0.4.0"  "0.4.0"
run_case "existing tags increment the max"          "0.3.0"  "0.3.22" v0.3.20 v0.3.21
run_case "VERSION bumped past tags wins"            "0.3.30" "0.3.30" v0.3.20 v0.3.21
run_case "non-contiguous tags use max not count"    "0.3.0"  "0.3.10" v0.3.1 v0.3.9
run_case "other version lines are ignored"          "0.3.0"  "0.3.2"  v0.3.1 v1.5.99
run_case "glob-admitted but regex-invalid tags are rejected" "0.3.0" "0.3.2" v0.3.1 v0.3.1.2 v0.3.0-rc1
run_case "double-digit builds sort numerically"     "0.3.0"  "0.3.24" v0.3.9 v0.3.23

if [ "$failures" -ne 0 ]; then
  echo ""
  echo "$failures test case(s) failed"
  exit 1
fi
echo ""
echo "All next-version tests passed"
