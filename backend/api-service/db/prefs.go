package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserPreferences holds user UI preferences.
type UserPreferences struct {
	UserID      int64
	CardStyle   string
	Personalize bool
	UpdatedAt   time.Time
}

// PrefsStore handles user_preferences DB operations.
type PrefsStore struct{ pool *pgxpool.Pool }

func NewPrefsStore(pool *pgxpool.Pool) *PrefsStore { return &PrefsStore{pool: pool} }

// Get returns preferences for the user, or defaults when no row exists.
func (s *PrefsStore) Get(ctx context.Context, userID int64) (*UserPreferences, error) {
	var p UserPreferences
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, card_style, personalize, updated_at
		 FROM user_preferences WHERE user_id = $1`, userID,
	).Scan(&p.UserID, &p.CardStyle, &p.Personalize, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return &UserPreferences{
			UserID:      userID,
			CardStyle:   "framed",
			Personalize: true,
			UpdatedAt:   time.Now(),
		}, nil
	}
	return &p, err
}

// Upsert inserts or updates user preferences.
func (s *PrefsStore) Upsert(ctx context.Context, userID int64, cardStyle string, personalize bool) (*UserPreferences, error) {
	var p UserPreferences
	err := s.pool.QueryRow(ctx,
		`INSERT INTO user_preferences (user_id, card_style, personalize)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE
		     SET card_style   = EXCLUDED.card_style,
		         personalize  = EXCLUDED.personalize,
		         updated_at   = now()
		 RETURNING user_id, card_style, personalize, updated_at`,
		userID, cardStyle, personalize,
	).Scan(&p.UserID, &p.CardStyle, &p.Personalize, &p.UpdatedAt)
	return &p, err
}
