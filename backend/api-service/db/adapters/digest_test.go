package adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/digest"

	"github.com/jackc/pgx/v5"
)

type fakeDigestStore struct {
	userID            int64
	resolveMembership string
	resolveErr        error

	getUserID int64
	getState  *db.DigestState
	getErr    error

	touchUserID         int64
	touchLastActivityAt time.Time
	touchVisitStartedAt *time.Time
	touchErr            error

	saveUserID int64
	saveState  db.DigestState
	saveErr    error
}

func (f *fakeDigestStore) GetUserID(_ context.Context, membershipID string) (int64, error) {
	f.resolveMembership = membershipID
	return f.userID, f.resolveErr
}

func (f *fakeDigestStore) Get(_ context.Context, userID int64) (*db.DigestState, error) {
	f.getUserID = userID
	return f.getState, f.getErr
}

func (f *fakeDigestStore) TouchActivity(_ context.Context, userID int64, lastActivityAt time.Time, visitStartedAt *time.Time) error {
	f.touchUserID = userID
	f.touchLastActivityAt = lastActivityAt
	f.touchVisitStartedAt = visitStartedAt
	return f.touchErr
}

func (f *fakeDigestStore) Save(_ context.Context, userID int64, state db.DigestState) error {
	f.saveUserID = userID
	f.saveState = state
	return f.saveErr
}

func TestDigestRepository_GetResolvesMembershipAndProjectsStoredState(t *testing.T) {
	stamp := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	store := &fakeDigestStore{
		userID: 42,
		getState: &db.DigestState{
			UserID:          42,
			LastActivityAt:  stamp,
			VisitStartedAt:  stamp,
			Snapshot:        []uint32{1, 2, 3},
			SnapshotTakenAt: stamp,
		},
	}
	repository := NewDigestRepository(store)

	got, found, err := repository.Get(context.Background(), "membership-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if store.resolveMembership != "membership-1" || store.getUserID != 42 {
		t.Fatalf("membership resolution = (%q, %d), want (membership-1, 42)", store.resolveMembership, store.getUserID)
	}
	if !found {
		t.Fatal("Get reported stored row absent")
	}
	want := digest.Snapshot{
		LastActivityAt:  stamp,
		VisitStartedAt:  stamp,
		OwnedItemHashes: []uint32{1, 2, 3},
		SnapshotTakenAt: stamp,
	}
	if got.LastActivityAt != want.LastActivityAt || got.VisitStartedAt != want.VisitStartedAt || got.SnapshotTakenAt != want.SnapshotTakenAt {
		t.Fatalf("Get = %#v, want %#v", got, want)
	}
	if len(got.OwnedItemHashes) != 3 {
		t.Fatalf("OwnedItemHashes = %v, want [1 2 3]", got.OwnedItemHashes)
	}
}

func TestDigestRepository_GetLeavesFirstVisitToService(t *testing.T) {
	store := &fakeDigestStore{userID: 42, getErr: pgx.ErrNoRows}

	_, found, err := NewDigestRepository(store).Get(context.Background(), "membership-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatal("Get reported a stored row for a fresh account")
	}
}

func TestDigestRepository_TouchActivityResolvesMembershipAndForwardsArgs(t *testing.T) {
	store := &fakeDigestStore{userID: 7}
	repository := NewDigestRepository(store)

	when := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	if err := repository.TouchActivity(context.Background(), "membership-2", when, &when); err != nil {
		t.Fatalf("TouchActivity: %v", err)
	}
	if store.touchUserID != 7 {
		t.Fatalf("touch user ID = %d, want 7", store.touchUserID)
	}
	if !store.touchLastActivityAt.Equal(when) {
		t.Fatalf("touch lastActivityAt = %v, want %v", store.touchLastActivityAt, when)
	}
	if store.touchVisitStartedAt == nil || !store.touchVisitStartedAt.Equal(when) {
		t.Fatalf("touch visitStartedAt = %v, want %v", store.touchVisitStartedAt, when)
	}
}

func TestDigestRepository_SaveResolvesMembershipAndTranslatesState(t *testing.T) {
	store := &fakeDigestStore{userID: 9}
	repository := NewDigestRepository(store)

	stamp := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	err := repository.Save(context.Background(), "membership-3", digest.Snapshot{
		LastActivityAt:  stamp,
		VisitStartedAt:  stamp,
		OwnedItemHashes: []uint32{5, 6},
		SnapshotTakenAt: stamp,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if store.saveUserID != 9 {
		t.Fatalf("save user ID = %d, want 9", store.saveUserID)
	}
	if len(store.saveState.Snapshot) != 2 || store.saveState.Snapshot[0] != 5 {
		t.Fatalf("save snapshot = %v, want [5 6]", store.saveState.Snapshot)
	}
}

func TestDigestRepository_TranslatesUnavailableWithoutLeakingDBSentinel(t *testing.T) {
	operations := map[string]func(*fakeDigestStore) error{
		"resolve on get": func(store *fakeDigestStore) error {
			store.resolveErr = db.ErrUnavailable
			_, _, err := NewDigestRepository(store).Get(context.Background(), "membership")
			return err
		},
		"read": func(store *fakeDigestStore) error {
			store.getErr = db.ErrUnavailable
			_, _, err := NewDigestRepository(store).Get(context.Background(), "membership")
			return err
		},
		"resolve on touch": func(store *fakeDigestStore) error {
			store.resolveErr = db.ErrUnavailable
			return NewDigestRepository(store).TouchActivity(context.Background(), "membership", time.Now(), nil)
		},
		"touch": func(store *fakeDigestStore) error {
			store.touchErr = db.ErrUnavailable
			return NewDigestRepository(store).TouchActivity(context.Background(), "membership", time.Now(), nil)
		},
		"resolve on save": func(store *fakeDigestStore) error {
			store.resolveErr = db.ErrUnavailable
			return NewDigestRepository(store).Save(context.Background(), "membership", digest.Snapshot{})
		},
		"save": func(store *fakeDigestStore) error {
			store.saveErr = db.ErrUnavailable
			return NewDigestRepository(store).Save(context.Background(), "membership", digest.Snapshot{})
		},
	}

	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			err := operation(&fakeDigestStore{})
			if !errors.Is(err, digest.ErrUnavailable) {
				t.Fatalf("error = %v, want digest.ErrUnavailable", err)
			}
			if errors.Is(err, db.ErrUnavailable) {
				t.Fatal("db.ErrUnavailable leaked across the digest seam")
			}
		})
	}
}

func TestDigestRepository_PassesOtherFailuresThrough(t *testing.T) {
	transient := errors.New("connection reset")
	store := &fakeDigestStore{getErr: transient}

	_, _, err := NewDigestRepository(store).Get(context.Background(), "membership")
	if !errors.Is(err, transient) {
		t.Fatalf("Get error = %v, want original error", err)
	}
	if errors.Is(err, digest.ErrUnavailable) {
		t.Fatal("real failure was reported as unavailable persistence")
	}
}
