package digest

import (
	"context"
	"errors"
	"testing"
	"time"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/items"
)

// stubRepository is the fake Repository every test drives directly, mirroring
// preferences' stubRepository pattern. It stands in for a real row: state set
// on it before a call is what Get returns; state recorded during a call is
// what a real UPDATE/INSERT would have written.
type stubRepository struct {
	getSnap  Snapshot
	getFound bool
	getErr   error

	touchCalls          int
	touchLastActivityAt time.Time
	touchErr            error

	recordCalls           int
	recordAt              time.Time
	recordPreviousVisitAt *time.Time
	recordErr             error

	saveCalls int
	saveSnap  Snapshot
	saveErr   error
}

func (s *stubRepository) Get(context.Context, string) (Snapshot, bool, error) {
	return s.getSnap, s.getFound, s.getErr
}

func (s *stubRepository) TouchActivity(_ context.Context, _ string, at time.Time) error {
	s.touchCalls++
	s.touchLastActivityAt = at
	return s.touchErr
}

func (s *stubRepository) RecordUnavailableVisit(_ context.Context, _ string, at time.Time, previousVisitAt *time.Time) error {
	s.recordCalls++
	s.recordAt = at
	s.recordPreviousVisitAt = previousVisitAt
	return s.recordErr
}

func (s *stubRepository) Save(_ context.Context, _ string, snap Snapshot) error {
	s.saveCalls++
	s.saveSnap = snap
	return s.saveErr
}

// stubCollections is the fake CollectionsReader.
type stubCollections struct {
	owned   map[uint32]bool
	privacy int
	err     error
	calls   int
}

func (s *stubCollections) CollectedState(context.Context, int, string, string) (map[uint32]bool, int, time.Time, error) {
	s.calls++
	return s.owned, s.privacy, time.Time{}, s.err
}

// stubItems is the fake ItemLookup.
type stubItems struct {
	facts map[uint32]items.AcquisitionFacts
	err   error
	calls int
}

func (s *stubItems) Lookup(_ context.Context, hashes []uint32) (map[uint32]items.AcquisitionFacts, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	out := make(map[uint32]items.AcquisitionFacts, len(hashes))
	for _, h := range hashes {
		if f, ok := s.facts[h]; ok {
			out[h] = f
		}
	}
	return out, nil
}

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

var baseTime = time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)

// --- Required property 1: a private read writes no snapshot. ---

func TestGetDigest_PrivateReadWritesNoSnapshot(t *testing.T) {
	repo := &stubRepository{
		getFound: true,
		getSnap: Snapshot{
			LastActivityAt:  baseTime.Add(-3 * time.Hour),
			VisitStartedAt:  baseTime.Add(-3 * time.Hour),
			OwnedItemHashes: []uint32{1, 2, 3},
			SnapshotTakenAt: baseTime.Add(-3 * time.Hour),
		},
	}
	collections := &stubCollections{owned: map[uint32]bool{1: true, 2: true, 3: true, 4: true}, privacy: bungie.CollectiblesPrivacyPrivate}
	svc := NewServiceWithClock(repo, collections, &stubItems{}, fixedClock(baseTime))

	result, err := svc.GetDigest(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("GetDigest: %v", err)
	}
	if result.Status != StatusUnavailable {
		t.Fatalf("Status = %q, want %q", result.Status, StatusUnavailable)
	}
	if repo.saveCalls != 0 {
		t.Fatalf("Save calls = %d, want 0 — a private read must never write a snapshot", repo.saveCalls)
	}
}

// --- Required property 2: a failed read writes no snapshot. ---

func TestGetDigest_FailedReadWritesNoSnapshot(t *testing.T) {
	repo := &stubRepository{
		getFound: true,
		getSnap: Snapshot{
			LastActivityAt:  baseTime.Add(-3 * time.Hour),
			VisitStartedAt:  baseTime.Add(-3 * time.Hour),
			OwnedItemHashes: []uint32{1, 2, 3},
			SnapshotTakenAt: baseTime.Add(-3 * time.Hour),
		},
	}
	collections := &stubCollections{err: errors.New("bungie unavailable")}
	svc := NewServiceWithClock(repo, collections, &stubItems{}, fixedClock(baseTime))

	result, err := svc.GetDigest(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("GetDigest: %v", err)
	}
	if result.Status != StatusUnavailable {
		t.Fatalf("Status = %q, want %q", result.Status, StatusUnavailable)
	}
	if repo.saveCalls != 0 {
		t.Fatalf("Save calls = %d, want 0 — a failed read must never write a snapshot", repo.saveCalls)
	}
}

// --- Required property 3: first visit reports first-visit, not an empty
// digest and not every collectible. ---

func TestGetDigest_FirstVisitReportsFirstVisitNotAnEmptyOrFullDigest(t *testing.T) {
	repo := &stubRepository{} // no prior row: found=false
	collections := &stubCollections{owned: map[uint32]bool{10: true, 20: true}, privacy: bungie.CollectiblesPrivacyPublic}
	svc := NewServiceWithClock(repo, collections, &stubItems{}, fixedClock(baseTime))

	result, err := svc.GetDigest(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("GetDigest: %v", err)
	}
	if result.Status != StatusFirstVisit {
		t.Fatalf("Status = %q, want %q", result.Status, StatusFirstVisit)
	}
	if len(result.Acquired) != 0 {
		t.Fatalf("Acquired = %v, want empty — a first visit must not dump the whole collection", result.Acquired)
	}
	if repo.saveCalls != 1 {
		t.Fatalf("Save calls = %d, want 1 — the first visit establishes the baseline", repo.saveCalls)
	}
	if len(repo.saveSnap.OwnedItemHashes) != 2 {
		t.Fatalf("saved snapshot = %v, want both currently owned hashes", repo.saveSnap.OwnedItemHashes)
	}
	if repo.saveSnap.CurrentVisitStatus != StatusFirstVisit {
		t.Fatalf("saved current-visit status = %q, want %q", repo.saveSnap.CurrentVisitStatus, StatusFirstVisit)
	}
}

// --- Required property 4: a second request inside the same visit returns the
// identical frozen digest and does not re-snapshot. ---

func TestGetDigest_SecondRequestInSameVisitReturnsFrozenDigestWithoutResnapshotting(t *testing.T) {
	repo := &stubRepository{
		getFound: true,
		getSnap: Snapshot{
			LastActivityAt:  baseTime.Add(-3 * time.Hour),
			VisitStartedAt:  baseTime.Add(-3 * time.Hour),
			OwnedItemHashes: []uint32{1},
			SnapshotTakenAt: baseTime.Add(-3 * time.Hour),
		},
	}
	collections := &stubCollections{owned: map[uint32]bool{1: true, 2: true}, privacy: bungie.CollectiblesPrivacyPublic}
	itemLookup := &stubItems{facts: map[uint32]items.AcquisitionFacts{
		2: {ItemHash: 2, Name: "New Gun", Icon: "/icon.png", ItemType: "Auto Rifle"},
	}}
	svc := NewServiceWithClock(repo, collections, itemLookup, fixedClock(baseTime))

	first, err := svc.GetDigest(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("first GetDigest: %v", err)
	}
	if first.Status != StatusReady || len(first.Acquired) != 1 || first.Acquired[0].ItemHash != 2 {
		t.Fatalf("first result = %+v, want ready with item 2 acquired", first)
	}
	if repo.saveCalls != 1 {
		t.Fatalf("Save calls after first request = %d, want 1", repo.saveCalls)
	}

	// The repository now reflects the just-written row, as it would in
	// production: existing.OwnedItemHashes == the current live read (so a
	// naive recompute of the *baseline* diff would show zero acquisitions),
	// while CurrentVisit* carries the frozen outcome forward.
	repo.getSnap = repo.saveSnap
	collections.owned = map[uint32]bool{1: true, 2: true} // unchanged mid-visit

	second, err := svc.GetDigest(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("second GetDigest: %v", err)
	}
	if second.Status != first.Status || len(second.Acquired) != 1 || second.Acquired[0].ItemHash != first.Acquired[0].ItemHash {
		t.Fatalf("second result = %+v, want identical to first %+v", second, first)
	}
	if second.Acquired[0].Name != first.Acquired[0].Name {
		t.Fatalf("second result item facts = %+v, want identical to first %+v", second.Acquired[0], first.Acquired[0])
	}
	if !second.VisitStartedAt.Equal(first.VisitStartedAt) {
		t.Fatalf("second VisitStartedAt = %v, want %v", second.VisitStartedAt, first.VisitStartedAt)
	}
	if second.PreviousVisitAt == nil || first.PreviousVisitAt == nil || !second.PreviousVisitAt.Equal(*first.PreviousVisitAt) {
		t.Fatalf("second PreviousVisitAt = %v, want %v", second.PreviousVisitAt, first.PreviousVisitAt)
	}
	if repo.saveCalls != 1 {
		t.Fatalf("Save calls after second request = %d, want still 1 — a same-visit request must not re-snapshot", repo.saveCalls)
	}
	if collections.calls != 1 {
		t.Fatalf("Collections read calls = %d, want still 1 — a same-visit request must not re-read Bungie", collections.calls)
	}
}

// This is the property the process-local cache design could not hold: a
// second Service instance — standing in for the process having restarted
// between the two requests — reads the same row and must reconstruct the
// identical frozen digest purely from persistence, with no shared memory.
func TestGetDigest_SecondRequestAfterARestartStillReturnsTheFrozenDigest(t *testing.T) {
	repo := &stubRepository{
		getFound: true,
		getSnap: Snapshot{
			LastActivityAt:  baseTime.Add(-3 * time.Hour),
			VisitStartedAt:  baseTime.Add(-3 * time.Hour),
			OwnedItemHashes: []uint32{1},
			SnapshotTakenAt: baseTime.Add(-3 * time.Hour),
		},
	}
	collections := &stubCollections{owned: map[uint32]bool{1: true, 2: true}, privacy: bungie.CollectiblesPrivacyPublic}
	itemLookup := &stubItems{facts: map[uint32]items.AcquisitionFacts{
		2: {ItemHash: 2, Name: "New Gun", Icon: "/icon.png", ItemType: "Auto Rifle"},
	}}

	first, err := NewServiceWithClock(repo, collections, itemLookup, fixedClock(baseTime)).
		GetDigest(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("first GetDigest: %v", err)
	}

	// Simulate a restart: the row is exactly what persistence now holds
	// (repo.saveSnap from the write above), and a brand-new Service is built
	// around it — no field, map, or mutex is carried over from the first one.
	repo.getSnap = repo.saveSnap
	restarted := NewServiceWithClock(repo, collections, itemLookup, fixedClock(baseTime))

	second, err := restarted.GetDigest(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("second GetDigest (post-restart): %v", err)
	}
	if second.Status != first.Status {
		t.Fatalf("post-restart Status = %q, want %q", second.Status, first.Status)
	}
	if len(second.Acquired) != len(first.Acquired) || second.Acquired[0].ItemHash != first.Acquired[0].ItemHash || second.Acquired[0].Name != first.Acquired[0].Name {
		t.Fatalf("post-restart Acquired = %+v, want identical to pre-restart %+v", second.Acquired, first.Acquired)
	}
	if !second.VisitStartedAt.Equal(first.VisitStartedAt) {
		t.Fatalf("post-restart VisitStartedAt = %v, want %v", second.VisitStartedAt, first.VisitStartedAt)
	}
	if second.PreviousVisitAt == nil || first.PreviousVisitAt == nil || !second.PreviousVisitAt.Equal(*first.PreviousVisitAt) {
		t.Fatalf("post-restart PreviousVisitAt = %v, want %v", second.PreviousVisitAt, first.PreviousVisitAt)
	}
	if repo.saveCalls != 1 {
		t.Fatalf("Save calls after the post-restart request = %d, want still 1 — it must read the frozen row, not recompute", repo.saveCalls)
	}
	if collections.calls != 1 {
		t.Fatalf("Collections read calls after the post-restart request = %d, want still 1", collections.calls)
	}
}

// --- Required property 5: a shrinking public read accepts the new baseline
// and surfaces no loss. ---

func TestGetDigest_ShrinkingPublicReadAcceptsBaselineWithoutSurfacingLoss(t *testing.T) {
	repo := &stubRepository{
		getFound: true,
		getSnap: Snapshot{
			LastActivityAt:  baseTime.Add(-3 * time.Hour),
			VisitStartedAt:  baseTime.Add(-3 * time.Hour),
			OwnedItemHashes: []uint32{1, 2, 3},
			SnapshotTakenAt: baseTime.Add(-3 * time.Hour),
		},
	}
	// The account now owns fewer items than the stored snapshot — e.g. every
	// character was deleted.
	collections := &stubCollections{owned: map[uint32]bool{1: true}, privacy: bungie.CollectiblesPrivacyPublic}
	svc := NewServiceWithClock(repo, collections, &stubItems{}, fixedClock(baseTime))

	result, err := svc.GetDigest(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("GetDigest: %v", err)
	}
	if result.Status != StatusReady {
		t.Fatalf("Status = %q, want %q — a shrinking read is accepted, not refused", result.Status, StatusReady)
	}
	if len(result.Acquired) != 0 {
		t.Fatalf("Acquired = %v, want empty — nothing new was acquired", result.Acquired)
	}
	if repo.saveCalls != 1 {
		t.Fatalf("Save calls = %d, want 1 — the new (smaller) baseline is still accepted", repo.saveCalls)
	}
	if len(repo.saveSnap.OwnedItemHashes) != 1 || repo.saveSnap.OwnedItemHashes[0] != 1 {
		t.Fatalf("saved snapshot = %v, want the new smaller set [1]", repo.saveSnap.OwnedItemHashes)
	}
}

// --- Supporting coverage: a normal second visit (past the gap) recomputes. ---

func TestGetDigest_NewVisitAfterGapRecomputesAgainstStoredSnapshot(t *testing.T) {
	repo := &stubRepository{
		getFound: true,
		getSnap: Snapshot{
			LastActivityAt:  baseTime.Add(-3 * time.Hour),
			VisitStartedAt:  baseTime.Add(-3 * time.Hour),
			OwnedItemHashes: []uint32{1},
			SnapshotTakenAt: baseTime.Add(-3 * time.Hour),
		},
	}
	collections := &stubCollections{owned: map[uint32]bool{1: true, 2: true}, privacy: bungie.CollectiblesPrivacyPublic}
	itemLookup := &stubItems{facts: map[uint32]items.AcquisitionFacts{
		2: {ItemHash: 2, Name: "New Gun"},
	}}
	svc := NewServiceWithClock(repo, collections, itemLookup, fixedClock(baseTime))

	result, err := svc.GetDigest(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("GetDigest: %v", err)
	}
	if result.Status != StatusReady {
		t.Fatalf("Status = %q, want %q", result.Status, StatusReady)
	}
	if len(result.Acquired) != 1 || result.Acquired[0].ItemHash != 2 {
		t.Fatalf("Acquired = %+v, want exactly item 2", result.Acquired)
	}
	if result.PreviousVisitAt == nil || !result.PreviousVisitAt.Equal(baseTime.Add(-3*time.Hour)) {
		t.Fatalf("PreviousVisitAt = %v, want the prior last-activity time", result.PreviousVisitAt)
	}
}

// A request inside the 2h gap must not touch Bungie or Collections at all,
// and must read the frozen result straight off the row.
func TestGetDigest_WithinGapNeverReadsCollectionsAndReturnsPersistedResult(t *testing.T) {
	frozenAcquired := baseTime.Add(-30 * time.Minute)
	prev := baseTime.Add(-4 * time.Hour)
	repo := &stubRepository{
		getFound: true,
		getSnap: Snapshot{
			LastActivityAt:              frozenAcquired,
			VisitStartedAt:              frozenAcquired,
			OwnedItemHashes:             []uint32{1, 2},
			SnapshotTakenAt:             frozenAcquired,
			CurrentVisitStatus:          StatusReady,
			CurrentVisitPreviousVisitAt: &prev,
			CurrentVisitAcquired:        []uint32{2},
		},
	}
	collections := &stubCollections{}
	itemLookup := &stubItems{facts: map[uint32]items.AcquisitionFacts{2: {ItemHash: 2, Name: "Frozen Gun"}}}
	svc := NewServiceWithClock(repo, collections, itemLookup, fixedClock(baseTime))

	result, err := svc.GetDigest(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("GetDigest: %v", err)
	}
	if collections.calls != 0 {
		t.Fatalf("Collections read calls = %d, want 0 — a within-visit request must never read Bungie", collections.calls)
	}
	if result.Status != StatusReady || len(result.Acquired) != 1 || result.Acquired[0].ItemHash != 2 || result.Acquired[0].Name != "Frozen Gun" {
		t.Fatalf("result = %+v, want the persisted frozen outcome (item 2, Frozen Gun)", result)
	}
	if result.PreviousVisitAt == nil || !result.PreviousVisitAt.Equal(prev) {
		t.Fatalf("PreviousVisitAt = %v, want the persisted %v", result.PreviousVisitAt, prev)
	}
	if repo.touchCalls != 1 {
		t.Fatalf("touch calls = %d, want 1 — only last_activity_at advances", repo.touchCalls)
	}
	if repo.saveCalls != 0 || repo.recordCalls != 0 {
		t.Fatalf("saveCalls=%d recordCalls=%d, want both 0 — a same-visit request writes neither", repo.saveCalls, repo.recordCalls)
	}
}

func TestGetDigest_RepositoryUnavailableReportsUnavailable(t *testing.T) {
	repo := &stubRepository{getErr: ErrUnavailable}
	svc := NewServiceWithClock(repo, &stubCollections{}, &stubItems{}, fixedClock(baseTime))

	result, err := svc.GetDigest(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("GetDigest: %v", err)
	}
	if result.Status != StatusUnavailable {
		t.Fatalf("Status = %q, want %q", result.Status, StatusUnavailable)
	}
}

// A failed read on an established visit records the frozen "unavailable"
// outcome (via RecordUnavailableVisit) rather than leaving the row silently
// pointed at a stale successful visit's result.
func TestGetDigest_FailedReadOnExistingRowRecordsUnavailableVisit(t *testing.T) {
	repo := &stubRepository{
		getFound: true,
		getSnap: Snapshot{
			LastActivityAt:  baseTime.Add(-3 * time.Hour),
			VisitStartedAt:  baseTime.Add(-3 * time.Hour),
			OwnedItemHashes: []uint32{1},
			SnapshotTakenAt: baseTime.Add(-3 * time.Hour),
		},
	}
	collections := &stubCollections{err: errors.New("bungie down")}
	svc := NewServiceWithClock(repo, collections, &stubItems{}, fixedClock(baseTime))

	if _, err := svc.GetDigest(context.Background(), 3, "member-1", "token"); err != nil {
		t.Fatalf("GetDigest: %v", err)
	}
	if repo.recordCalls != 1 {
		t.Fatalf("RecordUnavailableVisit calls = %d, want 1", repo.recordCalls)
	}
	if !repo.recordAt.Equal(baseTime) {
		t.Fatalf("record at = %v, want %v", repo.recordAt, baseTime)
	}
	if repo.recordPreviousVisitAt == nil || !repo.recordPreviousVisitAt.Equal(baseTime.Add(-3*time.Hour)) {
		t.Fatalf("record previousVisitAt = %v, want %v", repo.recordPreviousVisitAt, baseTime.Add(-3*time.Hour))
	}
}

func TestNewService_PanicsOnMissingDependency(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewService did not panic on a nil dependency")
		}
	}()
	NewService(nil, &stubCollections{}, &stubItems{})
}
