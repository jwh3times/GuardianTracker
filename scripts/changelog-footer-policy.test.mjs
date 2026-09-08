import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const root = join(import.meta.dirname, "..");
const changelog = readFileSync(join(root, "CHANGELOG.md"), "utf8");

const COMPARE = "https://github.com/jwh3times/GuardianTracker/compare";

/** Released versions, newest first, in the order their sections appear. */
const versions = [...changelog.matchAll(/^## \[(\d+\.\d+\.\d+)\]/gm)].map(
  (match) => match[1],
);

/** The reference-link definitions at the foot of the file. */
const definitions = new Map(
  [...changelog.matchAll(/^\[(\d+\.\d+\.\d+)\]:\s*(\S+)$/gm)].map((match) => [
    match[1],
    match[2],
  ]),
);

// `Changelog Version` checks the top heading against scripts/next-version.sh and
// never looks down here, so without this test the footer drifts one version per
// release: a heading with no definition renders as plain text instead of a link,
// and `[Unreleased]` keeps comparing from a stale tag.
test("every released version has a matching reference-link definition", () => {
  assert.ok(versions.length > 0, "CHANGELOG.md must list released versions");

  const undefined_ = versions.filter((version) => !definitions.has(version));
  assert.deepEqual(
    undefined_,
    [],
    `CHANGELOG.md has version headings with no reference-link definition: ${undefined_.join(", ")}. Add "[<version>]: ${COMPARE}/v<previous>...v<version>" to the footer.`,
  );

  const orphaned = [...definitions.keys()].filter(
    (version) => !versions.includes(version),
  );
  assert.deepEqual(
    orphaned,
    [],
    `CHANGELOG.md defines reference links for versions it has no section for: ${orphaned.join(", ")}. Older releases live in docs/changelog/, and their definitions belong there too.`,
  );
});

test("each definition compares the previous release against its own tag", () => {
  for (const [index, version] of versions.entries()) {
    const url = definitions.get(version);
    const previous = versions[index + 1];

    if (previous) {
      assert.equal(
        url,
        `${COMPARE}/v${previous}...v${version}`,
        `[${version}] must compare from v${previous}, the section directly below it`,
      );
      continue;
    }

    // The oldest section in this file: its predecessor's section was moved to
    // docs/changelog/, so the "from" tag cannot be derived here. Check the shape
    // and that it still targets this version's own tag.
    assert.ok(
      url.startsWith(`${COMPARE}/v`) && url.endsWith(`...v${version}`),
      `[${version}] must be a compare link ending at v${version}, got ${url}`,
    );
  }
});

test("the Unreleased link compares from the newest release", () => {
  const unreleased = changelog.match(/^\[Unreleased\]:\s*(\S+)$/m);

  assert.ok(unreleased, "CHANGELOG.md must define an [Unreleased] link");
  assert.equal(
    unreleased[1],
    `${COMPARE}/v${versions[0]}...HEAD`,
    `[Unreleased] must compare from v${versions[0]}, the newest released section. /ship rewrites this line every release.`,
  );
});
