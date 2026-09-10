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
