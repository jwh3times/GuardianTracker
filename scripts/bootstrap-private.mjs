#!/usr/bin/env node
// Clone the optional private companion repository into private/ from any
// checkout or git worktree, resolving its location through 1Password.
//
//   npm run bootstrap:private                       # reference from .private-workspace/repository.env.ref
//   npm run bootstrap:private -- --op-reference "op://<vault>/<item>/<field>"
//   npm run bootstrap:private -- --url https://github.com/<owner>/<repo>.git
//
// The reference file is looked up in this checkout first and then in the main
// checkout that owns the git worktree, so a fresh `git worktree add` can reuse
// the machine-local file without copying it. The resolved URL is kept out of
// this helper's direct child arguments and terminal output: `op run` injects it
// into a child process
// that hands it to git through a temporary `insteadOf` alias. Git transport
// subprocesses can receive the expanded URL; --url is also visible in caller argv.

import {
  existsSync,
  lstatSync,
  mkdirSync,
  renameSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const VARIABLE = "GUARDIAN_PRIVATE_REPOSITORY_URL";
const ALIAS = "guardian-private:";
const REFERENCE_FILE = join(".private-workspace", "repository.env.ref");
const REFERENCE_ASSIGNMENT = `${VARIABLE}=`;
const REFERENCE_PART = /^[A-Za-z0-9._ -]+$/u;

const scriptPath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(scriptPath), "..");
const privateRoot = join(repositoryRoot, "private");

class SafeBootstrapError extends Error {}

function fail(message) {
  throw new SafeBootstrapError(message);
}

// Preserve ordinary credential-helper configuration while removing process
// tracing, repository redirection, and injected Git configuration controls.
function childEnvironment(environment, { forOp = false } = {}) {
  const clean = { ...environment };
  for (const name of Object.keys(clean)) {
    const key = name.toUpperCase();
    if (
      /^GIT_TRACE|^GIT_CURL_VERBOSE$/u.test(key) ||
      /^GIT_(DIR|WORK_TREE|COMMON_DIR|INDEX_FILE|OBJECT_DIRECTORY|ALTERNATE_OBJECT_DIRECTORIES|CONFIG_COUNT|CONFIG_PARAMETERS|CONFIG_KEY_.*|CONFIG_VALUE_.*)$/u.test(
        key,
      ) ||
      key === VARIABLE ||
      (!forOp && key.startsWith("OP_")) ||
      /op:\/\//iu.test(clean[name])
    )
      delete clean[name];
  }
  // Explicit zero overrides trace2 settings inherited from Git config files.
  clean.GIT_TRACE2 = "0";
  clean.GIT_TRACE2_EVENT = "0";
  clean.GIT_TRACE2_PERF = "0";
  return clean;
}

function git(args, input) {
  return spawnSync("git", args, {
    encoding: "utf8",
    windowsHide: true,
    env: childEnvironment(process.env),
    stdio: [input === undefined ? "ignore" : "pipe", "pipe", "pipe"],
    input,
  });
}

function metadata(path) {
  try {
    return lstatSync(path);
  } catch (error) {
    if (error.code === "ENOENT") return null;
    throw error;
  }
}

function assertSafePath(root, path) {
  const part = relative(root, path);
  if (
    !part ||
    part === ".." ||
    part.startsWith(`..${sep}`) ||
    resolve(root, part) !== resolve(path)
  ) {
    fail("A private workspace path failed validation.");
  }
  let current = path;
  while (current !== root) {
    if (metadata(current)?.isSymbolicLink())
      fail(
        "A private workspace path contains a symbolic link or reparse point.",
      );
    current = dirname(current);
  }
}

function assertRepository(root) {
  assertSafePath(root, join(root, ".git"));
  if (!metadata(join(root, ".git")))
    fail("Run this helper from a Guardian Tracker repository checkout.");
  const result = git(["-C", root, "rev-parse", "--show-toplevel"]);
  if (result.status !== 0 || resolve(result.stdout.trim()) !== root)
    fail("The public repository root could not be validated.");
}

function assertUntracked(root, path) {
  for (const args of [
    ["ls-files", "-z", "--", path],
    ["ls-tree", "-r", "-z", "HEAD", "--", path],
  ]) {
    const result = git(["-C", root, ...args]);
    if (result.status !== 0 || result.stdout.length !== 0)
      fail(
        "Private workspace paths must not be tracked by the public repository.",
      );
  }
}

function ignoredByRoot(args, path) {
  const result = git(
    [
      ...args,
      "-c",
      "core.excludesFile=",
      "check-ignore",
      "-v",
      "-z",
      "--no-index",
      "--stdin",
    ],
    `${path}\0`,
  );
  const fields = result.stdout?.split("\0");
  return (
    result.status === 0 &&
    fields.length === 5 &&
    fields[0] === ".gitignore" &&
    !fields[2].startsWith("!")
  );
}

function assertProtected(root, path) {
  assertSafePath(root, join(root, ".gitignore"));
  if (!ignoredByRoot(["-C", root], path))
    fail("Private workspace paths must be protected by the root .gitignore.");
  const committed = git(["-C", root, "show", "HEAD:.gitignore"]);
  const gitDir = git(["-C", root, "rev-parse", "--absolute-git-dir"]);
  if (committed.status !== 0 || gitDir.status !== 0)
    fail("Committed public ignore rules could not be verified.");
  const temporaryRoot = mkdtempSync(join(tmpdir(), "guardian-private-ignore-"));
  try {
    writeFileSync(join(temporaryRoot, ".gitignore"), committed.stdout);
    if (
      !ignoredByRoot(
        [`--git-dir=${gitDir.stdout.trim()}`, `--work-tree=${temporaryRoot}`],
        path,
      )
    )
      fail(
        "Private workspace paths must be protected by committed public ignore rules.",
      );
  } finally {
    rmSync(temporaryRoot, { recursive: true, force: true });
  }
}

function validatePrivateClone(path) {
  assertSafePath(repositoryRoot, join(path, ".git", "config"));
  if (
    !metadata(join(path, ".git"))?.isDirectory() ||
    !metadata(join(path, ".git", "config"))?.isFile()
  )
    fail("The private workspace must contain an independent Git repository.");
  const result = git(["-C", path, "rev-parse", "--show-toplevel"]);
  if (result.status !== 0 || resolve(result.stdout.trim()) !== path)
    fail("The private repository root could not be validated.");
}

function preflight() {
  assertRepository(repositoryRoot);
  assertSafePath(repositoryRoot, privateRoot);
  assertUntracked(repositoryRoot, "private");
  assertProtected(repositoryRoot, "private/");
  if (metadata(privateRoot)) {
    validatePrivateClone(privateRoot);
    return true;
  }
  return false;
}

function isValidCloneUrl(value) {
  if (!value || /[\s\p{Cc}]/u.test(value)) return false;
  if (/^https:\/\//iu.test(value)) {
    let parsed;
    try {
      parsed = new URL(value);
    } catch {
      return false;
    }
    return (
      parsed.hostname.toLowerCase() === "github.com" &&
      !parsed.username &&
      !parsed.password &&
      !parsed.port &&
      !parsed.search &&
      !parsed.hash &&
      /^\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+(?:\.git)?$/u.test(parsed.pathname)
    );
  }
  return /^ssh:\/\/git@github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+(?:\.git)?$/u.test(
    value,
  );
}

function isValidSecretReference(value) {
  if (!/^op:\/\//iu.test(value ?? "")) return false;
  const parts = value.slice("op://".length).split("/");
  return (
    [3, 4].includes(parts.length) &&
    parts.every(
      (part) => REFERENCE_PART.test(part) && /[A-Za-z0-9._-]/u.test(part),
    )
  );
}

function parseReferenceAssignment(line) {
  if (!line.startsWith(REFERENCE_ASSIGNMENT)) return null;
  let reference = line.slice(REFERENCE_ASSIGNMENT.length);
  if (reference.startsWith('"')) {
    if (reference.length < 2 || !reference.endsWith('"')) return null;
    reference = reference.slice(1, -1);
    if (reference.includes('"')) return null;
  } else if (reference.includes('"') || /\s/u.test(reference)) {
    return null;
  }
  return isValidSecretReference(reference) ? reference : null;
}

function parseArguments() {
  const options = { url: null, reference: null, internal: false };
  for (let index = 2; index < process.argv.length; index += 1) {
    const argument = process.argv[index];
    if (argument === "--internal-clone-from-environment") {
      if (options.internal) fail("Bootstrap modes cannot be repeated.");
      options.internal = true;
      continue;
    }
    if (!["--url", "--op-reference"].includes(argument))
      fail("Unsupported bootstrap argument.");
    const value = process.argv[index + 1];
    if (!value) fail(`${argument} requires a value.`);
    if (argument === "--url") {
      if (options.url) fail("Bootstrap modes cannot be repeated.");
      options.url = value;
    } else {
      if (options.reference) fail("Bootstrap modes cannot be repeated.");
      options.reference = value;
    }
    index += 1;
  }
  if (
    (options.url && options.reference) ||
    (options.internal && (options.url || options.reference))
  )
    fail("Choose either --url or --op-reference, not both.");
  return options;
}

// Find the machine-local reference file: this checkout, then the main checkout
// that owns this worktree.
function findReferenceFile() {
  const roots = [repositoryRoot];
  const common = git([
    "-C",
    repositoryRoot,
    "rev-parse",
    "--path-format=absolute",
    "--git-common-dir",
  ]);
  if (common.status === 0) {
    const mainRoot = resolve(common.stdout.trim(), "..");
    if (mainRoot !== repositoryRoot) roots.push(mainRoot);
  }
  for (const root of roots) {
    const path = join(root, REFERENCE_FILE);
    // Check the entire path before opening it, including dangling symlinks.
    assertSafePath(root, path);
    if (!metadata(path)) continue;
    assertRepository(root);
    assertUntracked(root, REFERENCE_FILE);
    assertProtected(root, REFERENCE_FILE);
    if (!metadata(path).isFile())
      fail("The private repository reference must be a regular file.");
    return path;
  }
  return null;
}

function isWellFormedReferenceFile(path) {
  const lines = readFileSync(path, "utf8").split(/\r?\n/u);
  let assignments = 0;
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    if (!parseReferenceAssignment(trimmed)) return false;
    assignments += 1;
  }
  return assignments === 1;
}

function cloneFromUrl(url) {
  if (!isValidCloneUrl(url)) {
    fail(
      "The private repository location must be a credential-free GitHub HTTPS or ssh://git@github.com URL.",
    );
  }
  if (metadata(privateRoot))
    fail(
      "The private workspace path already exists. Preserve its contents before cloning.",
    );
  const stagingPath = join(
    repositoryRoot,
    `.private-bootstrap-${crypto.randomUUID()}`,
  );
  assertSafePath(repositoryRoot, stagingPath);
  assertUntracked(repositoryRoot, relative(repositoryRoot, stagingPath));
  assertProtected(repositoryRoot, `${relative(repositoryRoot, stagingPath)}/`);
  const temporaryDirectory = mkdtempSync(join(tmpdir(), "guardian-private-"));
  const configPath = join(temporaryDirectory, "git.config");
  let ownsStaging = false;
  const quoted = `"${url.replace(/\\/gu, "\\\\").replace(/"/gu, '\\"')}"`;
  try {
    mkdirSync(stagingPath, { mode: 0o700 });
    ownsStaging = true;
    writeFileSync(configPath, `[url ${quoted}]\n\tinsteadOf = ${ALIAS}\n`, {
      mode: 0o600,
    });
    const clone = git([
      "-c",
      `include.path=${configPath}`,
      "clone",
      "--origin",
      "origin",
      "--",
      ALIAS,
      stagingPath,
    ]);
    if (clone.status !== 0)
      fail(
        "The private repository could not be cloned. Existing workspace files were not changed.",
      );
    validatePrivateClone(stagingPath);
    // Rewrite origin to the real URL by editing the clone's config file so the
    // URL stays out of this helper's direct Git argument list.
    const gitConfig = join(stagingPath, ".git", "config");
    const lines = readFileSync(gitConfig, "utf8").split("\n");
    let inOrigin = false;
    let updated = false;
    for (let index = 0; index < lines.length; index += 1) {
      if (/^\s*\[/u.test(lines[index])) {
        inOrigin = /^\s*\[remote\s+"origin"\]\s*$/u.test(lines[index]);
        continue;
      }
      if (inOrigin && /^\s*url\s*=/u.test(lines[index])) {
        lines[index] = `\turl = ${quoted}`;
        updated = true;
        break;
      }
    }
    if (!updated)
      fail(
        "The private clone did not contain the expected origin configuration.",
      );
    writeFileSync(gitConfig, lines.join("\n"));
    // Recheck publication immediately before the rename; preserve any newly
    // created private/ path instead of replacing it.
    if (metadata(privateRoot))
      fail(
        "The private workspace path changed during setup. Existing files were preserved.",
      );
    assertUntracked(repositoryRoot, "private");
    assertProtected(repositoryRoot, "private/");
    renameSync(stagingPath, privateRoot);
  } finally {
    try {
      if (ownsStaging) rmSync(stagingPath, { recursive: true, force: true });
    } finally {
      rmSync(temporaryDirectory, { recursive: true, force: true });
    }
  }
}

function main() {
  const options = parseArguments();
  let internalUrl;
  if (options.internal) {
    internalUrl = process.env[VARIABLE];
    delete process.env[VARIABLE];
    for (const name of Object.keys(process.env))
      if (name.toUpperCase().startsWith("OP_")) delete process.env[name];
    if (!internalUrl)
      fail(
        "1Password did not provide the private repository location. No workspace was installed.",
      );
  }
  if (preflight()) {
    console.log("The private companion is already installed at private/.");
    return;
  }
  if (options.internal || options.url) {
    cloneFromUrl(options.internal ? internalUrl : options.url);
    console.log("Private companion installed at private/.");
    return;
  }

  let envFile = null;
  let temporaryEnvDirectory = null;
  try {
    if (options.reference) {
      if (!isValidSecretReference(options.reference))
        fail(
          "--op-reference must be a single op://<vault>/<item>/<field> reference.",
        );
      temporaryEnvDirectory = mkdtempSync(
        join(tmpdir(), "guardian-private-ref-"),
      );
      envFile = join(temporaryEnvDirectory, "repository.env.ref");
      writeFileSync(envFile, `${VARIABLE}="${options.reference}"\n`, {
        mode: 0o600,
      });
    } else {
      envFile = findReferenceFile();
      if (!envFile) {
        fail(
          `Create ${REFERENCE_FILE} (containing only ${VARIABLE}=op://<vault>/<item>/<field>) in this checkout or its main worktree, or pass --op-reference / --url.`,
        );
      }
      if (!isWellFormedReferenceFile(envFile)) {
        fail(
          "The private repository reference file must contain exactly one approved variable mapping.",
        );
      }
    }
    let failureMessage = null;
    if (
      spawnSync("op", ["--version"], {
        encoding: "utf8",
        windowsHide: true,
        env: childEnvironment(process.env, { forOp: true }),
      }).status !== 0
    ) {
      failureMessage =
        "1Password CLI is not available. Pass --url to clone without it.";
    } else {
      const run = spawnSync(
        "op",
        [
          "run",
          "--env-file",
          envFile,
          "--",
          process.execPath,
          scriptPath,
          "--internal-clone-from-environment",
        ],
        {
          stdio: ["ignore", "pipe", "pipe"],
          encoding: "utf8",
          windowsHide: true,
          env: childEnvironment(process.env, { forOp: true }),
        },
      );
      if (run.status !== 0 || !existsSync(join(privateRoot, ".git"))) {
        failureMessage =
          "1Password authorization or private workspace setup failed. Existing workspace files were not changed.";
      }
    }
    if (failureMessage) fail(failureMessage);
    validatePrivateClone(privateRoot);
  } finally {
    if (temporaryEnvDirectory)
      rmSync(temporaryEnvDirectory, { recursive: true, force: true });
  }
  console.log("Private companion installed at private/ through 1Password.");
}

try {
  main();
} catch (error) {
  console.error(
    error instanceof SafeBootstrapError
      ? error.message
      : "Private workspace setup stopped because of a local error. Existing workspace files were preserved.",
  );
  process.exitCode = 1;
}
