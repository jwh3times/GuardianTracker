package preferences

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type stubRepository struct {
	getValues       Values
	getFound        bool
	getErr          error
	applyValues     Values
	applyErr        error
	applyCalls      int
	applyMembership string
	applyInitial    Values
	applyPatch      Patch
}

func (s *stubRepository) Get(context.Context, string) (Values, bool, error) {
	return s.getValues, s.getFound, s.getErr
}

func (s *stubRepository) Apply(_ context.Context, membershipID string, initial Values, patch Patch) (Values, error) {
	s.applyCalls++
	s.applyMembership = membershipID
	s.applyInitial = initial
	s.applyPatch = patch
	return s.applyValues, s.applyErr
}

func TestGetUnavailableReturnsUnstoredDefaults(t *testing.T) {
	service := NewService(&stubRepository{getErr: fmt.Errorf("read: %w", ErrUnavailable)})

	got, err := service.Get(context.Background(), "membership-1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Persisted {
		t.Fatal("unavailable read reported persisted provenance")
	}
	if got.Values.CardStyle != CardStyleFramed {
		t.Errorf("CardStyle = %q, want %q", got.Values.CardStyle, CardStyleFramed)
	}
	if !got.Values.Personalize {
		t.Error("Personalize = false, want true")
	}
	if got.Values.OnboardedAt != nil {
		t.Errorf("OnboardedAt = %v, want nil", got.Values.OnboardedAt)
	}
}

func TestGetFreshAccountDefaultsAreAuthoritative(t *testing.T) {
	service := NewService(&stubRepository{})

	got, err := service.Get(context.Background(), "membership-1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !got.Persisted {
		t.Fatal("successful fresh-account read reported non-persisted provenance")
	}
	if got.Values != Defaults() {
		t.Errorf("Values = %#v, want %#v", got.Values, Defaults())
	}
}

func TestApplyDelegatesOneAtomicFieldPresencePatch(t *testing.T) {
	style := CardStyleCompact
	personalize := false
	repository := &stubRepository{
		applyValues: Values{CardStyle: style, Personalize: personalize},
	}
	service := NewService(repository)

	got, err := service.Apply(context.Background(), "membership-1", Patch{
		CardStyle:   &style,
		Personalize: &personalize,
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if repository.applyCalls != 1 {
		t.Fatalf("repository Apply calls = %d, want 1", repository.applyCalls)
	}
	if repository.applyMembership != "membership-1" {
		t.Errorf("repository membership = %q, want membership-1", repository.applyMembership)
	}
	if repository.applyInitial != Defaults() {
		t.Errorf("repository initial values = %#v, want service defaults %#v", repository.applyInitial, Defaults())
	}
	if repository.applyPatch.CardStyle != &style || repository.applyPatch.Personalize != &personalize {
		t.Errorf("repository patch = %#v, want supplied field-presence patch", repository.applyPatch)
	}
	if got != repository.applyValues {
		t.Errorf("Values = %#v, want %#v", got, repository.applyValues)
	}
}

func TestApplyRejectsInvalidCardStyleWithoutWriting(t *testing.T) {
	invalid := CardStyle("giant")
	repository := &stubRepository{}
	service := NewService(repository)

	_, err := service.Apply(context.Background(), "membership-1", Patch{CardStyle: &invalid})
	if !errors.Is(err, ErrInvalidCardStyle) {
		t.Fatalf("Apply error = %v, want ErrInvalidCardStyle", err)
	}
	if repository.applyCalls != 0 {
		t.Fatalf("repository Apply calls = %d, want 0", repository.applyCalls)
	}
}

func TestApplyRejectsOnboardingResetWithoutWriting(t *testing.T) {
	reset := false
	repository := &stubRepository{}
	service := NewService(repository)

	_, err := service.Apply(context.Background(), "membership-1", Patch{OnboardingComplete: &reset})
	if !errors.Is(err, ErrOnboardingReset) {
		t.Fatalf("Apply error = %v, want ErrOnboardingReset", err)
	}
	if repository.applyCalls != 0 {
		t.Fatalf("repository Apply calls = %d, want 0", repository.applyCalls)
	}
}

func TestApplyOnboardingCompletionUsesRepositoryStamp(t *testing.T) {
	complete := true
	stamp := time.Date(2026, time.August, 24, 15, 30, 0, 0, time.UTC)
	repository := &stubRepository{
		applyValues: Values{CardStyle: CardStyleFramed, Personalize: true, OnboardedAt: &stamp},
	}
	service := NewService(repository)

	got, err := service.Apply(context.Background(), "membership-1", Patch{OnboardingComplete: &complete})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if repository.applyPatch.OnboardingComplete == nil || !*repository.applyPatch.OnboardingComplete {
		t.Errorf("OnboardingComplete patch = %v, want present true", repository.applyPatch.OnboardingComplete)
	}
	if got.OnboardedAt != &stamp {
		t.Errorf("OnboardedAt = %v, want repository-stamped %v", got.OnboardedAt, stamp)
	}
}

func TestApplyUnavailableRemainsTypedFailure(t *testing.T) {
	repository := &stubRepository{applyErr: fmt.Errorf("write: %w", ErrUnavailable)}
	service := NewService(repository)

	_, err := service.Apply(context.Background(), "membership-1", Patch{})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Apply error = %v, want ErrUnavailable", err)
	}
}
