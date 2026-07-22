import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import {
  normalizeEol,
  parseFrontmatter,
  tomlBasicString,
  tomlMultilineString,
  renderAgentToml,
  renderSkillMarkdown,
  computeOutputs,
  computeExisting,
  diff,
  main,
} from "./sync-agent-configs.mjs";

test("normalizeEol converts CRLF to LF", () => {
  assert.equal(normalizeEol("a\r\nb\r\n"), "a\nb\n");
});

test("parseFrontmatter splits fields from body", () => {
  const { fields, body } = parseFrontmatter(
    "---\nname: demo\ndescription: A demo agent.\ntools: Read, Grep\nmodel: sonnet\n---\n\nBody line one.\n",
  );
  assert.equal(fields.name, "demo");
  assert.equal(fields.description, "A demo agent.");
  assert.equal(fields.tools, "Read, Grep");
  assert.equal(body, "\nBody line one.\n");
});

test("parseFrontmatter keeps colons in a value", () => {
  const { fields } = parseFrontmatter("---\nname: x\ndescription: Use for a: thing\n---\nb\n");
  assert.equal(fields.description, "Use for a: thing");
});

test("parseFrontmatter throws without a frontmatter block", () => {
  assert.throws(() => parseFrontmatter("no frontmatter here\n"), /no YAML frontmatter/);
});

test("tomlBasicString escapes quotes and backslashes", () => {
  assert.equal(tomlBasicString('say "hi"'), '"say \\"hi\\""');
  assert.equal(tomlBasicString("a\\b"), '"a\\\\b"');
});

test("tomlMultilineString uses a literal delimiter so backslashes survive", () => {
  const body = "run .\\startup.ps1\ngrep `(GET)\\(` here";
  const out = tomlMultilineString(body);
  assert.ok(out.startsWith("'''\n"), "expected a literal ''' string");
  assert.ok(out.endsWith("\n'''"));
  assert.ok(out.includes(".\\startup.ps1"), "backslash must not be escaped or dropped");
  assert.ok(out.includes("grep `(GET)\\(` here"));
});

test("tomlMultilineString falls back to a basic string when the body contains '''", () => {
  const body = "here is ''' inside\nand a \\ backslash";
  const out = tomlMultilineString(body);
  assert.ok(out.startsWith('"""\n'), "expected a basic \"\"\" string");
  assert.ok(out.includes("and a \\\\ backslash"), "backslash must be escaped in basic form");
});

test("renderAgentToml emits name and description and drops tools and model", () => {
  const out = renderAgentToml(".claude/agents/demo.md", { name: "demo", description: "A demo." }, "\nBody.\n");
  assert.ok(out.includes('name = "demo"'));
  assert.ok(out.includes('description = "A demo."'));
  assert.ok(out.includes("developer_instructions = '''\nBody.\n'''"));
  // Match on the TOML key at line start: the provenance header legitimately
  // mentions the words "tools" and "model" while explaining why they are dropped.
  assert.ok(!/^tools =/m.test(out), "tools must not be emitted");
  assert.ok(!/^model =/m.test(out), "model must not be emitted");
});

test("renderAgentToml writes a provenance header naming the source", () => {
  const out = renderAgentToml(".claude/agents/demo.md", { name: "demo", description: "d" }, "b");
  assert.ok(out.includes("# GENERATED FILE"));
  assert.ok(out.includes("# Source: .claude/agents/demo.md"));
  assert.ok(out.includes("# Regenerate: node scripts/sync-agent-configs.mjs"));
});

test("renderAgentToml throws when a required field is missing", () => {
  assert.throws(() => renderAgentToml(".claude/agents/demo.md", { name: "demo" }, "b"), /description/);
});

test("renderAgentToml never rewrites CLAUDE.md or .claude/ in the body", () => {
  const body = "Edit `.claude/agents/go-services.md`. `CLAUDE.md` is a thin importer.";
  const out = renderAgentToml(".claude/agents/demo.md", { name: "demo", description: "d" }, body);
  assert.ok(out.includes(".claude/agents/go-services.md"));
  assert.ok(out.includes("`CLAUDE.md` is a thin importer."));
  assert.ok(!out.includes(".Codex/"), "the 2026-07-22 corruption must not reappear");
});

test("renderSkillMarkdown injects provenance inside the frontmatter", () => {
  const out = renderSkillMarkdown(".claude/skills/ship/SKILL.md", "---\nname: ship\ndescription: d\n---\n\nBody.\n");
  assert.ok(out.startsWith("---\n# GENERATED FILE"));
  assert.ok(out.includes("# Source: .claude/skills/ship/SKILL.md"));
  assert.ok(out.includes("name: ship"));
  assert.ok(out.endsWith("\nBody.\n"));
});

test("renderSkillMarkdown copies a file without frontmatter verbatim", () => {
  assert.equal(renderSkillMarkdown(".claude/skills/ship/notes.md", "plain\n"), "plain\n");
});

function fixtureRoot() {
  const root = mkdtempSync(join(tmpdir(), "sync-agent-"));
  mkdirSync(join(root, ".claude/agents"), { recursive: true });
  mkdirSync(join(root, ".claude/skills/demo"), { recursive: true });
  writeFileSync(
    join(root, ".claude/agents/alpha.md"),
    "---\nname: alpha\ndescription: Alpha agent.\ntools: Read\nmodel: sonnet\n---\n\nRun .\\startup.ps1 then edit `.claude/agents/alpha.md`.\n",
  );
  writeFileSync(
    join(root, ".claude/skills/demo/SKILL.md"),
    "---\nname: demo\ndescription: Demo skill.\n---\n\nDemo body.\n",
  );
  return root;
}

test("computeOutputs maps each source to its generated path", () => {
  const root = fixtureRoot();
  const outputs = computeOutputs(root);
  assert.deepEqual([...outputs.keys()].sort(), [".agents/skills/demo/SKILL.md", ".codex/agents/alpha.toml"]);
  assert.ok(outputs.get(".codex/agents/alpha.toml").includes("Run .\\startup.ps1"));
  rmSync(root, { recursive: true, force: true });
});

test("diff reports added, changed, and orphaned files", () => {
  const outputs = new Map([["a", "1"], ["b", "2"]]);
  const existing = new Map([["b", "CHANGED"], ["c", "3"]]);
  assert.deepEqual(diff(outputs, existing), { added: ["a"], changed: ["b"], removed: ["c"] });
});

test("main writes the mirrors, then --check reports no drift", () => {
  const root = fixtureRoot();
  assert.equal(main(["--root", root]), 0);
  const written = readFileSync(join(root, ".codex/agents/alpha.toml"), "utf8");
  assert.ok(written.includes("developer_instructions = '''"));
  assert.equal(computeExisting(root).size, 2);
  assert.equal(main(["--check", "--root", root]), 0);
  rmSync(root, { recursive: true, force: true });
});

test("main --check returns 1 after a source changes", () => {
  const root = fixtureRoot();
  main(["--root", root]);
  writeFileSync(
    join(root, ".claude/agents/alpha.md"),
    "---\nname: alpha\ndescription: Alpha agent, edited.\n---\n\nNew body.\n",
  );
  assert.equal(main(["--check", "--root", root]), 1);
  rmSync(root, { recursive: true, force: true });
});

test("main removes an orphaned generated file", () => {
  const root = fixtureRoot();
  main(["--root", root]);
  writeFileSync(join(root, ".codex/agents/ghost.toml"), "stale\n");
  assert.equal(main(["--check", "--root", root]), 1);
  assert.equal(main(["--root", root]), 0);
  assert.equal(main(["--check", "--root", root]), 0);
  rmSync(root, { recursive: true, force: true });
});
