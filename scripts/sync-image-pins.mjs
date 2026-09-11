#!/usr/bin/env node
// Rewrites every frontend container image pin whose tag is dictated by a version
// file in this repository, resolving each tag to its current registry digest.
//
//   .nvmrc                        -> node:<version>-alpine3.24      (Dockerfile, Dockerfile.dev)
//                                 -> node:<version>-bookworm-slim   (Dockerfile.playwright)
//   frontend/package-lock.json    -> mcr.microsoft.com/playwright:v<version>-noble
//
// Why this exists: npm and docker are separate Dependabot ecosystems and cannot
// land in one PR. A `@playwright/test` bump therefore arrives with the image pin
// untouched, and a Node bump edits only the two plain frontend Dockerfiles. Both
// leave `Format Check` red on scripts/node-version-policy.test.mjs until someone
// hand-edits a tag and pastes a fresh digest. That edit is always mechanical, so
// it belongs in a script rather than in a reviewer's memory.
//
// Only pins a repo version file *owns* are in scope. nginx, alpine, and postgres
// are pinned too, but Dependabot's docker ecosystem moves their tag and digest
// together, so nothing here can drift. (The postgres compose/workflow split is a
// version-consistency problem, and scripts/postgres-pin-policy.test.mjs owns it.)
//
// This talks to the registry, so unlike sync-agent-configs.mjs it is NOT wired
// into CI — a required check must not depend on the network. The offline policy
// test stays the gate; this is the tool that makes it green.
//
// Usage:
//   node scripts/sync-image-pins.mjs           resolve and rewrite the pins
//   node scripts/sync-image-pins.mjs --check   report drift, write nothing

import { readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const ROOT = join(import.meta.dirname, "..");

/** Manifest-list types first: a multi-arch tag must resolve to the index digest. */
const ACCEPT = [
  "application/vnd.oci.image.index.v1+json",
  "application/vnd.docker.distribution.manifest.list.v2+json",
  "application/vnd.oci.image.manifest.v1+json",
  "application/vnd.docker.distribution.manifest.v2+json",
].join(", ");

/** Reads the versions that own the tags. Pure; the caller supplies the text. */
export function parseVersions({ nvmrc, packageLock }) {
  const node = nvmrc.trim();
  if (!/^\d+\.\d+\.\d+$/.test(node)) {
    throw new Error(`.nvmrc is not an exact version: ${JSON.stringify(node)}`);
  }
  const entry =
    JSON.parse(packageLock).packages?.["node_modules/@playwright/test"];
  if (!entry?.version) {
    throw new Error("package-lock.json has no @playwright/test entry");
  }
  return { node, playwright: entry.version };
}

/**
 * The pins to rewrite. `occurrences` is asserted, not discovered: a regex that
 * silently stops matching would turn this script into a no-op that still exits
 * 0, which is a worse failure than the drift it exists to fix.
 */
export function plan({ node, playwright }) {
  return [
    {
      file: "frontend/Dockerfile",
      image: "node",
      tag: `${node}-alpine3.24`,
      registry: "dockerhub",
      repository: "library/node",
      occurrences: 1,
    },
    {
      file: "frontend/Dockerfile.dev",
      image: "node",
      tag: `${node}-alpine3.24`,
      registry: "dockerhub",
      repository: "library/node",
      occurrences: 1,
    },
    {
      file: "frontend/Dockerfile.playwright",
      image: "node",
      tag: `${node}-bookworm-slim`,
      registry: "dockerhub",
      repository: "library/node",
      occurrences: 1,
    },
    {
      file: "frontend/Dockerfile.playwright",
      image: "mcr.microsoft.com/playwright",
      tag: `v${playwright}-noble`,
      registry: "mcr",
      repository: "playwright",
      occurrences: 1,
    },
  ];
}

/**
 * Matches any `<image>:<tag>@sha256:<digest>`, capturing the image name. The
 * whole reference is matched — not just the digest — so a stale *tag* is
 * corrected too; a Node bump invalidates both halves.
 *
 * The image name is captured and compared as a string rather than interpolated
 * into this pattern. Building the regex per image would mean escaping a name
 * that contains dots (`mcr.microsoft.com/...`), where a missed escape turns
 * each dot into a wildcard and silently widens what the pin matches. Plain
 * equality cannot be got wrong that way: `someorg/node` is captured whole and
 * simply does not equal `node`.
 *
 * What makes the capture whole is greedy leftmost matching — the engine starts
 * at the first character of the name and takes all of it. The leading boundary
 * is therefore redundant today (verified: it changes no match across the real
 * Dockerfiles and every adversarial prefix), and is kept only so that a later
 * edit making the name group lazy or alternated cannot quietly reintroduce
 * suffix matching. Do not read it as the thing doing the work.
 */
const PIN = /(?<![\w./-])([\w./-]+):([^@\s"']+)@sha256:([0-9a-f]{64})/g;

/** Replaces every pin of `image` with `tag@digest`. Returns the new text. */
export function applyPin(contents, image, tag, digest) {
  return contents.replace(PIN, (reference, name) =>
    name === image ? `${image}:${tag}@${digest}` : reference,
  );
}

export function countPins(contents, image) {
  let found = 0;
  for (const [, name] of contents.matchAll(PIN)) {
    if (name === image) found += 1;
  }
  return found;
}

async function dockerHubToken(repository, fetchImpl) {
  const url = `https://auth.docker.io/token?service=registry.docker.io&scope=repository:${repository}:pull`;
  const res = await fetchImpl(url);
  if (!res.ok) {
    throw new Error(`Docker Hub token request failed: ${res.status}`);
  }
  const { token } = await res.json();
  if (!token) throw new Error("Docker Hub returned no pull token");
  return token;
}

/**
 * Resolves one tag to its immutable digest over the registry v2 API. HEAD is
 * enough: the digest is a response header, so no manifest body is transferred.
 *
 * `fetchImpl` is injectable so the contract this depends on — the request shape
 * and every way a registry can answer unusably — is covered offline.
 */
export async function resolveDigest(
  { registry, repository, tag },
  fetchImpl = fetch,
) {
  const headers = { Accept: ACCEPT };
  let base;
  if (registry === "mcr") {
    base = "https://mcr.microsoft.com";
  } else if (registry === "dockerhub") {
    base = "https://registry-1.docker.io";
    headers.Authorization = `Bearer ${await dockerHubToken(repository, fetchImpl)}`;
  } else {
    throw new Error(`unknown registry: ${registry}`);
  }

  const res = await fetchImpl(`${base}/v2/${repository}/manifests/${tag}`, {
    method: "HEAD",
    headers,
  });
  if (res.status === 404) {
    throw new Error(
      `${repository}:${tag} does not exist in the registry — check the version file that owns this tag`,
    );
  }
  if (!res.ok) {
    throw new Error(`${repository}:${tag} lookup failed: ${res.status}`);
  }
  const digest = res.headers.get("docker-content-digest");
  if (!/^sha256:[0-9a-f]{64}$/.test(digest ?? "")) {
    throw new Error(`${repository}:${tag} returned no usable digest`);
  }
  return digest;
}

export async function main(argv = [], deps = {}) {
  const check = argv.includes("--check");
  const read = deps.read ?? ((rel) => readFileSync(join(ROOT, rel), "utf8"));
  const write =
    deps.write ?? ((rel, text) => writeFileSync(join(ROOT, rel), text, "utf8"));
  const resolve = deps.resolveDigest ?? resolveDigest;

  const versions = parseVersions({
    nvmrc: read(".nvmrc"),
    packageLock: read("frontend/package-lock.json"),
  });

  // One read per file, so two pins in the same Dockerfile compose rather than
  // the second overwriting the first.
  const originals = new Map();
  const updated = new Map();
  for (const target of plan(versions)) {
    if (!originals.has(target.file)) {
      const contents = read(target.file);
      originals.set(target.file, contents);
      updated.set(target.file, contents);
    }
    const found = countPins(originals.get(target.file), target.image);
    if (found !== target.occurrences) {
      console.error(
        `${target.file}: expected ${target.occurrences} ${target.image} pin(s), found ${found}.`,
      );
      console.error(
        "The file moved on without this script. Update plan() in scripts/sync-image-pins.mjs.",
      );
      return 1;
    }
    const digest = await resolve(target);
    updated.set(
      target.file,
      applyPin(updated.get(target.file), target.image, target.tag, digest),
    );
  }

  const drifted = [...updated.keys()].filter(
    (file) => updated.get(file) !== originals.get(file),
  );

  if (check) {
    if (drifted.length === 0) {
      console.log(`Image pins are in sync (${originals.size} files).`);
      return 0;
    }
    console.error("Image pins are out of date:");
    for (const file of drifted) console.error(`  stale: ${file}`);
    console.error("\nFix: node scripts/sync-image-pins.mjs");
    return 1;
  }

  for (const file of drifted) write(file, updated.get(file));

  console.log(
    drifted.length === 0
      ? `Image pins already up to date (${originals.size} files).`
      : `Updated ${drifted.length} file(s): ${drifted.join(", ")}`,
  );
  return 0;
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(process.argv[1]).href
) {
  process.exit(await main(process.argv.slice(2)));
}
