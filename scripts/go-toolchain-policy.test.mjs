import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const root = join(import.meta.dirname, "..");
const read = (...parts) => readFileSync(join(root, ...parts), "utf8");

// Staticcheck is compiled by whatever Go is active. Built by a toolchain newer
// than the one it supports, it cannot decode the standard library's export data
// and fails while loading — naming only stdlib packages, analyzing nothing, and
// leaving output that looks exactly like a clean run. Every documented local
// invocation therefore carries an explicit GOTOOLCHAIN, and this file is what
// keeps that value from drifting away from the Go the project actually pins.
const STATICCHECK = "go run honnef.co/go/tools/cmd/staticcheck@";

/** Files documenting a command a developer runs on their own machine. */
const LOCAL_COMMAND_DOCS = [
  "AGENTS.md",
  "SETUP.md",
  join(".claude", "agents", "go-services.md"),
  join(".codex", "agents", "go-services.toml"),
];

/** Workflows whose GO_VERSION the documented pin must match. */
const WORKFLOWS = [
  join(".github", "workflows", "ci-cd.yml"),
  join(".github", "workflows", "browser.yml"),
];

function goVersion(file) {
  const match = /^\s*GO_VERSION:\s*"([\d.]+)"\s*$/m.exec(read(file));
  assert.ok(match, `${file} must declare GO_VERSION`);
  return match[1];
}

test("both workflows pin the same Go version", () => {
  const [ci, ...rest] = WORKFLOWS.map(goVersion);
  for (const [i, version] of rest.entries()) {
    assert.equal(
      version,
      ci,
      `${WORKFLOWS[i + 1]} pins Go ${version} but ${WORKFLOWS[0]} pins ${ci}; ` +
        `the documented Staticcheck pin can only match one of them`,
    );
  }
});

test("the Go module's toolchain matches the CI pin", () => {
  const gomod = read(join("backend", "api-service", "go.mod"));
  const match = /^toolchain go([\d.]+)$/m.exec(gomod);
  assert.ok(match, "backend/api-service/go.mod must declare a toolchain");
  assert.equal(
    match[1],
    goVersion(WORKFLOWS[0]),
    "go.mod's toolchain and the workflows' GO_VERSION must name the same Go release",
  );
});

/** Dockerfiles whose `golang` builder compiles a binary that ships. */
const GO_DOCKERFILES = [join("backend", "api-service", "Dockerfile")];

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

// The tests run on GO_VERSION through actions/setup-go; only `Build Docker
// Images` compiles with the builder image, and that binary is the one that
// ships. Dependabot's docker ecosystem moves the golang tag on its own, so
// without this check a builder bump drifts silently away from the tested Go.
test("every Go builder image compiles with the CI Go release", () => {
  const expected = goVersion(WORKFLOWS[0]);

  for (const file of GO_DOCKERFILES) {
    const versions = golangBuilderVersions(read(file));
    assert.ok(
      versions.length > 0,
      `${file} no longer builds FROM golang; ` +
        `remove it from GO_DOCKERFILES or restore the builder stage`,
    );
    for (const version of versions) {
      assert.equal(
        version,
        expected,
        `${file} builds with golang:${version ?? "<floating tag>"} but the tests ` +
          `run on GO_VERSION ${expected}; the shipped binary must be compiled by ` +
          `the Go release CI tested. Move the builder tag and digest, go.mod's ` +
          `toolchain, both workflows' GO_VERSION and the GOTOOLCHAIN= lines together.`,
      );
    }
  }
});

test("every documented local Staticcheck command pins the toolchain", () => {
  const expected = `GOTOOLCHAIN=go${goVersion(WORKFLOWS[0])}`;

  for (const file of LOCAL_COMMAND_DOCS) {
    const contents = read(file);
    const lines = contents
      .split("\n")
      .filter((line) => line.includes(STATICCHECK));

    assert.ok(
      lines.length > 0,
      `${file} no longer documents the Staticcheck command; ` +
        `remove it from LOCAL_COMMAND_DOCS or restore the command`,
    );

    for (const line of lines) {
      assert.ok(
        line.includes(expected),
        `${file} runs Staticcheck without ${expected}:\n  ${line.trim()}\n` +
          `An unpinned run analyzes nothing and reports it as success.`,
      );
    }
  }
});

// The workflow's own invocation is deliberately exempt: the job has already
// installed the pinned Go through actions/setup-go, so a second pin on the
// command line would be a duplicate that could itself drift. This test exists so
// that exemption is a recorded decision rather than an oversight.
test("the CI workflow relies on setup-go rather than a second pin", () => {
  const workflow = read(WORKFLOWS[0]);
  const line = workflow
    .split("\n")
    .find((candidate) => candidate.includes(STATICCHECK));

  assert.ok(line, "ci-cd.yml must still run Staticcheck");
  assert.ok(
    !line.includes("GOTOOLCHAIN="),
    "ci-cd.yml pins Go through actions/setup-go; a second inline pin would be a " +
      "duplicate source of truth",
  );
  assert.match(
    workflow,
    /go-version:\s*\$\{\{\s*env\.GO_VERSION\s*\}\}/,
    "the Go job must install GO_VERSION through actions/setup-go",
  );
});
