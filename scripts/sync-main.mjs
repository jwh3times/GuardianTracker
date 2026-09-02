#!/usr/bin/env node
/**
 * Move the public checkout and optional private companion to main, then
 * fast-forward both from origin/main.
 *
 * Usage:
 *
 *   npm run sync:main
 *   npm run sync:main -- --skip-private
 *
 * The helper refuses uncommitted changes and non-fast-forward updates. It
 * never stashes, discards work, creates merge commits, or rewrites history.
 */

import { existsSync, lstatSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

export const TARGET_BRANCH = "main";

const scriptPath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(scriptPath), "..");
const privateRoot = join(repositoryRoot, "private");

export function parseArgs(argv) {
  const options = { syncPrivate: true };
  for (const argument of argv) {
    if (argument === "--skip-private") {
      options.syncPrivate = false;
      continue;
    }
    throw new Error(`Unknown argument: ${argument}`);
  }
  return options;
}

function withoutGitTracing(environment) {
  const clean = { ...environment };
  for (const name of Object.keys(clean)) {
    if (
      /^GIT_TRACE|^GIT_CURL_VERBOSE$/u.test(name) ||
      name.toUpperCase().startsWith("OP_")
    ) {
      delete clean[name];
    }
  }
  return clean;
}

export function runGit(root, args) {
  const result = spawnSync(
    "git",
    ["-c", "core.excludesFile=", "-C", root, ...args],
    {
      encoding: "utf8",
      env: withoutGitTracing(process.env),
      windowsHide: true,
    },
  );
  return {
    status: result.status ?? 1,
    stdout: result.stdout ?? "",
    stderr: result.stderr ?? "",
  };
}

function firstLine(...values) {
  for (const value of values) {
    const line = value
      .split(/\r?\n/u)
      .map((candidate) => candidate.trim())
      .find(Boolean);
    if (line) return line;
  }
  return "Git returned no diagnostic output";
}

export function isDirty(porcelain) {
  return porcelain.trim().length > 0;
}

const marks = {
  updated: "+",
  current: "=",
  skipped: "-",
  failed: "x",
};

export function describeResult(result) {
  return `${marks[result.status]} ${result.label}: ${result.detail}`;
}

function gitFailure(label, operation, result, redactErrors = false) {
  return {
    label,
    status: "failed",
    detail: redactErrors
      ? operation
      : `${operation}: ${firstLine(result.stderr, result.stdout)}`,
  };
}

function pathsEqual(left, right) {
  const resolvedLeft = resolve(left);
  const resolvedRight = resolve(right);
  return process.platform === "win32"
    ? resolvedLeft.toLowerCase() === resolvedRight.toLowerCase()
    : resolvedLeft === resolvedRight;
}

export function preflightRepository(root, label, options = {}, git = runGit) {
  const { requireIndependentGitDirectory = false, redactErrors = false } =
    options;

  if (!existsSync(join(root, ".git"))) {
    return { label, status: "skipped", detail: `no Git repository at ${root}` };
  }
  if (
    requireIndependentGitDirectory &&
    (lstatSync(root).isSymbolicLink() ||
      !lstatSync(join(root, ".git")).isDirectory() ||
      lstatSync(join(root, ".git")).isSymbolicLink())
  ) {
    return {
      label,
      status: "failed",
      detail: "repository path is not a safe independent Git clone",
    };
  }

  const topLevel = git(root, ["rev-parse", "--show-toplevel"]);
  if (topLevel.status !== 0) {
    return gitFailure(
      label,
      "repository validation failed",
      topLevel,
      redactErrors,
    );
  }
  if (!pathsEqual(topLevel.stdout.trim(), root)) {
    return {
      label,
      status: "failed",
      detail:
        "repository validation failed: path is not an independent Git root",
    };
  }

  const status = git(root, ["status", "--porcelain", "--untracked-files=all"]);
  if (status.status !== 0) {
    return gitFailure(label, "working-tree check failed", status, redactErrors);
  }
  if (isDirty(status.stdout)) {
    return {
      label,
      status: "failed",
      detail: "uncommitted changes; commit or stash them, then run this again",
    };
  }

  const branch = git(root, ["branch", "--show-current"]);
  if (branch.status !== 0) {
    return gitFailure(
      label,
      "current-branch check failed",
      branch,
      redactErrors,
    );
  }
  if (!branch.stdout.trim()) {
    return {
      label,
      status: "failed",
      detail:
        "detached HEAD; create or switch to a branch, then run this again",
    };
  }

  return null;
}

/**
 * Validate, fetch, switch, and fast-forward one repository.
 *
 * The Git runner is injectable so tests can use temporary local remotes and
 * exercise refusal paths without touching the working checkout.
 */
function updateRepository(root, label, options = {}, git = runGit) {
  const { redactErrors = false, redactRevision = false } = options;
  const fetch = git(root, ["fetch", "--prune", "origin"]);
  if (fetch.status !== 0) {
    return gitFailure(label, "fetch failed", fetch, redactErrors);
  }

  const remoteBranch = git(root, [
    "show-ref",
    "--verify",
    "--quiet",
    `refs/remotes/origin/${TARGET_BRANCH}`,
  ]);
  if (remoteBranch.status !== 0) {
    return {
      label,
      status: "failed",
      detail: `origin/${TARGET_BRANCH} does not exist`,
    };
  }

  const previousBranchResult = git(root, ["branch", "--show-current"]);
  if (previousBranchResult.status !== 0) {
    return gitFailure(
      label,
      "current-branch check failed",
      previousBranchResult,
      redactErrors,
    );
  }
  const previousBranch = previousBranchResult.stdout.trim() || "detached HEAD";

  if (previousBranch !== TARGET_BRANCH) {
    const localBranch = git(root, [
      "show-ref",
      "--verify",
      "--quiet",
      `refs/heads/${TARGET_BRANCH}`,
    ]);
    const checkoutArgs =
      localBranch.status === 0
        ? ["checkout", TARGET_BRANCH]
        : [
            "checkout",
            "--track",
            "-b",
            TARGET_BRANCH,
            `origin/${TARGET_BRANCH}`,
          ];
    const checkout = git(root, checkoutArgs);
    if (checkout.status !== 0) {
      return gitFailure(
        label,
        `checkout ${TARGET_BRANCH} failed`,
        checkout,
        redactErrors,
      );
    }
  }

  const beforeResult = git(root, ["rev-parse", "HEAD"]);
  if (beforeResult.status !== 0) {
    return gitFailure(
      label,
      "main revision check failed",
      beforeResult,
      redactErrors,
    );
  }
  const before = beforeResult.stdout.trim();

  const merge = git(root, ["merge", "--ff-only", `origin/${TARGET_BRANCH}`]);
  if (merge.status !== 0) {
    return gitFailure(label, "fast-forward failed", merge, redactErrors);
  }

  const afterResult = git(root, ["rev-parse", "HEAD"]);
  if (afterResult.status !== 0) {
    return gitFailure(
      label,
      "updated revision check failed",
      afterResult,
      redactErrors,
    );
  }
  const after = afterResult.stdout.trim();
  const switched =
    previousBranch === TARGET_BRANCH ? "" : ` (was on ${previousBranch})`;

  if (before === after) {
    return {
      label,
      status: "current",
      detail: redactRevision
        ? `${TARGET_BRANCH} already includes origin/${TARGET_BRANCH}${switched ? "; switched to main" : ""}`
        : `${TARGET_BRANCH} already includes origin/${TARGET_BRANCH} at ${after.slice(0, 7)}${switched}`,
    };
  }

  const countResult = git(root, ["rev-list", "--count", `${before}..${after}`]);
  const count = countResult.status === 0 ? countResult.stdout.trim() : "";
  const commits = count && count !== "0" ? `, ${count} new commit(s)` : "";
  return {
    label,
    status: "updated",
    detail: redactRevision
      ? `${TARGET_BRANCH} updated from origin/${TARGET_BRANCH}${switched ? "; switched to main" : ""}`
      : `${TARGET_BRANCH} now at ${after.slice(0, 7)}${commits}${switched}`,
  };
}

export function syncRepositories(repositories, git = runGit) {
  const preflights = repositories.map((repository) =>
    preflightRepository(
      repository.root,
      repository.label,
      repository.options,
      git,
    ),
  );

  if (preflights.some((result) => result?.status === "failed")) {
    return preflights.map(
      (result, index) =>
        result ?? {
          label: repositories[index].label,
          status: "skipped",
          detail: "not updated because another repository failed preflight",
        },
    );
  }

  return repositories.map(
    (repository, index) =>
      preflights[index] ??
      updateRepository(
        repository.root,
        repository.label,
        repository.options,
        git,
      ),
  );
}

export function syncRepository(root, label, git = runGit) {
  return syncRepositories([{ root, label, options: {} }], git)[0];
}

export function main(argv) {
  const options = parseArgs(argv);
  const repositories = [
    { root: repositoryRoot, label: "public repository", options: {} },
  ];
  const skipped = [];

  if (options.syncPrivate) {
    if (existsSync(join(privateRoot, ".git"))) {
      repositories.push({
        root: privateRoot,
        label: "private repository",
        options: {
          redactErrors: true,
          redactRevision: true,
          requireIndependentGitDirectory: true,
        },
      });
    } else {
      skipped.push({
        label: "private repository",
        status: "skipped",
        detail: "not installed at private/; run `npm run bootstrap:private`",
      });
    }
  }

  const results = [...syncRepositories(repositories), ...skipped];

  for (const result of results) console.log(describeResult(result));
  return results.some((result) => result.status === "failed") ? 1 : 0;
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(process.argv[1]).href
) {
  try {
    process.exit(main(process.argv.slice(2)));
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exit(1);
  }
}
