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
	OnboardedAt *time.Time
	UpdatedAt   time.Time
}

// PrefsStore handles user_preferences DB operations.
type PrefsStore struct{ pool *pgxpool.Pool }

func NewPrefsStore(pool *pgxpool.Pool) *PrefsStore { return &PrefsStore{pool: pool} }

// Get returns preferences for the user, or defaults when no row exists.
func (s *PrefsStore) Get(ctx context.Context, userID int64) (*UserPreferences, error) {
	var p UserPreferences
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, card_style, personalize, onboarded_at, updated_at
		 FROM user_preferences WHERE user_id = $1`, userID,
	).Scan(&p.UserID, &p.CardStyle, &p.Personalize, &p.OnboardedAt, &p.UpdatedAt)
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

// Upsert inserts or updates user preferences. completeOnboarding stamps
// onboarded_at using the database clock only when it is currently NULL.
func (s *PrefsStore) Upsert(ctx context.Context, userID int64, cardStyle string, personalize, completeOnboarding bool) (*UserPreferences, error) {
	var p UserPreferences
	err := s.pool.QueryRow(ctx,
		`INSERT INTO user_preferences (user_id, card_style, personalize, onboarded_at)
		 VALUES ($1, $2, $3, CASE WHEN $4 THEN now() ELSE NULL END)
		 ON CONFLICT (user_id) DO UPDATE
		     SET card_style   = EXCLUDED.card_style,
		         personalize  = EXCLUDED.personalize,
		         onboarded_at = CASE WHEN $4
		                              THEN COALESCE(user_preferences.onboarded_at, now())
		                              ELSE user_preferences.onboarded_at
		                         END,
		         updated_at   = now()
		 RETURNING user_id, card_style, personalize, onboarded_at, updated_at`,
		userID, cardStyle, personalize, completeOnboarding,
	).Scan(&p.UserID, &p.CardStyle, &p.Personalize, &p.OnboardedAt, &p.UpdatedAt)
	return &p, err
}
