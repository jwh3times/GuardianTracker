# Documentation Map

Guardian Tracker is a public GitHub repository. The committed documentation must
be safe for public readers and useful for contributors without exposing private
operations detail.

## Public Docs

These files are committed and maintained as public sources of truth:

| File                                        | Audience                     | Purpose                                                                                               |
| ------------------------------------------- | ---------------------------- | ----------------------------------------------------------------------------------------------------- |
| [README.md](../README.md)                   | Contributors and evaluators  | Project overview, feature summary, quick commands, doc index                                          |
| [SETUP.md](../SETUP.md)                     | Developers                   | Local setup, environment files, ports, tests                                                          |
| [docs/architecture.md](./architecture.md)   | Developers and reviewers     | Runtime architecture, data flow, security posture                                                     |
| [ROADMAP.md](../ROADMAP.md)                 | Contributors and maintainers | Not-yet-implemented work, gates, rough size                                                           |
| [SECURITY.md](../SECURITY.md)               | Users and security reporters | Reporting process, implemented controls, production checklist                                         |
| [CHANGELOG.md](../CHANGELOG.md)             | Users and maintainers        | Shipped changes by version                                                                            |
| [docs/adr/](./adr/README.md)                | Maintainers                  | Durable decisions future work must preserve or supersede                                              |
| [docs/agents/](./agents/)                   | AI coding agents             | Per-repo configuration for third-party engineering skills (issue tracker, triage labels, domain docs) |
| [AGENTS.md](../AGENTS.md)                   | All AI coding agents         | Canonical, tool-neutral operating context and repo rules                                              |
| [CLAUDE.md](../CLAUDE.md)                   | Claude Code                  | Imports AGENTS.md; adds Claude-only mechanics (subagents, skills)                                     |
| [PRD.md](../PRD.md)                         | Product/design work          | Product intent and UX requirements, not implementation truth                                          |
| [frontend/README.md](../frontend/README.md) | Frontend contributors        | React app structure, queries, scripts                                                                 |
| [k8s/README.md](../k8s/README.md)           | Infrastructure contributors  | Minikube validation scripts                                                                           |

## Private Docs

The `private/` directory is gitignored. Keep these categories there:

- deployment runbooks and cloud resource names
- production incident notes
- private security reviews and exploit-level analysis
- raw or derived Bungie API research dumps that are too large or too operational
- detailed implementation handoff plans
- credentials, secret-rotation records, and environment-specific commands

Public docs may reference that private notes exist, but they should not require
private context to build, test, or understand the public code.

## Maintenance Rules

- `README.md` stays concise and links to deeper docs.
- `SETUP.md` owns local setup, ports, and commands.
- `docs/architecture.md` describes implemented architecture only.
- `ROADMAP.md` lists unimplemented work only. Completed work belongs in
  `CHANGELOG.md`; durable choices belong in ADRs.
- ADRs record accepted decisions. If a decision changes, add or supersede an ADR
  instead of silently rewriting history.
- Deep implementation plans should not be committed under `docs/`; keep them
  under `private/`.
- `docs/agents/` is per-repo configuration for third-party engineering skills
  (`.agents/skills/`), not documentation of Guardian Tracker's own architecture
  or behavior. It is scaffolded by the `setup-matt-pocock-skills` skill and
  edited directly afterward, not derived from application code.
