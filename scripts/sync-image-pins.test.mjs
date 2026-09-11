import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import {
  applyPin,
  countPins,
  main,
  parseVersions,
  plan,
  resolveDigest,
} from "./sync-image-pins.mjs";

const root = join(import.meta.dirname, "..");
const read = (...parts) => readFileSync(join(root, ...parts), "utf8");

const DIGEST_A = `sha256:${"a".repeat(64)}`;
const DIGEST_B = `sha256:${"b".repeat(64)}`;

test("parseVersions reads the two files that own the tags", () => {
  const versions = parseVersions({
    nvmrc: "26.8.2\n",
    packageLock: JSON.stringify({
      packages: { "node_modules/@playwright/test": { version: "1.63.0" } },
    }),
  });
  assert.deepEqual(versions, { node: "26.8.2", playwright: "1.63.0" });
});

test("parseVersions rejects an inexact Node version rather than pinning a range", () => {
  assert.throws(
    () => parseVersions({ nvmrc: "26\n", packageLock: "{}" }),
    /not an exact version/,
  );
});

test("parseVersions fails loudly when Playwright is absent from the lockfile", () => {
  assert.throws(
    () => parseVersions({ nvmrc: "26.8.2", packageLock: "{}" }),
    /no @playwright\/test entry/,
  );
});

test("applyPin rewrites a stale tag as well as a stale digest", () => {
  // A Node bump invalidates both halves; replacing only the digest would leave
  // the tag lying about which image is pinned.
  const before = `FROM node:26.0.0-alpine3.24@${DIGEST_A} AS builder`;
  assert.equal(
    applyPin(before, "node", "26.8.2-alpine3.24", DIGEST_B),
    `FROM node:26.8.2-alpine3.24@${DIGEST_B} AS builder`,
  );
});

test("applyPin is idempotent", () => {
  const pinned = `ARG NODE_IMAGE=node:26.8.2-bookworm-slim@${DIGEST_A}`;
  const once = applyPin(pinned, "node", "26.8.2-bookworm-slim", DIGEST_A);
  assert.equal(once, pinned);
  assert.equal(applyPin(once, "node", "26.8.2-bookworm-slim", DIGEST_A), once);
});

test("applyPin leaves other images in the same file alone", () => {
  const before = [
    `FROM node:26.0.0-alpine3.24@${DIGEST_A} AS builder`,
    `FROM nginxinc/nginx-unprivileged:1.31.5-alpine3.24@${DIGEST_A}`,
  ].join("\n");
  const after = applyPin(before, "node", "26.8.2-alpine3.24", DIGEST_B);
  assert.match(after, /nginx-unprivileged:1\.31\.5-alpine3\.24@sha256:a{64}/);
  assert.equal(countPins(after, "nginxinc/nginx-unprivileged"), 1);
});

test("a pin pattern does not match an image whose name merely ends in it", () => {
  // Without the boundary guard, `node` would match inside `someorg/node` and
  // `mynode`, and the script would rewrite an unrelated image.
  for (const line of [
    `FROM someorg/node:1.0.0@${DIGEST_A}`,
    `FROM mynode:1.0.0@${DIGEST_A}`,
  ]) {
    assert.equal(countPins(line, "node"), 0, line);
  }
  assert.equal(countPins(`FROM node:1.0.0@${DIGEST_A}`, "node"), 1);
});

test("a dot in an image name is never treated as a wildcard", () => {
  // The name is compared as a string, so a host that differs only where the
  // real one has dots must not match.
  assert.equal(
    countPins(
      `FROM mcrxmicrosoftxcom/playwright:v1@${DIGEST_A}`,
      "mcr.microsoft.com/playwright",
    ),
    0,
  );
  assert.equal(
    countPins(
      `FROM mcr.microsoft.com/playwright:v1@${DIGEST_A}`,
      "mcr.microsoft.com/playwright",
    ),
    1,
  );
  assert.equal(
    applyPin(
      `FROM mcrxmicrosoftxcom/playwright:v1@${DIGEST_A}`,
      "mcr.microsoft.com/playwright",
      "v2-noble",
      DIGEST_B,
    ),
    `FROM mcrxmicrosoftxcom/playwright:v1@${DIGEST_A}`,
  );
});

test("plan still matches the real Dockerfiles", () => {
  // The regression that matters: if a Dockerfile is restructured and a pattern
  // stops matching, the script would report success while changing nothing.
  const versions = parseVersions({
    nvmrc: read(".nvmrc"),
    packageLock: read("frontend", "package-lock.json"),
  });
  for (const target of plan(versions)) {
    const contents = read(...target.file.split("/"));
    assert.equal(
      countPins(contents, target.image),
      target.occurrences,
      `${target.file} should carry ${target.occurrences} ${target.image} pin(s)`,
    );
  }
});

test("plan derives its tags from the version files, not from the Dockerfiles", () => {
  const targets = plan({ node: "26.8.2", playwright: "1.63.0" });
  assert.deepEqual(
    targets.map((t) => `${t.image}:${t.tag}`),
    [
      "node:26.8.2-alpine3.24",
      "node:26.8.2-alpine3.24",
      "node:26.8.2-bookworm-slim",
      "mcr.microsoft.com/playwright:v1.63.0-noble",
    ],
  );
});

test("plan covers every pin the Node policy test enforces", () => {
  // These two must not drift apart: a pin the policy gate checks but this script
  // does not rewrite is exactly the hand-edit this work exists to remove.
  const policy = read("scripts", "node-version-policy.test.mjs");
  const files = new Set(
    plan({ node: "1.2.3", playwright: "4.5.6" }).map((t) => t.file),
  );
  assert.deepEqual([...files].sort(), [
    "frontend/Dockerfile",
    "frontend/Dockerfile.dev",
    "frontend/Dockerfile.playwright",
  ]);
  assert.match(policy, /ARG PLAYWRIGHT_IMAGE=mcr/);
  assert.match(policy, /ARG NODE_IMAGE=node:/);

  // Image and file identity are not enough: the two sides also have to agree on
  // the tag *variant*. If the policy gate starts demanding a different flavour
  // (-noble -> -jammy, say) and plan() keeps writing the old one, every run of
  // this script would produce a file the gate rejects.
  // The policy test spells its variants inside regex literals, where dots are
  // escaped (`-alpine3\.24`), so compare against a de-escaped copy.
  const policyLiterals = policy.replaceAll("\\", "");
  for (const target of plan({ node: "1.2.3", playwright: "4.5.6" })) {
    const variant = target.tag.replace(/^v?[\d.]+/, "");
    assert.ok(
      policyLiterals.includes(variant),
      `${target.file} pins a ${variant} tag, which the policy test never mentions`,
    );
  }
});

function fakeRepo(overrides = {}) {
  const files = {
    ".nvmrc": "26.8.2\n",
    "frontend/package-lock.json": JSON.stringify({
      packages: { "node_modules/@playwright/test": { version: "1.63.0" } },
    }),
    "frontend/Dockerfile": `FROM node:26.0.0-alpine3.24@${DIGEST_A} AS builder\n`,
    "frontend/Dockerfile.dev": `FROM node:26.0.0-alpine3.24@${DIGEST_A}\n`,
    "frontend/Dockerfile.playwright": [
      `ARG NODE_IMAGE=node:26.0.0-bookworm-slim@${DIGEST_A}`,
      `ARG PLAYWRIGHT_IMAGE=mcr.microsoft.com/playwright:v1.62.1-noble@${DIGEST_A}`,
      "",
    ].join("\n"),
    ...overrides,
  };
  const written = {};
  return {
    files,
    written,
    deps: {
      read: (rel) => {
        if (!(rel in files)) throw new Error(`unexpected read: ${rel}`);
        return files[rel];
      },
      write: (rel, text) => {
        written[rel] = text;
      },
      resolveDigest: async () => DIGEST_B,
    },
  };
}

test("main rewrites both pins in a file that carries two", async () => {
  const repo = fakeRepo();
  assert.equal(await main([], repo.deps), 0);

  const playwright = repo.written["frontend/Dockerfile.playwright"];
  assert.match(
    playwright,
    new RegExp(
      `^ARG NODE_IMAGE=node:26\\.8\\.2-bookworm-slim@${DIGEST_B}$`,
      "m",
    ),
  );
  assert.match(
    playwright,
    new RegExp(
      `^ARG PLAYWRIGHT_IMAGE=mcr\\.microsoft\\.com/playwright:v1\\.63\\.0-noble@${DIGEST_B}$`,
      "m",
    ),
  );
  assert.match(
    repo.written["frontend/Dockerfile"],
    new RegExp(`node:26\\.8\\.2-alpine3\\.24@${DIGEST_B} AS builder`),
  );
});

test("main writes nothing when every pin already matches", async () => {
  const repo = fakeRepo({
    "frontend/Dockerfile": `FROM node:26.8.2-alpine3.24@${DIGEST_B} AS builder\n`,
    "frontend/Dockerfile.dev": `FROM node:26.8.2-alpine3.24@${DIGEST_B}\n`,
    "frontend/Dockerfile.playwright": [
      `ARG NODE_IMAGE=node:26.8.2-bookworm-slim@${DIGEST_B}`,
      `ARG PLAYWRIGHT_IMAGE=mcr.microsoft.com/playwright:v1.63.0-noble@${DIGEST_B}`,
      "",
    ].join("\n"),
  });
  assert.equal(await main([], repo.deps), 0);
  assert.deepEqual(repo.written, {});
});

test("--check reports drift and writes nothing", async () => {
  const repo = fakeRepo();
  assert.equal(await main(["--check"], repo.deps), 1);
  assert.deepEqual(repo.written, {});
});

test("--check passes once the pins are current", async () => {
  const repo = fakeRepo({
    "frontend/Dockerfile": `FROM node:26.8.2-alpine3.24@${DIGEST_B} AS builder\n`,
    "frontend/Dockerfile.dev": `FROM node:26.8.2-alpine3.24@${DIGEST_B}\n`,
    "frontend/Dockerfile.playwright": [
      `ARG NODE_IMAGE=node:26.8.2-bookworm-slim@${DIGEST_B}`,
      `ARG PLAYWRIGHT_IMAGE=mcr.microsoft.com/playwright:v1.63.0-noble@${DIGEST_B}`,
      "",
    ].join("\n"),
  });
  assert.equal(await main(["--check"], repo.deps), 0);
});

test("main refuses to proceed when a file no longer carries its pin", async () => {
  const repo = fakeRepo({
    "frontend/Dockerfile": "FROM scratch\n",
  });
  assert.equal(await main([], repo.deps), 1);
  assert.deepEqual(repo.written, {});
});

function response({ status = 200, digest, json } = {}) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: {
      get: (k) => (k.toLowerCase() === "docker-content-digest" ? digest : null),
    },
    json: async () => json,
  };
}

test("resolveDigest asks MCR anonymously with a HEAD and manifest-list Accept", async () => {
  const calls = [];
  const digest = await resolveDigest(
    { registry: "mcr", repository: "playwright", tag: "v1.63.0-noble" },
    async (url, init) => {
      calls.push({ url, init });
      return response({ digest: DIGEST_A });
    },
  );
  assert.equal(digest, DIGEST_A);
  assert.equal(calls.length, 1, "MCR needs no token request");
  assert.equal(
    calls[0].url,
    "https://mcr.microsoft.com/v2/playwright/manifests/v1.63.0-noble",
  );
  assert.equal(calls[0].init.method, "HEAD");
  assert.equal(calls[0].init.headers.Authorization, undefined);
  // A multi-arch tag must resolve to the index digest, so the index types have
  // to be offered ahead of the single-image ones.
  const accept = calls[0].init.headers.Accept;
  assert.ok(
    accept.indexOf("image.index") < accept.indexOf("image.manifest"),
    `manifest-list types must be preferred: ${accept}`,
  );
});

test("resolveDigest fetches a pull token before asking Docker Hub", async () => {
  const calls = [];
  const digest = await resolveDigest(
    {
      registry: "dockerhub",
      repository: "library/node",
      tag: "26.8.2-alpine3.24",
    },
    async (url, init) => {
      calls.push({ url, init });
      // Match on the parsed host, not a substring: "auth.docker.io" can appear
      // anywhere in a URL, including in a path or query of some other host.
      return new URL(url).hostname === "auth.docker.io"
        ? response({ json: { token: "tok" } })
        : response({ digest: DIGEST_B });
    },
  );
  assert.equal(digest, DIGEST_B);
  assert.match(calls[0].url, /auth\.docker\.io.*repository:library\/node:pull/);
  assert.equal(
    calls[1].url,
    "https://registry-1.docker.io/v2/library/node/manifests/26.8.2-alpine3.24",
  );
  assert.equal(calls[1].init.headers.Authorization, "Bearer tok");
});

test("resolveDigest explains a 404 by naming the version file that owns the tag", async () => {
  await assert.rejects(
    resolveDigest(
      { registry: "mcr", repository: "playwright", tag: "v9.9.9-noble" },
      async () => response({ status: 404 }),
    ),
    /does not exist in the registry.*version file/s,
  );
});

test("resolveDigest refuses a response with no usable digest", async () => {
  // Writing an empty or malformed digest into a Dockerfile would be worse than
  // the drift this script exists to fix, so it must throw rather than return.
  for (const digest of [undefined, "", "sha256:nothex", "sha256:abc"]) {
    await assert.rejects(
      resolveDigest(
        { registry: "mcr", repository: "playwright", tag: "v1.63.0-noble" },
        async () => response({ digest }),
      ),
      /no usable digest/,
      `digest ${JSON.stringify(digest)} must be rejected`,
    );
  }
});

test("resolveDigest surfaces a failed lookup and a failed token request", async () => {
  await assert.rejects(
    resolveDigest(
      { registry: "mcr", repository: "playwright", tag: "v1.63.0-noble" },
      async () => response({ status: 500 }),
    ),
    /lookup failed: 500/,
  );
  await assert.rejects(
    resolveDigest(
      { registry: "dockerhub", repository: "library/node", tag: "x" },
      async () => response({ status: 401 }),
    ),
    /token request failed: 401/,
  );
  await assert.rejects(
    resolveDigest(
      { registry: "dockerhub", repository: "library/node", tag: "x" },
      async () => response({ json: {} }),
    ),
    /no pull token/,
  );
});

test("resolveDigest rejects an unknown registry rather than guessing a host", async () => {
  await assert.rejects(
    resolveDigest({ registry: "quay", repository: "x", tag: "y" }, async () =>
      response({ digest: DIGEST_A }),
    ),
    /unknown registry: quay/,
  );
});
