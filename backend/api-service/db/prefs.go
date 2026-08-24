package db

import (
	"context"
	"time"

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

// PreferenceInitial carries the service-owned values used to create a row.
// Persistence does not define these values; its caller supplies them.
type PreferenceInitial struct {
	CardStyle   string
	Personalize bool
}

// PreferencePatch carries each mutable preference separately from its
// presence. A false Personalize value is therefore distinct from omitting the
// field, which lets one SQL statement update only the fields the caller sent.
type PreferencePatch struct {
	CardStyle             string
	SetCardStyle          bool
	Personalize           bool
	SetPersonalize        bool
	OnboardingComplete    bool
	SetOnboardingComplete bool
}

// PrefsStore handles user_preferences DB operations.
type PrefsStore struct{ pool *pgxpool.Pool }

func NewPrefsStore(pool *pgxpool.Pool) *PrefsStore { return &PrefsStore{pool: pool} }

// GetUserID resolves the internal Guardian Tracker user key used by the
// user_preferences foreign key. Membership-keyed consumers keep this detail
// behind the database adapter.
func (s *PrefsStore) GetUserID(ctx context.Context, membershipID string) (int64, error) {
	var userID int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM users WHERE membership_id = $1`, membershipID).Scan(&userID)
	return userID, err
}

// Get returns stored preferences for the user. A missing row remains
// pgx.ErrNoRows so the Preferences service, not persistence, supplies defaults.
func (s *PrefsStore) Get(ctx context.Context, userID int64) (*UserPreferences, error) {
	var p UserPreferences
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, card_style, personalize, onboarded_at, updated_at
		 FROM user_preferences WHERE user_id = $1`, userID,
	).Scan(&p.UserID, &p.CardStyle, &p.Personalize, &p.OnboardedAt, &p.UpdatedAt)
	return &p, err
}

// Apply atomically applies only the fields present in patch. Onboarding
// completion is server-stamped once and cannot be cleared through this seam.
func (s *PrefsStore) Apply(ctx context.Context, userID int64, initial PreferenceInitial, patch PreferencePatch) (*UserPreferences, error) {
	var p UserPreferences
	err := s.pool.QueryRow(ctx,
		`INSERT INTO user_preferences (user_id, card_style, personalize, onboarded_at)
		 VALUES ($1,
		         CASE WHEN $4::boolean THEN $5::text ELSE $2::text END,
		         CASE WHEN $6::boolean THEN $7::boolean ELSE $3::boolean END,
		         CASE WHEN $8::boolean AND $9::boolean THEN now() ELSE NULL END)
		 ON CONFLICT (user_id) DO UPDATE
		     SET card_style   = CASE WHEN $4::boolean
		                             THEN EXCLUDED.card_style
		                             ELSE user_preferences.card_style
		                        END,
		         personalize  = CASE WHEN $6::boolean
		                             THEN EXCLUDED.personalize
		                             ELSE user_preferences.personalize
		                        END,
		         onboarded_at = CASE WHEN $8::boolean AND $9::boolean
		                              THEN COALESCE(user_preferences.onboarded_at, now())
		                              ELSE user_preferences.onboarded_at
		                         END,
		         updated_at   = now()
		 RETURNING user_id, card_style, personalize, onboarded_at, updated_at`,
		userID,
		initial.CardStyle, initial.Personalize,
		patch.SetCardStyle, patch.CardStyle,
		patch.SetPersonalize, patch.Personalize,
		patch.SetOnboardingComplete, patch.OnboardingComplete,
	).Scan(&p.UserID, &p.CardStyle, &p.Personalize, &p.OnboardedAt, &p.UpdatedAt)
	return &p, err
}
