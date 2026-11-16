package sharedread

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type result struct {
	v   int
	err error
}

func start(ctx context.Context, g *Group[string, int], key string, load func(context.Context, func(func())) (int, error)) <-chan result {
	ch := make(chan result, 1)
	go func() {
		v, err := g.Do(ctx, key, load)
		ch <- result{v, err}
	}()
	return ch
}

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for fixture event")
		var zero T
		return zero
	}
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for !cond() {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s", what)
		default:
			runtime.Gosched()
		}
	}
}

func idle(t *testing.T, g *Group[string, int]) {
	t.Helper()
	eventually(t, "the group to go idle", func() bool { return g.Flights() == 0 && g.Running() == 0 })
}

// blockingLoad counts calls, reports each load's context, and returns the
// value sent on release.
type blockingLoad struct {
	calls   atomic.Int32
	entered chan context.Context
	release chan int
}

func newBlockingLoad() *blockingLoad {
	return &blockingLoad{entered: make(chan context.Context, 8), release: make(chan int, 8)}
}

func (b *blockingLoad) load(ctx context.Context, _ func(func())) (int, error) {
	b.calls.Add(1)
	b.entered <- ctx
	return <-b.release, nil
}

func TestGroup_OverlappingCallersShareOneLoad(t *testing.T) {
	g := &Group[string, int]{Timeout: time.Minute}
	b := newBlockingLoad()
	first := start(t.Context(), g, "k", b.load)
	receive(t, b.entered)
	second := start(t.Context(), g, "k", b.load)
	eventually(t, "two waiters", func() bool { return g.Waiters() == 2 })
	b.release <- 42
	for _, ch := range []<-chan result{first, second} {
		if got := receive(t, ch); got.err != nil || got.v != 42 {
			t.Fatalf("result=%+v", got)
		}
	}
	if b.calls.Load() != 1 {
		t.Fatalf("loads=%d want 1", b.calls.Load())
	}
	idle(t, g)
}

func TestGroup_CallerCancellationIsIndependentAndKeepsContextValues(t *testing.T) {
	for _, cancelFirst := range []bool{true, false} {
		name := "follower"
		if cancelFirst {
			name = "leader"
		}
		t.Run(name, func(t *testing.T) {
			type ctxKey struct{}
			g := &Group[string, int]{Timeout: time.Minute}
			b := newBlockingLoad()
			// The leader's own deadline is far shorter than the Group's; the
			// shared load must use the Group's bound, not inherit the leader's.
			base, baseCancel := context.WithTimeout(context.WithValue(t.Context(), ctxKey{}, "request-value"), 30*time.Second)
			defer baseCancel()
			leaderCtx, cancelLeader := context.WithCancel(base)
			defer cancelLeader()
			followerCtx, cancelFollower := context.WithCancel(t.Context())
			defer cancelFollower()

			leader := start(leaderCtx, g, "k", b.load)
			shared := receive(t, b.entered)
			follower := start(followerCtx, g, "k", b.load)
			eventually(t, "two waiters", func() bool { return g.Waiters() == 2 })

			if shared.Value(ctxKey{}) != "request-value" {
				t.Fatal("shared load lost the caller's context values")
			}
			if d, ok := shared.Deadline(); !ok || time.Until(d) < 45*time.Second || time.Until(d) > time.Minute {
				t.Fatalf("shared deadline=%v ok=%v; want the Group's one-minute bound", d, ok)
			}

			canceled, remaining := leader, follower
			if cancelFirst {
				cancelLeader()
			} else {
				cancelFollower()
				canceled, remaining = follower, leader
			}
			if got := receive(t, canceled); !errors.Is(got.err, context.Canceled) {
				t.Fatalf("canceled caller error=%v", got.err)
			}
			if shared.Err() != nil {
				t.Fatal("one caller's cancellation canceled shared work")
			}
			if g.Waiters() != 1 {
				t.Fatalf("waiters=%d after one caller left", g.Waiters())
			}
			b.release <- 7
			if got := receive(t, remaining); got.err != nil || got.v != 7 {
				t.Fatalf("remaining=%+v", got)
			}
			idle(t, g)
		})
	}
}

func TestGroup_LastWaiterLeavingCancelsTheLoadAndAllowsANewOne(t *testing.T) {
	g := &Group[string, int]{Timeout: time.Minute}
	loadCanceled := make(chan struct{})
	var calls atomic.Int32
	load := func(ctx context.Context, _ func(func())) (int, error) {
		if calls.Add(1) > 1 {
			return 9, nil
		}
		<-ctx.Done()
		close(loadCanceled)
		return 0, ctx.Err()
	}
	ctx, cancel := context.WithCancel(t.Context())
	first := start(ctx, g, "k", load)
	second := start(ctx, g, "k", load)
	eventually(t, "two waiters", func() bool { return g.Waiters() == 2 })
	cancel()
	for _, ch := range []<-chan result{first, second} {
		if got := receive(t, ch); !errors.Is(got.err, context.Canceled) {
			t.Fatalf("error=%v", got.err)
		}
	}
	receive(t, loadCanceled)
	idle(t, g)
	if v, err := g.Do(t.Context(), "k", load); err != nil || v != 9 {
		t.Fatalf("next call=%d,%v; want a new load", v, err)
	}
	if calls.Load() != 2 {
		t.Fatalf("loads=%d want 2", calls.Load())
	}
}

func TestGroup_TimeoutReleasesWaitersWithDeadlineExceeded(t *testing.T) {
	g := &Group[string, int]{Timeout: 50 * time.Millisecond}
	release := make(chan struct{})
	defer close(release)
	// A load that ignores its context still cannot hold callers past Timeout.
	load := func(context.Context, func(func())) (int, error) { <-release; return 1, nil }
	first := start(t.Context(), g, "k", load)
	second := start(t.Context(), g, "k", load)
	for _, ch := range []<-chan result{first, second} {
		if got := receive(t, ch); !errors.Is(got.err, context.DeadlineExceeded) {
			t.Fatalf("error=%v", got.err)
		}
	}
	if g.Flights() != 0 {
		t.Fatal("timed-out flight is still joinable")
	}
	if g.Running() != 1 {
		t.Fatalf("running=%d; the stalled load has not returned yet", g.Running())
	}
}

// An abandoned load whose dependency returns later can neither publish through
// ifActive nor, when it finishes, remove the replacement that superseded it.
func TestGroup_AbandonedLoadCannotPublishOrRemoveItsReplacement(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprintf("deadline=%v", deadline), func(t *testing.T) {
			g := &Group[string, int]{Timeout: time.Minute}
			if deadline {
				g.Timeout = 50 * time.Millisecond
			}
			var publishedMu sync.Mutex
			var published []int
			entered := make(chan chan int, 2)
			load := func(_ context.Context, ifActive func(func())) (int, error) {
				release := make(chan int)
				entered <- release
				v := <-release
				ifActive(func() {
					publishedMu.Lock()
					published = append(published, v)
					publishedMu.Unlock()
				})
				return v, nil
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			first := start(ctx, g, "k", load)
			oldRelease := receive(t, entered)
			second := start(ctx, g, "k", load)
			eventually(t, "two waiters", func() bool { return g.Waiters() == 2 })
			want := context.DeadlineExceeded
			if !deadline {
				want = context.Canceled
				cancel()
			}
			for _, ch := range []<-chan result{first, second} {
				if got := receive(t, ch); !errors.Is(got.err, want) {
					t.Fatalf("error=%v want %v", got.err, want)
				}
			}
			if g.Flights() != 0 {
				t.Fatal("abandoned flight is still joinable")
			}

			g.Timeout = time.Minute
			replacement := start(t.Context(), g, "k", load)
			newRelease := receive(t, entered)
			eventually(t, "the replacement's waiter", func() bool { return g.Waiters() == 1 })
			oldRelease <- 1
			eventually(t, "the abandoned load to settle", func() bool { return g.Running() == 1 })
			if g.Waiters() != 1 {
				t.Fatal("the abandoned load's completion removed its replacement")
			}
			newRelease <- 2
			if got := receive(t, replacement); got.err != nil || got.v != 2 {
				t.Fatalf("replacement=%+v", got)
			}
			idle(t, g)
			publishedMu.Lock()
			defer publishedMu.Unlock()
			if len(published) != 1 || published[0] != 2 {
				t.Fatalf("published=%v; want only the replacement's value", published)
			}
		})
	}
}

// A commit that passed the guard before the last waiter left is not undone:
// it completes, and the canceled caller still returns its own error.
func TestGroup_CommitInProgressWinsTheCancellationRace(t *testing.T) {
	g := &Group[string, int]{Timeout: time.Minute}
	committing, release := make(chan struct{}), make(chan struct{})
	var committed atomic.Bool
	load := func(_ context.Context, ifActive func(func())) (int, error) {
		ifActive(func() {
			close(committing)
			<-release
			committed.Store(true)
		})
		return 1, nil
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	got := start(ctx, g, "k", load)
	receive(t, committing)
	cancel()
	close(release)
	if r := receive(t, got); !errors.Is(r.err, context.Canceled) {
		t.Fatalf("canceled caller error=%v", r.err)
	}
	if !committed.Load() {
		t.Fatal("commit that passed the guard did not complete")
	}
	idle(t, g)
}

func TestGroup_FailureIsSharedButNotRetained(t *testing.T) {
	g := &Group[string, int]{Timeout: time.Minute}
	b := newBlockingLoad()
	boom := errors.New("upstream failed")
	failing := func(ctx context.Context, ifActive func(func())) (int, error) {
		_, _ = b.load(ctx, ifActive)
		return 0, boom
	}
	first := start(t.Context(), g, "k", failing)
	receive(t, b.entered)
	second := start(t.Context(), g, "k", failing)
	eventually(t, "two waiters", func() bool { return g.Waiters() == 2 })
	b.release <- 0
	for _, ch := range []<-chan result{first, second} {
		if got := receive(t, ch); !errors.Is(got.err, boom) {
			t.Fatalf("error=%v", got.err)
		}
	}
	idle(t, g)
	b.release <- 5
	if v, err := g.Do(t.Context(), "k", b.load); err != nil || v != 5 {
		t.Fatalf("after failure=%d,%v; want a fresh load", v, err)
	}
	if b.calls.Load() != 2 {
		t.Fatalf("loads=%d want 2", b.calls.Load())
	}
}

func TestGroup_DistinctKeysDoNotShare(t *testing.T) {
	g := &Group[string, int]{Timeout: time.Minute}
	b := newBlockingLoad()
	a := start(t.Context(), g, "a", b.load)
	c := start(t.Context(), g, "c", b.load)
	receive(t, b.entered)
	receive(t, b.entered)
	if g.Flights() != 2 {
		t.Fatalf("flights=%d want 2", g.Flights())
	}
	b.release <- 1
	b.release <- 1
	receive(t, a)
	receive(t, c)
	idle(t, g)
}

func TestGroup_CanceledCallerStartsNoLoad(t *testing.T) {
	g := &Group[string, int]{}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	called := false
	_, err := g.Do(ctx, "k", func(context.Context, func(func())) (int, error) { called = true; return 0, nil })
	if !errors.Is(err, context.Canceled) || called || g.Running() != 0 {
		t.Fatalf("err=%v called=%v running=%d", err, called, g.Running())
	}
}
