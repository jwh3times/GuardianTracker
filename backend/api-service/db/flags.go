package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
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

// Update patches the mutable fields of a flag. nil fields are left unchanged.
// Returns pgx.ErrNoRows when the key does not exist.
func (s *FlagStore) Update(ctx context.Context, key string, enabled *bool, minTier *int16) (*FeatureFlag, error) {
	var f FeatureFlag
	err := s.pool.QueryRow(ctx,
		`UPDATE feature_flags
		    SET enabled    = COALESCE($2, enabled),
		        min_tier   = COALESCE($3, min_tier),
		        updated_at = now()
		  WHERE key = $1
		RETURNING key, name, description, category, min_tier, enabled, sort_order, updated_at`,
		key, enabled, minTier,
	).Scan(&f.Key, &f.Name, &f.Description, &f.Category,
		&f.MinTier, &f.Enabled, &f.SortOrder, &f.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}
