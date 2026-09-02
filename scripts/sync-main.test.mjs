import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, test } from "node:test";

import {
  describeResult,
  isDirty,
  parseArgs,
  syncRepositories,
  syncRepository,
  TARGET_BRANCH,
} from "./sync-main.mjs";

const scratch = [];

function temporaryDirectory() {
  const directory = mkdtempSync(join(tmpdir(), "guardian-sync-main-"));
  scratch.push(directory);
  return directory;
}

afterEach(() => {
  for (const directory of scratch.splice(0)) {
    rmSync(directory, { recursive: true, force: true });
  }
});

function git(root, ...args) {
  return execFileSync(
    "git",
    ["-c", "core.excludesFile=", "-C", root, ...args],
    { encoding: "utf8" },
  );
}

function repositoryPair() {
  const fixture = temporaryDirectory();
  const origin = join(fixture, "origin.git");
  const seed = join(fixture, "seed");
  const clone = join(fixture, "clone");

  git(fixture, "init", "--bare", `--initial-branch=${TARGET_BRANCH}`, origin);
  git(fixture, "clone", origin, seed);
  git(seed, "config", "user.email", "test@example.com");
  git(seed, "config", "user.name", "Test");
  writeFileSync(join(seed, "README.md"), "one\n");
  git(seed, "add", ".");
  git(seed, "commit", "-m", "one");
  git(seed, "push", "origin", TARGET_BRANCH);
  git(fixture, "clone", origin, clone);
  git(clone, "config", "user.email", "test@example.com");
  git(clone, "config", "user.name", "Test");

  return { clone, origin };
}

function pushCommit(origin, message) {
  const work = join(temporaryDirectory(), "work");
  git(temporaryDirectory(), "clone", origin, work);
  git(work, "config", "user.email", "test@example.com");
  git(work, "config", "user.name", "Test");
  writeFileSync(join(work, `${message}.txt`), `${message}\n`);
  git(work, "add", ".");
  git(work, "commit", "-m", message);
  git(work, "push", "origin", TARGET_BRANCH);
}

test("arguments default to both repositories and allow public-only sync", () => {
  assert.deepEqual(parseArgs([]), { syncPrivate: true });
  assert.deepEqual(parseArgs(["--skip-private"]), { syncPrivate: false });
  assert.throws(() => parseArgs(["--branch", "release"]), /Unknown argument/u);
});

test("dirty checks and result descriptions are deterministic", () => {
  assert.equal(isDirty("\n"), false);
  assert.equal(isDirty(" M README.md\n"), true);
  assert.equal(
    describeResult({ label: "public", status: "updated", detail: "done" }),
    "+ public: done",
  );
});

test("a missing repository is skipped", () => {
  const result = syncRepository(temporaryDirectory(), "private");
  assert.equal(result.status, "skipped");
  assert.match(result.detail, /no Git repository/u);
});

test("main fast-forwards from origin/main", () => {
  const { clone, origin } = repositoryPair();
  pushCommit(origin, "two");

  const result = syncRepository(clone, "public");

  assert.equal(result.status, "updated");
  assert.match(result.detail, /1 new commit/u);
  assert.equal(git(clone, "branch", "--show-current").trim(), TARGET_BRANCH);
});

test("a clean feature branch switches to main before updating", () => {
  const { clone, origin } = repositoryPair();
  git(clone, "checkout", "-b", "feature/thing");
  pushCommit(origin, "three");

  const result = syncRepository(clone, "public");

  assert.equal(result.status, "updated");
  assert.match(result.detail, /was on feature\/thing/u);
  assert.equal(git(clone, "branch", "--show-current").trim(), TARGET_BRANCH);
});

test("switching from a distinct feature commit does not claim main advanced", () => {
  const { clone } = repositoryPair();
  git(clone, "checkout", "-b", "feature/thing");
  writeFileSync(join(clone, "feature.txt"), "feature\n");
  git(clone, "add", ".");
  git(clone, "commit", "-m", "feature only");

  const result = syncRepository(clone, "public");

  assert.equal(result.status, "current");
  assert.match(result.detail, /was on feature\/thing/u);
  assert.equal(git(clone, "branch", "--show-current").trim(), TARGET_BRANCH);
});

test("a missing local main branch is recreated from origin/main", () => {
  const { clone } = repositoryPair();
  git(clone, "checkout", "-b", "feature/thing");
  git(clone, "branch", "-D", TARGET_BRANCH);

  const result = syncRepository(clone, "public");

  assert.equal(result.status, "current");
  assert.equal(git(clone, "branch", "--show-current").trim(), TARGET_BRANCH);
  assert.equal(
    git(clone, "rev-parse", TARGET_BRANCH).trim(),
    git(clone, "rev-parse", `origin/${TARGET_BRANCH}`).trim(),
  );
});

test("uncommitted changes are refused without switching branches", () => {
  const { clone } = repositoryPair();
  git(clone, "checkout", "-b", "feature/thing");
  writeFileSync(join(clone, "README.md"), "edited\n");

  const result = syncRepository(clone, "public");

  assert.equal(result.status, "failed");
  assert.match(result.detail, /uncommitted changes/u);
  assert.equal(git(clone, "branch", "--show-current").trim(), "feature/thing");
});

test("a detached HEAD with a unique commit is refused during preflight", () => {
  const { clone } = repositoryPair();
  git(clone, "checkout", "--detach");
  writeFileSync(join(clone, "detached.txt"), "detached\n");
  git(clone, "add", ".");
  git(clone, "commit", "-m", "detached only");
  const detachedCommit = git(clone, "rev-parse", "HEAD").trim();

  const result = syncRepository(clone, "public");

  assert.equal(result.status, "failed");
  assert.match(result.detail, /detached HEAD/u);
  assert.equal(git(clone, "branch", "--show-current").trim(), "");
  assert.equal(git(clone, "rev-parse", "HEAD").trim(), detachedCommit);
});

test("a dirty companion prevents either repository from being updated", () => {
  const publicPair = repositoryPair();
  const privatePair = repositoryPair();
  git(publicPair.clone, "checkout", "-b", "feature/public");
  git(privatePair.clone, "checkout", "-b", "feature/private");
  writeFileSync(join(privatePair.clone, "README.md"), "edited\n");

  const results = syncRepositories([
    { root: publicPair.clone, label: "public", options: {} },
    { root: privatePair.clone, label: "private", options: {} },
  ]);

  assert.deepEqual(
    results.map((result) => result.status),
    ["skipped", "failed"],
  );
  assert.equal(
    git(publicPair.clone, "branch", "--show-current").trim(),
    "feature/public",
  );
  assert.equal(
    git(privatePair.clone, "branch", "--show-current").trim(),
    "feature/private",
  );
});

test("diverged main fails without creating a merge commit", () => {
  const { clone, origin } = repositoryPair();
  writeFileSync(join(clone, "local.txt"), "local\n");
  git(clone, "add", ".");
  git(clone, "commit", "-m", "local only");
  const localCommit = git(clone, "rev-parse", "HEAD").trim();
  pushCommit(origin, "remote-only");

  const result = syncRepository(clone, "public");

  assert.equal(result.status, "failed");
  assert.match(result.detail, /fast-forward failed/u);
  assert.equal(git(clone, "rev-parse", "HEAD").trim(), localCommit);
});
