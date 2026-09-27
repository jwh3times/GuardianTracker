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

// profileMu protects mutable flight state and serializes abandonment with cache
// publication. There is no Manifest generation: component 900 is raw profile
// data, so a Manifest swap must neither split this work nor retire its result.
type profileFlight struct {
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	waiters  int
	finished bool
	result   *cachedRecords
	err      error
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

func (s *Service) shareProfile(ctx context.Context, key profileFlightKey, token string) (*cachedRecords, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.profileMu.Lock()
	f := s.profileFlights[key]
	if f == nil || f.ctx.Err() != nil {
		shared, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.profileTimeout)
		f = &profileFlight{ctx: shared, cancel: cancel, done: make(chan struct{})}
		if s.profileFlights == nil {
			s.profileFlights = make(map[profileFlightKey]*profileFlight)
		}
		s.profileFlights[key] = f
		go s.loadProfile(key, f, token)
	}
	f.waiters++
	s.profileMu.Unlock()

	select {
	case <-ctx.Done():
	case <-f.ctx.Done():
	case <-f.done:
	}

	s.profileMu.Lock()
	defer s.profileMu.Unlock()
	err := ctx.Err()
	if f.finished {
		if err != nil {
			return nil, err
		}
		return f.result, f.err
	}
	if err == nil {
		err = f.ctx.Err()
	}
	f.waiters--
	if f.waiters == 0 {
		if s.profileFlights[key] == f {
			delete(s.profileFlights, key)
		}
		f.cancel()
	}
	return nil, err
}

func (s *Service) loadProfile(key profileFlightKey, f *profileFlight, token string) {
	cacheKey := recordsCacheKey(key.membershipType, key.membershipID)
	// A previous flight may have populated the entry after the caller's miss.
	// Reusing it must not extend TTL or restamp the original fetch time.
	r, cached := s.cachedProfile(f.ctx, cacheKey)
	var err error
	if !cached {
		var resp *bungie.RecordsProfileResponse
		resp, err = s.bungie.GetRecords(f.ctx, key.membershipType, key.membershipID, token)
		if err == nil {
			r = &cachedRecords{resp: resp, fetchedAt: time.Now().UTC()}
		}
	}
	if err == nil && !cached && s.cache != nil {
		// Lock order: membership publication -> profileMu -> cache. Joining or
		// leaving never calls a publication while holding profileMu. The actual
		// write stays under both guards, so refresh and abandonment cannot race
		// a detached worker back into the reusable cache.
		key.refresh.Publish(func() {
			s.profileMu.Lock()
			defer s.profileMu.Unlock()
			if s.profileFlights[key] == f && f.waiters > 0 && f.ctx.Err() == nil {
				s.cache.Set(cacheKey, r, s.ttl)
			}
		})
	}

	s.profileMu.Lock()
	if f.ctx.Err() != nil {
		r, err = nil, f.ctx.Err()
	}
	f.result, f.err, f.finished = r, err, true
	if s.profileFlights[key] == f {
		delete(s.profileFlights, key)
	}
	close(f.done)
	f.cancel()
	s.profileMu.Unlock()
}
