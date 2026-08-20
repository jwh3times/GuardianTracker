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

- [ADR 0001: Public and Private Documentation Boundary](./0001-public-private-documentation-boundary.md)
- [ADR 0002: Bungie OAuth and Token Storage](./0002-bungie-oauth-and-token-storage.md)
- [ADR 0003: Manifest-Derived Data and Verify-First Changes](./0003-manifest-derived-data-and-verify-first-changes.md)
- [ADR 0004: Local Development and Minikube Scope](./0004-local-development-and-minikube-scope.md)
- [ADR 0005: Versioning with Auto-Incremented Releases](./0005-versioning-with-build-tags.md)
- [ADR 0006: Roles, Feature Flags, and Admin Authorization](./0006-roles-feature-flags-and-admin-authorization.md)
- [ADR 0007: Bungie Public API Weekly and Vendor Limits](./0007-bungie-public-api-weekly-and-vendor-limits.md)
- [ADR 0008: Browser Refresh Credential Cookie](./0008-browser-refresh-cookie.md)
- [ADR 0009: Changelog-Version Gate and the Version Oracle](./0009-changelog-version-gate.md)
- [ADR 0010: Manifest Swap Participants and Observers](./0010-manifest-swap-participants-and-observers.md)
- [ADR 0011: Route Table as a Testable Composition Root](./0011-route-table-as-a-testable-composition-root.md)
- [ADR 0012: Session Issuance Owns the Session Lifecycle](./0012-session-issuance-owns-the-session-lifecycle.md)
- [ADR 0013: Own the Application Cache Contract](./0013-own-the-application-cache-contract.md)
- [ADR 0014: Own Manifest-Derived Publication](./0014-own-manifest-derived-publication.md)
- [ADR 0015: Own Item Acquisition Facts in Items](./0015-own-item-acquisition-facts-in-items.md)
- [ADR 0016: Own Acquisition Recommendation Outcomes](./0016-own-acquisition-recommendation-outcomes.md)
- [ADR 0017: Own the Browser Session Projection](./0017-own-the-browser-session-projection.md)
- [ADR 0018: Own Complete Membership Collections](./0018-own-complete-membership-collections.md)
- [ADR 0019: Own Wish List and Preferences](./0019-own-wish-list-and-preferences.md)
- [ADR 0020: Own Frontend Data Access](./0020-own-frontend-data-access.md)
- [ADR 0021: Own Preferences Synchronization](./0021-own-preferences-synchronization.md)
