package records

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/membershipstate"
)

// GetRecords permits four 30-second attempts and ten seconds of retry sleeps.
// Three minutes leaves queue margin but bounds longer Retry-After/limiter waits.
// A stalled context-insensitive dependency may return later; deadline/canceled
// waiters detach immediately and that abandoned flight cannot publish.
const profileLoadTimeout = 3 * time.Minute

type profileFlightKey struct {
	membershipType int
	membershipID   string
	refresh        membershipstate.Attempt
	credential     [sha256.Size]byte
}

// getProfileRecords shares overlapping misses while preserving the original
// cache TTL and the actual Bungie fetch time for every derived Records view.
func (s *Service) getProfileRecords(ctx context.Context, membershipType int, membershipID, bungieToken string) (*bungie.RecordsProfileResponse, time.Time, error) {
	if err := ctx.Err(); err != nil {
		return nil, time.Time{}, err
	}
	key := profileFlightKey{
		membershipType: membershipType, membershipID: membershipID,
		refresh: s.refresh.Begin(membershipType, membershipID),
	}
	if r, ok := s.cachedProfile(ctx, recordsCacheKey(membershipType, membershipID)); ok {
		return r.resp, r.fetchedAt, nil
	}
	key.credential = sha256.Sum256([]byte(bungieToken))
	r, err := s.shareProfile(ctx, key, bungieToken)
	if err != nil {
		return nil, time.Time{}, err
	}
	return r.resp, r.fetchedAt, nil
}

func (s *Service) cachedProfile(ctx context.Context, key string) (*cachedRecords, bool) {
	if s.cache != nil {
		if value, ok := s.cache.Get(key); ok {
			if r, ok := value.(*cachedRecords); ok {
				return r, true
			}
			// Preserve the load helper's wrong-type warning without logging the
			// membership-bearing cache key or any credential/response content.
			observability.Logger(ctx).WarnContext(ctx, "cached value has the wrong type; treating as a miss",
				"want_type", fmt.Sprintf("%T", (*cachedRecords)(nil)), "cached_type", fmt.Sprintf("%T", value))
		}
	}
	return nil, false
}

// shareProfile joins or starts the shared component-900 load for key. The
// flight group owns waiters, the detached deadline and cancellation; this
// owner keeps the key, the cache re-check, the fetch, and publication.
func (s *Service) shareProfile(ctx context.Context, key profileFlightKey, token string) (*cachedRecords, error) {
	return s.profileFlights.Do(ctx, key, func(ctx context.Context, ifActive func(func())) (*cachedRecords, error) {
		cacheKey := recordsCacheKey(key.membershipType, key.membershipID)
		// A previous flight may have populated the entry after the caller's miss.
		// Reusing it must not extend TTL or restamp the original fetch time.
		if r, cached := s.cachedProfile(ctx, cacheKey); cached {
			return r, nil
		}
		resp, err := s.bungie.GetRecords(ctx, key.membershipType, key.membershipID, token)
		if err != nil {
			return nil, err
		}
		r := &cachedRecords{resp: resp, fetchedAt: time.Now().UTC()}
		if s.cache != nil {
			// Lock order: membership publication -> flight group -> cache.
			// Joining or leaving never calls a publication while holding the
			// group's mutex. The actual write stays under both guards, so
			// refresh and abandonment cannot race a detached worker back into
			// the reusable cache.
			key.refresh.Publish(func() {
				ifActive(func() { s.cache.Set(cacheKey, r, s.ttl) })
			})
		}
		return r, nil
	})
}
