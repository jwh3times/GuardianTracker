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

  const playwrightDockerfile = read("frontend", "Dockerfile.playwright");
  const playwrightNodePattern = new RegExp(
    `^ARG NODE_IMAGE=node:${version.replaceAll(".", "\\.")}-bookworm-slim@sha256:[0-9a-f]{64}$`,
    "m",
  );
  assert.match(playwrightDockerfile, playwrightNodePattern);
  const lockfile = JSON.parse(read("frontend", "package-lock.json"));
  const playwrightVersion =
    lockfile.packages["node_modules/@playwright/test"].version;
  assert.match(
    playwrightDockerfile,
    new RegExp(
      `^ARG PLAYWRIGHT_IMAGE=mcr\\.microsoft\\.com/playwright:v${playwrightVersion.replaceAll(".", "\\.")}-noble@sha256:[0-9a-f]{64}$`,
      "m",
    ),
  );
  assert.match(
    playwrightDockerfile,
    /^ENV PATH="\/opt\/node\/bin:\$\{PATH\}"$/m,
  );

  const browserWorkflow = read(".github", "workflows", "browser.yml");
  assert.match(browserWorkflow, /--file frontend\/Dockerfile\.playwright/);
  assert.match(browserWorkflow, /guardian-tracker\/playwright-node26:ci/);
});

test("frontend package metadata supports only the Node 26 line", () => {
  const manifest = JSON.parse(read("frontend", "package.json"));
  const lockfile = JSON.parse(read("frontend", "package-lock.json"));

  assert.equal(manifest.engines?.node, ">=26.0.0 <27.0.0");
  assert.equal(lockfile.packages[""].engines?.node, manifest.engines.node);
  assert.match(manifest.devDependencies["@types/node"], /^\^26\./);
  assert.match(read("frontend", ".npmrc"), /^engine-strict=true\s*$/);
});
