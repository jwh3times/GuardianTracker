package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// User represents a Guardian Tracker user record.
type User struct {
	ID             int64
	MembershipID   string
	MembershipType int16
	DisplayName    string
	TokenVersion   int
	CreatedAt      time.Time
	LastLoginAt    time.Time
}

// UserStore handles user DB operations.
type UserStore struct{ pool *pgxpool.Pool }

func NewUserStore(pool *pgxpool.Pool) *UserStore { return &UserStore{pool: pool} }

// Upsert inserts or updates the user record and returns current id + token_version.
func (s *UserStore) Upsert(ctx context.Context, membershipID string, membershipType int16, displayName string) (id int64, tokenVersion int, err error) {
	err = s.pool.QueryRow(ctx, `
		INSERT INTO users (membership_id, membership_type, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (membership_id) DO UPDATE
			SET display_name = EXCLUDED.display_name,
				last_login_at = now()
		RETURNING id, token_version`,
		membershipID, membershipType, displayName,
	).Scan(&id, &tokenVersion)
	return
}

// GetTokenVersion returns the current token_version for revocation checks.
func (s *UserStore) GetTokenVersion(ctx context.Context, membershipID string) (int, error) {
	var v int
	err := s.pool.QueryRow(ctx, `SELECT token_version FROM users WHERE membership_id = $1`, membershipID).Scan(&v)
	return v, err
}

// BumpTokenVersion increments token_version to invalidate all issued JWTs.
func (s *UserStore) BumpTokenVersion(ctx context.Context, membershipID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET token_version = token_version + 1 WHERE membership_id = $1`, membershipID)
	return err
}
