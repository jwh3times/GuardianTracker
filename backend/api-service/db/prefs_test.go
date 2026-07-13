package db

import (
	"context"
	"testing"
)

func TestPrefsStore_OnboardingCompletionIsServerStampedAndIdempotent(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewPrefsStore(pool)
	ctx := context.Background()

	defaults, err := store.Get(ctx, userID)
	if err != nil {
		t.Fatalf("Get defaults: %v", err)
	}
	if defaults.CardStyle != "framed" || !defaults.Personalize || defaults.OnboardedAt != nil {
		t.Fatalf("defaults = %+v, want framed/personalized/not onboarded", defaults)
	}

	completed, err := store.Upsert(ctx, userID, "compact", false, true)
	if err != nil {
		t.Fatalf("complete onboarding: %v", err)
	}
	if completed.OnboardedAt == nil {
		t.Fatal("onboarded_at was not stamped")
	}
	firstStamp := *completed.OnboardedAt

	again, err := store.Upsert(ctx, userID, "framed", true, true)
	if err != nil {
		t.Fatalf("repeat completion: %v", err)
	}
	if again.OnboardedAt == nil || !again.OnboardedAt.Equal(firstStamp) {
		t.Fatalf("repeat completion changed onboarded_at: first=%v again=%v", firstStamp, again.OnboardedAt)
	}

	updated, err := store.Upsert(ctx, userID, "compact", false, false)
	if err != nil {
		t.Fatalf("ordinary preference update: %v", err)
	}
	if updated.OnboardedAt == nil || !updated.OnboardedAt.Equal(firstStamp) {
		t.Fatalf("ordinary update changed onboarded_at: first=%v updated=%v", firstStamp, updated.OnboardedAt)
	}
}
