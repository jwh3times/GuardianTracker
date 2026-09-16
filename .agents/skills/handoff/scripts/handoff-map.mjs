#!/usr/bin/env node
// Reads and updates handoff_map.json in the Proton Drive Handoffs folder.
// Shared by the handoff and lets-go skills so every machine resolves the
// folder, keys the map, and stamps Last_Updated the same way.
//
//   node handoff-map.mjs dir                  print the Handoffs folder
//   node handoff-map.mjs get [repo]           this repo's active entry
//   node handoff-map.mjs set <file> [repo]    make <file> the active entry
//   node handoff-map.mjs clear [repo]         mark the active entry consumed
import { execFileSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { basename, join } from "node:path";

const MAP_FILE = "handoff_map.json";

function fail(message) {
  console.error(`handoff-map: ${message}`);
  process.exit(1);
}

// HANDOFFS_DIR (or the older HANDOFF_DIR) wins; otherwise the single Proton
// Drive account folder that holds My files/Documents/Handoffs/handoff_map.json.
// On a machine without the desktop client, HANDOFFS_DIR is a local mirror the
// skills pull from and push to the cloud folder through the proton-drive CLI.
function resolveDir() {
  const override = process.env.HANDOFFS_DIR || process.env.HANDOFF_DIR;
  if (override) {
    if (!existsSync(join(override, MAP_FILE))) {
      fail(`HANDOFFS_DIR has no ${MAP_FILE}: ${override}`);
    }
    return override;
  }
  const root = join(homedir(), "Proton Drive");
  const accounts = existsSync(root)
    ? readdirSync(root, { withFileTypes: true }).filter((e) => e.isDirectory())
    : [];
  const matches = accounts
    .map((e) => join(root, e.name, "My files", "Documents", "Handoffs"))
    .filter((dir) => existsSync(join(dir, MAP_FILE)));
  if (matches.length === 1) return matches[0];
  if (matches.length === 0) {
    fail(
      `no ${MAP_FILE} under ${root}/<account>/My files/Documents/Handoffs; is Proton Drive running and synced? Set HANDOFFS_DIR to override.`,
    );
  }
  fail(
    `several Handoffs folders found; set HANDOFFS_DIR to one of: ${matches.join(", ")}`,
  );
}

function git(...args) {
  return execFileSync("git", args, {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "ignore"],
  }).trim();
}

// The map is keyed by repository name, taken from origin so a checkout or
// worktree directory name on either machine does not matter.
function repoName() {
  try {
    return git("remote", "get-url", "origin")
      .replace(/\/+$/, "")
      .split(/[/:]/)
      .pop()
      .replace(/\.git$/, "");
  } catch {
    // no origin; fall through to the checkout directory
  }
  try {
    return basename(git("rev-parse", "--show-toplevel"));
  } catch {
    fail("not in a git repository; pass the repo key explicitly");
  }
}

// Reuse an existing key that differs only by case rather than adding a twin.
function keyFor(active, repo) {
  if (Object.hasOwn(active, repo)) return repo;
  return (
    Object.keys(active).find((k) => k.toLowerCase() === repo.toLowerCase()) ??
    repo
  );
}

function stamp(d = new Date()) {
  const p = (n) => String(n).padStart(2, "0");
  return `${p(d.getMonth() + 1)}-${p(d.getDate())}-${d.getFullYear()} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
}

function load(dir) {
  const text = readFileSync(join(dir, MAP_FILE), "utf8");
  const map = JSON.parse(text);
  map.Active_Handoffs ??= {};
  return { map, eol: text.includes("\r\n") ? "\r\n" : "\n" };
}

function save(dir, map, eol) {
  map.Last_Updated = stamp();
  const body = JSON.stringify(map, null, 2).replace(/\n/g, eol);
  writeFileSync(join(dir, MAP_FILE), body + eol);
}

const [command, ...args] = process.argv.slice(2);
const dir = resolveDir();

switch (command) {
  case "dir": {
    console.log(dir);
    break;
  }
  case "get": {
    const { map } = load(dir);
    const repo = keyFor(map.Active_Handoffs, args[0] ?? repoName());
    const file = map.Active_Handoffs[repo] ?? null;
    const path = file && join(dir, file);
    console.log(
      JSON.stringify(
        { repo, file, path, exists: Boolean(path && existsSync(path)) },
        null,
        2,
      ),
    );
    break;
  }
  case "set": {
    const [file, explicitRepo] = args;
    if (!file || basename(file) !== file)
      fail("set needs a bare filename inside the Handoffs folder");
    if (!existsSync(join(dir, file)))
      fail(`no such handoff doc: ${join(dir, file)}`);
    const { map, eol } = load(dir);
    const repo = keyFor(map.Active_Handoffs, explicitRepo ?? repoName());
    const previous = map.Active_Handoffs[repo] ?? null;
    map.Active_Handoffs[repo] = file;
    save(dir, map, eol);
    console.log(JSON.stringify({ repo, file, previous }, null, 2));
    break;
  }
  case "clear": {
    const { map, eol } = load(dir);
    const repo = keyFor(map.Active_Handoffs, args[0] ?? repoName());
    const previous = map.Active_Handoffs[repo] ?? null;
    if (previous !== null) {
      map.Active_Handoffs[repo] = null;
      save(dir, map, eol);
    }
    console.log(JSON.stringify({ repo, previous }, null, 2));
    break;
  }
  default:
    fail(
      "usage: handoff-map.mjs dir | get [repo] | set <file> [repo] | clear [repo]",
    );
}
