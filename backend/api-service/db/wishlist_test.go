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
