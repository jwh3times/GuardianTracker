# Audit Logging — Design Spec

**Date:** 2026-06-20
**Status:** Approved (brainstorming complete; ready for implementation planning)
**Closes:** `private/security-limitations.md` item #4 — "Partial audit logging"

## 1. Problem & Goal

Today only admin-driven role changes are persisted to a durable audit trail
(`role_audit`, written transactionally inside `db.UserStore.SetRoleByID`).
Authentication events (login, token refresh, failed attempts), self-service role
opt-in, and feature-flag changes leave no persistent record. This is the one open
item in `security-limitations.md`.

**Goal:** a single, append-only, queryable audit trail covering authentication,
session-security, role, and feature-flag events, exposed to admins via an API
endpoint and an Audit panel in the existing `/admin` console.

## 2. Scope

**In scope (event groups):**

- **Auth lifecycle** — login success, login failure, logout (this device), logout-all.
- **Session security** — refresh-token reuse detection (session revoked) and refresh
  failures (expired/unknown/revoked session). Routine refresh *success* is intentionally
  **not** logged (fires ~daily per device; low forensic value, high volume).
- **Role changes** — admin-driven (migrate existing `role_audit` history in, keep the
  in-transaction write) + self opt-in via `PUT /api/account/role`.
- **Feature-flag changes** — admin toggles of `enabled` / `min_tier` via
  `PUT /api/admin/flags/:key`.

**Explicitly out of scope:**

- Password / forgot-password events — **do not exist** in this app. Authentication is
  fully delegated to Bungie OAuth; the app never sees or stores a credential (only
  AES-256-GCM-encrypted Bungie OAuth tokens). There is nothing to instrument.
- Account-deletion events — there is no delete-account feature today; building one is
  separate product work, not part of closing the audit-logging gap.
- Routine `refresh.success` (see above).

## 3. Key Decisions

| # | Decision | Choice | Why |
| --- | --- | --- | --- |
| A1 | Read access | Full feature: persist + admin read endpoint + Audit panel in `/admin` | A unified, admin-visible feed is the requested end state. |
| A2 | Schema shape | One generic append-only `audit_log` table; migrate `role_audit` rows in and **drop** `role_audit` | Single source of truth → one read path for the panel; preserves history. |
| A3 | `details` storage | `JSONB` column — **deliberately overrides decision D4** ("no JSONB") | D4's anti-JSONB reasoning was prefs-specific (two fixed fields). Audit payloads are heterogeneous per event type — JSONB's canonical use case. Scoped narrowly to `audit_log.details`; no other table gains JSONB. |
| A4 | Write durability | Two paths: **best-effort** for auth lifecycle/session events (never block the user request); **in-transaction** for role & flag changes (admin-action-always-audited invariant) | A failed audit insert must never break a login/logout/refresh. Admin mutations must be reliably recorded. |
| A5 | IP capture | Store client IP (`INET`, nullable) + truncated User-Agent | Best forensic value. Requires trusted-proxy config (A6) and a SECURITY.md disclosure of IP retention. |
| A6 | Trusted proxies | Configure gin `SetTrustedProxies` (config-driven) before trusting `X-Forwarded-For` | `gin.Default()` currently trusts all proxies, making `c.ClientIP()` spoofable. Correct + non-spoofable IP capture is a prerequisite for A5. |
| A7 | Retention | Hourly pruner deletes rows older than `AUDIT_RETENTION_DAYS` (default **180**) | Bounds table growth and the IP-retention window. Mirrors the existing session pruner. |
| A8 | `login.failure` for `invalid_state` | Logged (not suppressed) | It is the only event an unauthenticated caller can trigger at will (row-spam vector), but it is also a genuine CSRF-probing signal. Retention (A7) bounds growth; no request bodies are stored. |

## 4. Data Model — migration `0004_audit_log.sql`

```sql
CREATE TABLE audit_log (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_type          TEXT        NOT NULL,                 -- 'login.success', 'role.change.admin', ...
    outcome             TEXT        NOT NULL DEFAULT 'success' CHECK (outcome IN ('success','failure')),
    actor_user_id       BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    actor_membership_id TEXT        NOT NULL DEFAULT '',      -- denormalized: login.failure may have no user row
    target_user_id      BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    session_id          TEXT,                                 -- refresh_sessions.id (TEXT) when relevant
    ip                  INET,                                 -- nullable; native Postgres type
    user_agent          TEXT        NOT NULL DEFAULT '',      -- truncated like refresh_sessions
    details             JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_created_idx ON audit_log (created_at DESC, id DESC);  -- keyset paging / default feed
CREATE INDEX audit_log_event_idx   ON audit_log (event_type, created_at DESC);
CREATE INDEX audit_log_actor_idx   ON audit_log (actor_user_id, created_at DESC);
CREATE INDEX audit_log_target_idx  ON audit_log (target_user_id, created_at DESC);

-- Preserve existing role-change history, then retire the specialized table.
INSERT INTO audit_log (event_type, outcome, actor_user_id, target_user_id, details, created_at)
SELECT 'role.change.admin', 'success', actor_user_id, target_user_id,
       jsonb_build_object('oldRole', old_role, 'newRole', new_role), created_at
FROM role_audit;

DROP TABLE role_audit;
```

Notes:

- `actor_user_id` / `target_user_id` use `ON DELETE SET NULL` so deleting a user does
  not erase the audit trail (the denormalized `actor_membership_id` survives for context).
  This differs from `role_audit`'s `target … ON DELETE CASCADE`; the migration of old rows
  is unaffected (it copies historical FKs as-is).
- `session_id` is `TEXT` to match `refresh_sessions.id`.
- The migration runs inside the existing per-file transaction (see `db/migrate.go`), so the
  copy-then-drop is atomic.

## 5. Write Path — `db/audit.go`

```go
type AuditEvent struct {
    EventType         string
    Outcome           string          // "success" | "failure"; default "success" if empty
    ActorUserID       *int64
    ActorMembershipID string
    TargetUserID      *int64
    SessionID         string
    IP                string          // parsed to INET; parse-fail -> NULL
    UserAgent         string          // truncated via truncateUserAgent
    Details           map[string]any  // marshaled to JSONB; nil -> '{}'
}

type AuditStore struct{ pool *pgxpool.Pool }
func NewAuditStore(pool *pgxpool.Pool) *AuditStore

// Best-effort write on the store's own connection (auth lifecycle / session events).
func (s *AuditStore) Log(ctx context.Context, ev AuditEvent) error

// Shared INSERT usable standalone OR inside an existing transaction.
// execer is satisfied by both *pgxpool.Pool and pgx.Tx.
type execer interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
func insertAudit(ctx context.Context, q execer, ev AuditEvent) error

// Read path (admin endpoint).
func (s *AuditStore) List(ctx context.Context, f AuditFilter) (entries []AuditEntry, nextCursor string, err error)

// Retention pruner.
func (s *AuditStore) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
```

- The `execer` interface lets role and flag changes insert into the **same transaction** as
  the mutation, with no cross-store dependency:
  - `db.UserStore.SetRoleByID` replaces its `INSERT INTO role_audit` with
    `insertAudit(ctx, tx, AuditEvent{EventType:"role.change.admin", ...})`.
  - `db.FlagStore.Update` becomes a small transaction: `SELECT` old `enabled`/`min_tier`
    → `UPDATE … RETURNING` → `insertAudit(ctx, tx, AuditEvent{EventType:"flag.update", ...})`.
    Its signature gains the actor (membership id + resolved actor user id).
- Auth-lifecycle / session events call `stores.Audit.Log(...)` best-effort: on error, log via
  `log.Printf` and continue. Never affects the HTTP response.
- Degraded mode: `stores.Audit` is nil when there is no DB pool; all call sites nil-guard
  (no-op), exactly like the other stores. Wire `Audit *AuditStore` into `db.Stores` /
  `db.NewStores`.

## 6. Event Catalog (instrumentation points)

| event_type | outcome | site (file) | actor / target / details |
| --- | --- | --- | --- |
| `login.success` | success | `api/handlers/auth.go` `BungieCallback` (after session created) | actor = user; `details {role}` |
| `login.failure` | failure | `BungieCallback` early-return paths | actor null; `details {reason: "invalid_state" \| "code_exchange" \| "profile_fetch"}` |
| `logout` | success | `auth.go` `Logout` | actor = user; `session_id` |
| `logout.all` | success | `auth.go` `LogoutAll` | actor = user |
| `refresh.reuse` | failure | `auth.go` `RefreshToken` reuse branch | actor = user; `session_id` (highest-value event) |
| `refresh.failure` | failure | `RefreshToken` revoked / expired / unknown-session 401 paths | actor = user; `details {reason: "revoked" \| "expired" \| "unknown_session"}` |
| `role.change.admin` | success | `db/users.go` `SetRoleByID` (in-tx) | actor = admin, target = user; `details {oldRole, newRole}` |
| `role.optin` | success | `api/handlers/account.go` `SetRole` | actor = target = self; `details {oldRole, newRole}` |
| `flag.update` | success | `db/flags.go` `Update` (in-tx) | actor = admin; `details {key, enabled?:[old,new], minTier?:[old,new]}` |

Role labels in `details` use the numeric tier (0–3) consistent with the migrated
`role_audit` history; the read endpoint maps to `auth.RoleName` for display. `role.optin`
captures old→new even though opt-in is self-service, for a complete role-change history.

## 7. IP Capture & Trusted Proxies

- Add config `TrustedProxies []string` (env `TRUSTED_PROXIES`, comma-separated), defaulting
  to loopback for local dev. Call `router.SetTrustedProxies(cfg.TrustedProxies)` in `main.go`.
  On Azure Container Apps, set it to the ingress proxy range so `X-Forwarded-For` is trusted
  only from the real ingress and cannot be spoofed by clients.
- Capture IP via `c.ClientIP()` and store as `INET` (nullable; on parse failure store NULL).
- User-Agent via the existing `truncateUserAgent` helper (already used by `refresh_sessions`).

## 8. Read Path — `GET /api/admin/audit`

- Admin-gated: mounted under the existing `api.Group("/admin", jwtHelper.Middleware(revoker),
  authz.RequireAdmin())`. Returns 503 in degraded mode (consistent with other admin routes).
- Query params:
  - `type` — exact match, or prefix when it ends in `.` (e.g. `role.` matches
    `role.change.admin` + `role.optin`).
  - `actor` — actor membership id.
  - `target` — target membership id (resolved to `target_user_id`).
  - `outcome` — `success` | `failure`.
  - `before` / `after` — RFC3339 time bounds.
  - `cursor` — opaque keyset cursor (encodes last `created_at` + `id`).
  - `limit` — default 50, max 200.
- **Keyset pagination** on `(created_at DESC, id DESC)` → returns `nextCursor` (empty when no
  more rows). No OFFSET (stable under concurrent inserts).
- `LEFT JOIN users` on both `actor_user_id` and `target_user_id` to surface display names.
- Response shape:

  ```json
  {
    "entries": [
      {
        "id": "1024",
        "eventType": "role.change.admin",
        "outcome": "success",
        "actor":  { "membershipId": "4611…", "displayName": "Admin" },
        "target": { "membershipId": "4611…", "displayName": "Guardian" },
        "ip": "203.0.113.7",
        "userAgent": "Mozilla/5.0 …",
        "details": { "oldRole": 0, "newRole": 1 },
        "createdAt": "2026-06-20T17:00:00Z"
      }
    ],
    "nextCursor": "eyJ0IjoiMjAyNi0wNi0yMFQxNzowMDowMFoiLCJpZCI6MTAyNH0="
  }
  ```

Handler lives in `api/handlers/admin.go` (or a new `audit.go` in the same package) and uses a
small `auditStore` consumer interface (mirrors `adminUserStore` / `adminFlagStore`).

## 9. Frontend — Audit Panel in `/admin`

- New section/tab in `pages/Admin.tsx`, beside Users and Flags: a filterable, paginated table.
- Filters: event-type select (grouped: Auth / Session / Role / Flags), actor search box,
  outcome select, time range. "Load more" button drives cursor pagination (append).
- Data: React Query key `['admin', 'audit', filters]`; new `getAuditLog(filters)` in
  `lib/api.ts`; types in `types/api.ts`.
- Presentation: a small `AuditRow` / `AuditTable` in the admin kit
  (`components/kit/admin/`), reusing existing admin styles and `relTime` for timestamps.
  `details` rendered as compact key/value pills; outcome shown via the existing `Badge`
  (failure = warning kind).
- Gating: admin-only, already enforced server-side; the panel is reached only from the
  admin-gated `/admin` route.

## 10. Retention & Pruning

- `startAuditPruner` in `main.go`, mirroring the existing `startSessionPruner`: hourly tick,
  calls `stores.Audit.DeleteOlderThan(ctx, time.Now().AddDate(0,0,-cfg.AuditRetentionDays))`.
- Config `AuditRetentionDays int` (env `AUDIT_RETENTION_DAYS`, default 180). No-op in degraded
  mode (nil store).

## 11. Testing

- **`db/audit_test.go`** (integration, gated on `TEST_DATABASE_URL`):
  - insert → `List` round-trip; filters (type prefix, actor, target, outcome, time range);
    keyset pagination across a page boundary; `DeleteOlderThan` prunes the right rows.
  - `SetRoleByID` writes a `role.change.admin` row **in the same tx** (rolls back together on
    failure); `flag.update` writes in-tx with old→new.
  - migration `0004` copies `role_audit` rows then drops the table (assert row count + absence).
- **Handler tests** (`api/handlers`): auth events recorded via an audit-store spy
  (best-effort: a failing spy does not change the HTTP status); `GET /api/admin/audit`
  filter + pagination + 503 degraded mode; `flag.update` audited.
- **Frontend** (MSW): Audit panel renders rows, changing a filter refetches, "Load more"
  appends the next page.
- Keep `go test ./...`, `npm run type-check`, `npm test`, `npm run build` green; respect the
  CI Go coverage gate (≥60%) and vitest thresholds.

## 12. Documentation Updates

- `CLAUDE.md` — new `audit_log` table + migration `0004`, `db/audit.go`, the
  `GET /api/admin/audit` endpoint, `AUDIT_RETENTION_DAYS` / `TRUSTED_PROXIES` env vars, and the
  Admin page note.
- `SECURITY.md` — authentication + flag events are now audited; **disclose IP + User-Agent
  retention** in the audit log and the retention window; note the trusted-proxy requirement.
- `private/security-limitations.md` — mark item #4 **resolved** (note the residual: auth-event
  audit writes are best-effort, so a DB outage can drop an auth event, by design A4).
- `private/TODO.md` — record the audit-logging work in Completed Work.

## 13. Degraded Mode (no DB)

With no `DATABASE_URL`: `stores.Audit` is nil → all `Log` / in-tx audit calls are no-ops via
nil-guards; the pruner does not start; `GET /api/admin/audit` returns 503 like the other admin
endpoints. The app still runs with `go run .` and zero infrastructure.

## 14. Out-of-Scope / Future

- Exporting / streaming the audit log to an external SIEM.
- Tamper-evidence (hash chaining) — append-only + admin-only read is the v1 bar.
- Per-event rate limiting of `login.failure` beyond retention-based bounding (A8).
- An "active sessions" management UI (the `refresh_sessions` table already supports it; not
  part of this gap).
