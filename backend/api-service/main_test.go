package main

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/db/adapters"
)

type stubTokenRepo struct{}

func (*stubTokenRepo) Get(context.Context, string) (*auth.EncryptedTokenRecord, error) {
	return nil, auth.ErrTokensNotFound
}

func (*stubTokenRepo) Upsert(context.Context, string, *auth.EncryptedTokenRecord, time.Time) (time.Time, bool, error) {
	return time.Time{}, true, nil
}

func (*stubTokenRepo) Delete(context.Context, string) error { return nil }

func testTokenCipher(t *testing.T) *auth.TokenCipher {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	cipher, err := auth.NewTokenCipher(key, 1, "", 0)
	if err != nil {
		t.Fatalf("NewTokenCipher: %v", err)
	}
	return cipher
}

func TestTokenPersistenceDeps_DegradedStoresSelectMemoryOnlyMode(t *testing.T) {
	stores := db.NewStores(nil)
	// This is the exact non-nil degraded adapter main used to pass through.
	candidateRepo := adapters.NewTokenRepo(stores.Tokens)
	candidateCipher := testTokenCipher(t)

	repo, cipher := tokenPersistenceDeps(stores.Available(), candidateRepo, candidateCipher)
	if repo != nil || cipher != nil {
		t.Fatalf("degraded persistence = (%T, %v), want (nil, nil)", repo, cipher)
	}
}

func TestTokenPersistenceDeps_RealDatabaseRetainsEncryptedPair(t *testing.T) {
	candidateRepo := &stubTokenRepo{}
	candidateCipher := testTokenCipher(t)

	repo, cipher := tokenPersistenceDeps(true, candidateRepo, candidateCipher)
	if repo != candidateRepo || cipher != candidateCipher {
		t.Fatalf("configured persistence pair was not retained")
	}
}

func TestTokenPersistenceDeps_MissingCipherDisablesPersistencePair(t *testing.T) {
	repo, cipher := tokenPersistenceDeps(true, &stubTokenRepo{}, nil)
	if repo != nil || cipher != nil {
		t.Fatalf("unencrypted persistence = (%T, %v), want (nil, nil)", repo, cipher)
	}
}

// namedObserver identifies itself so a test can assert notification order
// without constructing the real services.
type namedObserver struct{ name string }

func (namedObserver) OnVersionChanged(string) error { return nil }

// ADR 0018: Collections pairs the Items catalog with its own presentation-tree
// analysis, so Items must be told about a new manifest first. Getting this
// backwards is silent — the mixture it produces looks like a valid result — so
// the order is pinned here rather than left to the order of the registration
// calls.
func TestManifestObservers_ItemsAdvanceBeforeCollections(t *testing.T) {
	set := manifestObservers{
		Records:     namedObserver{"records"},
		Weekly:      namedObserver{"weekly"},
		Items:       namedObserver{"items"},
		Collections: namedObserver{"collections"},
		Search:      namedObserver{"search"},
		Efficiency:  namedObserver{"efficiency"},
	}

	order := set.inNotificationOrder()

	position := map[string]int{}
	for i, o := range order {
		observer, ok := o.(namedObserver)
		if !ok {
			t.Fatalf("observer %d is not registered: %#v", i, o)
		}
		position[observer.name] = i
	}
	if len(position) != 6 {
		t.Fatalf("notification order = %v, want all six observers exactly once", position)
	}
	if position["items"] >= position["collections"] {
		t.Errorf("items notified at %d, collections at %d; Items must advance first",
			position["items"], position["collections"])
	}
}
