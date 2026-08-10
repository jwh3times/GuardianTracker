package db

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// countingUsers records DeleteExpiredSessions calls and fails the first one, so
// a loop that exits on error is distinguishable from one that keeps going.
type countingUsers struct {
	UserRepo // never called; only DeleteExpiredSessions is exercised

	mu    sync.Mutex
	calls int
	got   chan struct{}
}

func (c *countingUsers) DeleteExpiredSessions(context.Context) (int64, error) {
	c.mu.Lock()
	c.calls++
	n := c.calls
	c.mu.Unlock()

	select {
	case c.got <- struct{}{}:
	default:
	}

	if n == 1 {
		return 0, errors.New("connection reset")
	}
	return 3, nil
}

type countingAudit struct {
	AuditRepo

	mu      sync.Mutex
	cutoffs []time.Time
	got     chan struct{}
}

func (c *countingAudit) DeleteOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	c.mu.Lock()
	c.cutoffs = append(c.cutoffs, cutoff)
	n := len(c.cutoffs)
	c.mu.Unlock()

	select {
	case c.got <- struct{}{}:
	default:
	}

	if n == 1 {
		return 0, errors.New("deadlock detected")
	}
	return 7, nil
}

func waitForTicks(t *testing.T, got chan struct{}, n int) {
	t.Helper()
	for i := range n {
		select {
		case <-got:
		case <-time.After(2 * time.Second):
			t.Fatalf("pruner stopped after %d sweeps; it must survive a failed one", i)
		}
	}
}

// TestStartSessionPruner_SurvivesAFailedSweep pins the invariant that makes
// these loops worth having. A `return` where the code says `continue` ends
// pruning for the lifetime of the process, and nothing else would notice:
// sessions simply accumulate.
func TestStartSessionPruner_SurvivesAFailedSweep(t *testing.T) {
	users := &countingUsers{got: make(chan struct{}, 8)}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	StartSessionPruner(ctx, users, time.Millisecond)
	waitForTicks(t, users.got, 3)
}

func TestStartAuditPruner_SurvivesAFailedSweepAndUsesTheRetentionWindow(t *testing.T) {
	audit := &countingAudit{got: make(chan struct{}, 8)}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	const retentionDays = 30
	before := time.Now().AddDate(0, 0, -retentionDays)
	StartAuditPruner(ctx, audit, retentionDays, time.Millisecond)
	waitForTicks(t, audit.got, 3)
	after := time.Now().AddDate(0, 0, -retentionDays)

	audit.mu.Lock()
	cutoffs := append([]time.Time(nil), audit.cutoffs...)
	audit.mu.Unlock()

	for i, c := range cutoffs {
		if c.Before(before.Add(-time.Second)) || c.After(after.Add(time.Second)) {
			t.Errorf("cutoff %d = %v, want ~%d days before now", i, c, retentionDays)
		}
	}
}

// TestPruners_StopOnContextCancel guards the shutdown path: a pruner that
// ignores ctx keeps a goroutine and a database connection alive past shutdown.
func TestPruners_StopOnContextCancel(t *testing.T) {
	users := &countingUsers{got: make(chan struct{}, 8)}
	ctx, cancel := context.WithCancel(t.Context())

	StartSessionPruner(ctx, users, time.Millisecond)
	waitForTicks(t, users.got, 2)
	cancel()

	// Drain anything already in flight, then require quiet.
	time.Sleep(20 * time.Millisecond)
	users.mu.Lock()
	settled := users.calls
	users.mu.Unlock()

	time.Sleep(50 * time.Millisecond)
	users.mu.Lock()
	final := users.calls
	users.mu.Unlock()

	if final != settled {
		t.Fatalf("pruner ran %d more sweeps after cancel; it must stop at shutdown", final-settled)
	}
}
