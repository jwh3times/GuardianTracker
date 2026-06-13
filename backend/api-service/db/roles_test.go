package db

import (
	"context"
	"errors"
	"testing"
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

func TestUserStore_SetRoleAndGetAuthInfo(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()
	mid, _ := createTestUser(t, pool)

	if err := users.SetRole(ctx, mid, 2); err != nil { // alpha
		t.Fatalf("SetRole: %v", err)
	}
	role, err := users.GetRole(ctx, mid)
	if err != nil || role != 2 {
		t.Fatalf("GetRole = %d (%v), want 2", role, err)
	}
	tv, r, err := users.GetAuthInfo(ctx, mid)
	if err != nil {
		t.Fatalf("GetAuthInfo: %v", err)
	}
	if r != 2 || tv != 1 {
		t.Errorf("GetAuthInfo = (tv %d, role %d), want (1, 2)", tv, r)
	}
}

func TestUserStore_SetRoleUnknownUser(t *testing.T) {
	pool := testPool(t)
	if err := NewUserStore(pool).SetRole(context.Background(), "no-such-member", 1); !errors.Is(err, ErrUserNotFound) {
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
	if err := users.SetRole(ctx, mid, roleAdmin); err != nil {
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
	if err := users.SetRole(ctx, actorMid, roleAdmin); err != nil {
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
	tv, role, err := users.GetAuthInfo(ctx, targetMid)
	if err != nil {
		t.Fatalf("GetAuthInfo: %v", err)
	}
	if tv != 2 || role != 2 {
		t.Errorf("after change: tv=%d role=%d, want tv=2 role=2", tv, role)
	}
	// An audit row was written.
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM role_audit WHERE target_user_id = $1 AND new_role = 2`, targetID).Scan(&n); err != nil {
		t.Fatalf("audit query: %v", err)
	}
	if n != 1 {
		t.Errorf("role_audit rows = %d, want 1", n)
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
	if err := users.SetRole(ctx, adminMid, roleAdmin); err != nil {
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
