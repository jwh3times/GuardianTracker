// Package preferences owns Guardian Tracker user preference policy.
package preferences

import (
	"context"
	"errors"
	"time"
)

// CardStyle is the display density used for collection cards.
type CardStyle string

const (
	CardStyleFramed  CardStyle = "framed"
	CardStyleCompact CardStyle = "compact"
)

// Values is the authoritative domain projection of a user's preferences.
type Values struct {
	CardStyle   CardStyle
	Personalize bool
	OnboardedAt *time.Time
}

// Patch represents field presence independently from each field's value.
type Patch struct {
	CardStyle          *CardStyle
	Personalize        *bool
	OnboardingComplete *bool
}

// ReadResult carries both preference values and whether they came from
// persistence. A genuinely new account is persisted; unavailable persistence
// is the sole source of a non-persisted result.
type ReadResult struct {
	Values    Values
	Persisted bool
}

var (
	// ErrUnavailable distinguishes unavailable persistence from another failure.
	ErrUnavailable      = errors.New("preferences persistence unavailable")
	ErrInvalidCardStyle = errors.New("cardStyle must be 'framed' or 'compact'")
	ErrOnboardingReset  = errors.New("onboardingComplete can only be set to true")
)

// Repository is the membership-keyed persistence port used by Service.
type Repository interface {
	Get(ctx context.Context, membershipID string) (values Values, found bool, err error)
	Apply(ctx context.Context, membershipID string, initial Values, patch Patch) (Values, error)
}

// Service owns preference defaults and transitions.
type Service struct {
	repository Repository
}

// NewService constructs a Preferences service around its required repository.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// Defaults returns the domain value for a genuinely new account or degraded
// read: framed cards, personalization enabled, and onboarding incomplete.
func Defaults() Values {
	return Values{CardStyle: CardStyleFramed, Personalize: true}
}

// Get reads authoritative preferences. Unavailable persistence intentionally
// degrades to unstored defaults so the existing GET endpoint remains usable.
func (s *Service) Get(ctx context.Context, membershipID string) (ReadResult, error) {
	values, found, err := s.repository.Get(ctx, membershipID)
	if errors.Is(err, ErrUnavailable) {
		return ReadResult{Values: Defaults(), Persisted: false}, nil
	}
	if err != nil {
		return ReadResult{}, err
	}
	if !found {
		values = Defaults()
	}
	return ReadResult{Values: values, Persisted: true}, nil
}

// Apply validates and delegates one atomic partial patch.
func (s *Service) Apply(ctx context.Context, membershipID string, patch Patch) (Values, error) {
	if patch.CardStyle != nil && *patch.CardStyle != CardStyleFramed && *patch.CardStyle != CardStyleCompact {
		return Values{}, ErrInvalidCardStyle
	}
	if patch.OnboardingComplete != nil && !*patch.OnboardingComplete {
		return Values{}, ErrOnboardingReset
	}
	return s.repository.Apply(ctx, membershipID, Defaults(), patch)
}
