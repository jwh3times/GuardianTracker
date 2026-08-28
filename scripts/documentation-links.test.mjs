import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, statSync } from "node:fs";
import { dirname, extname, resolve, sep } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");

function repositoryMarkdownFiles() {
  return execFileSync(
    "git",
    ["ls-files", "--cached", "--others", "--exclude-standard", "--", "*.md"],
    {
      cwd: repositoryRoot,
      encoding: "utf8",
    },
  )
    .split(/\r?\n/)
    .filter((file) => file && existsSync(resolve(repositoryRoot, file)));
}

function withoutFencedCode(markdown) {
  let fence = null;

  return markdown
    .split(/\r?\n/)
    .map((line) => {
      const marker = line.match(/^\s*(`{3,}|~{3,})/);
      if (marker) {
        const character = marker[1][0];
        if (fence === null) fence = character;
        else if (fence === character) fence = null;
        return "";
      }
      return fence === null ? line : "";
    })
    .join("\n");
}

function linkDestinations(markdown) {
  const visibleMarkdown = withoutFencedCode(markdown);
  const destinations = [];
  const inlineLink = /!?\[[^\]]*\]\((<[^>]+>|[^\s)]+)(?:\s+['"][^)]*['"])?\)/g;
  const referenceLink = /^\s*\[[^\]]+\]:\s*(<[^>]+>|\S+)/gm;

  for (const pattern of [inlineLink, referenceLink]) {
    for (const match of visibleMarkdown.matchAll(pattern)) {
      destinations.push(match[1].replace(/^<|>$/g, ""));
    }
  }

  return destinations;
}

function slugify(heading) {
  return heading
    .trim()
    .toLowerCase()
    .replace(/<[^>]*>/g, "")
    .replace(/[`*_~]/g, "")
    .replace(/[^\p{L}\p{N}\s-]/gu, "")
    .replace(/\s/g, "-");
}

function anchors(markdown) {
  const visibleMarkdown = withoutFencedCode(markdown);
  const anchorsForFile = new Set();
  const occurrences = new Map();

  for (const line of visibleMarkdown.split(/\r?\n/)) {
    const heading = line.match(/^\s{0,3}#{1,6}\s+(.+?)\s*#*\s*$/);
    if (heading) {
      const base = slugify(heading[1]);
      const occurrence = occurrences.get(base) ?? 0;
      occurrences.set(base, occurrence + 1);
      anchorsForFile.add(occurrence === 0 ? base : `${base}-${occurrence}`);
    }

    for (const explicit of line.matchAll(
      /<(?:a\s+name|[^>]+\sid)=["']([^"']+)["'][^>]*>/gi,
    )) {
      anchorsForFile.add(explicit[1]);
    }
  }

  return anchorsForFile;
}

function localTarget(sourceFile, destination) {
  if (/^(?:[a-z][a-z0-9+.-]*:|\/\/)/i.test(destination)) return null;

  const [rawPath, rawFragment = ""] = destination.split("#", 2);
  const decodedPath = decodeURIComponent(rawPath);
  const sourcePath = resolve(repositoryRoot, sourceFile);
  const targetPath = decodedPath
    ? decodedPath.startsWith("/")
      ? resolve(repositoryRoot, `.${decodedPath}`)
      : resolve(dirname(sourcePath), decodedPath)
    : sourcePath;

  return {
    fragment: decodeURIComponent(rawFragment),
    path: targetPath,
  };
}

test("repository Markdown has valid local links and heading anchors", () => {
  const failures = [];
  const anchorCache = new Map();

  for (const sourceFile of repositoryMarkdownFiles()) {
    const markdown = readFileSync(resolve(repositoryRoot, sourceFile), "utf8");

    for (const destination of linkDestinations(markdown)) {
      const target = localTarget(sourceFile, destination);
      if (target === null) continue;

      const relativeDisplay = target.path.startsWith(repositoryRoot + sep)
        ? target.path.slice(repositoryRoot.length + 1)
        : target.path;

      if (
        target.path !== repositoryRoot &&
        !target.path.startsWith(repositoryRoot + sep)
      ) {
        failures.push(
          `${sourceFile}: link escapes the repository (${destination})`,
        );
        continue;
      }

      if (!existsSync(target.path)) {
        failures.push(
          `${sourceFile}: missing ${destination} (${relativeDisplay})`,
        );
        continue;
      }

      if (
        !target.fragment ||
        /^L\d+(?:-L\d+)?$/.test(target.fragment) ||
        statSync(target.path).isDirectory() ||
        extname(target.path).toLowerCase() !== ".md"
      ) {
        continue;
      }

      let targetAnchors = anchorCache.get(target.path);
      if (!targetAnchors) {
        targetAnchors = anchors(readFileSync(target.path, "utf8"));
        anchorCache.set(target.path, targetAnchors);
      }

      if (!targetAnchors.has(target.fragment.toLowerCase())) {
        failures.push(
          `${sourceFile}: missing anchor ${destination} in ${relativeDisplay}`,
        );
      }
    }
  }

  assert.deepEqual(failures, []);
});
