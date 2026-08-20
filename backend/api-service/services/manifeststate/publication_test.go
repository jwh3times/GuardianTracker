package manifeststate

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublish_CurrentGenerationCommits(t *testing.T) {
	p := New(nil)
	attempt := p.Begin()

	committed := false
	if !attempt.Publish(func() { committed = true }) {
		t.Fatal("Publish on the current generation = false, want true")
	}
	if !committed {
		t.Error("commit did not run")
	}
}

// The race ADR 0014 exists to close: work starts under one generation, a new
// manifest lands, and the old work finishes. It must not install anything.
func TestPublish_RetiredGenerationDoesNotCommit(t *testing.T) {
	p := New(nil)
	attempt := p.Begin()

	if err := p.Advance("v2"); err != nil {
		t.Fatalf("Advance: %v", err)
	}

	committed := false
	if attempt.Publish(func() { committed = true }) {
		t.Error("Publish on a retired generation = true, want false")
	}
	if committed {
		t.Error("commit ran for a retired generation")
	}
}

func TestPublish_ZeroAttemptIsInert(t *testing.T) {
	committed := false
	if (Attempt{}).Publish(func() { committed = true }) {
		t.Error("zero Attempt published, want inert")
	}
	if committed {
		t.Error("zero Attempt ran commit")
	}
}

func TestPublish_NilCommitStillReportsGeneration(t *testing.T) {
	p := New(nil)
	current := p.Begin()
	if !current.Publish(nil) {
		t.Error("nil commit on current generation = false, want true")
	}

	stale := p.Begin()
	if err := p.Advance("v2"); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	if stale.Publish(nil) {
		t.Error("nil commit on retired generation = true, want false")
	}
}

func TestAdvance_EmptyVersionRejectedAndChangesNothing(t *testing.T) {
	invalidations := 0
	p := New(func() { invalidations++ })
	attempt := p.Begin()

	if err := p.Advance(""); !errors.Is(err, ErrEmptyVersion) {
		t.Fatalf("Advance(\"\") = %v, want ErrEmptyVersion", err)
	}
	if invalidations != 0 {
		t.Errorf("invalidations = %d, want 0", invalidations)
	}
	if !attempt.Publish(nil) {
		t.Error("a rejected Advance retired an outstanding attempt")
	}
}

func TestAdvance_SameVersionIsIdempotent(t *testing.T) {
	invalidations := 0
	p := New(func() { invalidations++ })

	if err := p.Advance("v1"); err != nil {
		t.Fatalf("first Advance: %v", err)
	}
	if invalidations != 1 {
		t.Fatalf("invalidations after first Advance = %d, want 1", invalidations)
	}

	attempt := p.Begin()
	if err := p.Advance("v1"); err != nil {
		t.Fatalf("repeat Advance: %v", err)
	}
	if invalidations != 1 {
		t.Errorf("invalidations after repeating the current version = %d, want 1", invalidations)
	}
	if !attempt.Publish(nil) {
		t.Error("repeating the current version retired an outstanding attempt")
	}
}

// A version string coming back around is still a new generation. Work abandoned
// under the earlier visit must not become current again just because the name
// matches.
func TestAdvance_ReturningToAPreviousVersionRetiresWork(t *testing.T) {
	invalidations := 0
	p := New(func() { invalidations++ })

	if err := p.Advance("v1"); err != nil {
		t.Fatalf("Advance v1: %v", err)
	}
	attempt := p.Begin()
	if err := p.Advance("v2"); err != nil {
		t.Fatalf("Advance v2: %v", err)
	}
	if err := p.Advance("v1"); err != nil {
		t.Fatalf("Advance back to v1: %v", err)
	}

	if invalidations != 3 {
		t.Errorf("invalidations = %d, want 3", invalidations)
	}
	if attempt.Publish(nil) {
		t.Error("an attempt from the first v1 visit published after returning to v1")
	}
}

// Advancing the generation and clearing owner state are one transition. A
// commit must never run while the owner is midway through invalidating, or a
// publish could land in state that is about to be wiped — or worse, after it.
func TestAdvance_CommitCannotInterleaveWithInvalidation(t *testing.T) {
	var clearing, observed atomic.Bool
	p := New(func() {
		clearing.Store(true)
		time.Sleep(time.Millisecond)
		clearing.Store(false)
	})

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
			}
			// A fresh attempt each time, so publishes keep landing as the
			// generation moves rather than all being retired by the first
			// Advance.
			p.Begin().Publish(func() {
				if clearing.Load() {
					observed.Store(true)
				}
			})
		}
	}()

	for i, v := range []string{"v1", "v2", "v3", "v4"} {
		if err := p.Advance(v); err != nil {
			t.Fatalf("Advance %d: %v", i, err)
		}
	}
	close(stop)
	<-done

	if observed.Load() {
		t.Error("a commit ran while the owner was invalidating")
	}
}

func TestAdvance_ConcurrentWithBeginAndPublish(t *testing.T) {
	invalidations := 0
	var mu sync.Mutex
	p := New(func() {
		mu.Lock()
		invalidations++
		mu.Unlock()
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				p.Begin().Publish(func() {})
			}
		}()
	}
	versions := []string{"a", "b", "c", "d"}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if err := p.Advance(versions[(i+j)%len(versions)]); err != nil {
					t.Errorf("Advance: %v", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if invalidations == 0 {
		t.Error("no invalidation ran across concurrent advances")
	}
}
