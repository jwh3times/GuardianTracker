package auth

import (
	"context"
	"fmt"
	"log"
	"time"

	"guardian-tracker/api-service/cache"
)

// UserVersionStore is satisfied by db.UserStore.
type UserVersionStore interface {
	GetTokenVersion(ctx context.Context, membershipID string) (int, error)
}

// RevocationChecker validates JWT token_version against the DB, using a short-lived cache.
type RevocationChecker struct {
	store UserVersionStore // nil = skip check (degraded mode)
	cache cache.Cache
	ttl   time.Duration
}

// NewRevocationChecker creates a RevocationChecker. store may be nil for degraded mode.
func NewRevocationChecker(store UserVersionStore, c cache.Cache) *RevocationChecker {
	return &RevocationChecker{store: store, cache: c, ttl: 60 * time.Second}
}

// Check returns an error if the JWT's token version is stale.
func (r *RevocationChecker) Check(ctx context.Context, membershipID string, claimedVersion int) error {
	if r.store == nil {
		return nil // degraded mode — skip
	}
	cacheKey := "tver:" + membershipID
	if v, ok := r.cache.Get(cacheKey); ok {
		if v.(int) != claimedVersion {
			return fmt.Errorf("token revoked")
		}
		return nil
	}
	dbVersion, err := r.store.GetTokenVersion(ctx, membershipID)
	if err != nil {
		log.Printf("revocation check failed for %s: %v — allowing request", membershipID, err)
		return nil // fail open on DB error
	}
	r.cache.Set(cacheKey, dbVersion, r.ttl)
	if dbVersion != claimedVersion {
		return fmt.Errorf("token revoked")
	}
	return nil
}
