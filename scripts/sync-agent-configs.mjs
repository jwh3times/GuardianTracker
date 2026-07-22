#!/usr/bin/env node
// Generates the Codex agent and skill mirrors from the Claude Code sources.
//
//   .claude/agents/<name>.md  ->  .codex/agents/<name>.toml
//   .claude/skills/**         ->  .agents/skills/**
//
// The Claude Code tree is the source of truth. Generated files are committed so a
// fresh clone works for both tools; CI re-runs this with --check and fails on drift.
//
// Bodies are copied byte-for-byte. References to `CLAUDE.md` and `.claude/` inside a
// body are deliberately NOT rewritten: they name the source files both tools edit.
//
// Usage:
//   node scripts/sync-agent-configs.mjs           write the mirrors
//   node scripts/sync-agent-configs.mjs --check   report drift, write nothing

import { readFileSync, writeFileSync, mkdirSync, readdirSync, rmSync, existsSync } from "node:fs";
import { join, dirname, relative, sep } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

export const HEADER_LINE = "GENERATED FILE — DO NOT EDIT.";
export const REGEN_LINE = "Regenerate: node scripts/sync-agent-configs.mjs";

const FRONTMATTER = /^---\n([\s\S]*?)\n---\n([\s\S]*)$/;

export function normalizeEol(text) {
  return text.replace(/\r\n/g, "\n");
}

export function parseFrontmatter(text) {
  const match = FRONTMATTER.exec(text);
  if (!match) throw new Error("file has no YAML frontmatter block");
  const fields = {};
  for (const line of match[1].split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const colon = line.indexOf(":");
    if (colon === -1) continue;
    fields[line.slice(0, colon).trim()] = line.slice(colon + 1).trim();
  }
  return { fields, body: match[2] };
}

export function tomlBasicString(value) {
  const escaped = value
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\t/g, "\\t")
    .replace(/\n/g, "\\n");
  return `"${escaped}"`;
}

export function tomlMultilineString(body) {
  // A literal ''' string performs no escape processing, so Windows paths
  // (.\startup.ps1) and regex fragments (grep \() survive verbatim. A basic """
  // string would reject them outright: "Unescaped '\' in a string". Only a body
  // that itself contains ''' forces the escaping basic form.
  if (!body.includes("'''")) return `'''\n${body}\n'''`;
  const escaped = body.replace(/\\/g, "\\\\").replace(/"""/g, '""\\"');
  return `"""\n${escaped}\n"""`;
}

export function renderAgentToml(sourceRelPath, fields, body) {
  for (const key of ["name", "description"]) {
    if (!fields[key]) throw new Error(`${sourceRelPath}: frontmatter is missing '${key}'`);
  }
  return (
    `# ${HEADER_LINE}\n` +
    `# Source: ${sourceRelPath}\n` +
    `# ${REGEN_LINE}\n` +
    `#\n` +
    `# 'tools' and 'model' from the source frontmatter are dropped on purpose: Codex\n` +
    `# has no per-agent tool allowlist and no equivalent model selector.\n` +
    `\n` +
    `name = ${tomlBasicString(fields.name)}\n` +
    `description = ${tomlBasicString(fields.description)}\n` +
    `developer_instructions = ${tomlMultilineString(body.trim())}\n`
  );
}

export function renderSkillMarkdown(sourceRelPath, text) {
  const match = FRONTMATTER.exec(text);
  if (!match) return text;
  return (
    `---\n` +
    `# ${HEADER_LINE}\n` +
    `# Source: ${sourceRelPath}\n` +
    `# ${REGEN_LINE}\n` +
    `${match[1]}\n` +
    `---\n` +
    `${match[2]}`
  );
}

export const REPO_ROOT = fileURLToPath(new URL("..", import.meta.url));

export const AGENT_SRC_DIR = ".claude/agents";
export const AGENT_OUT_DIR = ".codex/agents";
export const SKILL_SRC_DIR = ".claude/skills";
export const SKILL_OUT_DIR = ".agents/skills";

// Skill assets are assumed to be UTF-8 text. The only skill today is markdown.
function walk(dir) {
  const found = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) found.push(...walk(full));
    else if (entry.isFile()) found.push(full);
  }
  return found;
}

const toPosix = (p) => p.split(sep).join("/");

export function computeOutputs(root) {
  const outputs = new Map();

  const agentDir = join(root, AGENT_SRC_DIR);
  if (existsSync(agentDir)) {
    for (const abs of walk(agentDir)) {
      if (!abs.endsWith(".md")) continue;
      const rel = toPosix(relative(agentDir, abs)).replace(/\.md$/, "");
      const sourceRelPath = `${AGENT_SRC_DIR}/${rel}.md`;
      const { fields, body } = parseFrontmatter(normalizeEol(readFileSync(abs, "utf8")));
      outputs.set(`${AGENT_OUT_DIR}/${rel}.toml`, renderAgentToml(sourceRelPath, fields, body));
    }
  }

  const skillDir = join(root, SKILL_SRC_DIR);
  if (existsSync(skillDir)) {
    for (const abs of walk(skillDir)) {
      const rel = toPosix(relative(skillDir, abs));
      const sourceRelPath = `${SKILL_SRC_DIR}/${rel}`;
      const text = normalizeEol(readFileSync(abs, "utf8"));
      const content = rel.endsWith("SKILL.md") ? renderSkillMarkdown(sourceRelPath, text) : text;
      outputs.set(`${SKILL_OUT_DIR}/${rel}`, content);
    }
  }

  return outputs;
}

export function computeExisting(root) {
  const existing = new Map();
  for (const outDir of [AGENT_OUT_DIR, SKILL_OUT_DIR]) {
    const abs = join(root, outDir);
    if (!existsSync(abs)) continue;
    for (const file of walk(abs)) {
      existing.set(`${outDir}/${toPosix(relative(abs, file))}`, normalizeEol(readFileSync(file, "utf8")));
    }
  }
  return existing;
}

export function diff(outputs, existing) {
  const added = [];
  const changed = [];
  const removed = [];
  for (const [rel, content] of outputs) {
    if (!existing.has(rel)) added.push(rel);
    else if (existing.get(rel) !== content) changed.push(rel);
  }
  for (const rel of existing.keys()) if (!outputs.has(rel)) removed.push(rel);
  return { added: added.sort(), changed: changed.sort(), removed: removed.sort() };
}

export function main(argv) {
  const rootFlag = argv.indexOf("--root");
  const root = rootFlag === -1 ? REPO_ROOT : argv[rootFlag + 1];
  const check = argv.includes("--check");

  const outputs = computeOutputs(root);
  const { added, changed, removed } = diff(outputs, computeExisting(root));
  const drifted = added.length + changed.length + removed.length;

  if (check) {
    if (drifted === 0) {
      console.log(`Agent config mirrors are in sync (${outputs.size} generated files).`);
      return 0;
    }
    console.error("Generated agent config mirrors are out of sync with .claude/:");
    for (const rel of added) console.error(`  missing: ${rel}`);
    for (const rel of changed) console.error(`  stale:   ${rel}`);
    for (const rel of removed) console.error(`  orphan:  ${rel}`);
    console.error("\nFix: node scripts/sync-agent-configs.mjs");
    return 1;
  }

  for (const [rel, content] of outputs) {
    const abs = join(root, rel);
    mkdirSync(dirname(abs), { recursive: true });
    writeFileSync(abs, content, "utf8");
  }
  for (const rel of removed) rmSync(join(root, rel), { force: true });

  console.log(
    drifted === 0
      ? `Agent config mirrors already up to date (${outputs.size} files).`
      : `Wrote ${outputs.size} generated files (${added.length} added, ${changed.length} updated, ${removed.length} removed).`,
  );
  return 0;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  process.exit(main(process.argv.slice(2)));
}
