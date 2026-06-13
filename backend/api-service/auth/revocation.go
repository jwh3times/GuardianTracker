package auth

import (
	"context"
	"fmt"
	"log"
	"time"

	"guardian-tracker/api-service/cache"
)

// UserAuthStore is satisfied by db.UserStore. It returns the data the middleware
// needs on every request: the current token_version (for account-wide revocation)
// and role (for tier gating) in one query, plus per-session validity (for
// this-device logout).
type UserAuthStore interface {
	GetAuthInfo(ctx context.Context, membershipID string) (tokenVersion int, role int16, err error)
	SessionExists(ctx context.Context, sessionID string) (bool, error)
}

// authInfo is the cached per-user value: token version + role, refreshed together
// so an admin role change (which evicts the key) re-syncs both at once.
type authInfo struct {
	version int
	role    int
}

// RevocationChecker validates JWT token_version against the DB and resolves the
// user's current role, using a short-lived cache so role changes propagate within
// the cache window with no token churn (TODO 13.2).
type RevocationChecker struct {
	store UserAuthStore // nil = skip check (degraded mode)
	cache cache.Cache
	ttl   time.Duration
}

// NewRevocationChecker creates a RevocationChecker. store may be nil for degraded mode.
func NewRevocationChecker(store UserAuthStore, c cache.Cache) *RevocationChecker {
	return &RevocationChecker{store: store, cache: c, ttl: 60 * time.Second}
}

// Resolve validates the JWT's token_version (account-wide revocation) and the
// session's validity (this-device logout), and returns the user's current role.
// It returns an error only when the token or session has been revoked; on a DB
// error it fails open (request allowed) with the standard role. sessionID may be
// empty for tokens issued before sessions existed — the session check is skipped
// for those (they age out within the access-token lifetime).
func (r *RevocationChecker) Resolve(ctx context.Context, membershipID string, claimedVersion int, sessionID string) (int, error) {
	if r.store == nil {
		return RoleStandard, nil // degraded mode — skip
	}

	// Account-wide token_version + role, cached per user.
	var role int
	if v, ok := r.cache.Get("tver:" + membershipID); ok {
		if ai, ok := v.(authInfo); ok {
			if ai.version != claimedVersion {
				return 0, fmt.Errorf("token revoked")
			}
			role = ai.role
		}
	} else {
		dbVersion, dbRole, err := r.store.GetAuthInfo(ctx, membershipID)
		if err != nil {
			log.Printf("revocation check failed for %s: %v — allowing request", membershipID, err)
			return RoleStandard, nil // fail open on DB error
		}
		r.cache.Set("tver:"+membershipID, authInfo{version: dbVersion, role: int(dbRole)}, r.ttl)
		if dbVersion != claimedVersion {
			return 0, fmt.Errorf("token revoked")
		}
		role = int(dbRole)
	}

	// Per-session validity (this-device logout), cached per session id.
	if sessionID != "" {
		if err := r.checkSession(ctx, sessionID); err != nil {
			return 0, err
		}
	}
	return role, nil
}

// checkSession returns an error when a session has been revoked (logged out or
// reuse-revoked). It fails open on DB errors, consistent with the version check.
func (r *RevocationChecker) checkSession(ctx context.Context, sessionID string) error {
	cacheKey := "sess:" + sessionID
	if v, ok := r.cache.Get(cacheKey); ok {
		if valid, ok := v.(bool); ok {
			if !valid {
				return fmt.Errorf("session revoked")
			}
			return nil
		}
	}
	exists, err := r.store.SessionExists(ctx, sessionID)
	if err != nil {
		log.Printf("session check failed for %s: %v — allowing request", sessionID, err)
		return nil // fail open on DB error
	}
	r.cache.Set(cacheKey, exists, r.ttl)
	if !exists {
		return fmt.Errorf("session revoked")
	}
	return nil
}

// Check returns an error if the JWT's token version is stale. It is a thin wrapper
// over Resolve for callers (e.g. token refresh) that only need account-wide
// revocation; session validity is enforced separately on rotation.
func (r *RevocationChecker) Check(ctx context.Context, membershipID string, claimedVersion int) error {
	_, err := r.Resolve(ctx, membershipID, claimedVersion, "")
	return err
}
