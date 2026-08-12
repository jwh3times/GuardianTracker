import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const root = join(import.meta.dirname, "..");
const read = (...parts) => readFileSync(join(root, ...parts), "utf8");

test("local, CI, and Docker tooling use one exact Node 26 patch", () => {
  const version = read(".nvmrc").trim();
  assert.match(version, /^26\.\d+\.\d+$/);

  for (const workflow of ["ci-cd.yml", "browser.yml"]) {
    const contents = read(".github", "workflows", workflow);
    const setupNodeCount = [...contents.matchAll(/uses: actions\/setup-node@/g)]
      .length;
    const versionFileCount = [
      ...contents.matchAll(/node-version-file: \.nvmrc/g),
    ].length;

    assert.ok(setupNodeCount > 0, `${workflow} must set up Node`);
    assert.equal(
      versionFileCount,
      setupNodeCount,
      `${workflow} must read every Node version from .nvmrc`,
    );
    assert.doesNotMatch(contents, /node-version:\s|NODE_VERSION:/);
  }

  const imagePattern = new RegExp(
    `^FROM node:${version.replaceAll(".", "\\.")}-alpine3\\.24@sha256:[0-9a-f]{64}(?: AS builder)?$`,
    "m",
  );
  assert.match(read("frontend", "Dockerfile"), imagePattern);
  assert.match(read("frontend", "Dockerfile.dev"), imagePattern);
});

test("frontend package metadata supports only the Node 26 line", () => {
  const manifest = JSON.parse(read("frontend", "package.json"));
  const lockfile = JSON.parse(read("frontend", "package-lock.json"));

  assert.equal(manifest.engines?.node, ">=26.0.0 <27.0.0");
  assert.equal(lockfile.packages[""].engines?.node, manifest.engines.node);
  assert.match(manifest.devDependencies["@types/node"], /^\^26\./);
  assert.match(read("frontend", ".npmrc"), /^engine-strict=true\s*$/);
});
