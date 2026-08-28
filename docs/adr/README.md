# Architecture Decision Records

This directory records durable architecture and operating decisions for Guardian
Tracker. ADRs should describe decisions future work must preserve or deliberately
supersede.

Do not include secrets, private infrastructure runbooks, exploit-level security
analysis, or raw private research notes in ADRs. Put that material under
`private/`.

## Status vocabulary

Each record's `Status` line states whether its decision is merely accepted or
actually built:

- **Accepted** — the decision holds and describes how the code works.
- **Accepted — implementation sequenced in #172** — the decision holds, but the
  code does not work this way yet. ADRs 0014–0021 are architecture decisions
  whose implementation is sequenced slice by slice; read them as plans, not as
  descriptions of current behavior.
- **Accepted — implemented in `vX.Y.Z`** — a sequenced decision whose
  implementation has landed.

A record may also carry a **Supersedes in part** or **Superseded in part by**
line. Supersession here is narrow and bidirectional: it names the exact statement
that changed rather than reissuing the record, and both records link to each
other. See [ADR 0021](./0021-own-preferences-synchronization.md), the first one.

## Index

| ADR                                                              | Decision                                       | Delivery status                                                                   |
| ---------------------------------------------------------------- | ---------------------------------------------- | --------------------------------------------------------------------------------- |
| [0001](./0001-public-private-documentation-boundary.md)          | Public and private documentation boundary      | Current                                                                           |
| [0002](./0002-bungie-oauth-and-token-storage.md)                 | Bungie OAuth and token storage                 | Current                                                                           |
| [0003](./0003-manifest-derived-data-and-verify-first-changes.md) | Manifest-derived data and verify-first changes | Current                                                                           |
| [0004](./0004-local-development-and-minikube-scope.md)           | Local development and Minikube scope           | Current                                                                           |
| [0005](./0005-versioning-with-build-tags.md)                     | Versioning with auto-incremented releases      | Current                                                                           |
| [0006](./0006-roles-feature-flags-and-admin-authorization.md)    | Roles, feature flags, and admin authorization  | Current                                                                           |
| [0007](./0007-bungie-public-api-weekly-and-vendor-limits.md)     | Bungie public API weekly and vendor limits     | Current                                                                           |
| [0008](./0008-browser-refresh-cookie.md)                         | Browser refresh credential cookie              | Current                                                                           |
| [0009](./0009-changelog-version-gate.md)                         | Changelog-version gate and version oracle      | Current                                                                           |
| [0010](./0010-manifest-swap-participants-and-observers.md)       | Manifest swap participants and observers       | Current                                                                           |
| [0011](./0011-route-table-as-a-testable-composition-root.md)     | Route table as a testable composition root     | Current                                                                           |
| [0012](./0012-session-issuance-owns-the-session-lifecycle.md)    | Session issuance owns the session lifecycle    | Current                                                                           |
| [0013](./0013-own-the-application-cache-contract.md)             | Application cache ownership                    | Current                                                                           |
| [0014](./0014-own-manifest-derived-publication.md)               | Manifest-derived publication ownership         | Sequenced; generation fence landed in `v1.0.14`, adoption continues               |
| [0015](./0015-own-item-acquisition-facts-in-items.md)            | Item acquisition facts owned by Items          | Sequenced; Items slice landed in `v1.0.15`, downstream adoption continues         |
| [0016](./0016-own-acquisition-recommendation-outcomes.md)        | Acquisition recommendation outcomes            | Sequenced                                                                         |
| [0017](./0017-own-the-browser-session-projection.md)             | Browser session projection                     | Sequenced                                                                         |
| [0018](./0018-own-complete-membership-collections.md)            | Complete membership collections                | Sequenced                                                                         |
| [0019](./0019-own-wish-list-and-preferences.md)                  | Wish list and Preferences ownership            | Sequenced; Preferences backend slice landed in `v1.1.0`                           |
| [0020](./0020-own-frontend-data-access.md)                       | Frontend data access                           | Sequenced                                                                         |
| [0021](./0021-own-preferences-synchronization.md)                | Preferences synchronization                    | Sequenced; backend semantics landed in `v1.1.0`, frontend synchronization remains |
