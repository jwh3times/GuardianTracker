#!/usr/bin/env node
// Rewrites every pin whose version is dictated by a version file in this
// repository: container image pins, each resolved to its current registry
// digest, and the Go release in the documented local Staticcheck commands.
//
//   .nvmrc                        -> node:<version>-alpine3.24      (frontend Dockerfile, Dockerfile.dev)
//                                 -> node:<version>-bookworm-slim   (frontend Dockerfile.playwright)
//   frontend/package-lock.json    -> mcr.microsoft.com/playwright:v<version>-noble
//   backend/api-service/go.mod    -> golang:<release>-alpine         (backend/api-service/Dockerfile)
//                                 -> GOTOOLCHAIN=go<release>         (GOTOOLCHAIN_PINS)
//
// Why this exists: npm, gomod and docker are separate Dependabot ecosystems and
// cannot land in one PR. A `@playwright/test` bump therefore arrives with the
// image pin untouched, a Node bump edits only the two plain frontend
// Dockerfiles, and a golang bump moves the builder without go.mod (Dependabot
// never moves a toolchain directive). Each leaves `Format Check` red on
// node-version-policy or go-toolchain-policy until someone hand-edits a tag and
// pastes a fresh digest. That edit is always mechanical, so it belongs in a
// script rather than in a reviewer's memory. go.mod's release changes only by
// `go mod edit -toolchain=go<X.Y.Z>`; this script never reads it back from a
// Dockerfile.
//
// Only pins a repo version file *owns* are in scope. nginx, alpine, and postgres
// are pinned too, but Dependabot's docker ecosystem moves their tag and digest
// together, so nothing here can drift. (The postgres compose/workflow split is a
// version-consistency problem, and scripts/postgres-pin-policy.test.mjs owns it.)
//
// This talks to the registry, so unlike sync-agents.mjs it is NOT wired
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

/**
 * The Go release the API module declares: its `toolchain` directive, or its `go`
 * directive when Go has dropped the `toolchain` line (it does whenever the two
 * would be equal). This is the same rule `go` and actions/setup-go follow, so
 * CI, the builder image and the documented commands all name one release.
 */
export function goRelease(gomod) {
  // A directive may carry a trailing `// comment`, which go.mod allows.
  const toolchain = /^toolchain\s+(\S+)\s*(?:\/\/.*)?$/m.exec(gomod)?.[1];
  if (toolchain !== undefined) {
    const release = /^go(\d+\.\d+\.\d+)$/.exec(toolchain)?.[1];
    if (!release) {
      throw new Error(
        `go.mod's toolchain ${JSON.stringify(toolchain)} is not an exact go<X.Y.Z> release`,
      );
    }
    return release;
  }
  const go = /^go\s+(\S+)\s*(?:\/\/.*)?$/m.exec(gomod)?.[1];
  if (go === undefined) throw new Error("go.mod declares no Go release");
  if (!/^\d+\.\d+\.\d+$/.test(go)) {
    throw new Error(
      `go.mod's go line ${JSON.stringify(go)} names no patch release; add a toolchain line such as "toolchain go${go}.0"`,
    );
  }
  return go;
}

/** Reads the versions that own the tags. Pure; the caller supplies the text. */
export function parseVersions({ nvmrc, packageLock, gomod }) {
  const node = nvmrc.trim();
  if (!/^\d+\.\d+\.\d+$/.test(node)) {
    throw new Error(`.nvmrc is not an exact version: ${JSON.stringify(node)}`);
  }
  const entry =
    JSON.parse(packageLock).packages?.["node_modules/@playwright/test"];
  if (!entry?.version) {
    throw new Error("package-lock.json has no @playwright/test entry");
  }
  return { node, playwright: entry.version, go: goRelease(gomod) };
}

/**
 * The pins to rewrite. `occurrences` is asserted, not discovered: a regex that
 * silently stops matching would turn this script into a no-op that still exits
 * 0, which is a worse failure than the drift it exists to fix.
 */
export function plan({ node, playwright, go }) {
  return [
    {
      file: "backend/api-service/Dockerfile",
      image: "golang",
      tag: `${go}-alpine`,
      registry: "dockerhub",
      repository: "library/golang",
      occurrences: 1,
    },
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
 * The authored files that carry the pinned local Staticcheck command, with how
 * many `GOTOOLCHAIN=go<X.Y.Z>` pins each holds. Staticcheck is built by whatever
 * Go is active and analyzes nothing on a newer one, so each documented command
 * pins the release go.mod names. `.claude/settings.json` holds the exact-match
 * allowlist entry for that command, which must change with it. The generated
 * Codex mirror is not listed: `npm run sync:agents` copies it from the agent
 * file. scripts/go-toolchain-policy.test.mjs checks this same list.
 */
export const GOTOOLCHAIN_PINS = [
  { file: "AGENTS.md", occurrences: 1 },
  { file: "SETUP.md", occurrences: 1 },
  { file: ".claude/agents/go-services.md", occurrences: 1 },
  { file: ".claude/settings.json", occurrences: 1 },
];

/** A `GOTOOLCHAIN=go<X.Y.Z>` pin, capturing the release. */
const GOTOOLCHAIN = /GOTOOLCHAIN=go(\d+\.\d+\.\d+)/g;

/** Every release named by a `GOTOOLCHAIN=` pin in `contents`, in order. */
export function gotoolchainReleases(contents) {
  return [...contents.matchAll(GOTOOLCHAIN)].map(([, release]) => release);
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
    gomod: read("backend/api-service/go.mod"),
  });

  // One read per file, so two pins in the same Dockerfile compose rather than
  // the second overwriting the first.
  const originals = new Map();
  const updated = new Map();
  const load = (file) => {
    if (!originals.has(file)) {
      const contents = read(file);
      originals.set(file, contents);
      updated.set(file, contents);
    }
  };

  for (const target of GOTOOLCHAIN_PINS) {
    load(target.file);
    const found = gotoolchainReleases(originals.get(target.file)).length;
    if (found !== target.occurrences) {
      console.error(
        `${target.file}: expected ${target.occurrences} GOTOOLCHAIN=go<X.Y.Z> pin(s), found ${found}.`,
      );
      console.error(
        "The file moved on without this script. Update GOTOOLCHAIN_PINS in scripts/sync-image-pins.mjs.",
      );
      return 1;
    }
    updated.set(
      target.file,
      updated
        .get(target.file)
        .replace(GOTOOLCHAIN, `GOTOOLCHAIN=go${versions.go}`),
    );
  }

  for (const target of plan(versions)) {
    load(target.file);
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
      console.log(`Pins are in sync (${originals.size} files).`);
      return 0;
    }
    console.error(
      "Pins are out of date (image tags and digests, GOTOOLCHAIN= lines):",
    );
    for (const file of drifted) console.error(`  stale: ${file}`);
    console.error("\nFix: node scripts/sync-image-pins.mjs");
    return 1;
  }

  for (const file of drifted) write(file, updated.get(file));

  console.log(
    drifted.length === 0
      ? `Pins already up to date (${originals.size} files).`
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
