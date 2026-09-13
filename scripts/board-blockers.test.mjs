import assert from "node:assert/strict";
import { test } from "node:test";

import {
  describeFinding,
  findStaleBlockers,
  loadBoard,
  normalizeItems,
  parseArgs,
  PROJECT_NUMBER,
  PROJECT_OWNER,
} from "./board-blockers.mjs";

function item(title, status, blockedBy, number = null) {
  return { title, status, blockedBy, number };
}

function references(findings) {
  return findings.map(({ item, reference, reason }) => [
    item.title,
    reference,
    reason,
  ]);
}

test("flags an open item still blocked by a slice that is Done (the C2/C3 recurrence)", () => {
  const findings = findStaleBlockers([
    item(
      "refactor(weekly): replace the Efficiency dependency (C2)",
      "Done",
      "",
      233,
    ),
    item(
      "refactor(frontend): add a tolerant difficulty adapter (C3)",
      "Todo",
      "C2",
    ),
  ]);
  assert.deepEqual(references(findings), [
    [
      "refactor(frontend): add a tolerant difficulty adapter (C3)",
      "C2",
      "done",
    ],
  ]);
});

test("flags a blocker named by an issue number that is Done", () => {
  const findings = findStaleBlockers([
    item("Accept the production hosting topology", "Done", "", 13),
    item("Production metrics", "Todo", "Private #13 production design"),
  ]);
  assert.deepEqual(references(findings), [
    ["Production metrics", "#13", "done"],
  ]);
});

test("flags a slice label that no board item carries (a phantom blocker)", () => {
  const findings = findStaleBlockers([
    item("refactor(data): own the Wish list resource (E4)", "Todo", "E3"),
  ]);
  assert.deepEqual(references(findings), [
    ["refactor(data): own the Wish list resource (E4)", "E3", "unknown"],
  ]);
});

test("only a title's closing suffix names its slice, not a mention mid-title", () => {
  const findings = findStaleBlockers([
    item("Notes on the (E3) cleanup, kept for reference", "Done", ""),
    item("own the Wish list resource (E4)", "Todo", "E3"),
  ]);
  assert.deepEqual(references(findings), [
    ["own the Wish list resource (E4)", "E3", "unknown"],
  ]);
});

test("leaves a blocker that is still open alone", () => {
  const findings = findStaleBlockers([
    item("make AuthProvider declarative (E3)", "In Progress", "", 239),
    item("own the Wish list resource (E4)", "Todo", "E3 (#239)"),
  ]);
  assert.deepEqual(findings, []);
});

test("does not check text that was already cleared", () => {
  const board = [
    item("make AuthProvider declarative (E3)", "Done", "", 239),
    item("one", "Todo", "Ready — E3 completed 2026-09-06 (#239)"),
    item("two", "Todo", "Unblocked — E3 completed (#239)"),
    item("three", "Todo", "shipped in v1.3.32 (#239)"),
  ];
  assert.deepEqual(findStaleBlockers(board), []);
});

test("ignores Done items, empty text, None, and prose with nothing to check", () => {
  const board = [
    item("finished (E3)", "Done", "E2"),
    item("empty", "Todo", ""),
    item("none", "Todo", "None"),
    item(
      "prose",
      "Todo",
      "Bungie response-shape verification; ADR 0016 decision",
    ),
  ];
  assert.deepEqual(findStaleBlockers(board), []);
});

test("does not judge an issue number more than one board item carries", () => {
  const findings = findStaleBlockers([
    item("public issue", "Done", "", 13),
    item("private companion issue", "Todo", "", 13),
    item("waiting", "Todo", "#13"),
  ]);
  assert.deepEqual(findings, []);
});

test("reports a reference once however often the text repeats it", () => {
  const findings = findStaleBlockers([
    item("done slice (B2)", "Done", "", 232),
    item("waiting", "Todo", "B2, then B2 again (#232 and #232)"),
  ]);
  assert.deepEqual(references(findings), [
    ["waiting", "B2", "done"],
    ["waiting", "#232", "done"],
  ]);
});

test("normalizes the gh project item-list JSON shape", () => {
  const items = normalizeItems({
    items: [
      {
        title: "Own the Seals resource (E14)",
        status: "Done",
        "blocked By": "Ready — E13 completed",
        content: { number: 304, type: "Issue" },
      },
      { status: "Todo", content: { title: "A draft", type: "DraftIssue" } },
    ],
  });
  assert.deepEqual(items, [
    {
      title: "Own the Seals resource (E14)",
      status: "Done",
      blockedBy: "Ready — E13 completed",
      number: 304,
    },
    { title: "A draft", status: "Todo", blockedBy: "", number: null },
  ]);
});

test("describes a finding with the item, the reference, and the text", () => {
  const text = describeFinding({
    item: item("waiting (C3)", "Todo", " C2 "),
    reference: "C2",
    reason: "done",
  });
  assert.equal(
    text,
    'waiting (C3)\n  Blocked By names C2, which is Done: "C2"',
  );
});

test("parses --input and rejects anything else", () => {
  assert.deepEqual(parseArgs([]), { input: null });
  assert.deepEqual(parseArgs(["--input", "board.json"]), {
    input: "board.json",
  });
  assert.throws(() => parseArgs(["--input"]), /needs a path/u);
  assert.throws(() => parseArgs(["--all"]), /Unknown argument/u);
});

test("reads the board through gh project item-list", () => {
  const calls = [];
  const board = loadBoard({ input: null }, (command, args) => {
    calls.push([command, args]);
    return { status: 0, stdout: '{"items":[]}', stderr: "" };
  });
  assert.deepEqual(board, { items: [] });
  assert.equal(calls.length, 1);
  assert.equal(calls[0][0], "gh");
  assert.deepEqual(calls[0][1].slice(0, 6), [
    "project",
    "item-list",
    PROJECT_NUMBER,
    "--owner",
    PROJECT_OWNER,
    "--format",
  ]);
});

test("fails loudly when gh cannot read the board", () => {
  assert.throws(
    () =>
      loadBoard({ input: null }, () => ({
        status: 1,
        stdout: "",
        stderr: "authentication required\n",
      })),
    /gh project item-list failed: authentication required/u,
  );
});
