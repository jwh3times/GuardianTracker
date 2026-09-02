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
// the machine-local file without copying it. The resolved URL never enters a
// process argument or the terminal: `op run` injects it into a child process
// that hands it to git through a temporary `insteadOf` alias.

import {
  existsSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
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

function fail(message) {
  console.error(message);
  process.exit(1);
}

function git(args, options = {}) {
  return spawnSync("git", args, {
    encoding: "utf8",
    windowsHide: true,
    ...options,
  });
}

// Git enables curl/trace output when these variables are merely present, so
// they must be removed rather than set to "0".
function withoutTracing(environment) {
  const clean = { ...environment };
  for (const name of Object.keys(clean)) {
    if (/^GIT_TRACE|^GIT_CURL_VERBOSE$/u.test(name)) delete clean[name];
  }
  return clean;
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
      options.internal = true;
      continue;
    }
    if (!["--url", "--op-reference"].includes(argument))
      fail(`Unknown argument: ${argument}`);
    const value = process.argv[index + 1];
    if (!value) fail(`${argument} requires a value.`);
    if (argument === "--url") options.url = value;
    else options.reference = value;
    index += 1;
  }
  if (options.url && options.reference)
    fail("Choose either --url or --op-reference, not both.");
  return options;
}

// Find the machine-local reference file: this checkout, then the main checkout
// that owns this worktree.
function findReferenceFile() {
  const candidates = [join(repositoryRoot, REFERENCE_FILE)];
  const common = git([
    "-C",
    repositoryRoot,
    "rev-parse",
    "--path-format=absolute",
    "--git-common-dir",
  ]);
  if (common.status === 0) {
    const mainRoot = resolve(common.stdout.trim(), "..");
    if (mainRoot !== repositoryRoot)
      candidates.push(join(mainRoot, REFERENCE_FILE));
  }
  return candidates.find((path) => existsSync(path)) ?? null;
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
  const temporaryDirectory = mkdtempSync(join(tmpdir(), "guardian-private-"));
  const configPath = join(temporaryDirectory, "git.config");
  const quoted = `"${url.replace(/\\/gu, "\\\\").replace(/"/gu, '\\"')}"`;
  try {
    writeFileSync(configPath, `[url ${quoted}]\n\tinsteadOf = ${ALIAS}\n`, {
      mode: 0o600,
    });
    const clone = git(
      ["-c", `include.path=${configPath}`, "clone", "--", ALIAS, privateRoot],
      {
        stdio: ["ignore", "inherit", "inherit"],
        env: withoutTracing(process.env),
      },
    );
    if (clone.status !== 0 || !existsSync(join(privateRoot, ".git"))) {
      fail(
        "The private repository could not be cloned. Existing workspace files were not changed.",
      );
    }
    // Rewrite origin to the real URL by editing the clone's config file so the
    // URL never appears in a process argument.
    const gitConfig = join(privateRoot, ".git", "config");
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
  } finally {
    rmSync(temporaryDirectory, { recursive: true, force: true });
  }
}

const options = parseArguments();

if (options.internal) {
  const url = process.env[VARIABLE];
  delete process.env[VARIABLE];
  for (const name of Object.keys(process.env)) {
    if (name.toUpperCase().startsWith("OP_")) delete process.env[name];
  }
  if (!url)
    fail(
      "1Password did not provide the private repository location. No workspace was installed.",
    );
  cloneFromUrl(url);
  process.exit(0);
}

if (
  git([
    "-C",
    repositoryRoot,
    "rev-parse",
    "--is-inside-work-tree",
  ]).stdout?.trim() !== "true"
) {
  fail("Run this helper from a Guardian Tracker repository checkout.");
}
if (existsSync(join(privateRoot, ".git"))) {
  console.log("The private companion is already installed at private/.");
  process.exit(0);
}
if (existsSync(privateRoot) && readdirSync(privateRoot).length > 0) {
  fail(
    "The private workspace path already exists and is not a Git clone. Preserve and migrate its contents before cloning.",
  );
}

if (options.url) {
  cloneFromUrl(options.url);
  console.log("Private companion installed at private/.");
  process.exit(0);
}

let envFile = null;
let temporaryEnvDirectory = null;
if (options.reference) {
  if (!isValidSecretReference(options.reference))
    fail(
      "--op-reference must be a single op://<vault>/<item>/<field> reference.",
    );
  temporaryEnvDirectory = mkdtempSync(join(tmpdir(), "guardian-private-ref-"));
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
try {
  if (
    spawnSync("op", ["--version"], { encoding: "utf8", windowsHide: true })
      .status !== 0
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
      { stdio: ["ignore", "inherit", "inherit"], windowsHide: true },
    );
    if (run.status !== 0 || !existsSync(join(privateRoot, ".git"))) {
      failureMessage =
        "1Password authorization or private workspace setup failed. Existing workspace files were not changed.";
    }
  }
} finally {
  if (temporaryEnvDirectory)
    rmSync(temporaryEnvDirectory, { recursive: true, force: true });
}
if (failureMessage) fail(failureMessage);
console.log("Private companion installed at private/ through 1Password.");
