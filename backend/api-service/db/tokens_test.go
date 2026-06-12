package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func sampleTokens(seed byte) *EncryptedTokens {
	return &EncryptedTokens{
		AccessTokenEnc:   []byte{seed, 1, 2, 3},
		RefreshTokenEnc:  []byte{seed, 4, 5, 6},
		AccessExpiresAt:  time.Now().Add(1 * time.Hour).UTC(),
		RefreshExpiresAt: time.Now().Add(30 * 24 * time.Hour).UTC(),
		KeyVersion:       1,
	}
}

func TestBungieTokenStore_InsertAndGet(t *testing.T) {
	pool := testPool(t)
	mid, _ := createTestUser(t, pool)
	store := NewBungieTokenStore(pool)
	ctx := context.Background()

	in := sampleTokens(1)
	updatedAt, updated, err := store.Upsert(ctx, mid, in, time.Time{})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if !updated || updatedAt.IsZero() {
		t.Fatalf("Upsert = (%v, %v), want non-zero time and updated=true", updatedAt, updated)
	}

	got, err := store.Get(ctx, mid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got.AccessTokenEnc) != string(in.AccessTokenEnc) {
		t.Errorf("access blob mismatch")
	}
	if !got.UpdatedAt.Equal(updatedAt) {
		t.Errorf("Get.UpdatedAt = %v, want %v (RETURNING value)", got.UpdatedAt, updatedAt)
	}
	if got.KeyVersion != 1 {
		t.Errorf("KeyVersion = %d, want 1", got.KeyVersion)
	}
}

// TestBungieTokenStore_UnconditionalUpdate is the B1 regression at the SQL level:
// a second zero-prev upsert must overwrite the existing row.
func TestBungieTokenStore_UnconditionalUpdate(t *testing.T) {
	pool := testPool(t)
	mid, _ := createTestUser(t, pool)
	store := NewBungieTokenStore(pool)
	ctx := context.Background()

	first, updated, err := store.Upsert(ctx, mid, sampleTokens(1), time.Time{})
	if err != nil || !updated {
		t.Fatalf("first Upsert: updated=%v err=%v", updated, err)
	}

	second, updated, err := store.Upsert(ctx, mid, sampleTokens(2), time.Time{})
	if err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	if !updated {
		t.Fatal("second unconditional Upsert wrote 0 rows (B1 regression)")
	}
	if !second.After(first) {
		t.Errorf("updated_at did not advance: first=%v second=%v", first, second)
	}

	got, err := store.Get(ctx, mid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AccessTokenEnc[0] != 2 {
		t.Error("row still holds the first tokens after unconditional update")
	}
}

func TestBungieTokenStore_CASSuccessAndMiss(t *testing.T) {
	pool := testPool(t)
	mid, _ := createTestUser(t, pool)
	store := NewBungieTokenStore(pool)
	ctx := context.Background()

	current, _, err := store.Upsert(ctx, mid, sampleTokens(1), time.Time{})
	if err != nil {
		t.Fatalf("seed Upsert: %v", err)
	}

	// CAS with the correct previous updated_at applies.
	next, updated, err := store.Upsert(ctx, mid, sampleTokens(2), current)
	if err != nil {
		t.Fatalf("CAS Upsert: %v", err)
	}
	if !updated {
		t.Fatal("CAS with matching prev updated_at did not apply")
	}

	// CAS with a stale previous updated_at must not apply and must not error.
	_, updated, err = store.Upsert(ctx, mid, sampleTokens(3), current)
	if err != nil {
		t.Fatalf("stale CAS Upsert: %v", err)
	}
	if updated {
		t.Fatal("CAS with stale prev updated_at applied; expected lost race")
	}

	got, err := store.Get(ctx, mid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AccessTokenEnc[0] != 2 {
		t.Errorf("row holds seed %d, want 2 (the CAS winner)", got.AccessTokenEnc[0])
	}
	if !got.UpdatedAt.Equal(next) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, next)
	}
}

func TestBungieTokenStore_NoUserRow(t *testing.T) {
	pool := testPool(t)
	store := NewBungieTokenStore(pool)

	_, updated, err := store.Upsert(context.Background(), "no-such-membership", sampleTokens(1), time.Time{})
	if !errors.Is(err, ErrNoUserRow) {
		t.Fatalf("Upsert err = %v, want ErrNoUserRow", err)
	}
	if updated {
		t.Fatal("Upsert reported a write for a membership with no users row")
	}
}

func TestBungieTokenStore_Delete(t *testing.T) {
	pool := testPool(t)
	mid, _ := createTestUser(t, pool)
	store := NewBungieTokenStore(pool)
	ctx := context.Background()

	if _, _, err := store.Upsert(ctx, mid, sampleTokens(1), time.Time{}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := store.Delete(ctx, mid); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx, mid); err == nil {
		t.Fatal("Get succeeded after Delete")
	}
}
