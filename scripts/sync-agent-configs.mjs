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
