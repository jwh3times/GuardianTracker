package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUserStore_RoleCheckConstraint(t *testing.T) {
	pool := testPool(t)
	mid, _ := createTestUser(t, pool)
	// role=4 violates CHECK (role BETWEEN 0 AND 3).
	_, err := pool.Exec(context.Background(), `UPDATE users SET role = 4 WHERE membership_id = $1`, mid)
	if err == nil {
		t.Fatal("expected CHECK violation setting role=4")
	}
}

func TestUserStore_SetSelfRoleAndGetAuthInfo(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()
	mid, _ := createTestUser(t, pool)

	if err := users.SetSelfRole(ctx, mid, 2, "", ""); err != nil { // alpha
		t.Fatalf("SetRole: %v", err)
	}
	role, err := users.GetRole(ctx, mid)
	if err != nil || role != 2 {
		t.Fatalf("GetRole = %d (%v), want 2", role, err)
	}
	tv, r, found, err := users.GetAuthInfo(ctx, mid)
	if err != nil {
		t.Fatalf("GetAuthInfo: %v", err)
	}
	if !found {
		t.Fatal("GetAuthInfo reported existing user as missing")
	}
	if r != 2 || tv != 1 {
		t.Errorf("GetAuthInfo = (tv %d, role %d), want (1, 2)", tv, r)
	}
}

func TestUserStore_GetAuthInfoMissingIsDefinitive(t *testing.T) {
	users := NewUserStore(testPool(t))
	_, _, found, err := users.GetAuthInfo(context.Background(), "no-such-membership")
	if err != nil {
		t.Fatalf("GetAuthInfo: %v", err)
	}
	if found {
		t.Fatal("GetAuthInfo found an unknown user")
	}
}

func TestUserStore_SetSelfRoleUnknownUser(t *testing.T) {
	pool := testPool(t)
	if err := NewUserStore(pool).SetSelfRole(context.Background(), "no-such-member", 1, "", ""); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("SetRole unknown user: got %v, want ErrUserNotFound", err)
	}
}

func TestUserStore_UpsertForceAdmin(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()
	mid, _ := createTestUser(t, pool) // starts standard

	// Re-login as a bootstrap admin pins the role to admin.
	_, _, role, err := users.Upsert(ctx, mid, 3, "Test Guardian", true)
	if err != nil {
		t.Fatalf("Upsert forceAdmin: %v", err)
	}
	if role != roleAdmin {
		t.Errorf("forceAdmin role = %d, want %d", role, roleAdmin)
	}
	// A subsequent normal login keeps admin (does not reset it).
	_, _, role, err = users.Upsert(ctx, mid, 3, "Test Guardian", false)
	if err != nil {
		t.Fatalf("Upsert normal: %v", err)
	}
	if role != roleAdmin {
		t.Errorf("normal login after admin: role = %d, want admin preserved", role)
	}
}

func TestUserStore_CountAdminsAndListUsers(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()

	before, err := users.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins: %v", err)
	}
	mid, _ := createTestUser(t, pool)
	if _, _, _, err := users.Upsert(ctx, mid, 3, "Test Guardian", true); err != nil {
		t.Fatalf("SetRole admin: %v", err)
	}
	after, err := users.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins: %v", err)
	}
	if after != before+1 {
		t.Errorf("CountAdmins after promote = %d, want %d", after, before+1)
	}

	list, err := users.ListUsers(ctx, "Test Guardian", 200)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	found := false
	for _, u := range list {
		if u.MembershipID == mid {
			found = true
		}
	}
	if !found {
		t.Error("ListUsers did not return the promoted test user")
	}
}

func TestUserStore_SetRoleByID_BumpsVersionAndAudits(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()
	actorMid, _ := createTestUser(t, pool)
	if _, _, _, err := users.Upsert(ctx, actorMid, 3, "Test Guardian", true); err != nil {
		t.Fatalf("promote actor: %v", err)
	}
	targetMid, targetID := createTestUser(t, pool)

	change, err := users.SetRoleByID(ctx, actorMid, targetID, 2) // -> alpha
	if err != nil {
		t.Fatalf("SetRoleByID: %v", err)
	}
	if change.TargetMembershipID != targetMid || change.OldRole != 0 || change.NewRole != 2 {
		t.Errorf("RoleChange = %+v, want target=%s old=0 new=2", change, targetMid)
	}
	// token_version bumped from the default 1 to 2.
	tv, role, found, err := users.GetAuthInfo(ctx, targetMid)
	if err != nil {
		t.Fatalf("GetAuthInfo: %v", err)
	}
	if !found {
		t.Fatal("GetAuthInfo reported existing user as missing")
	}
	if tv != 2 || role != 2 {
		t.Errorf("after change: tv=%d role=%d, want tv=2 role=2", tv, role)
	}
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
}

func TestUserStore_SetRoleByID_LastAdminRefused(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()

	// Demote every existing admin down to a single one we control, so this test
	// is deterministic regardless of bootstrap admins already present.
	existing, err := users.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins: %v", err)
	}
	if existing > 0 {
		t.Skip("environment already has admins; last-admin path is covered by the unit test")
	}
	adminMid, adminID := createTestUser(t, pool)
	if _, _, _, err := users.Upsert(ctx, adminMid, 3, "Test Guardian", true); err != nil {
		t.Fatalf("promote: %v", err)
	}
	// Demoting the only admin must be refused.
	if _, err := users.SetRoleByID(ctx, adminMid, adminID, 0); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("demote last admin: got %v, want ErrLastAdmin", err)
	}
}

func TestUserStore_SetRoleByID_NotFound(t *testing.T) {
	pool := testPool(t)
	if _, err := NewUserStore(pool).SetRoleByID(context.Background(), "", 999999999, 1); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("SetRoleByID unknown: got %v, want ErrUserNotFound", err)
	}
}

func TestUserStore_SetSelfRoleAuditsAuthoritativeRole(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()
	mid, id := createTestUser(t, pool)
	if _, err := pool.Exec(ctx, `UPDATE users SET role = 1 WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if err := users.SetSelfRole(ctx, mid, 2, "127.0.0.1", "fixture-agent"); err != nil {
		t.Fatal(err)
	}
	var oldRole, newRole int
	var actorID int64
	var ip, userAgent string
	err := pool.QueryRow(ctx, `SELECT actor_user_id, (details->>'oldRole')::int, (details->>'newRole')::int, host(ip), user_agent FROM audit_log WHERE actor_membership_id = $1 AND event_type = 'role.optin'`, mid).Scan(&actorID, &oldRole, &newRole, &ip, &userAgent)
	if err != nil {
		t.Fatal(err)
	}
	if actorID != id || oldRole != 1 || newRole != 2 || ip != "127.0.0.1" || userAgent != "fixture-agent" {
		t.Fatalf("audit = actor:%d roles:%d->%d ip:%s ua:%s", actorID, oldRole, newRole, ip, userAgent)
	}
	version, role, _, err := users.GetAuthInfo(ctx, mid)
	if err != nil || version != 1 || role != 2 {
		t.Fatalf("post-opt-in auth info: version=%d role=%d err=%v", version, role, err)
	}
}

func TestUserStore_SetSelfRoleRollsBackWhenAuditFails(t *testing.T) {
	pool := testPool(t)
	mid, _ := createTestUser(t, pool)
	users := NewUserStore(pool)
	ctx := context.Background()
	// An invalid fixture IP makes PostgreSQL reject the audit inet value after
	// the role UPDATE. The mutation must roll back with that failed audit insert.
	if err := users.SetSelfRole(ctx, mid, 2, "not-an-ip", "fixture-agent"); err == nil {
		t.Fatal("expected audit insert failure")
	}
	version, role, _, err := users.GetAuthInfo(ctx, mid)
	if err != nil || role != 0 || version != 1 {
		t.Fatalf("audit failure changed auth info: role=%d version=%d err=%v", role, version, err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE actor_membership_id=$1 AND event_type='role.optin'`, mid).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("failed opt-in left %d audit rows", count)
	}
}

func TestUserStore_SetSelfRoleRejectsAdminTargets(t *testing.T) {
	pool := testPool(t)
	mid, _ := createTestUser(t, pool)
	for _, role := range []int16{-1, 3, 4} {
		if err := NewUserStore(pool).SetSelfRole(context.Background(), mid, role, "", ""); !errors.Is(err, ErrRoleNotAllowed) {
			t.Fatalf("target %d: got %v, want ErrRoleNotAllowed", role, err)
		}
	}
}

func TestUserStore_SetSelfRoleWaitsForPromotionAndPreservesAdmin(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mid, _ := createTestUser(t, pool)
	before, err := users.CountAdmins(ctx)
	if err != nil {
		t.Fatal(err)
	}
	promotion, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer promotion.Rollback(context.Background())
	// Model a bootstrap/admin promotion already holding the user row. The
	// in-flight self request began under an earlier non-admin middleware role.
	if _, err = promotion.Exec(ctx, `UPDATE users SET role = 3 WHERE membership_id = $1`, mid); err != nil {
		t.Fatal(err)
	}
	promoterPID := promotion.Conn().PgConn().PID()
	result := make(chan error, 1)
	go func() { result <- users.SetSelfRole(ctx, mid, 0, "", "") }()
	// Wait for PostgreSQL to confirm the actual lock interleaving; elapsed time
	// alone is not proof that the self mutation reached the contested row.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE $1::integer = ANY(pg_blocking_pids(pid)))`, promoterPID).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		select {
		case err := <-result:
			t.Fatalf("self mutation did not wait for promotion: %v", err)
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("self mutation never reached promotion lock")
		}
	}
	if err := promotion.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, ErrAdminOptIn) {
			t.Fatalf("got %v, want authoritative admin refusal", err)
		}
	case <-ctx.Done():
		t.Fatal("self mutation did not finish after promotion")
	}
	role, err := users.GetRole(ctx, mid)
	if err != nil || role != roleAdmin {
		t.Fatalf("promoted admin was demoted: role=%d err=%v", role, err)
	}
	after, err := users.CountAdmins(ctx)
	if err != nil || after != before+1 || after < 1 {
		t.Fatalf("admin count=%d before=%d err=%v", after, before, err)
	}
	var auditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE actor_membership_id=$1 AND event_type='role.optin'`, mid).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 0 {
		t.Fatalf("refused self mutation wrote %d success audits", auditCount)
	}
}
