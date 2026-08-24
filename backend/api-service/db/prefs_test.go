package db

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

func preferenceInitialForTest() PreferenceInitial {
	return PreferenceInitial{CardStyle: "framed", Personalize: true}
}

func TestPrefsStore_GetReportsMissingRow(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)

	_, err := NewPrefsStore(pool).Get(context.Background(), userID)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Get error = %v, want pgx.ErrNoRows", err)
	}
}

func TestPrefsStore_ApplyUsesCallerInitialValuesForFreshAccount(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewPrefsStore(pool)

	got, err := store.Apply(context.Background(), userID, preferenceInitialForTest(), PreferencePatch{
		CardStyle:    "compact",
		SetCardStyle: true,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got.CardStyle != "compact" || !got.Personalize {
		t.Fatalf("Apply = %+v, want compact with caller-supplied initial personalize=true", got)
	}
}

func TestPrefsStore_ApplyPreservesOmittedFields(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewPrefsStore(pool)
	ctx := context.Background()

	_, err := store.Apply(ctx, userID, preferenceInitialForTest(), PreferencePatch{
		CardStyle:      "compact",
		SetCardStyle:   true,
		Personalize:    false,
		SetPersonalize: true,
	})
	if err != nil {
		t.Fatalf("seed preferences: %v", err)
	}

	got, err := store.Apply(ctx, userID, preferenceInitialForTest(), PreferencePatch{
		CardStyle:    "framed",
		SetCardStyle: true,
	})
	if err != nil {
		t.Fatalf("apply card style: %v", err)
	}
	if got.CardStyle != "framed" || got.Personalize {
		t.Fatalf("Apply = %+v, want framed with personalize preserved as false", got)
	}
}

func TestPrefsStore_ConcurrentDisjointPatchesDoNotRestoreStaleValues(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewPrefsStore(pool)
	ctx := context.Background()

	for range 10 {
		_, err := store.Apply(ctx, userID, preferenceInitialForTest(), PreferencePatch{
			CardStyle:      "compact",
			SetCardStyle:   true,
			Personalize:    true,
			SetPersonalize: true,
		})
		if err != nil {
			t.Fatalf("seed preferences: %v", err)
		}

		start := make(chan struct{})
		errs := make(chan error, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, err := store.Apply(ctx, userID, preferenceInitialForTest(), PreferencePatch{
				CardStyle:    "framed",
				SetCardStyle: true,
			})
			errs <- err
		}()
		go func() {
			defer wg.Done()
			<-start
			_, err := store.Apply(ctx, userID, preferenceInitialForTest(), PreferencePatch{
				Personalize:    false,
				SetPersonalize: true,
			})
			errs <- err
		}()

		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatalf("concurrent Apply: %v", err)
			}
		}

		got, err := store.Get(ctx, userID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.CardStyle != "framed" || got.Personalize {
			t.Fatalf("concurrent disjoint patches = %+v, want framed/false", got)
		}
	}
}

func TestPrefsStore_OnboardingCompletionIsServerStampedAndIdempotent(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewPrefsStore(pool)
	ctx := context.Background()

	_, err := store.Apply(ctx, userID, preferenceInitialForTest(), PreferencePatch{
		CardStyle:      "compact",
		SetCardStyle:   true,
		Personalize:    false,
		SetPersonalize: true,
	})
	if err != nil {
		t.Fatalf("seed preferences: %v", err)
	}

	completed, err := store.Apply(ctx, userID, preferenceInitialForTest(), PreferencePatch{
		OnboardingComplete:    true,
		SetOnboardingComplete: true,
	})
	if err != nil {
		t.Fatalf("complete onboarding: %v", err)
	}
	if completed.OnboardedAt == nil {
		t.Fatal("onboarded_at was not stamped")
	}
	if completed.CardStyle != "compact" || completed.Personalize {
		t.Fatalf("completion changed preferences: %+v", completed)
	}
	firstStamp := *completed.OnboardedAt

	again, err := store.Apply(ctx, userID, preferenceInitialForTest(), PreferencePatch{
		OnboardingComplete:    true,
		SetOnboardingComplete: true,
	})
	if err != nil {
		t.Fatalf("repeat completion: %v", err)
	}
	if again.OnboardedAt == nil || !again.OnboardedAt.Equal(firstStamp) {
		t.Fatalf("repeat completion changed onboarded_at: first=%v again=%v", firstStamp, again.OnboardedAt)
	}

	updated, err := store.Apply(ctx, userID, preferenceInitialForTest(), PreferencePatch{
		CardStyle:    "framed",
		SetCardStyle: true,
	})
	if err != nil {
		t.Fatalf("ordinary preference update: %v", err)
	}
	if updated.OnboardedAt == nil || !updated.OnboardedAt.Equal(firstStamp) {
		t.Fatalf("ordinary update changed onboarded_at: first=%v updated=%v", firstStamp, updated.OnboardedAt)
	}
}
