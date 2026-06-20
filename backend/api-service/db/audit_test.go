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
	_ = uid // uid used by createTestUser cleanup; silence unused warning
}
