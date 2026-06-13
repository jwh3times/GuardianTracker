package db

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestUserStore_UpsertAndTokenVersion(t *testing.T) {
	pool := testPool(t)
	mid, userID := createTestUser(t, pool)
	users := NewUserStore(pool)
	ctx := context.Background()

	if userID <= 0 {
		t.Fatalf("userID = %d, want > 0", userID)
	}

	// Re-upsert returns the same id and does not reset token_version.
	id2, tver, _, err := users.Upsert(ctx, mid, 3, "Renamed Guardian", false)
	if err != nil {
		t.Fatalf("re-Upsert: %v", err)
	}
	if id2 != userID {
		t.Errorf("re-upsert id = %d, want %d", id2, userID)
	}
	if tver != 1 {
		t.Errorf("token_version = %d, want default 1", tver)
	}

	if err := users.BumpTokenVersion(ctx, mid); err != nil {
		t.Fatalf("BumpTokenVersion: %v", err)
	}
	got, err := users.GetTokenVersion(ctx, mid)
	if err != nil {
		t.Fatalf("GetTokenVersion: %v", err)
	}
	if got != 2 {
		t.Errorf("token_version after bump = %d, want 2", got)
	}
}

func TestUserStore_SessionRotationAndReuse(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()
	mid, _ := createTestUser(t, pool)

	// Two independent sessions for the same user (two devices).
	if err := users.CreateSession(ctx, "sid-A", mid, "jti-a1", "deviceA", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession A: %v", err)
	}
	if err := users.CreateSession(ctx, "sid-B", mid, "jti-b1", "deviceB", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession B: %v", err)
	}

	// Rotating A with its current jti succeeds and does not touch B.
	rotated, reused, err := users.RotateSession(ctx, "sid-A", mid, "jti-a1", "jti-a2", time.Now().Add(time.Hour))
	if err != nil || !rotated || reused {
		t.Fatalf("rotate A = (rotated %v, reused %v, err %v), want (true,false,nil)", rotated, reused, err)
	}
	if ok, _ := users.SessionExists(ctx, "sid-B"); !ok {
		t.Error("rotating A must not affect session B (multi-device)")
	}

	// Replaying A's now-superseded jti is detected as reuse and revokes session A.
	rotated, reused, err = users.RotateSession(ctx, "sid-A", mid, "jti-a1", "jti-a3", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("rotate reused: %v", err)
	}
	if rotated || !reused {
		t.Fatalf("replay = (rotated %v, reused %v), want (false,true)", rotated, reused)
	}
	if ok, _ := users.SessionExists(ctx, "sid-A"); ok {
		t.Error("reuse must revoke (delete) session A")
	}
	// B is still usable — the reuse on A did not cascade.
	if ok, _ := users.SessionExists(ctx, "sid-B"); !ok {
		t.Error("session B must survive a reuse event on session A")
	}

	// An unknown session rotates to nothing (neither rotated nor reused).
	rotated, reused, err = users.RotateSession(ctx, "sid-missing", mid, "x", "y", time.Now().Add(time.Hour))
	if err != nil || rotated || reused {
		t.Fatalf("rotate missing = (rotated %v, reused %v, err %v), want (false,false,nil)", rotated, reused, err)
	}
}

func TestUserStore_SessionExpiryAndDeletion(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()
	mid, _ := createTestUser(t, pool)

	// An already-expired session is not "exists" and cannot be rotated.
	if err := users.CreateSession(ctx, "sid-exp", mid, "jti-x", "", time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("CreateSession expired: %v", err)
	}
	if ok, err := users.SessionExists(ctx, "sid-exp"); err != nil || ok {
		t.Fatalf("expired SessionExists = (%v, %v), want (false, nil)", ok, err)
	}
	rotated, reused, err := users.RotateSession(ctx, "sid-exp", mid, "jti-x", "jti-y", time.Now().Add(time.Hour))
	if err != nil || rotated || reused {
		t.Fatalf("rotate expired = (rotated %v, reused %v, err %v), want all-false/nil", rotated, reused, err)
	}

	// DeleteUserSessions clears every session for the user (sign-out-everywhere).
	_ = users.CreateSession(ctx, "sid-1", mid, "j1", "", time.Now().Add(time.Hour))
	_ = users.CreateSession(ctx, "sid-2", mid, "j2", "", time.Now().Add(time.Hour))
	if err := users.DeleteUserSessions(ctx, mid); err != nil {
		t.Fatalf("DeleteUserSessions: %v", err)
	}
	for _, sid := range []string{"sid-1", "sid-2"} {
		if ok, _ := users.SessionExists(ctx, sid); ok {
			t.Errorf("session %s should be gone after DeleteUserSessions", sid)
		}
	}
}

func TestUserStore_SessionSlidingExpiry(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()
	mid, _ := createTestUser(t, pool)

	// Session starts expiring soon; a rotation must slide expires_at forward so an
	// active session does not hard-expire at its original creation time.
	near := time.Now().Add(2 * time.Minute)
	far := time.Now().Add(720 * time.Hour)
	if err := users.CreateSession(ctx, "sid-slide", mid, "j1", "", near); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if rotated, _, err := users.RotateSession(ctx, "sid-slide", mid, "j1", "j2", far); err != nil || !rotated {
		t.Fatalf("rotate: rotated=%v err=%v", rotated, err)
	}
	var exp time.Time
	if err := pool.QueryRow(ctx, `SELECT expires_at FROM refresh_sessions WHERE id = $1`, "sid-slide").Scan(&exp); err != nil {
		t.Fatalf("read expires_at: %v", err)
	}
	if exp.Before(time.Now().Add(700 * time.Hour)) {
		t.Errorf("expires_at = %v, expected it to slide forward to ~%v", exp, far)
	}
}

func TestUserStore_SessionPerUserCap(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()
	mid, _ := createTestUser(t, pool)

	// Creating more than the cap evicts the oldest-by-use; the count never exceeds it.
	for i := range maxSessionsPerUser + 5 {
		id := fmt.Sprintf("sid-cap-%d", i)
		if err := users.CreateSession(ctx, id, mid, "j", "", time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("CreateSession %s: %v", id, err)
		}
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM refresh_sessions WHERE membership_id = $1`, mid).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != maxSessionsPerUser {
		t.Errorf("session count = %d, want capped at %d", n, maxSessionsPerUser)
	}
}

func TestUserStore_DeleteExpiredSessions(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	ctx := context.Background()
	mid, _ := createTestUser(t, pool)

	_ = users.CreateSession(ctx, "sid-dead", mid, "j", "", time.Now().Add(-time.Hour))
	_ = users.CreateSession(ctx, "sid-live", mid, "j", "", time.Now().Add(time.Hour))

	n, err := users.DeleteExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredSessions: %v", err)
	}
	if n < 1 {
		t.Errorf("removed %d, want at least the one expired session", n)
	}
	if ok, _ := users.SessionExists(ctx, "sid-live"); !ok {
		t.Error("live session must survive pruning")
	}
	// The expired row is gone from the table (SessionExists already filters expiry).
	var present bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM refresh_sessions WHERE id = $1)`, "sid-dead").Scan(&present); err != nil {
		t.Fatalf("check: %v", err)
	}
	if present {
		t.Error("expired session row should have been deleted by the pruner")
	}
}

func TestUserStore_GetTokenVersionUnknownUser(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	if _, err := users.GetTokenVersion(context.Background(), "no-such-membership"); err == nil {
		t.Fatal("expected error for unknown membership id")
	}
}
