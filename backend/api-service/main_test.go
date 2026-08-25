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
