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

// Every field except ctx/cancel/done is protected by MembershipAnalysis.flightsMu.
type analysisFlight struct {
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	waiters  int
	finished bool
	result   *analysis
	err      error
}

func (m *MembershipAnalysis) sharedAnalysis(ctx context.Context, key analysisFlightKey, accessToken string) (*analysis, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.flightsMu.Lock()
	f := m.flights[key]
	if f == nil || f.ctx.Err() != nil {
		shared, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.loadTimeout)
		f = &analysisFlight{ctx: shared, cancel: cancel, done: make(chan struct{})}
		if m.flights == nil {
			m.flights = make(map[analysisFlightKey]*analysisFlight)
		}
		m.flights[key] = f
		go m.runAnalysis(key, f, accessToken)
	}
	f.waiters++
	m.flightsMu.Unlock()

	select {
	case <-ctx.Done():
	case <-f.ctx.Done():
	case <-f.done:
	}

	m.flightsMu.Lock()
	defer m.flightsMu.Unlock()
	// Caller cancellation is independent even if completion became ready at
	// the same time. It never cancels work still needed by another waiter.
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
		if m.flights[key] == f {
			delete(m.flights, key)
		}
		f.cancel()
	}
	return nil, err
}

func (m *MembershipAnalysis) runAnalysis(key analysisFlightKey, f *analysisFlight, accessToken string) {
	// Publication always acquires generation locks BEFORE flightsMu. Neither
	// joining nor abandoning a flight holds flightsMu while acquiring a
	// generation lock. Cancellation and the actual cache/tree write therefore
	// serialize without reversing the Manifest -> membership lock order.
	ifActive := func(commit func()) {
		m.flightsMu.Lock()
		defer m.flightsMu.Unlock()
		if m.flights[key] == f && f.waiters > 0 && f.ctx.Err() == nil {
			commit()
		}
	}
	a, loaded, err := m.loadAnalysis(f.ctx, key, accessToken, ifActive)
	if err == nil && loaded {
		publishAnalysis(key.manifest, key.refresh, func() {
			ifActive(func() { m.cache.Set(analysisCacheKey(key.membershipType, key.membershipID), a, m.cacheTTL) })
		})
	}

	m.flightsMu.Lock()
	if f.ctx.Err() != nil {
		a, err = nil, f.ctx.Err()
	}
	f.result, f.err, f.finished = a, err, true
	if m.flights[key] == f {
		delete(m.flights, key)
	}
	close(f.done)
	f.cancel()
	m.flightsMu.Unlock()
}
