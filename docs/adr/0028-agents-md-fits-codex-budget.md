# ADR 0028: AGENTS.md Fits Codex's Instruction Budget

**Status:** Accepted
**Date:** 2026-11-23
**Supersedes in part:** [ADR 0022](./0022-github-owns-task-status.md); see the last paragraph of
the decision.

## Context

ADR 0022 kept operating context inline in `AGENTS.md` (ports, the environment table, test
commands, and the "Rules for every slice" it moved out of the retired implementation plan) on
the premise that tool-neutral agents cannot follow links out of that file. The file grew to
about 52 KB. Codex reads at most `project_doc_max_bytes` of the `AGENTS.md` chain, 32 KiB by
default, and drops the rest without a warning, so a Codex session was missing the last third of
the file, including the Agent Routing table, the shared-skills block and the MCP notes. The
premise no longer holds either: every agent that works here reads files on request, and the
shared skills already depend on that by reading their settings from `docs/agents/`.

## Decision

The root `AGENTS.md` stays at or below 30,000 bytes, leaving headroom under Codex's 32 KiB for
the blocks `scripts/sync-agents.mjs` writes. It keeps what every session needs: the overview,
ground rules, the development flow, the architecture invariants, the routing table and the
must-never rules. Reference detail lives in the doc that owns its topic, mostly under
`docs/agents/`, and `AGENTS.md` keeps a pointer that names when to read it. Moved text is moved,
not copied, and never goes into a nested `AGENTS.md`, which Codex counts against the same
budget. The `Doc map` in `docs/agents/docs.md` records the limit.

This supersedes exactly two statements in ADR 0022: that agents cannot follow links out of
`AGENTS.md`, and that the "Rules for every slice" live in `AGENTS.md`. They now live in
`docs/agents/dev-flow.md`, which the development flow already requires reading before a change.
Every other part of ADR 0022 stands.

## Consequences

- A Codex session reads the whole of `AGENTS.md` again.
- An agent working on CI, the environment, auth, MCP or local runs reads one more file, named by
  the pointer for that work.
- A change that would push `AGENTS.md` over the limit moves detail out first.
