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

/**
 * The changelog, its archived sections and its unreleased fragments: they record
 * what each release ran, so the retired-tag sweep leaves them alone.
 */
export function isReleaseHistory(file) {
  return (
    file === "CHANGELOG.md" ||
    file.startsWith("changelog.d/") ||
    file.startsWith("docs/changelog/")
  );
}

test("release history is exempt from the retired-tag sweep, and nothing else", () => {
  for (const file of [
    "CHANGELOG.md",
    "changelog.d/build-postgres-18-7.md",
    "docs/changelog/0.x.md",
  ]) {
    assert.equal(isReleaseHistory(file), true, file);
  }
  for (const file of [
    "docs/architecture.md",
    "docs/changelog.md",
    "SETUP.md",
    ".claude/agents/postgres-specialist.md",
    "frontend/CHANGELOG.md",
    // Siblings that share a folder's name without being inside it, and
    // folders of the same name below the repository root.
    "changelog.d.old/x.md",
    "changelog.dx",
    "frontend/changelog.d/x.md",
    "frontend/docs/changelog/x.md",
  ]) {
    assert.equal(isReleaseHistory(file), false, file);
  }
});

/**
 * Every PostgreSQL image reference in `files` (`{ file, contents }`, with
 * repository-relative `/` paths) that is not in `allowed`, as `file: reference`.
 */
export function retiredReferences(files, allowed) {
  const failures = [];

  for (const { file, contents } of files) {
    // Release history records the image each release ran, and no branch may edit
    // a released section (`Changelog Version`), so a retired tag there is
    // history, not stale guidance. Without this, every PostgreSQL tag bump fails.
    if (isReleaseHistory(file)) continue;
    // Requires the `<major>.<minor>` tag shape so `postgres:5432` host:port
    // pairs inside connection strings are not mistaken for image references.
    for (const [reference] of contents.matchAll(
      /postgres:\d+\.\d+[^\s`'"]*/g,
    )) {
      if (allowed.has(reference)) continue;
      failures.push(`${file}: ${reference}`);
    }
  }

  return failures;
}

// The repository sweep below passes or fails on whatever release history holds
// today, so this fixture pins the exemption's use in the sweep on its own.
test("the retired-tag sweep skips release history and reports a retired tag anywhere else", () => {
  const retired = "postgres:18.6-alpine3.24";
  const files = [
    "CHANGELOG.md",
    "changelog.d/build-postgres-19-0.md",
    "docs/changelog/1.0-1.1.md",
    "SETUP.md",
    "changelog.d.old/x.md",
  ].map((file) => ({ file, contents: `image: ${retired}\n` }));
  files.push({
    file: "docs/architecture.md",
    contents: "image: postgres:19.0-alpine3.24\nhost: postgres:5432\n",
  });

  assert.deepEqual(
    retiredReferences(files, new Set(["postgres:19.0-alpine3.24"])),
    [`SETUP.md: ${retired}`, `changelog.d.old/x.md: ${retired}`],
  );
});

// A version bump also touches coupling guidance when a tag has been copied into
// prose. This sweep prevents a documented reference from quietly describing a
// retired server. Release history is exempt (`isReleaseHistory`).
test("no tracked file references a retired PostgreSQL image", () => {
  const allowed = new Set([
    composeRef,
    composeRef.split("@")[0],
    ciRef,
    ciRef.split("@")[0],
  ]);
  const files = function* () {
    for (const path of textFiles(root)) {
      yield {
        file: relative(root, path).split(sep).join("/"),
        contents: readFileSync(path, "utf8"),
      };
    }
  };

  assert.deepEqual(retiredReferences(files(), allowed), []);
});

test("SETUP.md delegates PostgreSQL pins to their owning configuration", () => {
  const setup = read("SETUP.md");
  assert.ok(setup.includes("Dockerfiles, Compose file, and\nworkflows"));
  assert.ok(setup.includes("repository policy tests"));
  assert.doesNotMatch(setup, /docker buildx imagetools inspect postgres:/);
});
