// Package sharedread owns the lifetime of a shared read: concurrent misses
// for the same key join one detached load instead of each starting their own.
//
// A Group owns the waiters, the detached deadline, cancellation, replacement
// safety and the "still active" publication guard. Its owner keeps everything
// else: the key, what a load reads, cache policy, and generation/publication
// decisions (ADRs 0013, 0014, 0018). The Group holds no cache and never
// publishes anything itself.
//
// Lock order. The Group never holds its mutex while calling a load or any owner
// code other than a commit passed to ifActive. An owner that publishes under
// its own generation locks takes those locks first and calls ifActive inside
// them, so the order is always owner locks -> Group mutex -> whatever the
// commit touches. Joining or leaving a flight never acquires an owner lock.
package sharedread

import (
	"context"
	"sync"
	"time"
)

// Load produces a value for one flight. ctx is the flight's detached context:
// it carries the first caller's values, not its cancellation, and ends at the
// Group's Timeout or when the last waiter leaves.
//
// ifActive runs commit under the Group's mutex only while this flight is still
// the key's current flight, still has a waiter, and its context is live. An
// owner routes every publication of the load's result through it, so abandoned
// work can never publish. commit must not call back into the Group.
type Load[V any] func(ctx context.Context, ifActive func(commit func())) (V, error)

// Group coalesces concurrent reads keyed by K. The zero value is ready to use.
type Group[K comparable, V any] struct {
	// Timeout bounds every detached load, so a stalled dependency cannot keep
	// work alive indefinitely. Zero means no deadline. It is read when a
	// flight starts.
	Timeout time.Duration

	mu      sync.Mutex
	flights map[K]*flight[V]
	running int
}

// Every field except ctx/cancel/done is protected by Group.mu.
type flight[V any] struct {
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	waiters  int
	finished bool
	result   V
	err      error
}

// Do returns the result of key's current flight, starting one when there is
// none or the current one's context has ended.
//
// Caller cancellation is independent: it returns the caller's own error, even
// when the result became ready at the same moment, and never cancels work
// another waiter still needs. When the last waiter leaves before the load
// finishes, the flight's context is canceled and the flight is retired, so
// the next caller starts a replacement. A result is shared only with the
// callers waiting on it; nothing is retained after the flight finishes.
func (g *Group[K, V]) Do(ctx context.Context, key K, load Load[V]) (V, error) {
	var zero V
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	g.mu.Lock()
	f := g.flights[key]
	if f == nil || f.ctx.Err() != nil {
		f = &flight[V]{done: make(chan struct{})}
		if g.Timeout > 0 {
			f.ctx, f.cancel = context.WithTimeout(context.WithoutCancel(ctx), g.Timeout)
		} else {
			f.ctx, f.cancel = context.WithCancel(context.WithoutCancel(ctx))
		}
		if g.flights == nil {
			g.flights = make(map[K]*flight[V])
		}
		g.flights[key] = f
		g.running++
		go g.run(key, f, load)
	}
	f.waiters++
	g.mu.Unlock()

	select {
	case <-ctx.Done():
	case <-f.ctx.Done():
	case <-f.done:
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	err := ctx.Err()
	if f.finished {
		if err != nil {
			return zero, err
		}
		return f.result, f.err
	}
	if err == nil {
		err = f.ctx.Err()
	}
	f.waiters--
	if f.waiters == 0 {
		g.retire(key, f)
		f.cancel()
	}
	return zero, err
}

func (g *Group[K, V]) run(key K, f *flight[V], load Load[V]) {
	// A live context is exactly "still current with a waiter": every path that
	// retires a flight (finish, last waiter leaving) cancels its context under
	// g.mu, and a replacement starts only after the old context has ended.
	ifActive := func(commit func()) {
		g.mu.Lock()
		defer g.mu.Unlock()
		if f.ctx.Err() == nil {
			commit()
		}
	}
	v, err := load(f.ctx, ifActive)

	g.mu.Lock()
	defer g.mu.Unlock()
	if f.ctx.Err() != nil {
		var zero V
		v, err = zero, f.ctx.Err()
	}
	f.result, f.err, f.finished = v, err, true
	g.retire(key, f)
	close(f.done)
	f.cancel()
	g.running--
}

// retire removes f only while it is still key's current flight, so a finishing
// or abandoned flight never removes the replacement that superseded it.
func (g *Group[K, V]) retire(key K, f *flight[V]) {
	if g.flights[key] == f {
		delete(g.flights, key)
	}
}

// Waiters reports how many callers are waiting across all current flights.
// Like Flights and Running it is a point-in-time reading for observability
// and tests.
func (g *Group[K, V]) Waiters() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	n := 0
	for _, f := range g.flights {
		n += f.waiters
	}
	return n
}

// Flights reports how many keys have a current, joinable flight.
func (g *Group[K, V]) Flights() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.flights)
}

// Running reports how many loads are still executing, including abandoned
// loads whose waiters have all left but whose load function has not returned.
func (g *Group[K, V]) Running() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.running
}
