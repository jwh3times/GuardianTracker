package db

import (
	"context"
	"testing"
	"time"
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

func TestAuditStore_LogAndPrune(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewAuditStore(pool)
	mid, _ := createTestUser(t, pool)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM audit_log WHERE actor_membership_id = $1`, mid) })

	if err := store.Log(ctx, AuditEvent{EventType: "logout.session", ActorMembershipID: mid}); err != nil {
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

func TestInsertAudit_EmptyIPStoresNull(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	mid, uid := createTestUser(t, pool)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM audit_log WHERE actor_membership_id = $1`, mid) })

	if err := insertAudit(ctx, pool, AuditEvent{EventType: "logout.session", ActorMembershipID: mid}); err != nil {
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
	_ = uid // uid used by createTestUser cleanup; silence unused warning
}

func TestFlagStore_UpdateWritesAudit(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	flags := NewFlagStore(pool)
	_, adminUID := createTestUser(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM audit_log WHERE event_type = 'flag.update' AND actor_user_id = $1`, adminUID)
	})

	// Snapshot the flag so we can restore it after the test.
	orig, err := flags.Get(ctx, "god-roll")
	if err != nil {
		t.Fatalf("Get god-roll: %v", err)
	}
	// Restore via direct SQL so no second audit row is written.
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx,
			`UPDATE feature_flags SET enabled = $1, min_tier = $2 WHERE key = 'god-roll'`,
			orig.Enabled, orig.MinTier)
	})
	// Widen audit cleanup to catch any orphan rows (e.g. NULL-actor rows from a
	// restore that accidentally went through FlagStore.Update).
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx,
			`DELETE FROM audit_log WHERE event_type = 'flag.update' AND details->>'key' = 'god-roll'`)
	})

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
