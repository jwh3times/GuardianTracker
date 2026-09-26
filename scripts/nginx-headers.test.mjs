import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  chmodSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { setTimeout } from "node:timers/promises";
import { test } from "node:test";

const root = join(import.meta.dirname, "..");
const securityHeaders = {
  "x-frame-options": "SAMEORIGIN",
  "x-content-type-options": "nosniff",
  "x-xss-protection": "1; mode=block",
  "referrer-policy": "strict-origin-when-cross-origin",
  "permissions-policy": "geolocation=(), microphone=(), camera=()",
  "content-security-policy":
    "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' https://www.bungie.net data:; connect-src 'self' http://localhost:8081 https://www.bungie.net https://*.ngrok-free.app; font-src 'self' https://fonts.gstatic.com; object-src 'none'; base-uri 'self'; frame-ancestors 'self';",
};

function docker(...args) {
  const result = spawnSync("docker", args, {
    encoding: "utf8",
    timeout: 120_000,
    maxBuffer: 4 * 1024 * 1024,
  });
  assert.equal(
    result.status,
    0,
    `Docker nginx probe failed: ${result.error ?? ""}\n${result.stdout}\n${result.stderr}`,
  );
  return result.stdout.trim();
}

// Exercise the actual production configuration and runtime image with synthetic
// public assets. No application build, running app stack, or credentials needed.
test("nginx preserves security headers and cache policy over HTTP", async (t) => {
  const dockerfile = readFileSync(join(root, "frontend", "Dockerfile"), "utf8");
  const image = dockerfile.match(
    /^FROM (nginxinc\/nginx-unprivileged:\S+@sha256:[a-f0-9]{64})\s*$/m,
  )?.[1];
  assert.ok(image, "frontend Dockerfile must supply the pinned nginx runtime");
  const temporary = mkdtempSync(join(tmpdir(), "guardian-nginx-headers-"));
  let container;
  try {
    // mkdtemp uses 0700 on POSIX; nginx runs as a different, non-root UID.
    // These synthetic public fixtures need explicit traversal/read permissions,
    // independent of the host umask (Windows bind mounts can hide this issue).
    chmodSync(temporary, 0o755);
    function writeFixture(name, content) {
      const path = join(temporary, name);
      writeFileSync(path, content);
      chmodSync(path, 0o644);
    }
    const html = "<!doctype html><title>Guardian nginx probe</title>";
    writeFixture("index.html", html);
    writeFixture("site.webmanifest", '{"name":"Probe"}');
    for (const extension of ["js", "css", "svg", "woff2"]) {
      writeFixture(`fixture.${extension}`, "synthetic asset");
    }
    container = docker(
      "create",
      "--publish",
      "127.0.0.1::8080",
      "--mount",
      `type=bind,source=${temporary},target=/usr/share/nginx/html,readonly`,
      "--mount",
      `type=bind,source=${join(root, "frontend", "nginx.conf")},target=/etc/nginx/conf.d/default.conf,readonly`,
      image,
    );
    docker("start", container);
    const port = docker("port", container, "8080/tcp");
    assert.match(port, /^127\.0\.0\.1:\d+$/);
    const base = `http://${port}`;
    let ready = false;
    for (let attempt = 0; attempt < 50; attempt++) {
      try {
        const response = await fetch(base, {
          signal: AbortSignal.timeout(1_000),
        });
        await response.arrayBuffer();
        if (response.ok) {
          ready = true;
          break;
        }
      } catch {
        /* nginx may still be starting */
      }
      await setTimeout(100);
    }
    assert.ok(ready, "nginx must become ready");

    function assertHeaders(response) {
      for (const [name, value] of Object.entries(securityHeaders)) {
        assert.equal(
          response.headers.get(name),
          value,
          `${response.url}: ${name}`,
        );
      }
    }
    for (const path of [
      "/",
      "/index.html",
      "/collections/deep/link",
      "/site.webmanifest",
    ]) {
      await t.test(
        `${path} keeps headers without immutable caching`,
        async () => {
          const response = await fetch(`${base}${path}`, {
            signal: AbortSignal.timeout(5_000),
          });
          assert.equal(response.status, 200);
          assertHeaders(response);
          assert.equal(response.headers.get("cache-control"), null);
          assert.equal(response.headers.get("expires"), null);
          if (path === "/site.webmanifest") {
            assert.equal(
              response.headers.get("content-type"),
              "application/manifest+json",
            );
            assert.deepEqual(await response.json(), { name: "Probe" });
          } else {
            assert.match(response.headers.get("content-type"), /^text\/html/);
            assert.equal(await response.text(), html);
          }
        },
      );
    }
    for (const extension of ["js", "css", "svg", "woff2"]) {
      await t.test(
        `${extension} assets retain headers and one-year immutable caching`,
        async () => {
          const url = `${base}/fixture.${extension}`;
          const response = await fetch(url, {
            signal: AbortSignal.timeout(5_000),
          });
          assert.equal(response.status, 200);
          assertHeaders(response);
          assert.equal(
            response.headers.get("cache-control"),
            "max-age=31536000, public, immutable",
          );
          assert.ok(Date.parse(response.headers.get("expires")) > Date.now());
          assert.equal(await response.text(), "synthetic asset");
          const etag = response.headers.get("etag");
          assert.ok(etag);
          const unchanged = await fetch(url, {
            headers: { "If-None-Match": etag },
            signal: AbortSignal.timeout(5_000),
          });
          assert.equal(unchanged.status, 304);
          assertHeaders(unchanged);
          assert.equal(
            unchanged.headers.get("cache-control"),
            "max-age=31536000, public, immutable",
          );
        },
      );
    }
    await t.test(
      "missing static assets keep security headers without immutable caching",
      async () => {
        const response = await fetch(`${base}/missing.js`, {
          signal: AbortSignal.timeout(5_000),
        });
        assert.equal(response.status, 404);
        assertHeaders(response);
        assert.equal(response.headers.get("cache-control"), null);
        assert.equal(response.headers.get("expires"), null);
        await response.arrayBuffer();
      },
    );
  } finally {
    if (container) docker("rm", "--force", container);
    rmSync(temporary, { recursive: true, force: true });
  }
});
