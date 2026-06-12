package db

import (
	"context"
	"testing"
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
	id2, tver, err := users.Upsert(ctx, mid, 3, "Renamed Guardian")
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

func TestUserStore_GetTokenVersionUnknownUser(t *testing.T) {
	pool := testPool(t)
	users := NewUserStore(pool)
	if _, err := users.GetTokenVersion(context.Background(), "no-such-membership"); err == nil {
		t.Fatal("expected error for unknown membership id")
	}
}
