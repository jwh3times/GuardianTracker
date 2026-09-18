package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DigestState is one user's since-last-visit digest state (ADR 0023): the
// visit clock, the snapshot of owned item hashes it was last computed
// against, and the outcome frozen for the current visit.
type DigestState struct {
	UserID          int64
	LastActivityAt  time.Time
	VisitStartedAt  time.Time
	Snapshot        []uint32
	SnapshotTakenAt time.Time

	// CurrentVisitStatus, CurrentVisitPreviousVisitAt, and
	// CurrentVisitAcquired together are the complete digest outcome computed
	// at this visit's start, persisted so a repeated request — including one
	// after a process restart — reconstructs it without recomputing anything.
	// CurrentVisitAcquired holds raw item hashes, not resolved item facts:
	// display projection is re-resolved on every read, never baked into
	// storage.
	CurrentVisitStatus          string
	CurrentVisitPreviousVisitAt *time.Time
	CurrentVisitAcquired        []uint32
}

// currentVisitResult is the JSON shape of the current_visit_result column.
type currentVisitResult struct {
	Status          string     `json:"status"`
	PreviousVisitAt *time.Time `json:"previousVisitAt,omitempty"`
	Acquired        []uint32   `json:"acquired"`
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
	var snapshotRaw, resultRaw []byte
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, last_activity_at, visit_started_at, snapshot, snapshot_taken_at, current_visit_result
		 FROM digest_state WHERE user_id = $1`, userID,
	).Scan(&d.UserID, &d.LastActivityAt, &d.VisitStartedAt, &snapshotRaw, &d.SnapshotTakenAt, &resultRaw)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(snapshotRaw, &d.Snapshot); err != nil {
		return nil, err
	}
	var result currentVisitResult
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		return nil, err
	}
	d.CurrentVisitStatus = result.Status
	d.CurrentVisitPreviousVisitAt = result.PreviousVisitAt
	d.CurrentVisitAcquired = result.Acquired
	return &d, nil
}

// TouchActivity advances only last_activity_at, for a request that continues
// the existing visit. It touches nothing else — not the snapshot, not
// visit_started_at, not the frozen current-visit result — so it is a no-op
// (zero rows affected, no error) when no row exists yet.
func (s *DigestStore) TouchActivity(ctx context.Context, userID int64, at time.Time) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE digest_state SET last_activity_at = $2 WHERE user_id = $1`,
		userID, at)
	return err
}

// RecordUnavailableVisit starts a new visit whose Bungie read failed or was
// private: it advances last_activity_at and visit_started_at and records the
// frozen "unavailable" outcome (so a same-visit repeat request reads the same
// answer back rather than retrying), but — ADR 0023's privacy gate — never
// touches the snapshot baseline columns.
//
// It is a no-op (zero rows affected, no error) when no row exists yet: the
// row cannot be created without a snapshot value (NOT NULL), so a brand-new
// account whose first-ever read fails leaves no trace at all, and its next
// request retries from scratch.
func (s *DigestStore) RecordUnavailableVisit(ctx context.Context, userID int64, at time.Time, previousVisitAt *time.Time) error {
	raw, err := json.Marshal(currentVisitResult{Status: "unavailable", PreviousVisitAt: previousVisitAt, Acquired: []uint32{}})
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`UPDATE digest_state
		 SET last_activity_at = $2, visit_started_at = $2, current_visit_result = $3::jsonb
		 WHERE user_id = $1`,
		userID, at, string(raw))
	return err
}

// Save atomically replaces the whole row: the visit clock, the snapshot
// baseline, and the frozen current-visit outcome together. Used only when a
// new visit begins and the Bungie read is public and successful.
func (s *DigestStore) Save(ctx context.Context, userID int64, state DigestState) error {
	snapshot := state.Snapshot
	if snapshot == nil {
		snapshot = []uint32{}
	}
	snapshotRaw, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	acquired := state.CurrentVisitAcquired
	if acquired == nil {
		acquired = []uint32{}
	}
	resultRaw, err := json.Marshal(currentVisitResult{
		Status:          state.CurrentVisitStatus,
		PreviousVisitAt: state.CurrentVisitPreviousVisitAt,
		Acquired:        acquired,
	})
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO digest_state (user_id, last_activity_at, visit_started_at, snapshot, snapshot_taken_at, current_visit_result)
		 VALUES ($1, $2, $3, $4::jsonb, $5, $6::jsonb)
		 ON CONFLICT (user_id) DO UPDATE
		     SET last_activity_at     = EXCLUDED.last_activity_at,
		         visit_started_at     = EXCLUDED.visit_started_at,
		         snapshot             = EXCLUDED.snapshot,
		         snapshot_taken_at    = EXCLUDED.snapshot_taken_at,
		         current_visit_result = EXCLUDED.current_visit_result`,
		userID, state.LastActivityAt, state.VisitStartedAt, string(snapshotRaw), state.SnapshotTakenAt, string(resultRaw))
	return err
}
