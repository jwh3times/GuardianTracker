package collections

import (
	"context"
	"crypto/sha256"
	"time"

	"guardian-tracker/api-service/services/manifeststate"
	"guardian-tracker/api-service/services/membershipstate"
)

// Three minutes allow the client's four 30-second profile attempts and ten
// seconds of retry sleeps, with time for normal queue/readiness work. A long
// Retry-After or a stalled limiter cannot keep detached work alive indefinitely.
// Context-free Manifest readers still have to return before their goroutine
// exits; the deadline releases waiters and prevents their late publication.
const analysisLoadTimeout = 3 * time.Minute

type analysisFlightKey struct {
	membershipType int
	membershipID   string
	manifest       manifeststate.Attempt
	refresh        membershipstate.Attempt
	credential     [sha256.Size]byte
}

// sharedAnalysis joins or starts the shared load for key. The flight group
// owns waiters, the detached deadline and cancellation; this owner keeps the
// key, the load, and publication.
//
// Publication always acquires generation locks BEFORE the group's mutex: the
// Manifest -> membership publication runs first and calls ifActive inside it,
// and joining or abandoning a flight never acquires a generation lock.
// Cancellation and the actual cache/tree write therefore serialize without
// reversing the Manifest -> membership lock order.
func (m *MembershipAnalysis) sharedAnalysis(ctx context.Context, key analysisFlightKey, accessToken string) (*analysis, error) {
	return m.flights.Do(ctx, key, func(ctx context.Context, ifActive func(func())) (*analysis, error) {
		a, loaded, err := m.loadAnalysis(ctx, key, accessToken, ifActive)
		if err == nil && loaded {
			publishAnalysis(key.manifest, key.refresh, func() {
				ifActive(func() { m.cache.Set(analysisCacheKey(key.membershipType, key.membershipID), a, m.cacheTTL) })
			})
		}
		return a, err
	})
}
