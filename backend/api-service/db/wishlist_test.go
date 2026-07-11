package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestWishlistStore_CRUDAndOwnership(t *testing.T) {
	pool := testPool(t)
	mid, userID := createTestUser(t, pool)
	_, otherID := createTestUser(t, pool)
	store := NewWishlistStore(pool)
	ctx := context.Background()

	gotID, err := store.GetUserID(ctx, mid)
	if err != nil || gotID != userID {
		t.Fatalf("GetUserID = %d, %v; want %d, nil", gotID, err, userID)
	}

	it, err := store.Add(ctx, userID, 1234567890, 2, "test notes")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if it.ItemHash != 1234567890 || it.Priority != 2 || it.Notes != "test notes" {
		t.Errorf("Add returned %+v", it)
	}

	// Duplicate (user_id, item_hash) violates the unique constraint.
	if _, err := store.Add(ctx, userID, 1234567890, 1, ""); !IsDuplicate(err) {
		t.Errorf("duplicate Add err = %v, want unique violation", err)
	}

	// Update by a different user must not match the row.
	prio := int16(3)
	if _, err := store.Update(ctx, otherID, it.ID, &prio, nil); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("cross-user Update err = %v, want pgx.ErrNoRows", err)
	}
	updated, err := store.Update(ctx, userID, it.ID, &prio, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Priority != 3 || updated.Notes != "test notes" {
		t.Errorf("Update returned %+v, want priority 3 with notes preserved", updated)
	}

	// Delete is ownership-scoped as well.
	if ok, err := store.Delete(ctx, otherID, it.ID); err != nil || ok {
		t.Errorf("cross-user Delete = %v, %v; want false, nil", ok, err)
	}
	if ok, err := store.Delete(ctx, userID, it.ID); err != nil || !ok {
		t.Errorf("owner Delete = %v, %v; want true, nil", ok, err)
	}

	items, err := store.List(ctx, userID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("List after delete = %d items, want 0", len(items))
	}
}

func TestWishlistStore_ListOrdering(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewWishlistStore(pool)
	ctx := context.Background()

	if _, err := store.Add(ctx, userID, 100, 0, ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := store.Add(ctx, userID, 200, 3, ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := store.Add(ctx, userID, 300, 1, ""); err != nil {
		t.Fatalf("Add: %v", err)
	}

	items, err := store.List(ctx, userID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("List = %d items, want 3", len(items))
	}
	if items[0].ItemHash != 200 || items[1].ItemHash != 300 || items[2].ItemHash != 100 {
		t.Errorf("List order = %d,%d,%d; want priority DESC 200,300,100",
			items[0].ItemHash, items[1].ItemHash, items[2].ItemHash)
	}
}

func TestWishlistStore_BulkDelete_OwnershipScoped(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	s := NewWishlistStore(pool)

	_, me := createTestUser(t, pool)
	_, other := createTestUser(t, pool)
	a, _ := s.Add(ctx, me, 1001, 1, "")
	b, _ := s.Add(ctx, me, 1002, 1, "")
	foreign, _ := s.Add(ctx, other, 1003, 1, "")

	// Delete two owned + one foreign id; only the two owned are removed.
	updated, err := s.BulkDelete(ctx, me, []int64{a.ID, b.ID, foreign.ID})
	if err != nil {
		t.Fatalf("BulkDelete: %v", err)
	}
	if updated != 2 {
		t.Errorf("updated = %d, want 2 (foreign id skipped)", updated)
	}
	remaining, _ := s.List(ctx, me)
	if len(remaining) != 0 {
		t.Errorf("owned items remaining = %d, want 0", len(remaining))
	}
	stillForeign, _ := s.List(ctx, other)
	if len(stillForeign) != 1 {
		t.Errorf("foreign item wrongly deleted; remaining = %d, want 1", len(stillForeign))
	}
}

func TestWishlistStore_BulkSetPriority_OwnershipScoped(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	s := NewWishlistStore(pool)

	_, me := createTestUser(t, pool)
	_, other := createTestUser(t, pool)
	a, _ := s.Add(ctx, me, 2001, 0, "")
	_, _ = s.Add(ctx, other, 2002, 0, "")

	updated, err := s.BulkSetPriority(ctx, me, []int64{a.ID}, 3)
	if err != nil {
		t.Fatalf("BulkSetPriority: %v", err)
	}
	if updated != 1 {
		t.Errorf("updated = %d, want 1", updated)
	}
	mine, _ := s.List(ctx, me)
	if len(mine) != 1 || mine[0].Priority != 3 {
		t.Errorf("owned item priority = %+v, want priority 3", mine)
	}
	theirs, _ := s.List(ctx, other)
	if len(theirs) != 1 || theirs[0].Priority != 0 {
		t.Errorf("foreign item priority changed: %+v", theirs)
	}
}

func TestWishlistStore_BulkDelete_EmptyIDs(t *testing.T) {
	pool := testPool(t)
	updated, err := NewWishlistStore(pool).BulkDelete(context.Background(), 1, []int64{})
	if err != nil || updated != 0 {
		t.Fatalf("empty ids: updated=%d err=%v, want 0, nil", updated, err)
	}
}
