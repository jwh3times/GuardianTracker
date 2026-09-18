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
	prev := stamp.Add(-3 * time.Hour)
	state := DigestState{
		LastActivityAt:              stamp,
		VisitStartedAt:              stamp,
		Snapshot:                    []uint32{1, 2, 3},
		SnapshotTakenAt:             stamp,
		CurrentVisitStatus:          "ready",
		CurrentVisitPreviousVisitAt: &prev,
		CurrentVisitAcquired:        []uint32{3},
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
	if got.CurrentVisitStatus != "ready" {
		t.Fatalf("CurrentVisitStatus = %q, want ready", got.CurrentVisitStatus)
	}
	if got.CurrentVisitPreviousVisitAt == nil || !got.CurrentVisitPreviousVisitAt.Equal(prev) {
		t.Fatalf("CurrentVisitPreviousVisitAt = %v, want %v", got.CurrentVisitPreviousVisitAt, prev)
	}
	if len(got.CurrentVisitAcquired) != 1 || got.CurrentVisitAcquired[0] != 3 {
		t.Fatalf("CurrentVisitAcquired = %v, want [3]", got.CurrentVisitAcquired)
	}
}

func TestDigestStore_GetOnAFreshRowDefaultsCurrentVisitResultToUnavailable(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewDigestStore(pool)
	ctx := context.Background()

	// Save with the zero-value current-visit fields, as a caller that only
	// cares about the snapshot columns might (defensive: the column default
	// on direct SQL INSERTs is exercised by the DEFAULT clause itself, but
	// Save always sends an explicit value).
	stamp := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	if err := store.Save(ctx, userID, DigestState{
		LastActivityAt: stamp, VisitStartedAt: stamp, Snapshot: []uint32{1}, SnapshotTakenAt: stamp,
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(ctx, userID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.CurrentVisitAcquired) != 0 {
		t.Fatalf("CurrentVisitAcquired = %v, want empty", got.CurrentVisitAcquired)
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
		CurrentVisitStatus: "first-visit",
	}); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	second := first.Add(3 * time.Hour)
	if err := store.Save(ctx, userID, DigestState{
		LastActivityAt: second, VisitStartedAt: second, Snapshot: []uint32{1, 2}, SnapshotTakenAt: second,
		CurrentVisitStatus: "ready", CurrentVisitAcquired: []uint32{2},
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
	if got.CurrentVisitStatus != "ready" || len(got.CurrentVisitAcquired) != 1 || got.CurrentVisitAcquired[0] != 2 {
		t.Fatalf("current-visit fields = (%q, %v), want (ready, [2]) — the first visit's frozen outcome must not linger", got.CurrentVisitStatus, got.CurrentVisitAcquired)
	}
}

func TestDigestStore_TouchActivityNeverChangesAnythingElse(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewDigestStore(pool)
	ctx := context.Background()

	stamp := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	if err := store.Save(ctx, userID, DigestState{
		LastActivityAt: stamp, VisitStartedAt: stamp, Snapshot: []uint32{1, 2}, SnapshotTakenAt: stamp,
		CurrentVisitStatus: "ready", CurrentVisitAcquired: []uint32{2},
	}); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	later := stamp.Add(30 * time.Minute)
	if err := store.TouchActivity(ctx, userID, later); err != nil {
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
	if got.CurrentVisitStatus != "ready" || len(got.CurrentVisitAcquired) != 1 {
		t.Fatalf("current-visit fields = (%q, %v), want unchanged (ready, [2]) — TouchActivity must never touch the frozen outcome", got.CurrentVisitStatus, got.CurrentVisitAcquired)
	}
}

func TestDigestStore_TouchActivityIsANoOpWithoutAnExistingRow(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewDigestStore(pool)
	ctx := context.Background()

	if err := store.TouchActivity(ctx, userID, time.Now()); err != nil {
		t.Fatalf("TouchActivity on a missing row returned an error: %v", err)
	}
	if _, err := store.Get(ctx, userID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Get after a no-op touch = %v, want pgx.ErrNoRows — no row must be created without a snapshot", err)
	}
}

func TestDigestStore_RecordUnavailableVisitAdvancesClockAndResultButNeverTheSnapshot(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewDigestStore(pool)
	ctx := context.Background()

	stamp := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	if err := store.Save(ctx, userID, DigestState{
		LastActivityAt: stamp, VisitStartedAt: stamp, Snapshot: []uint32{1, 2}, SnapshotTakenAt: stamp,
		CurrentVisitStatus: "ready", CurrentVisitAcquired: []uint32{2},
	}); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	newVisit := stamp.Add(3 * time.Hour)
	prev := stamp
	if err := store.RecordUnavailableVisit(ctx, userID, newVisit, &prev); err != nil {
		t.Fatalf("RecordUnavailableVisit: %v", err)
	}

	got, err := store.Get(ctx, userID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.LastActivityAt.Equal(newVisit) || !got.VisitStartedAt.Equal(newVisit) {
		t.Fatalf("visit clock = (%v, %v), want both %v", got.LastActivityAt, got.VisitStartedAt, newVisit)
	}
	if len(got.Snapshot) != 2 || got.Snapshot[0] != 1 || got.Snapshot[1] != 2 {
		t.Fatalf("Snapshot = %v, want unchanged [1 2] — RecordUnavailableVisit must never write the snapshot baseline", got.Snapshot)
	}
	if !got.SnapshotTakenAt.Equal(stamp) {
		t.Fatalf("SnapshotTakenAt = %v, want unchanged %v", got.SnapshotTakenAt, stamp)
	}
	if got.CurrentVisitStatus != "unavailable" {
		t.Fatalf("CurrentVisitStatus = %q, want unavailable", got.CurrentVisitStatus)
	}
	if got.CurrentVisitPreviousVisitAt == nil || !got.CurrentVisitPreviousVisitAt.Equal(prev) {
		t.Fatalf("CurrentVisitPreviousVisitAt = %v, want %v", got.CurrentVisitPreviousVisitAt, prev)
	}
	if len(got.CurrentVisitAcquired) != 0 {
		t.Fatalf("CurrentVisitAcquired = %v, want empty", got.CurrentVisitAcquired)
	}
}

func TestDigestStore_RecordUnavailableVisitIsANoOpWithoutAnExistingRow(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewDigestStore(pool)
	ctx := context.Background()

	if err := store.RecordUnavailableVisit(ctx, userID, time.Now(), nil); err != nil {
		t.Fatalf("RecordUnavailableVisit on a missing row returned an error: %v", err)
	}
	if _, err := store.Get(ctx, userID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Get after a no-op record = %v, want pgx.ErrNoRows — no row must be created without a snapshot", err)
	}
}
