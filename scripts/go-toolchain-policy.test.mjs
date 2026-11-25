import { test } from "node:test";
import assert from "node:assert/strict";
import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import {
  GOTOOLCHAIN_PINS,
  goRelease,
  gotoolchainReleases,
} from "./sync-image-pins.mjs";

const root = join(import.meta.dirname, "..");
const read = (...parts) => readFileSync(join(root, ...parts), "utf8");

// One Go release, written once: the API module's toolchain directive (or its go
// directive, when Go has dropped the toolchain line). CI's actions/setup-go reads
// it from go.mod; `npm run sync:image-pins` writes it into the builder image and
// the documented Staticcheck commands; this file checks every copy against it,
// offline, and names the fix. A Go bump is `go mod edit -toolchain=go<X.Y.Z>` in
// backend/api-service, then `npm run sync:image-pins` (#544, #564).
const GOMOD = "backend/api-service/go.mod";
const SYNC = "npm run sync:image-pins";

const expected = () => goRelease(read(GOMOD));

// Staticcheck is compiled by whatever Go is active. Built by a toolchain newer
// than the one it supports, it cannot decode the standard library's export data
// and fails while loading — naming only stdlib packages, analyzing nothing, and
// leaving output that looks exactly like a clean run. Every documented local
// invocation therefore carries an explicit GOTOOLCHAIN.
const STATICCHECK = "go run honnef.co/go/tools/cmd/staticcheck@";

/**
 * Every file carrying a `GOTOOLCHAIN=go<X.Y.Z>` pin, and how many. This list is
 * the gate's own, not imported from the sync script, so dropping a file from the
 * script cannot also drop it from the check; a test below requires the script's
 * GOTOOLCHAIN_PINS to cover every authored file here. `everyCommandPinned` is
 * false only for the allowlist, which also permits CI's unpinned form.
 */
const GOTOOLCHAIN_FILES = [
  { file: "AGENTS.md", occurrences: 1, everyCommandPinned: true },
  { file: "SETUP.md", occurrences: 1, everyCommandPinned: true },
  {
    file: ".claude/agents/go-services.md",
    occurrences: 1,
    everyCommandPinned: true,
  },
  // Generated from the agent file by `npm run sync:agents`, so the sync script
  // does not write it.
  {
    file: ".codex/agents/go-services.toml",
    occurrences: 1,
    everyCommandPinned: true,
    generated: true,
  },
  { file: ".claude/settings.json", occurrences: 1, everyCommandPinned: false },
];

/** Dockerfiles whose `golang` builder compiles a binary that ships. */
const GO_DOCKERFILES = ["backend/api-service/Dockerfile"];

const WORKFLOW_DIR = ".github/workflows";
const workflows = () =>
  readdirSync(join(root, WORKFLOW_DIR))
    .filter((name) => /\.ya?ml$/.test(name))
    .map((name) => `${WORKFLOW_DIR}/${name}`);

/**
 * The Go release each `FROM golang:<tag>` line in a Dockerfile names. A tag
 * whose version is not a full `X.Y.Z` release (`golang:1.26-alpine`,
 * `golang:alpine`) yields null, so the caller can reject it: a floating tag
 * picks up a new Go release without any change in this repository.
 */
function golangBuilderVersions(dockerfile) {
  return [
    ...dockerfile.matchAll(/^\s*FROM\s+(?:--\S+\s+)*golang:(\S+)/gim),
  ].map(([, ref]) => /^(\d+\.\d+\.\d+)(?:-|@|$)/.exec(ref)?.[1] ?? null);
}

/**
 * Each YAML list item in `text`, whole: from its `- ` line to the last line
 * indented deeper than that dash. Nested items are returned too, as their own
 * entries, so a caller filters for the items it means.
 */
function listItems(text) {
  const lines = text.split("\n");
  const items = [];
  lines.forEach((line, index) => {
    const dash = /^(\s*)-\s/.exec(line);
    if (!dash) return;
    const item = [line];
    for (const next of lines.slice(index + 1)) {
      if (next.trim() !== "" && next.search(/\S/) <= dash[1].length) break;
      item.push(next);
    }
    items.push(item.join("\n"));
  });
  return items;
}

/**
 * Each actions/setup-go step in a workflow, whole: keys written before `uses:`
 * (a `with:` block first, say) belong to it too, and the action may be quoted.
 */
function setupGoSteps(workflow) {
  return listItems(workflow).filter((item) =>
    /^\s*(?:-\s+)?uses:\s*["']?actions\/setup-go@/m.test(item),
  );
}

test("golangBuilderVersions reads the release from each golang FROM line", () => {
  assert.deepEqual(
    golangBuilderVersions(
      "FROM golang:1.26.11-alpine@sha256:abc AS builder\nFROM alpine:3.24.4\n",
    ),
    ["1.26.11"],
  );
  assert.deepEqual(
    golangBuilderVersions(
      "FROM --platform=$BUILDPLATFORM golang:1.27.4 AS b\n",
    ),
    ["1.27.4"],
  );
  // A digest pin with no variant suffix still names a full release.
  assert.deepEqual(
    golangBuilderVersions("FROM golang:1.27.4@sha256:abc AS b\n"),
    ["1.27.4"],
  );
  // Docker reads instructions case-insensitively, and a commented-out FROM
  // is not a build stage.
  assert.deepEqual(
    golangBuilderVersions(
      "# FROM golang:1.20.0 AS old\nfrom golang:1.27.4 AS b\n",
    ),
    ["1.27.4"],
  );
  assert.deepEqual(golangBuilderVersions("FROM golang:1.26-alpine AS b\n"), [
    null,
  ]);
  assert.deepEqual(golangBuilderVersions("FROM golang:alpine AS b\n"), [null]);
  assert.deepEqual(golangBuilderVersions("FROM alpine:3.24.4\n"), []);
});

test("setupGoSteps reads each setup-go step to its end", () => {
  const workflow = [
    "jobs:",
    "  test:",
    "    steps:",
    "      - name: Set up Go",
    "        uses: actions/setup-go@abc # v7",
    "        with:",
    "          go-version-file: backend/api-service/go.mod",
    "",
    "      - name: Test",
    "        run: go test ./...",
    "      - uses: actions/setup-go@abc",
    "        with:",
    "          go-version: 1.27.4",
    "      - with:",
    "          go-version: 1.27.3",
    '        uses: "actions/setup-go@abc"',
    "      - run: echo done",
    "",
  ].join("\n");
  const steps = setupGoSteps(workflow);
  assert.equal(steps.length, 3);
  assert.match(steps[0], /go-version-file: backend\/api-service\/go\.mod/);
  assert.doesNotMatch(steps[0], /go test/, "the next step is not part of it");
  assert.match(steps[1], /go-version: 1\.27\.4/);
  // Keys before `uses:` belong to the step, and a quoted action still counts.
  assert.match(steps[2], /go-version: 1\.27\.3/);
  assert.doesNotMatch(steps[2], /echo done/);
});

// The tests run on the Go that setup-go installs; only `Build Docker Images`
// compiles with the builder image, and that binary is the one that ships.
// Dependabot's docker ecosystem moves the golang tag on its own, so without this
// check a builder bump drifts silently away from the tested Go.
test("every Go builder image compiles with go.mod's Go release", () => {
  const release = expected();

  for (const file of GO_DOCKERFILES) {
    const versions = golangBuilderVersions(read(file));
    assert.ok(
      versions.length > 0,
      `${file} no longer builds FROM golang; ` +
        `remove it from GO_DOCKERFILES or restore the builder stage`,
    );
    for (const version of versions) {
      const fix =
        version === null
          ? `Run ${SYNC} to pin golang:${release}-alpine and its digest.`
          : `If the Dockerfile's release is the one you want (a Dependabot bump), ` +
            `run \`go mod edit -toolchain=go${version}\` in backend/api-service, ` +
            `then ${SYNC}. Otherwise run ${SYNC} to put the builder back on ` +
            `${release}.`;
      assert.equal(
        version,
        release,
        `${file} builds with golang:${version ?? "<floating tag>"} but ${GOMOD} ` +
          `names Go ${release}, which CI tests on; the shipped binary must be ` +
          `compiled by the Go release CI tested. ${fix}`,
      );
    }
  }
});

test("the sync script rewrites every authored GOTOOLCHAIN= pin", () => {
  // A file the gate checks but the script skips is exactly the hand-edit the
  // script exists to remove.
  const key = ({ file, occurrences }) => `${file} x${occurrences}`;
  assert.deepEqual(
    GOTOOLCHAIN_PINS.map(key).sort(),
    GOTOOLCHAIN_FILES.filter((entry) => !entry.generated)
      .map(key)
      .sort(),
    "GOTOOLCHAIN_PINS in scripts/sync-image-pins.mjs must list every authored " +
      "file in this test's GOTOOLCHAIN_FILES",
  );
});

test("every GOTOOLCHAIN= pin names go.mod's Go release", () => {
  const release = expected();

  for (const { file, occurrences } of GOTOOLCHAIN_FILES) {
    const releases = gotoolchainReleases(read(file));
    assert.equal(
      releases.length,
      occurrences,
      `${file} should carry ${occurrences} GOTOOLCHAIN=go<X.Y.Z> pin(s), found ` +
        `${releases.length}; update GOTOOLCHAIN_FILES here and GOTOOLCHAIN_PINS in ` +
        `scripts/sync-image-pins.mjs, or restore the pin`,
    );
    for (const pinned of releases) {
      assert.equal(
        pinned,
        release,
        `${file} pins GOTOOLCHAIN=go${pinned} but ${GOMOD} names Go ${release}. ` +
          `Run ${SYNC}, then npm run sync:agents.`,
      );
    }
  }
});

test("every documented local Staticcheck command carries its pin", () => {
  const docs = GOTOOLCHAIN_FILES.filter((entry) => entry.everyCommandPinned);

  for (const { file } of docs) {
    const lines = read(file)
      .split("\n")
      .filter((line) => line.includes(STATICCHECK));

    assert.ok(
      lines.length > 0,
      `${file} no longer documents the Staticcheck command; ` +
        `remove it from GOTOOLCHAIN_FILES or restore the command`,
    );
    for (const line of lines) {
      assert.match(
        line,
        /GOTOOLCHAIN=go\d+\.\d+\.\d+ /,
        `${file} runs Staticcheck without a GOTOOLCHAIN pin:\n  ${line.trim()}\n` +
          `An unpinned run analyzes nothing and reports it as success.`,
      );
    }
  }
});

test("every setup-go step installs the Go release go.mod names", () => {
  let total = 0;
  for (const file of workflows()) {
    for (const step of setupGoSteps(read(file))) {
      total += 1;
      assert.match(
        step,
        /^\s*go-version-file:\s*["']?backend\/api-service\/go\.mod["']?\s*$/m,
        `${file} has a setup-go step that does not read ${GOMOD}:\n${step}`,
      );
      assert.doesNotMatch(
        step,
        /^\s*go-version:/m,
        `${file} has a setup-go step with a go-version input; ${GOMOD} is the only ` +
          `place the Go release is written`,
      );
    }
  }
  assert.ok(total > 0, "no workflow sets up Go; this check would be vacuous");
});

test("no workflow keeps a second copy of the Go release", () => {
  for (const file of workflows()) {
    assert.doesNotMatch(
      read(file),
      /GO_VERSION/,
      `${file} mentions GO_VERSION; setup-go reads the Go release from ${GOMOD}`,
    );
  }
});

// Dependabot proposes only the highest golang tag, so once a new Go minor ships
// a patch for the current one would never arrive as a PR. A new minor can also
// need a Staticcheck release that supports it, so moving minors is deliberate.
test("Dependabot proposes only patch releases of the golang builder", () => {
  // The innermost list items holding a key: an outer item contains the line too.
  const owning = (items, key) => {
    const matching = items.filter((item) => key.test(item));
    return matching.filter(
      (item) =>
        !matching.some((other) => other !== item && item.includes(other)),
    );
  };
  const items = listItems(read(".github/dependabot.yml"));
  const docker = owning(
    items,
    /^\s*(?:-\s+)?package-ecosystem:\s*["']?docker["']?\s*$/m,
  );
  assert.equal(
    docker.length,
    1,
    "dependabot.yml must keep one docker ecosystem entry",
  );
  const golang = owning(
    listItems(docker[0]),
    /^\s*(?:-\s+)?dependency-name:\s*["']?golang["']?\s*$/m,
  );
  assert.equal(
    golang.length,
    1,
    'the docker ecosystem must carry one ignore entry for "golang"',
  );
  assert.deepEqual(
    [...golang[0].matchAll(/version-update:semver-\w+/g)]
      .map(([type]) => type)
      .sort(),
    ["version-update:semver-major", "version-update:semver-minor"],
    'the "golang" ignore must cover minor and major updates, and only those',
  );
});

// The workflow's own invocation is deliberately exempt: the job has already
// installed go.mod's release through actions/setup-go, so a second pin on the
// command line would be a duplicate that could itself drift. This test exists so
// that exemption is a recorded decision rather than an oversight.
test("the CI Staticcheck step relies on setup-go rather than a second pin", () => {
  const line = read(`${WORKFLOW_DIR}/ci-cd.yml`)
    .split("\n")
    .find((candidate) => candidate.includes(STATICCHECK));

  assert.ok(line, "ci-cd.yml must still run Staticcheck");
  assert.ok(
    !line.includes("GOTOOLCHAIN="),
    "ci-cd.yml installs Go from go.mod through actions/setup-go; a second inline " +
      "pin would be a duplicate source of truth",
  );
});
