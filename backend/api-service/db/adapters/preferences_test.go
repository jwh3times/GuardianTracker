package adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/preferences"

	"github.com/jackc/pgx/v5"
)

type fakePreferenceStore struct {
	userID            int64
	resolveMembership string
	resolveErr        error
	values            *db.UserPreferences
	getUserID         int64
	getErr            error
	applyUserID       int64
	applyInitial      db.PreferenceInitial
	applyPatch        db.PreferencePatch
	applyValues       *db.UserPreferences
	applyErr          error
}

func (f *fakePreferenceStore) GetUserID(_ context.Context, membershipID string) (int64, error) {
	f.resolveMembership = membershipID
	return f.userID, f.resolveErr
}

func (f *fakePreferenceStore) Get(_ context.Context, userID int64) (*db.UserPreferences, error) {
	f.getUserID = userID
	return f.values, f.getErr
}

func (f *fakePreferenceStore) Apply(_ context.Context, userID int64, initial db.PreferenceInitial, patch db.PreferencePatch) (*db.UserPreferences, error) {
	f.applyUserID = userID
	f.applyInitial = initial
	f.applyPatch = patch
	return f.applyValues, f.applyErr
}

func TestPreferencesRepository_GetResolvesMembershipAndProjectsStoredValues(t *testing.T) {
	store := &fakePreferenceStore{
		userID: 42,
		values: &db.UserPreferences{
			UserID:      42,
			CardStyle:   "framed",
			Personalize: true,
			UpdatedAt:   time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
		},
	}
	repository := NewPreferencesRepository(store)

	got, found, err := repository.Get(context.Background(), "membership-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if store.resolveMembership != "membership-1" || store.getUserID != 42 {
		t.Fatalf("membership resolution = (%q, %d), want (membership-1, 42)", store.resolveMembership, store.getUserID)
	}
	if !found {
		t.Fatal("Get reported stored row absent")
	}
	if got != preferences.Defaults() {
		t.Fatalf("Get = %#v, want stored values %#v", got, preferences.Defaults())
	}
}

func TestPreferencesRepository_GetLeavesFreshAccountDefaultsToService(t *testing.T) {
	store := &fakePreferenceStore{userID: 42, getErr: pgx.ErrNoRows}

	got, found, err := NewPreferencesRepository(store).Get(context.Background(), "membership-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatal("Get reported a stored row for a fresh account")
	}
	if got != (preferences.Values{}) {
		t.Fatalf("Get = %#v, want zero value so Service supplies defaults", got)
	}
}

func TestPreferencesRepository_ApplyPreservesDomainPatchPresence(t *testing.T) {
	style := preferences.CardStyleCompact
	personalize := false
	complete := true
	stamp := time.Date(2026, 8, 24, 12, 30, 0, 0, time.UTC)
	store := &fakePreferenceStore{
		userID: 7,
		applyValues: &db.UserPreferences{
			UserID:      7,
			CardStyle:   "compact",
			Personalize: false,
			OnboardedAt: &stamp,
		},
	}
	repository := NewPreferencesRepository(store)

	got, err := repository.Apply(context.Background(), "membership-2", preferences.Defaults(), preferences.Patch{
		CardStyle:          &style,
		Personalize:        &personalize,
		OnboardingComplete: &complete,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if store.applyUserID != 7 {
		t.Fatalf("Apply user ID = %d, want 7", store.applyUserID)
	}
	if store.applyInitial.CardStyle != "framed" || !store.applyInitial.Personalize {
		t.Fatalf("initial values = %#v, want service defaults", store.applyInitial)
	}
	if !store.applyPatch.SetCardStyle || store.applyPatch.CardStyle != "compact" {
		t.Fatalf("card style patch = %#v, want present compact", store.applyPatch)
	}
	if !store.applyPatch.SetPersonalize || store.applyPatch.Personalize {
		t.Fatalf("personalize patch = %#v, want present false", store.applyPatch)
	}
	if !store.applyPatch.SetOnboardingComplete || !store.applyPatch.OnboardingComplete {
		t.Fatalf("onboarding patch = %#v, want completion", store.applyPatch)
	}
	if got.CardStyle != style || got.Personalize || got.OnboardedAt == nil || !got.OnboardedAt.Equal(stamp) {
		t.Fatalf("Apply = %#v, want compact/false/completed", got)
	}
}

func TestPreferencesRepository_ApplyCarriesServiceInitialValuesForOmittedFields(t *testing.T) {
	store := &fakePreferenceStore{
		userID: 9,
		applyValues: &db.UserPreferences{
			UserID:      9,
			CardStyle:   "framed",
			Personalize: true,
		},
	}
	repository := NewPreferencesRepository(store)

	_, err := repository.Apply(context.Background(), "membership-3", preferences.Defaults(), preferences.Patch{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if store.applyInitial.CardStyle != "framed" || !store.applyInitial.Personalize {
		t.Fatalf("initial values = %#v, want service defaults", store.applyInitial)
	}
	if store.applyPatch.SetCardStyle || store.applyPatch.SetPersonalize {
		t.Fatalf("patch = %#v, want both fields omitted", store.applyPatch)
	}
}

func TestPreferencesRepository_TranslatesUnavailableWithoutLeakingDBSentinel(t *testing.T) {
	operations := map[string]func(*fakePreferenceStore) error{
		"resolve on get": func(store *fakePreferenceStore) error {
			store.resolveErr = db.ErrUnavailable
			_, _, err := NewPreferencesRepository(store).Get(context.Background(), "membership")
			return err
		},
		"read": func(store *fakePreferenceStore) error {
			store.getErr = db.ErrUnavailable
			_, _, err := NewPreferencesRepository(store).Get(context.Background(), "membership")
			return err
		},
		"resolve on apply": func(store *fakePreferenceStore) error {
			store.resolveErr = db.ErrUnavailable
			_, err := NewPreferencesRepository(store).Apply(context.Background(), "membership", preferences.Defaults(), preferences.Patch{})
			return err
		},
		"apply": func(store *fakePreferenceStore) error {
			store.applyErr = db.ErrUnavailable
			_, err := NewPreferencesRepository(store).Apply(context.Background(), "membership", preferences.Defaults(), preferences.Patch{})
			return err
		},
	}

	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			err := operation(&fakePreferenceStore{})
			if !errors.Is(err, preferences.ErrUnavailable) {
				t.Fatalf("error = %v, want preferences.ErrUnavailable", err)
			}
			if errors.Is(err, db.ErrUnavailable) {
				t.Fatal("db.ErrUnavailable leaked across the preferences seam")
			}
		})
	}
}

func TestPreferencesRepository_PassesOtherFailuresThrough(t *testing.T) {
	transient := errors.New("connection reset")
	store := &fakePreferenceStore{getErr: transient}

	_, _, err := NewPreferencesRepository(store).Get(context.Background(), "membership")
	if !errors.Is(err, transient) {
		t.Fatalf("Get error = %v, want original error", err)
	}
	if errors.Is(err, preferences.ErrUnavailable) {
		t.Fatal("real failure was reported as unavailable persistence")
	}
}
