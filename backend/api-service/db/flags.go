package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FeatureFlag mirrors one row of the feature_flags table.
type FeatureFlag struct {
	Key         string
	Name        string
	Description string
	Category    string
	MinTier     int16
	Enabled     bool
	SortOrder   int16
	UpdatedAt   time.Time
}

// FlagStore handles feature_flags DB operations.
type FlagStore struct{ pool *pgxpool.Pool }

func NewFlagStore(pool *pgxpool.Pool) *FlagStore { return &FlagStore{pool: pool} }

// List returns all feature flags ordered by their seeded display order.
func (s *FlagStore) List(ctx context.Context) ([]FeatureFlag, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT key, name, description, category, min_tier, enabled, sort_order, updated_at
		 FROM feature_flags ORDER BY sort_order, key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FeatureFlag
	for rows.Next() {
		var f FeatureFlag
		if err := rows.Scan(&f.Key, &f.Name, &f.Description, &f.Category,
			&f.MinTier, &f.Enabled, &f.SortOrder, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Get returns a single flag by key, or pgx.ErrNoRows when absent.
func (s *FlagStore) Get(ctx context.Context, key string) (*FeatureFlag, error) {
	var f FeatureFlag
	err := s.pool.QueryRow(ctx,
		`SELECT key, name, description, category, min_tier, enabled, sort_order, updated_at
		 FROM feature_flags WHERE key = $1`, key,
	).Scan(&f.Key, &f.Name, &f.Description, &f.Category,
		&f.MinTier, &f.Enabled, &f.SortOrder, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// Update patches the mutable fields of a flag and records a flag.update audit row
// in the same transaction. nil enabled/minTier are left unchanged. The audit
// details capture only the fields that actually changed (old->new). Returns
// pgx.ErrNoRows when the key does not exist.
func (s *FlagStore) Update(ctx context.Context, key string, enabled *bool, minTier *int16, actorUserID *int64, actorMembershipID string) (*FeatureFlag, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful commit

	var oldEnabled bool
	var oldMinTier int16
	err = tx.QueryRow(ctx,
		`SELECT enabled, min_tier FROM feature_flags WHERE key = $1 FOR UPDATE`, key,
	).Scan(&oldEnabled, &oldMinTier)
	if err != nil {
		return nil, err // pgx.ErrNoRows when missing
	}

	var f FeatureFlag
	if err := tx.QueryRow(ctx,
		`UPDATE feature_flags
		    SET enabled    = COALESCE($2, enabled),
		        min_tier   = COALESCE($3, min_tier),
		        updated_at = now()
		  WHERE key = $1
		RETURNING key, name, description, category, min_tier, enabled, sort_order, updated_at`,
		key, enabled, minTier,
	).Scan(&f.Key, &f.Name, &f.Description, &f.Category,
		&f.MinTier, &f.Enabled, &f.SortOrder, &f.UpdatedAt); err != nil {
		return nil, err
	}

	details := map[string]any{"key": key}
	if enabled != nil && *enabled != oldEnabled {
		details["enabled"] = []any{oldEnabled, *enabled}
	}
	if minTier != nil && *minTier != oldMinTier {
		details["minTier"] = []any{oldMinTier, *minTier}
	}
	if err := insertAudit(ctx, tx, AuditEvent{
		EventType:         "flag.update",
		ActorUserID:       actorUserID,
		ActorMembershipID: actorMembershipID,
		Details:           details,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &f, nil
}
