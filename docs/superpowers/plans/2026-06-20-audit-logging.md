# Audit Logging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a unified, append-only `audit_log` covering authentication, session-security, role, and feature-flag events, exposed to admins via a `GET /api/admin/audit` endpoint and an Audit panel in `/admin`.

**Architecture:** One generic `audit_log` table is the source of truth (the existing `role_audit` is migrated in and dropped). A `db.AuditStore` provides a best-effort write (auth/session events, never blocks the request) and an in-transaction write primitive (`insertAudit`, used by role and flag mutations). Handlers are instrumented at event sites; a keyset-paginated admin endpoint + a React panel read the feed. Everything is nil-safe in degraded (no-DB) mode.

**Tech Stack:** Go 1.25 / Gin / pgx v5 / Postgres (backend); React 18 / TypeScript / TanStack Query / Vitest + MSW (frontend).

Spec: `docs/superpowers/specs/2026-06-20-audit-logging-design.md`.

## Global Constraints

- **Backend module:** `guardian-tracker/api-service`; all backend paths below are under `backend/api-service/`.
- **DB access:** parameterized pgx queries only — never string-interpolate values into SQL.
- **Degraded mode:** when `DATABASE_URL` is empty, `db.NewStores` returns nil stores. Every new call site must nil-guard. When passing a nil `*db.AuditStore` into an interface parameter, assign through a typed interface variable so the interface is **true-nil** (mirror the `accountHandler`/`adminHandler` nil-branch in `main.go:224-232`) — never pass a typed-nil pointer, which makes `!= nil` true and panics.
- **New deps:** none. Use stdlib `encoding/json`, `encoding/base64`, existing pgx/gin.
- **Migrations:** add exactly one file, `db/migrations/0004_audit_log.sql`; the runner applies it inside a transaction (`db/migrate.go`). Numeric filename prefix drives ordering.
- **Integration tests** live in package `db`, use `testPool(t)` / `createTestUser(t, pool)` (skip when `TEST_DATABASE_URL` unset), and clean up their own rows.
- **CI gates:** Go statement coverage ≥ 60% (`go test -race ./...` with Postgres + cgo); frontend vitest thresholds (lines ≥ 70%, branches ≥ 65%); `gofmt`/Prettier clean.
- **Event type strings (verbatim):** `login.success`, `login.failure`, `logout`, `logout.all`, `refresh.reuse`, `refresh.failure`, `role.change.admin`, `role.optin`, `flag.update`.
- **Outcome values:** `success` | `failure` only.
- **Retention default:** `AUDIT_RETENTION_DAYS=180`. **Trusted proxies env:** `TRUSTED_PROXIES` (comma-separated; empty default).

---

### Task 1: Schema + write primitive + migrate role-change auditing onto `audit_log`

This task is atomic on purpose: the migration **drops** `role_audit`, so `SetRoleByID` must stop writing to it in the same change or the repo breaks.

**Files:**
- Create: `db/migrations/0004_audit_log.sql`
- Create: `db/audit.go`
- Modify: `db/users.go` (`SetRoleByID` — replace the `role_audit` insert)
- Test: `db/audit_test.go` (new), `db/roles_test.go:108-141` (update existing assertion)

**Interfaces:**
- Produces:
  - `type AuditEvent struct { EventType, Outcome, ActorMembershipID, SessionID, IP, UserAgent string; ActorUserID, TargetUserID *int64; Details map[string]any }`
  - `type execer interface { Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) }`
  - `func insertAudit(ctx context.Context, q execer, ev AuditEvent) error`

- [ ] **Step 1: Write the migration file**

Create `db/migrations/0004_audit_log.sql`:

```sql
-- Unified append-only audit trail (closes security-limitations #4).
-- Supersedes the role-change-only role_audit table: its rows are copied in and it
-- is dropped. details is JSONB (deliberately overriding decision D4's no-JSONB rule,
-- which was preferences-specific; audit payloads are heterogeneous per event type).
CREATE TABLE audit_log (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_type          TEXT        NOT NULL,
    outcome             TEXT        NOT NULL DEFAULT 'success' CHECK (outcome IN ('success','failure')),
    actor_user_id       BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    actor_membership_id TEXT        NOT NULL DEFAULT '',
    target_user_id      BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    session_id          TEXT,
    ip                  INET,
    user_agent          TEXT        NOT NULL DEFAULT '',
    details             JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_created_idx ON audit_log (created_at DESC, id DESC);
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

- [ ] **Step 2: Write `db/audit.go` with the write primitive**

Create `db/audit.go`:

```go
package db

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgconn"
)

// AuditEvent is one entry written to audit_log. Zero-value Outcome defaults to
// "success"; empty IP is stored as NULL; nil Details is stored as '{}'.
type AuditEvent struct {
	EventType         string
	Outcome           string
	ActorUserID       *int64
	ActorMembershipID string
	TargetUserID      *int64
	SessionID         string
	IP                string
	UserAgent         string
	Details           map[string]any
}

// execer is satisfied by both *pgxpool.Pool and pgx.Tx, so insertAudit can run
// standalone (best-effort) or inside a caller's transaction (role/flag changes).
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// insertAudit writes one audit row using the given querier (pool or tx).
func insertAudit(ctx context.Context, q execer, ev AuditEvent) error {
	outcome := ev.Outcome
	if outcome == "" {
		outcome = "success"
	}
	detailsJSON := []byte("{}")
	if ev.Details != nil {
		b, err := json.Marshal(ev.Details)
		if err != nil {
			return err
		}
		detailsJSON = b
	}
	var ip any // nil -> NULL::inet; non-empty -> text cast to inet
	if ev.IP != "" {
		ip = ev.IP
	}
	var sessionID any
	if ev.SessionID != "" {
		sessionID = ev.SessionID
	}
	_, err := q.Exec(ctx, `
		INSERT INTO audit_log
			(event_type, outcome, actor_user_id, actor_membership_id,
			 target_user_id, session_id, ip, user_agent, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7::inet, $8, $9::jsonb)`,
		ev.EventType, outcome, ev.ActorUserID, ev.ActorMembershipID,
		ev.TargetUserID, sessionID, ip, truncateUserAgent(ev.UserAgent), string(detailsJSON))
	return err
}
```

(`truncateUserAgent` already exists in `db/users.go`.)

- [ ] **Step 3: Route admin role changes into `audit_log`**

In `db/users.go`, inside `SetRoleByID`, replace the `INSERT INTO role_audit (...)` block (currently `users.go:380-385`) with:

```go
	if err := insertAudit(ctx, tx, AuditEvent{
		EventType:    "role.change.admin",
		ActorUserID:  actorID,
		TargetUserID: &targetUserID,
		Details:      map[string]any{"oldRole": oldRole, "newRole": newRole},
	}); err != nil {
		return nil, err
	}
```

- [ ] **Step 4: Update the existing role-change audit test**

In `db/roles_test.go`, in `TestUserStore_SetRoleByID_BumpsVersionAndAudits`, replace the role_audit assertion (`roles_test.go:133-140`) with:

```go
	// An audit row was written to the unified audit_log (in the same tx).
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_log
		 WHERE event_type = 'role.change.admin' AND target_user_id = $1
		   AND (details->>'newRole')::int = 2`, targetID).Scan(&n); err != nil {
		t.Fatalf("audit query: %v", err)
	}
	if n != 1 {
		t.Errorf("audit_log rows = %d, want 1", n)
	}
```

- [ ] **Step 5: Write the migration + primitive integration test**

Create `db/audit_test.go`:

```go
package db

import (
	"context"
	"testing"
)

func TestMigrate_AuditLogReplacesRoleAudit(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	var auditExists, roleAuditExists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='audit_log')`,
	).Scan(&auditExists); err != nil {
		t.Fatalf("check audit_log: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='role_audit')`,
	).Scan(&roleAuditExists); err != nil {
		t.Fatalf("check role_audit: %v", err)
	}
	if !auditExists {
		t.Error("audit_log table missing after migrate")
	}
	if roleAuditExists {
		t.Error("role_audit table should have been dropped by 0004")
	}
}

func TestInsertAudit_WritesRow(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	mid, uid := createTestUser(t, pool)

	if err := insertAudit(ctx, pool, AuditEvent{
		EventType:         "login.success",
		ActorUserID:       &uid,
		ActorMembershipID: mid,
		IP:                "203.0.113.7",
		UserAgent:         "test-agent",
		Details:           map[string]any{"role": "standard"},
	}); err != nil {
		t.Fatalf("insertAudit: %v", err)
	}

	var eventType, outcome, ip string
	var details []byte
	if err := pool.QueryRow(ctx,
		`SELECT event_type, outcome, host(ip), details
		 FROM audit_log WHERE actor_user_id = $1`, uid,
	).Scan(&eventType, &outcome, &ip, &details); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if eventType != "login.success" || outcome != "success" || ip != "203.0.113.7" {
		t.Errorf("got (%s,%s,%s), want (login.success,success,203.0.113.7)", eventType, outcome, ip)
	}
	// clean up audit rows referencing the test user before its cleanup deletes it
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM audit_log WHERE actor_user_id = $1`, uid) })
}

func TestInsertAudit_EmptyIPStoresNull(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	mid, uid := createTestUser(t, pool)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM audit_log WHERE actor_membership_id = $1`, mid) })

	if err := insertAudit(ctx, pool, AuditEvent{EventType: "logout", ActorMembershipID: mid}); err != nil {
		t.Fatalf("insertAudit: %v", err)
	}
	var ipIsNull bool
	if err := pool.QueryRow(ctx,
		`SELECT ip IS NULL FROM audit_log WHERE actor_membership_id = $1`, mid).Scan(&ipIsNull); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !ipIsNull {
		t.Error("empty IP should store NULL")
	}
}
```

> Note: `audit_log.actor_user_id`/`target_user_id` use `ON DELETE SET NULL`, so a test user's deletion does not cascade-delete audit rows. The `t.Cleanup` above removes audit rows explicitly to keep the table tidy between runs.

- [ ] **Step 6: Run the tests (CI/local with Postgres)**

Run: `cd backend/api-service && ./test-local.ps1 -NoRace` (or set `TEST_DATABASE_URL` and `go test ./db/...`)
Expected: `TestMigrate_AuditLogReplacesRoleAudit`, `TestInsertAudit_WritesRow`, `TestInsertAudit_EmptyIPStoresNull`, and the updated `TestUserStore_SetRoleByID_BumpsVersionAndAudits` PASS. Without `TEST_DATABASE_URL` they SKIP.

- [ ] **Step 7: Verify build + format**

Run: `cd backend/api-service && gofmt -l . && go build ./...`
Expected: no files listed, build succeeds.

- [ ] **Step 8: Commit**

```bash
git add backend/api-service/db/migrations/0004_audit_log.sql backend/api-service/db/audit.go backend/api-service/db/users.go backend/api-service/db/audit_test.go backend/api-service/db/roles_test.go
git commit -m "feat(audit): add audit_log schema + route role changes onto it"
```

---

### Task 2: `AuditStore.Log` + `DeleteOlderThan` + wire into `db.Stores`

**Files:**
- Modify: `db/audit.go` (add `AuditStore`, `Log`, `DeleteOlderThan`)
- Modify: `db/stores.go` (add `Audit` field + construction)
- Test: `db/audit_test.go` (add cases)

**Interfaces:**
- Consumes: `insertAudit`, `AuditEvent` (Task 1).
- Produces:
  - `type AuditStore struct{ pool *pgxpool.Pool }`
  - `func NewAuditStore(pool *pgxpool.Pool) *AuditStore`
  - `func (s *AuditStore) Log(ctx context.Context, ev AuditEvent) error`
  - `func (s *AuditStore) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)`
  - `db.Stores.Audit *AuditStore`

- [ ] **Step 1: Write the failing tests**

Add to `db/audit_test.go`:

```go
func TestAuditStore_LogAndPrune(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewAuditStore(pool)
	mid, _ := createTestUser(t, pool)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM audit_log WHERE actor_membership_id = $1`, mid) })

	if err := store.Log(ctx, AuditEvent{EventType: "logout", ActorMembershipID: mid}); err != nil {
		t.Fatalf("Log: %v", err)
	}
	// Backdate it well past the cutoff and prune.
	if _, err := pool.Exec(ctx,
		`UPDATE audit_log SET created_at = now() - interval '400 days' WHERE actor_membership_id = $1`, mid); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	removed, err := store.DeleteOlderThan(ctx, time.Now().AddDate(0, 0, -180))
	if err != nil {
		t.Fatalf("DeleteOlderThan: %v", err)
	}
	if removed < 1 {
		t.Errorf("removed = %d, want >= 1", removed)
	}
}
```

Add `"time"` to the `db/audit_test.go` imports.

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend/api-service && go build ./db/...`
Expected: FAIL — `NewAuditStore`, `Log`, `DeleteOlderThan` undefined.

- [ ] **Step 3: Implement the store methods**

Append to `db/audit.go`:

```go
import (
	// add to the existing import block:
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditStore is the DB layer for the unified audit trail.
type AuditStore struct{ pool *pgxpool.Pool }

func NewAuditStore(pool *pgxpool.Pool) *AuditStore { return &AuditStore{pool: pool} }

// Log writes one audit event best-effort on the store's own connection. Callers
// that need the write to share a mutation's transaction use insertAudit directly.
func (s *AuditStore) Log(ctx context.Context, ev AuditEvent) error {
	return insertAudit(ctx, s.pool, ev)
}

// DeleteOlderThan removes audit rows created before cutoff, returning the count.
func (s *AuditStore) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM audit_log WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
```

(Merge the new imports into the existing `import (...)` block; `pgconn` and `encoding/json` are already imported.)

- [ ] **Step 4: Wire `Audit` into `db.Stores`**

In `db/stores.go`, add the field and construction:

```go
type Stores struct {
	Users    *UserStore
	Tokens   *BungieTokenStore
	Wishlist *WishlistStore
	Prefs    *PrefsStore
	Flags    *FlagStore
	Audit    *AuditStore
}

func NewStores(pool *pgxpool.Pool) *Stores {
	if pool == nil {
		return &Stores{}
	}
	return &Stores{
		Users:    NewUserStore(pool),
		Tokens:   NewBungieTokenStore(pool),
		Wishlist: NewWishlistStore(pool),
		Prefs:    NewPrefsStore(pool),
		Flags:    NewFlagStore(pool),
		Audit:    NewAuditStore(pool),
	}
}
```

- [ ] **Step 5: Run tests + build**

Run: `cd backend/api-service && ./test-local.ps1 -NoRace` (or `go test ./db/...` with `TEST_DATABASE_URL`)
Expected: `TestAuditStore_LogAndPrune` PASS; build green.

- [ ] **Step 6: Commit**

```bash
git add backend/api-service/db/audit.go backend/api-service/db/stores.go backend/api-service/db/audit_test.go
git commit -m "feat(audit): AuditStore Log + DeleteOlderThan + Stores wiring"
```

---

### Task 3: `AuditStore.List` with filters + keyset pagination

**Files:**
- Modify: `db/audit.go` (add `AuditFilter`, `AuditEntry`, `List`, cursor helpers)
- Test: `db/audit_test.go` (add cases)

**Interfaces:**
- Produces:
  - `type AuditFilter struct { EventType, Actor, Target, Outcome, Cursor string; After, Before time.Time; Limit int }`
  - `type AuditEntry struct { ID int64; EventType, Outcome, ActorMembershipID, ActorDisplayName, TargetMembershipID, TargetDisplayName, IP, UserAgent string; Details map[string]any; CreatedAt time.Time }`
  - `func (s *AuditStore) List(ctx context.Context, f AuditFilter) (entries []AuditEntry, nextCursor string, err error)`

- [ ] **Step 1: Write the failing tests**

Add to `db/audit_test.go`:

```go
func TestAuditStore_ListFiltersAndPaginates(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewAuditStore(pool)
	mid, uid := createTestUser(t, pool)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM audit_log WHERE actor_membership_id = $1`, mid) })

	// Three login.success + one flag.update for this actor.
	for i := 0; i < 3; i++ {
		if err := store.Log(ctx, AuditEvent{EventType: "login.success", ActorUserID: &uid, ActorMembershipID: mid}); err != nil {
			t.Fatalf("seed login: %v", err)
		}
	}
	if err := store.Log(ctx, AuditEvent{EventType: "flag.update", ActorUserID: &uid, ActorMembershipID: mid,
		Details: map[string]any{"key": "god-roll"}}); err != nil {
		t.Fatalf("seed flag: %v", err)
	}

	// Filter by exact event type + actor.
	got, _, err := store.List(ctx, AuditFilter{EventType: "login.success", Actor: mid, Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("login.success rows = %d, want 3", len(got))
	}
	if got[0].ActorDisplayName != "Test Guardian" {
		t.Errorf("actor display = %q, want Test Guardian", got[0].ActorDisplayName)
	}

	// Prefix filter "login." matches the same 3; keyset paginate at limit 2.
	page1, cursor, err := store.List(ctx, AuditFilter{EventType: "login.", Actor: mid, Limit: 2})
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	if len(page1) != 2 || cursor == "" {
		t.Fatalf("page1 = %d rows, cursor=%q; want 2 rows + cursor", len(page1), cursor)
	}
	page2, _, err := store.List(ctx, AuditFilter{EventType: "login.", Actor: mid, Limit: 2, Cursor: cursor})
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("page2 = %d rows, want 1", len(page2))
	}
	// Pages must not overlap (DESC by created_at,id).
	if page1[1].ID <= page2[0].ID {
		t.Errorf("pagination overlap: page1 last id %d, page2 first id %d", page1[1].ID, page2[0].ID)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend/api-service && go build ./db/...`
Expected: FAIL — `AuditFilter`, `AuditEntry`, `List` undefined.

- [ ] **Step 3: Implement `List` + cursor helpers**

Append to `db/audit.go` (add `"encoding/base64"`, `"fmt"`, `"strings"` to the import block):

```go
// AuditFilter narrows the audit feed. EventType matches exactly, or as a prefix
// when it ends in '.' (e.g. "role." -> role.change.admin + role.optin). Cursor is
// an opaque keyset token returned by a prior List call.
type AuditFilter struct {
	EventType string
	Actor     string // actor membership id
	Target    string // target membership id
	Outcome   string
	After     time.Time
	Before    time.Time
	Cursor    string
	Limit     int
}

// AuditEntry is one row joined to user display names for the admin feed.
type AuditEntry struct {
	ID                 int64
	EventType          string
	Outcome            string
	ActorMembershipID  string
	ActorDisplayName   string
	TargetMembershipID string
	TargetDisplayName  string
	IP                 string
	UserAgent          string
	Details            map[string]any
	CreatedAt          time.Time
}

type auditCursor struct {
	T  time.Time `json:"t"`
	ID int64     `json:"id"`
}

func encodeCursor(t time.Time, id int64) string {
	b, _ := json.Marshal(auditCursor{T: t, ID: id})
	return base64.URLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (auditCursor, bool) {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return auditCursor{}, false
	}
	var c auditCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return auditCursor{}, false
	}
	return c, true
}

// List returns audit entries newest-first with keyset pagination. nextCursor is
// non-empty when more rows remain.
func (s *AuditStore) List(ctx context.Context, f AuditFilter) ([]AuditEntry, string, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var conds []string
	var args []any
	add := func(cond string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}

	if f.EventType != "" {
		if strings.HasSuffix(f.EventType, ".") {
			add("a.event_type LIKE $%d", f.EventType+"%")
		} else {
			add("a.event_type = $%d", f.EventType)
		}
	}
	if f.Actor != "" {
		add("a.actor_membership_id = $%d", f.Actor)
	}
	if f.Target != "" {
		add("a.target_user_id = (SELECT id FROM users WHERE membership_id = $%d)", f.Target)
	}
	if f.Outcome != "" {
		add("a.outcome = $%d", f.Outcome)
	}
	if !f.After.IsZero() {
		add("a.created_at >= $%d", f.After)
	}
	if !f.Before.IsZero() {
		add("a.created_at <= $%d", f.Before)
	}
	if c, ok := decodeCursor(f.Cursor); ok {
		args = append(args, c.T, c.ID)
		conds = append(conds, fmt.Sprintf("(a.created_at, a.id) < ($%d, $%d)", len(args)-1, len(args)))
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	args = append(args, limit+1) // fetch one extra to detect "more"

	query := fmt.Sprintf(`
		SELECT a.id, a.event_type, a.outcome,
		       a.actor_membership_id, COALESCE(au.display_name, ''),
		       COALESCE(tu.membership_id, ''), COALESCE(tu.display_name, ''),
		       COALESCE(host(a.ip), ''), a.user_agent, a.details, a.created_at
		FROM audit_log a
		LEFT JOIN users au ON au.id = a.actor_user_id
		LEFT JOIN users tu ON tu.id = a.target_user_id
		%s
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $%d`, where, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var detailsRaw []byte
		if err := rows.Scan(&e.ID, &e.EventType, &e.Outcome,
			&e.ActorMembershipID, &e.ActorDisplayName,
			&e.TargetMembershipID, &e.TargetDisplayName,
			&e.IP, &e.UserAgent, &detailsRaw, &e.CreatedAt); err != nil {
			return nil, "", err
		}
		if len(detailsRaw) > 0 {
			_ = json.Unmarshal(detailsRaw, &e.Details)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(out) > limit {
		last := out[limit-1]
		nextCursor = encodeCursor(last.CreatedAt, last.ID)
		out = out[:limit]
	}
	return out, nextCursor, nil
}
```

- [ ] **Step 4: Run tests + build + format**

Run: `cd backend/api-service && gofmt -l . && ./test-local.ps1 -NoRace`
Expected: `TestAuditStore_ListFiltersAndPaginates` PASS; no gofmt output; build green.

- [ ] **Step 5: Commit**

```bash
git add backend/api-service/db/audit.go backend/api-service/db/audit_test.go
git commit -m "feat(audit): AuditStore.List with filters + keyset pagination"
```

---

### Task 4: Audit feature-flag changes (in-transaction)

**Files:**
- Modify: `db/flags.go` (`Update` → transaction + actor + audit)
- Modify: `api/handlers/admin.go` (`adminFlagStore` interface + `UpdateFlag` call site)
- Test: `db/audit_test.go` (add case)

**Interfaces:**
- Consumes: `insertAudit`, `AuditEvent`.
- Produces (changed signature):
  - `func (s *FlagStore) Update(ctx context.Context, key string, enabled *bool, minTier *int16, actorUserID *int64, actorMembershipID string) (*FeatureFlag, error)`

- [ ] **Step 1: Write the failing test**

Add to `db/audit_test.go`:

```go
func TestFlagStore_UpdateWritesAudit(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	flags := NewFlagStore(pool)
	_, adminUID := createTestUser(t, pool)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM audit_log WHERE event_type = 'flag.update' AND actor_user_id = $1`, adminUID) })

	enabled := true
	if _, err := flags.Update(ctx, "god-roll", &enabled, nil, &adminUID, "admin-mid"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_log
		 WHERE event_type='flag.update' AND actor_user_id=$1
		   AND details->>'key'='god-roll'`, adminUID).Scan(&n); err != nil {
		t.Fatalf("audit query: %v", err)
	}
	if n != 1 {
		t.Errorf("flag.update audit rows = %d, want 1", n)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend/api-service && go build ./db/...`
Expected: FAIL — `Update` arity mismatch (wants 4 args, test passes 6).

- [ ] **Step 3: Rewrite `FlagStore.Update` as a transaction with audit**

Replace `FlagStore.Update` in `db/flags.go` with:

```go
// Update patches the mutable fields of a flag and records a flag.update audit row
// in the same transaction. nil enabled/minTier are left unchanged. The audit
// details capture only the fields that actually changed (old->new). Returns
// pgx.ErrNoRows when the key does not exist.
func (s *FlagStore) Update(ctx context.Context, key string, enabled *bool, minTier *int16, actorUserID *int64, actorMembershipID string) (*FeatureFlag, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful commit

	var oldEnabled bool
	var oldMinTier int16
	err = tx.QueryRow(ctx,
		`SELECT enabled, min_tier FROM feature_flags WHERE key = $1 FOR UPDATE`, key,
	).Scan(&oldEnabled, &oldMinTier)
	if err != nil {
		return nil, err // pgx.ErrNoRows when missing
	}

	var f FeatureFlag
	if err := tx.QueryRow(ctx,
		`UPDATE feature_flags
		    SET enabled    = COALESCE($2, enabled),
		        min_tier   = COALESCE($3, min_tier),
		        updated_at = now()
		  WHERE key = $1
		RETURNING key, name, description, category, min_tier, enabled, sort_order, updated_at`,
		key, enabled, minTier,
	).Scan(&f.Key, &f.Name, &f.Description, &f.Category,
		&f.MinTier, &f.Enabled, &f.SortOrder, &f.UpdatedAt); err != nil {
		return nil, err
	}

	details := map[string]any{"key": key}
	if enabled != nil && *enabled != oldEnabled {
		details["enabled"] = []any{oldEnabled, *enabled}
	}
	if minTier != nil && *minTier != oldMinTier {
		details["minTier"] = []any{oldMinTier, *minTier}
	}
	if err := insertAudit(ctx, tx, AuditEvent{
		EventType:         "flag.update",
		ActorUserID:       actorUserID,
		ActorMembershipID: actorMembershipID,
		Details:           details,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &f, nil
}
```

(The `pgx.ErrNoRows` special-case the old code had is preserved: the `SELECT ... FOR UPDATE` returns `pgx.ErrNoRows` for an unknown key, which the handler already maps to 404.)

- [ ] **Step 4: Update the admin handler interface + call site**

In `api/handlers/admin.go`, change the `adminFlagStore` interface `Update` signature and the `UpdateFlag` call. Replace the interface (`admin.go:26-29`):

```go
type adminFlagStore interface {
	List(ctx context.Context) ([]db.FeatureFlag, error)
	Update(ctx context.Context, key string, enabled *bool, minTier *int16, actorUserID *int64, actorMembershipID string) (*db.FeatureFlag, error)
}
```

In `UpdateFlag`, replace the `h.flags.Update(...)` call (`admin.go:181`) with:

```go
	actorMID := c.GetString("membership_id")
	flag, err := h.flags.Update(c.Request.Context(), key, body.Enabled, minTier, actorUserIDFromContext(c), actorMID)
```

Add this helper at the bottom of `admin.go`:

```go
// actorUserIDFromContext returns the numeric user id when the middleware stored
// one, else nil. The audit actor falls back to the denormalized membership id.
func actorUserIDFromContext(c *gin.Context) *int64 {
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(int64); ok {
			return &id
		}
	}
	return nil
}
```

> Note: the JWT middleware does not currently set `user_id`, so `actorUserIDFromContext` returns nil today; the audit row still records `actor_membership_id`. Wiring `user_id` into the middleware is out of scope (the denormalized membership id is sufficient for the panel). The helper is future-proof for when it lands.

- [ ] **Step 5: Run tests + build + format**

Run: `cd backend/api-service && gofmt -l . && go build ./... && ./test-local.ps1 -NoRace`
Expected: `TestFlagStore_UpdateWritesAudit` PASS; existing admin handler tests still compile/pass; build green.

- [ ] **Step 6: Commit**

```bash
git add backend/api-service/db/flags.go backend/api-service/api/handlers/admin.go backend/api-service/db/audit_test.go
git commit -m "feat(audit): audit feature-flag changes in-transaction"
```

---

### Task 5: Config — retention window + trusted proxies

**Files:**
- Modify: `config/config.go` (struct + `Load`)
- Test: `config/config_test.go` (add cases; create the file if absent)

**Interfaces:**
- Produces: `Config.AuditRetentionDays int`, `Config.TrustedProxies []string`.

- [ ] **Step 1: Write the failing test**

Create or append to `config/config_test.go`:

```go
package config

import (
	"testing"
)

func TestLoad_AuditDefaults(t *testing.T) {
	t.Setenv("AUDIT_RETENTION_DAYS", "")
	t.Setenv("TRUSTED_PROXIES", "")
	c := Load()
	if c.AuditRetentionDays != 180 {
		t.Errorf("AuditRetentionDays = %d, want 180", c.AuditRetentionDays)
	}
	if len(c.TrustedProxies) != 0 {
		t.Errorf("TrustedProxies = %v, want empty", c.TrustedProxies)
	}
}

func TestLoad_AuditOverrides(t *testing.T) {
	t.Setenv("AUDIT_RETENTION_DAYS", "30")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8, 127.0.0.1")
	c := Load()
	if c.AuditRetentionDays != 30 {
		t.Errorf("AuditRetentionDays = %d, want 30", c.AuditRetentionDays)
	}
	if len(c.TrustedProxies) != 2 || c.TrustedProxies[0] != "10.0.0.0/8" || c.TrustedProxies[1] != "127.0.0.1" {
		t.Errorf("TrustedProxies = %v, want [10.0.0.0/8 127.0.0.1]", c.TrustedProxies)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend/api-service && go test ./config/... -run TestLoad_Audit -v`
Expected: FAIL — `AuditRetentionDays` / `TrustedProxies` undefined.

- [ ] **Step 3: Add the config fields**

In `config/config.go`, add to the `Config` struct (after `AdminMembershipIDs`):

```go
	// AuditRetentionDays bounds how long audit_log rows (and the IPs they carry)
	// are retained; an hourly pruner deletes older rows.
	AuditRetentionDays int
	// TrustedProxies are CIDRs/IPs gin trusts for X-Forwarded-For when resolving
	// the client IP recorded in the audit log. Empty in local dev.
	TrustedProxies []string
```

In `Load()`, add to the returned struct literal (after `AdminMembershipIDs: ...`):

```go
		AuditRetentionDays: getIntEnv("AUDIT_RETENTION_DAYS", 180),
		TrustedProxies:     parseCSV(os.Getenv("TRUSTED_PROXIES")),
```

- [ ] **Step 4: Run tests**

Run: `cd backend/api-service && go test ./config/... -run TestLoad_Audit -v`
Expected: PASS (both cases).

- [ ] **Step 5: Commit**

```bash
git add backend/api-service/config/config.go backend/api-service/config/config_test.go
git commit -m "feat(audit): config for retention window + trusted proxies"
```

---

### Task 6: Instrument auth handler (login, logout, refresh)

**Files:**
- Modify: `api/handlers/auth.go` (constructor + instrumentation + `AuditLogger` interface)
- Modify: `main.go` (update `NewAuthHandler` call to pass audit logger true-nil-safe)
- Test: `api/handlers/auth_audit_test.go` (new)

**Interfaces:**
- Consumes: `db.AuditEvent`, `db.AuditStore` (satisfies `AuditLogger`).
- Produces:
  - `type AuditLogger interface { Log(ctx context.Context, ev db.AuditEvent) error }` (in `handlers`)
  - `NewAuthHandler(..., audit AuditLogger)` — audit is the new trailing param.

- [ ] **Step 1: Write the failing test**

Create `api/handlers/auth_audit_test.go`:

```go
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
)

// spyAudit records events for assertions and can simulate a write failure.
type spyAudit struct {
	mu     sync.Mutex
	events []db.AuditEvent
	err    error
}

func (s *spyAudit) Log(_ context.Context, ev db.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	return s.err
}

func (s *spyAudit) types() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.events))
	for i, e := range s.events {
		out[i] = e.EventType
	}
	return out
}

func TestRefreshToken_InvalidToken_AuditsFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spy := &spyAudit{}
	// jwt/tokenStore/cfg are unused on the invalid-token early-return path; a
	// minimal handler with only the audit logger set exercises it.
	h := &AuthHandler{audit: spy}

	r := gin.New()
	r.POST("/api/auth/refresh", h.RefreshToken)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh",
		strings.NewReader(`{"refreshToken":"not-a-jwt"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if got := spy.types(); len(got) != 1 || got[0] != "refresh.failure" {
		t.Errorf("audit events = %v, want [refresh.failure]", got)
	}
}

func TestAudit_BestEffort_DoesNotBlockResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spy := &spyAudit{err: context.DeadlineExceeded} // audit write fails
	h := &AuthHandler{audit: spy}

	r := gin.New()
	r.POST("/api/auth/refresh", h.RefreshToken)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh",
		strings.NewReader(`{"refreshToken":"not-a-jwt"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Audit failure must not change the HTTP outcome.
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 despite audit error", w.Code)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend/api-service && go build ./api/handlers/...`
Expected: FAIL — `AuthHandler` has no field `audit`; `AuditLogger` undefined.

- [ ] **Step 3: Add the `AuditLogger` interface, handler field, and helper**

In `api/handlers/auth.go`, add the interface and a best-effort helper, and the field. Add near the top (after imports):

```go
// AuditLogger records audit events best-effort. Satisfied by *db.AuditStore;
// nil in degraded mode (no DB).
type AuditLogger interface {
	Log(ctx context.Context, ev db.AuditEvent) error
}
```

Add `"guardian-tracker/api-service/db"` to the imports.

Add `audit AuditLogger` to the `AuthHandler` struct (after `revoker`):

```go
	audit AuditLogger // nil in degraded mode
```

Change `NewAuthHandler` to accept and store it (new trailing param):

```go
func NewAuthHandler(j *auth.JWT, ts *auth.TokenStore, cfg *config.Config, userStore UserStore, revokeCache cache.Cache, revoker *auth.RevocationChecker, audit AuditLogger) *AuthHandler {
	return &AuthHandler{
		jwt:         j,
		tokenStore:  ts,
		cfg:         cfg,
		state:       auth.NewStateSigner(cfg.JWTSecret),
		userStore:   userStore,
		revokeCache: revokeCache,
		revoker:     revoker,
		audit:       audit,
	}
}
```

Add the helper at the bottom of `auth.go`:

```go
// logAudit writes one event best-effort: it never blocks or fails the request.
// IP and User-Agent are taken from the request; the caller sets the rest.
func (h *AuthHandler) logAudit(c *gin.Context, ev db.AuditEvent) {
	if h.audit == nil {
		return
	}
	ev.IP = c.ClientIP()
	ev.UserAgent = c.Request.UserAgent()
	if err := h.audit.Log(c.Request.Context(), ev); err != nil {
		log.Printf("audit %s: %v", ev.EventType, err)
	}
}
```

- [ ] **Step 4: Instrument the event sites**

In `BungieCallback`, capture the upserted user id and log success. Change the upsert block (`auth.go:112-121`) to keep `id`:

```go
	var actorUserID *int64
	if h.userStore != nil {
		forceAdmin := h.cfg.IsBootstrapAdmin(profile.MembershipID)
		id, tv, r, err := h.userStore.Upsert(c.Request.Context(), profile.MembershipID, int16(profile.MembershipType), profile.DisplayName, forceAdmin)
		if err != nil {
			log.Printf("user upsert failed for %s: %v", profile.MembershipID, err)
		} else {
			tokenVersion = tv
			role = int(r)
			uid := id
			actorUserID = &uid
		}
	}
```

Add `login.failure` logging on the two pre-exchange early returns in `BungieCallback`. Replace the invalid-code return (`auth.go:85-88`):

```go
	if code == "" || len(code) > 500 {
		h.logAudit(c, db.AuditEvent{EventType: "login.failure", Outcome: "failure",
			Details: map[string]any{"reason": "invalid_code"}})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid authorization code"})
		return
	}
```

Replace the invalid-state return (`auth.go:89-92`):

```go
	if state == "" || !h.state.Verify(state, time.Now(), oauthStateTTL) {
		h.logAudit(c, db.AuditEvent{EventType: "login.failure", Outcome: "failure",
			Details: map[string]any{"reason": "invalid_state"}})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired state. Please try logging in again."})
		return
	}
```

Replace the code-exchange failure return (`auth.go:94-99`):

```go
	tokenResp, err := h.exchangeCode(code)
	if err != nil {
		log.Printf("Error exchanging code for token: %v", err)
		h.logAudit(c, db.AuditEvent{EventType: "login.failure", Outcome: "failure",
			Details: map[string]any{"reason": "code_exchange"}})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete authentication"})
		return
	}
```

Replace the profile-fetch failure return (`auth.go:101-106`):

```go
	profile, err := h.getBungieProfile(tokenResp.AccessToken)
	if err != nil {
		log.Printf("Error getting user profile: %v", err)
		h.logAudit(c, db.AuditEvent{EventType: "login.failure", Outcome: "failure",
			Details: map[string]any{"reason": "profile_fetch"}})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		return
	}
```

Add `login.success` just before the final `c.JSON(http.StatusOK, ...)` in `BungieCallback` (after the session is created, ~`auth.go:163`):

```go
	h.logAudit(c, db.AuditEvent{
		EventType:         "login.success",
		ActorUserID:       actorUserID,
		ActorMembershipID: profile.MembershipID,
		SessionID:         sessionID,
		Details:           map[string]any{"role": auth.RoleName(role)},
	})
```

In `RefreshToken`, add audit on the failure/reuse paths. Replace the invalid-token return (`auth.go:188-192`):

```go
	claims, err := h.jwt.ValidateToken(body.RefreshToken)
	if err != nil || claims.TokenType != "refresh" {
		h.logAudit(c, db.AuditEvent{EventType: "refresh.failure", Outcome: "failure",
			Details: map[string]any{"reason": "invalid_token"}})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}
```

Replace the revoked-by-token-version return (`auth.go:195-200`):

```go
	if h.revoker != nil {
		if err := h.revoker.Check(c.Request.Context(), claims.MembershipID, claims.TokenVersion); err != nil {
			h.logAudit(c, db.AuditEvent{EventType: "refresh.failure", Outcome: "failure",
				ActorMembershipID: claims.MembershipID, SessionID: claims.SessionID,
				Details: map[string]any{"reason": "revoked"}})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session has been revoked. Please log in again."})
			return
		}
	}
```

Replace the reuse-detected branch (`auth.go:238-242`):

```go
			case reused:
				h.logAudit(c, db.AuditEvent{EventType: "refresh.reuse", Outcome: "failure",
					ActorMembershipID: claims.MembershipID, SessionID: sessionID})
				log.Printf("refresh-token reuse detected for %s (session %s) — session revoked", claims.MembershipID, sessionID)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Session ended for security reasons. Please log in again."})
				return
```

Replace the expired/unknown-session branch (`auth.go:247-249`):

```go
			case !rotated:
				h.logAudit(c, db.AuditEvent{EventType: "refresh.failure", Outcome: "failure",
					ActorMembershipID: claims.MembershipID, SessionID: sessionID,
					Details: map[string]any{"reason": "expired_session"}})
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Session has expired. Please log in again."})
				return
```

In `Logout`, add before the final `c.JSON` (`auth.go:318`):

```go
	h.logAudit(c, db.AuditEvent{EventType: "logout", ActorMembershipID: c.GetString("membership_id"), SessionID: sessionID})
```

In `LogoutAll`, add before the final `c.JSON` (`auth.go:346`):

```go
	h.logAudit(c, db.AuditEvent{EventType: "logout.all", ActorMembershipID: membershipID})
```

- [ ] **Step 5: Update the `main.go` call site (true-nil-safe)**

In `main.go`, just before constructing `authHandler` (currently `main.go:216`), introduce a true-nil audit logger and pass it:

```go
	// Pass audit as a true-nil interface in degraded mode so handlers' nil-guards
	// engage (a typed-nil *db.AuditStore would make `!= nil` true and panic).
	var auditLogger handlers.AuditLogger
	if stores.Audit != nil {
		auditLogger = stores.Audit
	}
	authHandler := handlers.NewAuthHandler(jwtHelper, tokenStore, cfg, stores.Users, appCache, revoker, auditLogger)
```

- [ ] **Step 6: Run tests + build + format**

Run: `cd backend/api-service && gofmt -l . && go build ./... && go test ./api/handlers/... -run 'Refresh|Audit' -v`
Expected: `TestRefreshToken_InvalidToken_AuditsFailure` + `TestAudit_BestEffort_DoesNotBlockResponse` PASS; build green; no gofmt output.

- [ ] **Step 7: Commit**

```bash
git add backend/api-service/api/handlers/auth.go backend/api-service/api/handlers/auth_audit_test.go backend/api-service/main.go
git commit -m "feat(audit): instrument login/logout/refresh events"
```

---

### Task 7: Instrument self opt-in role change (`role.optin`)

**Files:**
- Modify: `api/handlers/account.go` (constructor + `SetRole` instrumentation)
- Modify: `main.go` (update both `NewAccountHandler` call sites)
- Test: `api/handlers/account_audit_test.go` (new)

**Interfaces:**
- Consumes: `AuditLogger` (Task 6), `db.AuditEvent`.
- Produces: `NewAccountHandler(users roleSelfStore, flags flagLister, c cache.Cache, audit AuditLogger)`.

- [ ] **Step 1: Write the failing test**

Create `api/handlers/account_audit_test.go`:

```go
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeRoleStore struct{ err error }

func (f *fakeRoleStore) SetRole(_ context.Context, _ string, _ int16) error { return f.err }

func TestAccountSetRole_AuditsOptIn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spy := &spyAudit{}
	h := NewAccountHandler(&fakeRoleStore{}, nil, nil, spy)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("membership_id", "mid-1")
		c.Set("role", 0) // standard (not admin) so opt-in is allowed
		c.Next()
	})
	r.PUT("/api/account/role", h.SetRole)

	req := httptest.NewRequest(http.MethodPut, "/api/account/role",
		strings.NewReader(`{"role":"beta"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if got := spy.types(); len(got) != 1 || got[0] != "role.optin" {
		t.Errorf("audit events = %v, want [role.optin]", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend/api-service && go build ./api/handlers/...`
Expected: FAIL — `NewAccountHandler` wants 3 args, test passes 4.

- [ ] **Step 3: Add audit to `AccountHandler`**

In `api/handlers/account.go`, add the field + constructor param + import. Add `audit AuditLogger` to the struct (after `cache`); add `"guardian-tracker/api-service/db"` to imports (already imported — confirm). Change the constructor:

```go
func NewAccountHandler(users roleSelfStore, flags flagLister, c cache.Cache, audit AuditLogger) *AccountHandler {
	return &AccountHandler{users: users, flags: flags, cache: c, audit: audit}
}
```

In `SetRole`, after the successful `h.users.SetRole(...)` (after the cache eviction, before the final `c.JSON`, ~`account.go:81`), add:

```go
	if h.audit != nil {
		oldRole := c.GetInt("role")
		_ = h.audit.Log(c.Request.Context(), db.AuditEvent{
			EventType:         "role.optin",
			ActorMembershipID: membershipID,
			IP:                c.ClientIP(),
			UserAgent:         c.Request.UserAgent(),
			Details:           map[string]any{"oldRole": oldRole, "newRole": int(role)},
		})
	}
```

- [ ] **Step 4: Update both `main.go` call sites**

In `main.go`, the account handler is built in both branches (`main.go:227` and `main.go:230`). Update them to pass `auditLogger` (defined in Task 6, which is in scope here):

```go
	if stores.Users != nil {
		accountHandler = handlers.NewAccountHandler(stores.Users, stores.Flags, appCache, auditLogger)
		adminHandler = handlers.NewAdminHandler(stores.Users, stores.Flags, appCache)
	} else {
		accountHandler = handlers.NewAccountHandler(nil, nil, appCache, auditLogger)
		adminHandler = handlers.NewAdminHandler(nil, nil, appCache)
	}
```

> Ensure the `var auditLogger handlers.AuditLogger` block from Task 6 appears **before** this account/admin construction block. If the account/admin block currently precedes the `authHandler` construction in `main.go`, move the `auditLogger` declaration up so it is in scope for both. (In the current file, account/admin handlers are built at lines 224-232, before `authHandler` at 216 — verify ordering and place the `auditLogger` declaration above line 224.)

- [ ] **Step 5: Run tests + build + format**

Run: `cd backend/api-service && gofmt -l . && go build ./... && go test ./api/handlers/... -run 'AccountSetRole' -v`
Expected: `TestAccountSetRole_AuditsOptIn` PASS; build green.

- [ ] **Step 6: Commit**

```bash
git add backend/api-service/api/handlers/account.go backend/api-service/api/handlers/account_audit_test.go backend/api-service/main.go
git commit -m "feat(audit): instrument self opt-in role change"
```

---

### Task 8: Admin read endpoint `GET /api/admin/audit`

**Files:**
- Create: `api/handlers/audit.go`
- Test: `api/handlers/audit_test.go`

**Interfaces:**
- Consumes: `db.AuditEntry`, `db.AuditFilter`.
- Produces:
  - `type auditReadStore interface { List(ctx context.Context, f db.AuditFilter) ([]db.AuditEntry, string, error) }`
  - `type AuditHandler struct{ ... }`, `func NewAuditHandler(store auditReadStore) *AuditHandler`, method `func (h *AuditHandler) ListAudit(c *gin.Context)`.

- [ ] **Step 1: Write the failing test**

Create `api/handlers/audit_test.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
)

type mockAuditRead struct {
	gotFilter db.AuditFilter
	entries   []db.AuditEntry
	cursor    string
}

func (m *mockAuditRead) List(_ context.Context, f db.AuditFilter) ([]db.AuditEntry, string, error) {
	m.gotFilter = f
	return m.entries, m.cursor, nil
}

func TestListAudit_ReturnsEntriesAndPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &mockAuditRead{
		entries: []db.AuditEntry{{
			ID: 7, EventType: "role.change.admin", Outcome: "success",
			ActorMembershipID: "a", ActorDisplayName: "Admin",
			TargetMembershipID: "b", TargetDisplayName: "Guardian",
			Details: map[string]any{"oldRole": 0, "newRole": 1}, CreatedAt: time.Now(),
		}},
		cursor: "next-cursor",
	}
	h := NewAuditHandler(store)
	r := gin.New()
	r.GET("/api/admin/audit", h.ListAudit)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/audit?type=role.&outcome=success&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Entries []struct {
			ID        string `json:"id"`
			EventType string `json:"eventType"`
		} `json:"entries"`
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 1 || resp.Entries[0].ID != "7" || resp.Entries[0].EventType != "role.change.admin" {
		t.Errorf("entries = %+v, want one role.change.admin id=7", resp.Entries)
	}
	if resp.NextCursor != "next-cursor" {
		t.Errorf("nextCursor = %q, want next-cursor", resp.NextCursor)
	}
	// Query params propagated to the filter.
	if store.gotFilter.EventType != "role." || store.gotFilter.Outcome != "success" || store.gotFilter.Limit != 10 {
		t.Errorf("filter = %+v, want type=role. outcome=success limit=10", store.gotFilter)
	}
}

func TestListAudit_DegradedMode503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuditHandler(nil) // no store -> degraded
	r := gin.New()
	r.GET("/api/admin/audit", h.ListAudit)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/audit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend/api-service && go build ./api/handlers/...`
Expected: FAIL — `NewAuditHandler`, `AuditHandler` undefined.

- [ ] **Step 3: Implement the handler**

Create `api/handlers/audit.go`:

```go
package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
)

// auditReadStore is the read slice of db.AuditStore.
type auditReadStore interface {
	List(ctx context.Context, f db.AuditFilter) ([]db.AuditEntry, string, error)
}

// AuditHandler serves the admin audit feed. Mounted behind RequireAdmin.
type AuditHandler struct {
	store auditReadStore // nil in degraded mode
}

func NewAuditHandler(store auditReadStore) *AuditHandler { return &AuditHandler{store: store} }

type auditPartyResponse struct {
	MembershipID string `json:"membershipId"`
	DisplayName  string `json:"displayName"`
}

type auditEntryResponse struct {
	ID        string             `json:"id"`
	EventType string             `json:"eventType"`
	Outcome   string             `json:"outcome"`
	Actor     auditPartyResponse `json:"actor"`
	Target    *auditPartyResponse `json:"target,omitempty"`
	IP        string             `json:"ip,omitempty"`
	UserAgent string             `json:"userAgent,omitempty"`
	Details   map[string]any     `json:"details"`
	CreatedAt string             `json:"createdAt"`
}

// ListAudit handles GET /api/admin/audit (admin-gated upstream).
func (h *AuditHandler) ListAudit(c *gin.Context) {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Audit log requires the database, which is not configured.", "code": "DB_UNAVAILABLE"})
		return
	}

	f := db.AuditFilter{
		EventType: c.Query("type"),
		Actor:     c.Query("actor"),
		Target:    c.Query("target"),
		Outcome:   c.Query("outcome"),
		Cursor:    c.Query("cursor"),
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Limit = n
		}
	}
	if v := c.Query("after"); v != "" {
		if ts, err := time.Parse(time.RFC3339, v); err == nil {
			f.After = ts
		}
	}
	if v := c.Query("before"); v != "" {
		if ts, err := time.Parse(time.RFC3339, v); err == nil {
			f.Before = ts
		}
	}

	entries, nextCursor, err := h.store.List(c.Request.Context(), f)
	if err != nil {
		log.Printf("admin ListAudit: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	out := make([]auditEntryResponse, len(entries))
	for i, e := range entries {
		resp := auditEntryResponse{
			ID:        strconv.FormatInt(e.ID, 10),
			EventType: e.EventType,
			Outcome:   e.Outcome,
			Actor:     auditPartyResponse{MembershipID: e.ActorMembershipID, DisplayName: e.ActorDisplayName},
			IP:        e.IP,
			UserAgent: e.UserAgent,
			Details:   e.Details,
			CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
		}
		if resp.Details == nil {
			resp.Details = map[string]any{}
		}
		if e.TargetMembershipID != "" {
			resp.Target = &auditPartyResponse{MembershipID: e.TargetMembershipID, DisplayName: e.TargetDisplayName}
		}
		out[i] = resp
	}
	c.JSON(http.StatusOK, gin.H{"entries": out, "nextCursor": nextCursor})
}
```

- [ ] **Step 4: Run tests + build + format**

Run: `cd backend/api-service && gofmt -l . && go test ./api/handlers/... -run 'ListAudit' -v`
Expected: both PASS; no gofmt output.

- [ ] **Step 5: Commit**

```bash
git add backend/api-service/api/handlers/audit.go backend/api-service/api/handlers/audit_test.go
git commit -m "feat(audit): admin GET /api/admin/audit read endpoint"
```

---

### Task 9: Final `main.go` wiring — trusted proxies, pruner, route

**Files:**
- Modify: `main.go` (SetTrustedProxies, startAuditPruner, construct `AuditHandler`, register route)

**Interfaces:**
- Consumes: `cfg.TrustedProxies`, `cfg.AuditRetentionDays`, `stores.Audit`, `handlers.NewAuditHandler`.

- [ ] **Step 1: Configure trusted proxies on the router**

In `main.go`, immediately after `router := gin.Default()` (`main.go:235`) and before `router.Use(corsMiddleware(...))`, add:

```go
	// Trust only configured proxies for X-Forwarded-For so the audit-logged client
	// IP can't be spoofed by clients. Empty list (local dev) trusts none.
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Printf("Warning: SetTrustedProxies(%v): %v", cfg.TrustedProxies, err)
	}
```

- [ ] **Step 2: Start the audit pruner (DB mode only)**

In `main.go`, in the `if stores.Users != nil { ... }` block where `startSessionPruner` is called (`main.go:186-189`), add the audit pruner alongside it:

```go
	if stores.Users != nil {
		revoker = auth.NewRevocationChecker(stores.Users, appCache)
		startSessionPruner(ctx, stores.Users)
		if stores.Audit != nil {
			startAuditPruner(ctx, stores.Audit, cfg.AuditRetentionDays)
		}
	}
```

Add the pruner function near `startSessionPruner` (after it, ~`main.go:350`):

```go
// startAuditPruner periodically deletes audit_log rows older than retentionDays,
// bounding table growth and the retention window for stored client IPs. Runs until
// ctx is cancelled at shutdown.
func startAuditPruner(ctx context.Context, audit *db.AuditStore, retentionDays int) {
	const interval = time.Hour
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().AddDate(0, 0, -retentionDays)
				n, err := audit.DeleteOlderThan(ctx, cutoff)
				if err != nil {
					log.Printf("audit pruner: %v", err)
				} else if n > 0 {
					log.Printf("audit pruner: removed %d expired audit row(s)", n)
				}
			}
		}
	}()
}
```

- [ ] **Step 3: Construct the audit handler and register the route**

In `main.go`, after the `adminHandler` is constructed (~`main.go:232`), add:

```go
	var auditHandler *handlers.AuditHandler
	if stores.Audit != nil {
		auditHandler = handlers.NewAuditHandler(stores.Audit)
	} else {
		auditHandler = handlers.NewAuditHandler(nil)
	}
```

In the admin route group (`main.go:269-275`), add the audit route:

```go
		admin := api.Group("/admin", jwtHelper.Middleware(revoker), authz.RequireAdmin())
		{
			admin.GET("/users", adminHandler.ListUsers)
			admin.PUT("/users/:id/role", adminHandler.SetUserRole)
			admin.GET("/flags", adminHandler.ListFlags)
			admin.PUT("/flags/:key", adminHandler.UpdateFlag)
			admin.GET("/audit", auditHandler.ListAudit)
		}
```

- [ ] **Step 4: Build + vet + full test**

Run: `cd backend/api-service && gofmt -l . && go vet ./... && go build ./...`
Expected: no gofmt output, vet clean, build succeeds.

- [ ] **Step 5: Manual smoke (optional but recommended)**

Run: `cd backend/api-service && ./test-local.ps1 -NoRace`
Expected: all Go tests pass; coverage printed (≥ 60%).

- [ ] **Step 6: Commit**

```bash
git add backend/api-service/main.go
git commit -m "feat(audit): wire trusted proxies, pruner, and admin audit route"
```

---

### Task 10: Frontend — Audit panel in `/admin`

**Files:**
- Modify: `frontend/src/types/api.ts` (add audit types)
- Modify: `frontend/src/pages/Admin.tsx` (add audit tab + panel)
- Create: `frontend/src/components/kit/AuditTable.tsx`
- Modify: `frontend/src/components/kit/index.ts` (export `AuditTable`)

**Interfaces:**
- Consumes: `GET /api/admin/audit` (Task 8) response `{ entries: APIAuditEntry[]; nextCursor: string }`.
- Produces: `APIAuditEntry`, `APIAuditPage` types; `<AuditTable>` component.

- [ ] **Step 1: Add the API types**

In `frontend/src/types/api.ts`, append:

```ts
export interface APIAuditParty {
  membershipId: string;
  displayName: string;
}

export interface APIAuditEntry {
  id: string;
  eventType: string;
  outcome: "success" | "failure";
  actor: APIAuditParty;
  target?: APIAuditParty;
  ip?: string;
  userAgent?: string;
  details: Record<string, unknown>;
  createdAt: string;
}

export interface APIAuditPage {
  entries: APIAuditEntry[];
  nextCursor: string;
}
```

- [ ] **Step 2: Create the `AuditTable` component**

Create `frontend/src/components/kit/AuditTable.tsx`:

```tsx
import React from "react";
import { Badge } from "./index";
import { relTime } from "../../lib/adapters";
import type { APIAuditEntry } from "../../types/api";

const EVENT_LABEL: Record<string, string> = {
  "login.success": "Login",
  "login.failure": "Login failed",
  logout: "Logout",
  "logout.all": "Logout (all devices)",
  "refresh.reuse": "Token reuse",
  "refresh.failure": "Refresh failed",
  "role.change.admin": "Role change (admin)",
  "role.optin": "Role opt-in",
  "flag.update": "Flag update",
};

function label(type: string): string {
  return EVENT_LABEL[type] ?? type;
}

export function AuditTable({
  entries,
  loading,
}: {
  entries: APIAuditEntry[];
  loading: boolean;
}) {
  if (loading && entries.length === 0) {
    return <div className="gt-audit-empty mono">Loading audit events…</div>;
  }
  if (entries.length === 0) {
    return <div className="gt-audit-empty mono">No audit events match.</div>;
  }
  return (
    <div className="gt-card gt-pad0">
      <div className="gt-audit-row gt-audit-row--head">
        <span className="gt-userrow-h">Event</span>
        <span className="gt-userrow-h">Actor</span>
        <span className="gt-userrow-h">Details</span>
        <span className="gt-userrow-h" style={{ textAlign: "right" }}>
          When
        </span>
      </div>
      {entries.map((e) => (
        <div className="gt-audit-row" key={e.id}>
          <span>
            <Badge kind={e.outcome === "failure" ? "warn" : "ok"}>
              {label(e.eventType)}
            </Badge>
          </span>
          <span className="gt-audit-actor">
            {e.actor.displayName || e.actor.membershipId || "—"}
            {e.target ? (
              <span className="gt-audit-target mono">
                {" → "}
                {e.target.displayName || e.target.membershipId}
              </span>
            ) : null}
          </span>
          <span className="gt-audit-details mono">
            {Object.entries(e.details).map(([k, v]) => (
              <span className="gt-audit-pill" key={k}>
                {k}: {JSON.stringify(v)}
              </span>
            ))}
            {e.ip ? <span className="gt-audit-pill">ip: {e.ip}</span> : null}
          </span>
          <span className="gt-audit-when mono" style={{ textAlign: "right" }}>
            {relTime(e.createdAt)}
          </span>
        </div>
      ))}
    </div>
  );
}
```

> Confirm the `Badge` component accepts a `kind` prop with values including `"ok"` and `"warn"`. If the kit's `Badge` uses different kind names, use the existing ones (check `frontend/src/components/kit/`); the only requirement is a visually distinct failure style.

- [ ] **Step 3: Export `AuditTable` from the kit**

In `frontend/src/components/kit/index.ts`, add an export line alongside the other kit exports:

```ts
export { AuditTable } from "./AuditTable";
```

- [ ] **Step 4: Add the Audit tab + panel to `Admin.tsx`**

In `frontend/src/pages/Admin.tsx`:

1. Extend the `Tab` type (`Admin.tsx:19`):

```ts
type Tab = "users" | "flags" | "audit";
```

2. Add imports — extend the kit import to include `AuditTable`, and the types import to include `APIAuditPage`:

```ts
import {
  AuditTable,
  EmptyState,
  FilterChip,
  FlagCard,
  Icon,
  PageHead,
  StatTile,
  UserRow,
} from "../components/kit";
import type { APIAdminFlag, APIAdminUser, APIAuditPage } from "../types/api";
```

3. Add audit state + query inside the `Admin` component (after the `flagsQuery` definition, ~`Admin.tsx:36`):

```ts
  const [auditType, setAuditType] = useState<string>("");
  const auditQuery = useQuery({
    queryKey: ["admin", "audit", auditType],
    queryFn: () =>
      apiFetch<APIAuditPage>(
        `/api/admin/audit?limit=100${auditType ? `&type=${encodeURIComponent(auditType)}` : ""}`,
      ),
    enabled: tab === "audit",
  });
```

4. Add an Audit subtab button in the `right` prop of `<PageHead>` (after the Feature Flags button, ~`Admin.tsx:131`):

```tsx
            <button
              className="gt-subtab"
              data-on={tab === "audit"}
              onClick={() => setTab("audit")}
            >
              <Icon name="shield" size="0.95rem" /> Audit Log
            </button>
```

5. Add the audit panel. Change the final ternary so the audit tab renders its panel. Replace the closing of the flags branch — locate the outer `{tab === "users" ? (...) : (...)}` (`Admin.tsx:137` opens it, `Admin.tsx:280` closes the flags branch) and convert it to handle three tabs by wrapping the flags branch:

Replace `) : (` at the start of the flags branch (`Admin.tsx:230`) with:

```tsx
      ) : tab === "flags" ? (
```

Then, immediately before the final `</div>` that closes `<div className="gt-page">` (`Admin.tsx:281`), insert the audit branch by closing the flags branch with the audit alternative. Concretely, change the flags-branch closer (`Admin.tsx:279-280`, the `</>` then `)}`) to:

```tsx
        </>
      ) : (
        <>
          <div className="gt-coll-toolbar">
            <div className="gt-filterbar">
              {[
                ["", "All"],
                ["login.", "Logins"],
                ["logout", "Logouts"],
                ["refresh.", "Sessions"],
                ["role.", "Roles"],
                ["flag.update", "Flags"],
              ].map(([val, lbl]) => (
                <FilterChip
                  key={val || "all"}
                  on={auditType === val}
                  onClick={() => setAuditType(val)}
                >
                  {lbl}
                </FilterChip>
              ))}
            </div>
          </div>
          <AuditTable
            entries={auditQuery.data?.entries ?? []}
            loading={auditQuery.isLoading}
          />
        </>
      )}
```

> This is a focused JSX edit; if the surrounding structure differs slightly at apply time, the rule is: `users` tab keeps its block, `flags` tab keeps its block guarded by `tab === "flags"`, and a third `: ( <audit panel> )` branch handles `audit`. The audit panel renders a `FilterChip` row bound to `setAuditType` and an `<AuditTable>` fed by `auditQuery`.

- [ ] **Step 5: Add minimal styles**

In `frontend/src/styles/admin.css`, append:

```css
.gt-audit-row {
  display: grid;
  grid-template-columns: 13rem 1fr 2fr 7rem;
  gap: var(--s-3);
  align-items: center;
  padding: var(--s-3) var(--s-4);
  border-top: 1px solid var(--c-border);
}
.gt-audit-row--head {
  border-top: none;
  color: var(--c-text-3);
}
.gt-audit-details {
  display: flex;
  flex-wrap: wrap;
  gap: var(--s-1);
}
.gt-audit-pill {
  background: var(--c-surface-2);
  border-radius: var(--r-1);
  padding: 0 var(--s-1);
  font-size: 0.78rem;
}
.gt-audit-target {
  opacity: 0.7;
}
.gt-audit-empty {
  padding: var(--s-6);
  text-align: center;
  color: var(--c-text-3);
}
```

> Token names (`--s-3`, `--c-border`, `--r-1`, `--c-surface-2`, etc.) follow the existing design tokens. If a referenced token does not exist, substitute the nearest existing one from `frontend/src/styles/tokens.css`.

- [ ] **Step 6: Type-check + lint + build**

Run: `cd frontend && npm run type-check && npm run lint && npm run build`
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/types/api.ts frontend/src/pages/Admin.tsx frontend/src/components/kit/AuditTable.tsx frontend/src/components/kit/index.ts frontend/src/styles/admin.css
git commit -m "feat(audit): admin Audit Log panel"
```

---

### Task 11: Frontend — MSW test for the Audit panel

**Files:**
- Create: `frontend/src/__tests__/Admin.audit.test.tsx`

**Interfaces:**
- Consumes: the `<Admin>` page + MSW mock of `GET /api/admin/audit`.

- [ ] **Step 1: Write the test**

Create `frontend/src/__tests__/Admin.audit.test.tsx`. Mirror the existing MSW page-test setup (check another test in `src/__tests__/` for the exact `server`, `QueryClientProvider`, and provider-wrapper imports used in this repo, and copy that harness):

```tsx
import { describe, it, expect, beforeAll, afterEach, afterAll } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { QueryClientProvider, QueryClient } from "@tanstack/react-query";
import { Admin } from "../pages/Admin";
// NOTE: match the provider wrappers (Auth/Toast/etc.) used by sibling tests in
// src/__tests__/ — import and wrap with the same ones here.

const server = setupServer(
  http.get("*/api/admin/users", () => HttpResponse.json([])),
  http.get("*/api/admin/flags", () => HttpResponse.json([])),
  http.get("*/api/admin/audit", ({ request }) => {
    const url = new URL(request.url);
    const type = url.searchParams.get("type") ?? "";
    return HttpResponse.json({
      entries: [
        {
          id: "1",
          eventType: type === "flag.update" ? "flag.update" : "login.success",
          outcome: "success",
          actor: { membershipId: "mid-1", displayName: "Tester" },
          details: {},
          createdAt: new Date().toISOString(),
          ip: "203.0.113.7",
        },
      ],
      nextCursor: "",
    });
  }),
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function renderAdmin() {
  const qc = new QueryClient();
  // localStorage user must look like an admin for the page to mount in app flow;
  // Admin renders its panels directly here, so a bare render suffices for the panel test.
  return render(
    <QueryClientProvider client={qc}>
      <Admin />
    </QueryClientProvider>,
  );
}

describe("Admin Audit panel", () => {
  it("renders audit events after switching to the Audit Log tab", async () => {
    renderAdmin();
    fireEvent.click(screen.getByText(/Audit Log/i));
    await waitFor(() =>
      expect(screen.getByText(/Login/i)).toBeInTheDocument(),
    );
    expect(screen.getByText(/203\.0\.113\.7/)).toBeInTheDocument();
  });

  it("refetches when the Flags filter chip is clicked", async () => {
    renderAdmin();
    fireEvent.click(screen.getByText(/Audit Log/i));
    await screen.findByText(/Login/i);
    fireEvent.click(screen.getByText(/^Flags$/));
    await waitFor(() =>
      expect(screen.getByText(/Flag update/i)).toBeInTheDocument(),
    );
  });
});
```

> If `Admin` requires `AuthContext`/`ToastContext`/`PreferencesContext` to render (it calls `useAuth` and `useToast`), wrap it with the same provider stack the sibling page tests use. Copy that wrapper verbatim from an existing `src/__tests__/*.test.tsx` rather than guessing.

- [ ] **Step 2: Run the test**

Run: `cd frontend && npm test -- Admin.audit`
Expected: both cases PASS.

- [ ] **Step 3: Run the full suite + coverage gate**

Run: `cd frontend && npm test -- --coverage`
Expected: green; coverage thresholds (lines ≥ 70%, branches ≥ 65%) hold.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/__tests__/Admin.audit.test.tsx
git commit -m "test(audit): MSW test for admin Audit Log panel"
```

---

### Task 12: Documentation + env examples

**Files:**
- Modify: `CLAUDE.md`, `SECURITY.md`, `private/security-limitations.md`, `private/TODO.md`
- Modify: `.env.example`, `backend/api-service/.env.example`

- [ ] **Step 1: Update `CLAUDE.md`**

- In the API Service key-files table, add a row: `db/audit.go` — "Unified append-only audit trail store (`audit_log`): best-effort `Log` + in-tx `insertAudit`, filtered/keyset `List`, retention prune".
- In the migrations row, add `0004_audit_log.sql` (audit_log; role_audit migrated in then dropped).
- In the endpoints list, add: `GET /api/admin/audit` — admin only: filtered, keyset-paginated audit feed (`type`, `actor`, `target`, `outcome`, `before`/`after`, `cursor`, `limit`).
- In the Authentication Security / roles section, note that authentication, session, role, and feature-flag events are now persisted to `audit_log` (auth/session writes best-effort; role/flag writes in-transaction).
- Add env vars `AUDIT_RETENTION_DAYS` (default 180) and `TRUSTED_PROXIES` to the relevant config notes.

- [ ] **Step 2: Update `SECURITY.md`**

- State that authentication (login/logout/refresh), self opt-in role changes, admin role changes, and feature-flag changes are recorded in `audit_log`.
- **Disclose** that the audit log stores client **IP address and User-Agent** for security forensics, retained for `AUDIT_RETENTION_DAYS` (default 180) and pruned hourly. Note IP is trusted only from configured `TRUSTED_PROXIES` (gin `SetTrustedProxies`) so it cannot be spoofed.
- Note the residual tradeoff: auth/session audit writes are best-effort, so a DB outage can drop an auth-event row (role/flag writes are transactional and cannot).

- [ ] **Step 3: Resolve the known-limitation**

In `private/security-limitations.md`, rewrite item #4 to strike through the original and mark resolved, e.g.:

```markdown
4. ~~**Partial audit logging**~~ — **RESOLVED.** Authentication (login success/failure, logout, logout-all), session security (refresh reuse/failure), self opt-in and admin role changes, and feature-flag changes are persisted to the unified append-only `audit_log` and exposed to admins via `GET /api/admin/audit` + the `/admin` Audit Log panel. Role and flag changes are written in the mutation's transaction; auth/session events are best-effort (a DB outage can drop one). Client IP + User-Agent are retained for `AUDIT_RETENTION_DAYS` (default 180), hourly-pruned.
```

- [ ] **Step 4: Update `private/TODO.md`**

Add an entry under Completed Work summarizing the audit-logging slice (table `0004`, `db/audit.go`, endpoint, panel, retention pruner, IP capture behind trusted proxies, docs), and note it closed `security-limitations.md` #4.

- [ ] **Step 5: Update `.env.example` files**

In both `.env.example` (root) and `backend/api-service/.env.example`, add:

```bash
# Audit log retention in days (rows + stored client IPs pruned hourly past this)
AUDIT_RETENTION_DAYS=180
# Comma-separated CIDRs/IPs trusted for X-Forwarded-For client-IP resolution
# (set to the platform ingress range in production; empty trusts none)
TRUSTED_PROXIES=
```

- [ ] **Step 6: Commit**

```bash
git add CLAUDE.md SECURITY.md private/security-limitations.md private/TODO.md .env.example backend/api-service/.env.example
git commit -m "docs(audit): document audit log, IP retention, and resolve limitation #4"
```

---

## Self-Review

**Spec coverage:**

- §2 scope (4 event groups, no password events, no refresh.success) → Tasks 1,4,6,7 (catalog matches; refresh.success absent). ✓
- §3 A1 read endpoint + UI → Tasks 8,10. ✓
- A2 unify + drop role_audit → Task 1. ✓
- A3 JSONB details → Task 1 schema + `insertAudit`. ✓
- A4 best-effort vs in-tx → Task 2 (`Log`) / Task 1,4 (`insertAudit` in-tx); best-effort proven in Task 6 test. ✓
- A5 IP + UA → Task 1 columns, Task 6 capture; A6 trusted proxies → Tasks 5,9. ✓
- A7 retention pruner → Tasks 2,5,9. ✓
- A8 invalid_state logged → Task 6 (`reason: "invalid_state"`). ✓
- §4 schema incl. `ON DELETE SET NULL` + migrate-then-drop → Task 1. ✓
- §5 write path (`AuditEvent`, `execer`, `insertAudit`, `Log`, `List`, `DeleteOlderThan`) → Tasks 1–3. ✓
- §6 event catalog (9 events) → Tasks 1 (role.change.admin), 4 (flag.update), 6 (5 auth/session), 7 (role.optin). ✓
- §8 endpoint with filters + keyset paging + 503 degraded → Tasks 3,8. ✓
- §9 frontend panel → Tasks 10,11. ✓
- §10 retention → Task 9. §11 tests → every backend task + 11. §12 docs → Task 12. §13 degraded mode → nil-guards in Tasks 6–9. ✓

**Placeholder scan:** No "TBD/TODO/handle edge cases"; each code step shows full code. Frontend Tasks 10–11 carry explicit "verify against existing kit/provider patterns" notes (real ambiguity in an unseen file), not placeholders for logic. ✓

**Type consistency:** `AuditEvent`, `execer`, `insertAudit` defined in Task 1 and reused verbatim in Tasks 2–7. `AuditStore.List` signature `([]AuditEntry, string, error)` (Task 3) matches `auditReadStore.List` (Task 8) and the handler call. `AuditLogger.Log(ctx, db.AuditEvent) error` (Task 6) matches `spyAudit`, `AuditStore.Log` (Task 2), and the `NewAccountHandler` param (Task 7). `FlagStore.Update` new 6-arg signature (Task 4) matches the `adminFlagStore` interface change and the `UpdateFlag` call site. Frontend `APIAuditPage`/`APIAuditEntry` (Task 10) match the endpoint JSON shaped in Task 8 (`entries`, `nextCursor`, `actor`, `target?`, `details`, `createdAt`). ✓

No issues found requiring new tasks.
