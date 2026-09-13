#!/usr/bin/env node
/**
 * Report project-board `Blocked By` text that went stale.
 *
 * Usage:
 *
 *   npm run board:blockers
 *   npm run board:blockers -- --input board.json
 *
 * `Blocked By` is free text. A draft has no number, so it cannot carry a native
 * dependency, and nothing clears the text when its blocker closes. An open item
 * whose blocker already finished then drops silently out of the computed ready
 * frontier — which happened twice in one day (issue #314).
 *
 * This lists every open item whose `Blocked By` names a slice label (the
 * `(E3)` suffix on a title) or an issue number that is already Done, or a slice
 * label no board item carries. Text that starts with "Ready", "Unblocked", or
 * "Shipped" has been cleared and names its finished blocker on purpose, so it is
 * not checked. Prose conditions with no label or number cannot be checked.
 *
 * It reads the private board through the local `gh` session, so it runs on a
 * maintainer machine rather than in CI. Exit status: 0 clean, 1 stale text
 * found, 2 the board could not be read.
 */

import { readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { pathToFileURL } from "node:url";

export const PROJECT_NUMBER = "4";
export const PROJECT_OWNER = "jwh3times";

const CLEARED = /^\s*(?:ready|unblocked|shipped)\b/iu;
const TITLE_LABEL = /\(([A-Z]\d{1,2})\)\s*$/u;
const LABEL_REFERENCE = /\b([A-Z]\d{1,2})\b/gu;
const NUMBER_REFERENCE = /#(\d+)\b/gu;

export function parseArgs(argv) {
  const options = { input: null };
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === "--input") {
      const value = argv[index + 1];
      if (!value) throw new Error("--input needs a path");
      options.input = value;
      index += 1;
      continue;
    }
    throw new Error(`Unknown argument: ${argument}`);
  }
  return options;
}

/** Reduces `gh project item-list --format json` output to what the check reads. */
export function normalizeItems(board) {
  return (board?.items ?? []).map((item) => ({
    title: item.title ?? item.content?.title ?? "",
    status: item.status ?? "",
    blockedBy: item["blocked By"] ?? "",
    number: item.content?.number ?? null,
  }));
}

function isDone(item) {
  return item.status === "Done";
}

function indexBy(items, keyOf) {
  const index = new Map();
  for (const item of items) {
    const key = keyOf(item);
    if (key == null) continue;
    index.set(key, [...(index.get(key) ?? []), item]);
  }
  return index;
}

/**
 * Every stale reference in an open item's `Blocked By`, as
 * `{ item, reference, reason }` where reason is "done" or "unknown".
 */
export function findStaleBlockers(items) {
  const byLabel = indexBy(items, (item) => TITLE_LABEL.exec(item.title)?.[1]);
  const byNumber = indexBy(items, (item) => item.number);
  const findings = [];

  for (const item of items) {
    const text = item.blockedBy.trim();
    if (isDone(item) || !text || CLEARED.test(text)) {
      continue;
    }
    const seen = new Set();
    const report = (reference, reason) => {
      if (seen.has(reference)) return;
      seen.add(reference);
      findings.push({ item, reference, reason });
    };

    for (const [, label] of text.matchAll(LABEL_REFERENCE)) {
      const owners = byLabel.get(label) ?? [];
      if (owners.length === 0) report(label, "unknown");
      else if (owners.every(isDone)) report(label, "done");
    }
    for (const [, digits] of text.matchAll(NUMBER_REFERENCE)) {
      // Issue numbers repeat across the public and private repositories, so
      // only a number exactly one board item carries can be judged.
      const owners = byNumber.get(Number(digits)) ?? [];
      if (owners.length === 1 && isDone(owners[0]))
        report(`#${digits}`, "done");
    }
  }
  return findings;
}

export function describeFinding({ item, reference, reason }) {
  const why =
    reason === "done" ? `which is Done` : `which no board item's title carries`;
  return `${item.title}\n  Blocked By names ${reference}, ${why}: "${item.blockedBy.trim()}"`;
}

export function loadBoard(options, run = spawnSync) {
  if (options.input) {
    return JSON.parse(readFileSync(options.input, "utf8"));
  }
  const result = run(
    "gh",
    [
      "project",
      "item-list",
      PROJECT_NUMBER,
      "--owner",
      PROJECT_OWNER,
      "--format",
      "json",
      "--limit",
      "1000",
    ],
    { encoding: "utf8", windowsHide: true, maxBuffer: 64 * 1024 * 1024 },
  );
  if (result.status !== 0) {
    const detail = (result.stderr ?? "").trim() || "no diagnostic output";
    throw new Error(`gh project item-list failed: ${detail}`);
  }
  return JSON.parse(result.stdout);
}

function main() {
  let findings;
  try {
    const options = parseArgs(process.argv.slice(2));
    findings = findStaleBlockers(normalizeItems(loadBoard(options)));
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 2;
    return;
  }

  if (findings.length === 0) {
    console.log("No stale Blocked By text on open board items.");
    return;
  }
  for (const finding of findings) {
    console.log(describeFinding(finding));
  }
  console.log(
    `\n${findings.length} stale Blocked By reference(s). Clear each to ` +
      `"Ready — <blocker> completed <date> (#<n>)", or correct it.`,
  );
  process.exitCode = 1;
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(process.argv[1]).href
) {
  main();
}
