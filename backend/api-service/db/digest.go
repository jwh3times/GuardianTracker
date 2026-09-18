package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DigestState is one user's since-last-visit digest state (ADR 0023): the
// visit clock plus the snapshot of owned item hashes it was last computed
// against.
type DigestState struct {
	UserID          int64
	LastActivityAt  time.Time
	VisitStartedAt  time.Time
	Snapshot        []uint32
	SnapshotTakenAt time.Time
}

// DigestStore handles digest_state DB operations.
type DigestStore struct{ pool *pgxpool.Pool }

func NewDigestStore(pool *pgxpool.Pool) *DigestStore { return &DigestStore{pool: pool} }

// GetUserID resolves the internal Guardian Tracker user key used by the
// digest_state foreign key, mirroring PrefsStore.GetUserID.
func (s *DigestStore) GetUserID(ctx context.Context, membershipID string) (int64, error) {
	var userID int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM users WHERE membership_id = $1`, membershipID).Scan(&userID)
	return userID, err
}

// Get returns the stored digest state for the user. A missing row remains
// pgx.ErrNoRows so the Digest service, not persistence, decides what a first
// visit looks like.
func (s *DigestStore) Get(ctx context.Context, userID int64) (*DigestState, error) {
	var d DigestState
	var raw []byte
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, last_activity_at, visit_started_at, snapshot, snapshot_taken_at
		 FROM digest_state WHERE user_id = $1`, userID,
	).Scan(&d.UserID, &d.LastActivityAt, &d.VisitStartedAt, &raw, &d.SnapshotTakenAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &d.Snapshot); err != nil {
		return nil, err
	}
	return &d, nil
}

// TouchActivity updates only the visit-clock columns — never the snapshot —
// so it is safe to call after a failed or private Bungie read (ADR 0023's
// privacy gate covers only the snapshot columns). visitStartedAt is nil when
// this request continues the existing visit rather than starting a new one.
//
// It is a no-op (zero rows affected, no error) when no row exists yet: the
// row cannot be created without a snapshot value (NOT NULL), so the first
// visit's write path is always Save, never this one.
func (s *DigestStore) TouchActivity(ctx context.Context, userID int64, lastActivityAt time.Time, visitStartedAt *time.Time) error {
	if visitStartedAt != nil {
		_, err := s.pool.Exec(ctx,
			`UPDATE digest_state SET last_activity_at = $2, visit_started_at = $3 WHERE user_id = $1`,
			userID, lastActivityAt, *visitStartedAt)
		return err
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE digest_state SET last_activity_at = $2 WHERE user_id = $1`,
		userID, lastActivityAt)
	return err
}

// Save atomically replaces the whole row: the visit clock and the snapshot
// together. Used only when a new visit begins and the Bungie read is public
// and successful.
func (s *DigestStore) Save(ctx context.Context, userID int64, state DigestState) error {
	snapshot := state.Snapshot
	if snapshot == nil {
		snapshot = []uint32{}
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO digest_state (user_id, last_activity_at, visit_started_at, snapshot, snapshot_taken_at)
		 VALUES ($1, $2, $3, $4::jsonb, $5)
		 ON CONFLICT (user_id) DO UPDATE
		     SET last_activity_at  = EXCLUDED.last_activity_at,
		         visit_started_at  = EXCLUDED.visit_started_at,
		         snapshot          = EXCLUDED.snapshot,
		         snapshot_taken_at = EXCLUDED.snapshot_taken_at`,
		userID, state.LastActivityAt, state.VisitStartedAt, string(raw), state.SnapshotTakenAt)
	return err
}
