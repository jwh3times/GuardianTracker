import { test } from "node:test";
import assert from "node:assert/strict";
import { readdirSync, readFileSync } from "node:fs";
import { extname, join, relative, sep } from "node:path";

const root = join(import.meta.dirname, "..");
const read = (...parts) => readFileSync(join(root, ...parts), "utf8");

const COMPOSE_SERVICES = ["postgres", "test-postgres", "e2e-postgres"];

// `postgres:<major>.<minor>[-alpine<major>.<minor>]@sha256:<64 hex>`. Both refs
// must stay tag-qualified *and* digest-pinned; the variants differ on purpose
// (Compose runs Alpine, the CI service container runs the Debian image), so the
// alignment below compares versions rather than digests.
const PINNED_REF =
  /^postgres:(\d+\.\d+)(-alpine\d+\.\d+)?@sha256:[0-9a-f]{64}$/;

/** Maps each top-level Compose service name to its `image:` value. */
function composeImages(contents) {
  const images = new Map();
  let service = null;

  for (const line of contents.split("\n")) {
    const serviceKey = /^ {2}([A-Za-z0-9_-]+):\s*$/.exec(line);
    if (serviceKey) {
      service = serviceKey[1];
      continue;
    }
    const image = /^ {4}image:\s*(\S+)\s*$/.exec(line);
    if (image && service) images.set(service, image[1]);
  }

  return images;
}

/** The `image:` of a service container declared inside a workflow job. */
function workflowServiceImage(contents, service) {
  const match = new RegExp(
    String.raw`^ {6}${service}:\n(?:[^\n]*\n)*?^ {8}image:\s*(\S+)\s*$`,
    "m",
  ).exec(contents);
  assert.ok(match, `ci-cd.yml must declare a '${service}' service container`);
  return match[1];
}

const compose = composeImages(read("docker-compose.yml"));
const ciRef = workflowServiceImage(
  read(".github", "workflows", "ci-cd.yml"),
  "postgres",
);
const composeRef = compose.get("postgres");

const version = (ref) => {
  const match = PINNED_REF.exec(ref);
  assert.ok(
    match,
    `${ref} must be a tag-qualified, digest-pinned postgres ref`,
  );
  return match[1];
};

test("every Compose PostgreSQL service shares one pinned Alpine ref", () => {
  for (const service of COMPOSE_SERVICES) {
    const ref = compose.get(service);
    assert.ok(ref, `docker-compose.yml must declare a '${service}' service`);
    assert.match(ref, PINNED_REF);
    assert.match(ref, /-alpine\d+\.\d+@/, `${service} must use the Alpine tag`);
    assert.equal(
      ref,
      composeRef,
      `${service} must use the same ref as the 'postgres' service`,
    );
  }
});

// Dependabot's docker-compose ecosystem does not see service-container refs in
// workflow YAML, so a Compose bump would otherwise leave CI running its database
// integration tests against a different server version than local dev and E2E.
test("the CI integration database matches the Compose PostgreSQL version", () => {
  assert.match(ciRef, PINNED_REF);
  assert.doesNotMatch(ciRef, /-alpine/, "CI uses the Debian postgres image");
  assert.equal(version(ciRef), version(composeRef));
});

const TEXT_EXTENSIONS = new Set([
  ".md",
  ".yml",
  ".yaml",
  ".toml",
  ".ps1",
  ".sh",
  ".sql",
  ".go",
  ".ts",
  ".tsx",
]);
const SKIPPED_DIRECTORIES = new Set([
  ".git",
  "node_modules",
  "private",
  "dist",
  "coverage",
  "test-results",
  "playwright-report",
]);

function* textFiles(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) {
      if (SKIPPED_DIRECTORIES.has(entry.name)) continue;
      yield* textFiles(path);
    } else if (
      TEXT_EXTENSIONS.has(extname(entry.name)) ||
      entry.name.startsWith("Dockerfile")
    ) {
      yield path;
    }
  }
}

// A version bump also touches coupling guidance when a tag has been copied into
// prose. This sweep prevents a documented reference from quietly describing a
// retired server.
test("no tracked file references a retired PostgreSQL image", () => {
  const allowed = new Set([
    composeRef,
    composeRef.split("@")[0],
    ciRef,
    ciRef.split("@")[0],
  ]);
  const failures = [];

  for (const path of textFiles(root)) {
    const contents = readFileSync(path, "utf8");
    // Requires the `<major>.<minor>` tag shape so `postgres:5432` host:port
    // pairs inside connection strings are not mistaken for image references.
    for (const [reference] of contents.matchAll(
      /postgres:\d+\.\d+[^\s`'"]*/g,
    )) {
      if (allowed.has(reference)) continue;
      failures.push(
        `${relative(root, path).split(sep).join("/")}: ${reference}`,
      );
    }
  }

  assert.deepEqual(failures, []);
});

test("SETUP.md delegates PostgreSQL pins to their owning configuration", () => {
  const setup = read("SETUP.md");
  assert.ok(setup.includes("Dockerfiles, Compose file, and\nworkflows"));
  assert.ok(setup.includes("repository policy tests"));
  assert.doesNotMatch(setup, /docker buildx imagetools inspect postgres:/);
});
