package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestDigestStore_GetReportsMissingRow(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)

	_, err := NewDigestStore(pool).Get(context.Background(), userID)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Get error = %v, want pgx.ErrNoRows", err)
	}
}

func TestDigestStore_SaveThenGetRoundTripsTheWholeRow(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewDigestStore(pool)
	ctx := context.Background()

	stamp := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	state := DigestState{
		LastActivityAt:  stamp,
		VisitStartedAt:  stamp,
		Snapshot:        []uint32{1, 2, 3},
		SnapshotTakenAt: stamp,
	}
	if err := store.Save(ctx, userID, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(ctx, userID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.LastActivityAt.Equal(stamp) || !got.VisitStartedAt.Equal(stamp) || !got.SnapshotTakenAt.Equal(stamp) {
		t.Fatalf("Get timestamps = %+v, want all %v", got, stamp)
	}
	if len(got.Snapshot) != 3 || got.Snapshot[0] != 1 || got.Snapshot[1] != 2 || got.Snapshot[2] != 3 {
		t.Fatalf("Get snapshot = %v, want [1 2 3]", got.Snapshot)
	}
}

func TestDigestStore_SaveReplacesThePreviousRowAtomically(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewDigestStore(pool)
	ctx := context.Background()

	first := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	if err := store.Save(ctx, userID, DigestState{
		LastActivityAt: first, VisitStartedAt: first, Snapshot: []uint32{1}, SnapshotTakenAt: first,
	}); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	second := first.Add(3 * time.Hour)
	if err := store.Save(ctx, userID, DigestState{
		LastActivityAt: second, VisitStartedAt: second, Snapshot: []uint32{1, 2}, SnapshotTakenAt: second,
	}); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	got, err := store.Get(ctx, userID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.VisitStartedAt.Equal(second) {
		t.Fatalf("VisitStartedAt = %v, want the second visit's %v", got.VisitStartedAt, second)
	}
	if len(got.Snapshot) != 2 {
		t.Fatalf("Snapshot = %v, want the replaced [1 2]", got.Snapshot)
	}
}

func TestDigestStore_TouchActivityNeverChangesTheSnapshot(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewDigestStore(pool)
	ctx := context.Background()

	stamp := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	if err := store.Save(ctx, userID, DigestState{
		LastActivityAt: stamp, VisitStartedAt: stamp, Snapshot: []uint32{1, 2}, SnapshotTakenAt: stamp,
	}); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	later := stamp.Add(30 * time.Minute)
	if err := store.TouchActivity(ctx, userID, later, nil); err != nil {
		t.Fatalf("TouchActivity: %v", err)
	}

	got, err := store.Get(ctx, userID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.LastActivityAt.Equal(later) {
		t.Fatalf("LastActivityAt = %v, want %v", got.LastActivityAt, later)
	}
	if !got.VisitStartedAt.Equal(stamp) {
		t.Fatalf("VisitStartedAt = %v, want unchanged %v", got.VisitStartedAt, stamp)
	}
	if len(got.Snapshot) != 2 || got.Snapshot[0] != 1 || got.Snapshot[1] != 2 {
		t.Fatalf("Snapshot = %v, want unchanged [1 2] — TouchActivity must never write the snapshot", got.Snapshot)
	}
	if !got.SnapshotTakenAt.Equal(stamp) {
		t.Fatalf("SnapshotTakenAt = %v, want unchanged %v", got.SnapshotTakenAt, stamp)
	}
}

func TestDigestStore_TouchActivityWithNewVisitAdvancesVisitStartedAt(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewDigestStore(pool)
	ctx := context.Background()

	stamp := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	if err := store.Save(ctx, userID, DigestState{
		LastActivityAt: stamp, VisitStartedAt: stamp, Snapshot: []uint32{1}, SnapshotTakenAt: stamp,
	}); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	newVisit := stamp.Add(3 * time.Hour)
	if err := store.TouchActivity(ctx, userID, newVisit, &newVisit); err != nil {
		t.Fatalf("TouchActivity: %v", err)
	}

	got, err := store.Get(ctx, userID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.VisitStartedAt.Equal(newVisit) {
		t.Fatalf("VisitStartedAt = %v, want %v", got.VisitStartedAt, newVisit)
	}
	if len(got.Snapshot) != 1 || got.Snapshot[0] != 1 {
		t.Fatalf("Snapshot = %v, want unchanged [1]", got.Snapshot)
	}
}

func TestDigestStore_TouchActivityIsANoOpWithoutAnExistingRow(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewDigestStore(pool)
	ctx := context.Background()

	if err := store.TouchActivity(ctx, userID, time.Now(), nil); err != nil {
		t.Fatalf("TouchActivity on a missing row returned an error: %v", err)
	}
	if _, err := store.Get(ctx, userID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Get after a no-op touch = %v, want pgx.ErrNoRows — no row must be created without a snapshot", err)
	}
}
