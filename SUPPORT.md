# Support

Thanks for using Guardian Tracker! This page explains where to get help and what
to expect. Guardian Tracker is a side project maintained by a single developer,
so please be patient — but every report and question is read.

## Before you ask

- Check the [README](README.md) for setup, ports, and the three ways to run the
  stack (Docker Compose, Minikube, individual services).
- Check [CONTRIBUTING.md](CONTRIBUTING.md) for build, test, and CI-gate
  requirements, and [CLAUDE.md](CLAUDE.md) for the developer guide.
- Search [existing issues](https://github.com/jwh3times/GuardianTracker/issues?q=is%3Aissue)
  — your question may already be answered.

## Questions and general discussion

For "how do I…?", "is this expected?", or "what's the best way to…?" questions,
open an [issue](https://github.com/jwh3times/GuardianTracker/issues/new/choose)
and add as much context as you can. Please use public channels for general
questions rather than email, so the answers are searchable and help the next
person.

## Reporting bugs

Found something broken? Open an issue using the **Bug report** template and
include reproduction steps, expected vs. actual behavior, the affected component
(API, frontend, database, k8s), and the branch or commit you tested. Check the
**Known Limitations** section of [CLAUDE.md](CLAUDE.md#known-limitations) first —
a few behaviors (e.g. Xûr's location always "Unknown") are upstream API
limitations, not bugs.

## Requesting features

Have an idea? Open an issue using the **Feature request** template. Describe the
problem you're trying to solve, not just the proposed solution — it helps shape
the best fix.

## Security issues

**Do not** report security vulnerabilities through issues, pull requests, or
discussions. Follow the private disclosure process in [SECURITY.md](SECURITY.md)
(email **<jerryholland00@gmail.com>**).

## Code of Conduct concerns

To report behavior that violates our [Code of Conduct](CODE_OF_CONDUCT.md), email
**<conduct@holland.vip>**.

## What to expect

Guardian Tracker is maintained in spare time, so there is no guaranteed support
SLA for general questions or feature requests. As a rough guide:

- **Bugs and questions** — a best-effort response, typically within about a week.
- **Security reports** — acknowledged within 72 hours, per
  [SECURITY.md](SECURITY.md).
- **Feature requests** — reviewed and triaged, but implementation depends on
  available time and project direction.

If you don't hear back on a non-security issue after a couple of weeks, a polite
bump on the thread is welcome — things occasionally get missed.
