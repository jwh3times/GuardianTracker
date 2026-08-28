# Documentation Map

Guardian Tracker is a public repository. Its committed documentation must be
safe for public readers and sufficient to build, test, review, and understand
the project without access to a private workspace.

## Sources of Truth

Each kind of information has one living owner:

| Owner                                                                         | Purpose                                                                           |
| ----------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| [README.md](../README.md)                                                     | Concise project overview and quick start                                          |
| [SETUP.md](../SETUP.md)                                                       | Local setup, environment files, ports, runtime options, and test commands         |
| [docs/maintainers/workspace-recovery.md](./maintainers/workspace-recovery.md) | Value-free entry point for authorized-maintainer workspace recovery               |
| [CONTRIBUTING.md](../CONTRIBUTING.md)                                         | Contribution workflow, formatting, and CI expectations                            |
| [CONTEXT.md](../CONTEXT.md)                                                   | Canonical domain vocabulary and seam names                                        |
| [docs/product.md](./product.md)                                               | Durable product goals, users, and experience principles                           |
| [docs/architecture.md](./architecture.md)                                     | Current implemented runtime architecture and data flow                            |
| [docs/adr/](./adr/README.md)                                                  | Accepted decisions future work must preserve or supersede                         |
| [ROADMAP.md](../ROADMAP.md)                                                   | Public work that has not shipped                                                  |
| [CHANGELOG.md](../CHANGELOG.md)                                               | Current shipped changes, with older releases in [`docs/changelog/`](./changelog/) |
| [SECURITY.md](../SECURITY.md)                                                 | Security model, reporting process, and production checklist                       |
| [SUPPORT.md](../SUPPORT.md)                                                   | Public support and issue-reporting guidance                                       |
| [frontend/README.md](../frontend/README.md)                                   | Frontend structure, scripts, and browser-test procedure                           |
| [k8s/README.md](../k8s/README.md)                                             | Minikube validation procedure                                                     |
| [AGENTS.md](../AGENTS.md)                                                     | Canonical, self-contained operating guide for coding agents                       |
| [CLAUDE.md](../CLAUDE.md)                                                     | Thin AGENTS.md importer plus Claude Code-only mechanics                           |
| [docs/agents/](./agents/)                                                     | Per-repository configuration for third-party engineering skills                   |

Community policy is defined by the
[Code of Conduct](../CODE_OF_CONDUCT.md), [support policy](../SUPPORT.md),
[security policy](../SECURITY.md), and GitHub issue and pull-request templates.

## Public and Private Boundary

The ignored `private/` directory may be restored by authorized maintainers as
an independent documentation repository. It is not a submodule, its remote
location is not public configuration, and public contributors do not need it.

Keep these categories private:

- deployment runbooks, cloud resource names, and incident notes;
- private security reviews and exploit-level analysis;
- oversized or operational Bungie API and manifest research;
- detailed implementation handoffs and audit evidence;
- credential-handling and rotation runbooks, value-free rotation records, and
  environment-specific commands.

Never store credential values in documentation. Public docs may acknowledge
that a private runbook exists, but must not depend on private context.

## Maintenance Rules

- Link to the owning document instead of copying its commands, version pins,
  backlog state, or implementation facts into secondary guides.
- Keep `README.md` brief. It is an entry point, not a second setup guide.
- Keep `docs/product.md` durable. Current behavior belongs in architecture or
  feature docs; future work belongs in the roadmap.
- Remove completed work from `ROADMAP.md` and record it in `CHANGELOG.md`.
- Preserve accepted ADRs as history. Supersede a decision with a new ADR rather
  than silently rewriting it.
- Keep deep implementation plans and private operational detail out of
  committed `docs/`.
- `AGENTS.md` deliberately repeats the operating context agents need in one
  self-contained file. Do not move that context into `CLAUDE.md`.
- Edit `.claude/agents/` and `.agents/skills/`, then run the repository sync;
  never hand-edit their generated mirrors.
