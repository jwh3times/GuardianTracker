package adapters

import (
	"context"
	"errors"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/preferences"

	"github.com/jackc/pgx/v5"
)

// preferencesRepository adapts the internal user-keyed preference store to the
// membership-keyed domain repository. PostgreSQL rows and internal user IDs do
// not cross this boundary.
type preferencesRepository struct{ store db.PrefsRepo }

// NewPreferencesRepository wraps the preference store for preferences.Service.
func NewPreferencesRepository(store db.PrefsRepo) preferences.Repository {
	return &preferencesRepository{store: store}
}

func (r *preferencesRepository) Get(ctx context.Context, membershipID string) (preferences.Values, bool, error) {
	userID, err := r.store.GetUserID(ctx, membershipID)
	if err != nil {
		return preferences.Values{}, false, preferenceError(err)
	}

	stored, err := r.store.Get(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return preferences.Values{}, false, nil
	}
	if err != nil {
		return preferences.Values{}, false, preferenceError(err)
	}
	return preferenceValues(stored), true, nil
}

func (r *preferencesRepository) Apply(ctx context.Context, membershipID string, initial preferences.Values, patch preferences.Patch) (preferences.Values, error) {
	userID, err := r.store.GetUserID(ctx, membershipID)
	if err != nil {
		return preferences.Values{}, preferenceError(err)
	}

	storageInitial := db.PreferenceInitial{
		CardStyle:   string(initial.CardStyle),
		Personalize: initial.Personalize,
	}
	storagePatch := db.PreferencePatch{}
	if patch.CardStyle != nil {
		storagePatch.CardStyle = string(*patch.CardStyle)
		storagePatch.SetCardStyle = true
	}
	if patch.Personalize != nil {
		storagePatch.Personalize = *patch.Personalize
		storagePatch.SetPersonalize = true
	}
	if patch.OnboardingComplete != nil {
		storagePatch.OnboardingComplete = *patch.OnboardingComplete
		storagePatch.SetOnboardingComplete = true
	}

	stored, err := r.store.Apply(ctx, userID, storageInitial, storagePatch)
	if err != nil {
		return preferences.Values{}, preferenceError(err)
	}
	return preferenceValues(stored), nil
}

func preferenceValues(stored *db.UserPreferences) preferences.Values {
	return preferences.Values{
		CardStyle:   preferences.CardStyle(stored.CardStyle),
		Personalize: stored.Personalize,
		OnboardedAt: stored.OnboardedAt,
	}
}

func preferenceError(err error) error {
	if errors.Is(err, db.ErrUnavailable) {
		return preferences.ErrUnavailable
	}
	return err
}
