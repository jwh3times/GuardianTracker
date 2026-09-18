package adapters

import (
	"context"
	"errors"
	"time"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/digest"

	"github.com/jackc/pgx/v5"
)

// digestRepository adapts the internal user-keyed digest store to the
// membership-keyed domain repository (ADR 0023). PostgreSQL rows and internal
// user IDs do not cross this boundary.
type digestRepository struct{ store db.DigestRepo }

// NewDigestRepository wraps the digest store for digest.Service.
func NewDigestRepository(store db.DigestRepo) digest.Repository {
	return &digestRepository{store: store}
}

func (r *digestRepository) Get(ctx context.Context, membershipID string) (digest.Snapshot, bool, error) {
	userID, err := r.store.GetUserID(ctx, membershipID)
	if err != nil {
		return digest.Snapshot{}, false, digestError(err)
	}

	stored, err := r.store.Get(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return digest.Snapshot{}, false, nil
	}
	if err != nil {
		return digest.Snapshot{}, false, digestError(err)
	}
	return digestSnapshot(stored), true, nil
}

func (r *digestRepository) TouchActivity(ctx context.Context, membershipID string, lastActivityAt time.Time, visitStartedAt *time.Time) error {
	userID, err := r.store.GetUserID(ctx, membershipID)
	if err != nil {
		return digestError(err)
	}
	return digestError(r.store.TouchActivity(ctx, userID, lastActivityAt, visitStartedAt))
}

func (r *digestRepository) Save(ctx context.Context, membershipID string, snap digest.Snapshot) error {
	userID, err := r.store.GetUserID(ctx, membershipID)
	if err != nil {
		return digestError(err)
	}
	return digestError(r.store.Save(ctx, userID, db.DigestState{
		LastActivityAt:  snap.LastActivityAt,
		VisitStartedAt:  snap.VisitStartedAt,
		Snapshot:        snap.OwnedItemHashes,
		SnapshotTakenAt: snap.SnapshotTakenAt,
	}))
}

func digestSnapshot(stored *db.DigestState) digest.Snapshot {
	return digest.Snapshot{
		LastActivityAt:  stored.LastActivityAt,
		VisitStartedAt:  stored.VisitStartedAt,
		OwnedItemHashes: stored.Snapshot,
		SnapshotTakenAt: stored.SnapshotTakenAt,
	}
}

func digestError(err error) error {
	if errors.Is(err, db.ErrUnavailable) {
		return digest.ErrUnavailable
	}
	return err
}
