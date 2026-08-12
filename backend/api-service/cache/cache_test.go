package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeCacheClock struct {
	mu          sync.Mutex
	now         time.Time
	ticker      *fakeCacheTicker
	tickerCalls int
	started     chan time.Duration
}

func newFakeCacheClock(now time.Time) *fakeCacheClock {
	return &fakeCacheClock{
		now:     now,
		ticker:  newFakeCacheTicker(),
		started: make(chan time.Duration, 1),
	}
}

func (c *fakeCacheClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeCacheClock) NewTicker(interval time.Duration) cacheTicker {
	c.mu.Lock()
	c.tickerCalls++
	c.mu.Unlock()
	c.started <- interval
	return c.ticker
}

func (c *fakeCacheClock) Advance(elapsed time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(elapsed)
	now := c.now
	tickerStarted := c.tickerCalls > 0
	c.mu.Unlock()
	if tickerStarted {
		c.ticker.tick <- now
	}
}

func (c *fakeCacheClock) TickerCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tickerCalls
}

type fakeCacheTicker struct {
	tick     chan time.Time
	stopped  chan struct{}
	stopOnce sync.Once
}

func newFakeCacheTicker() *fakeCacheTicker {
	return &fakeCacheTicker{
		tick:    make(chan time.Time),
		stopped: make(chan struct{}),
	}
}

func (t *fakeCacheTicker) C() <-chan time.Time { return t.tick }
func (t *fakeCacheTicker) Stop()               { t.stopOnce.Do(func() { close(t.stopped) }) }

func TestMemoryCacheContract(t *testing.T) {
	t.Run("missing keys and deletion", func(t *testing.T) {
		var c Cache = NewMemoryCache(time.Minute, 0)

		if _, ok := c.Get("missing"); ok {
			t.Fatal("Get on an empty cache returned a value")
		}
		c.Delete("missing")
	})

	t.Run("Set stores and overwrites values", func(t *testing.T) {
		var c Cache = NewMemoryCache(time.Minute, 0)
		c.Set("k", "first", time.Minute)
		c.Set("k", "second", time.Minute)

		got, ok := c.Get("k")
		if !ok || got != "second" {
			t.Fatalf("Get = %v, %v; want second, true", got, ok)
		}
	})

	t.Run("Clear removes every value", func(t *testing.T) {
		var c Cache = NewMemoryCache(time.Minute, 0)
		c.Set("a", 1, time.Minute)
		c.Set("b", 2, time.Minute)
		c.Clear()

		for _, key := range []string{"a", "b"} {
			if _, ok := c.Get(key); ok {
				t.Fatalf("%q survived Clear", key)
			}
		}
	})

	t.Run("positive TTL is absolute and Get does not refresh it", func(t *testing.T) {
		clock := newFakeCacheClock(time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC))
		var c Cache = newMemoryCacheWithClock(time.Minute, 0, clock, nil)
		c.Set("k", 42, 100*time.Millisecond)
		clock.Advance(60 * time.Millisecond)
		if _, ok := c.Get("k"); !ok {
			t.Fatal("value expired before its TTL")
		}
		clock.Advance(70 * time.Millisecond)
		if _, ok := c.Get("k"); ok {
			t.Fatal("Get refreshed the TTL")
		}
	})

	t.Run("zero TTL uses the constructor default", func(t *testing.T) {
		clock := newFakeCacheClock(time.Now())
		var c Cache = newMemoryCacheWithClock(20*time.Millisecond, 0, clock, nil)
		c.Set("k", "v", 0)
		clock.Advance(21 * time.Millisecond)

		if _, ok := c.Get("k"); ok {
			t.Fatal("zero TTL did not use the constructor default")
		}
	})

	t.Run("non-positive constructor default does not expire", func(t *testing.T) {
		for _, defaultExpiration := range []time.Duration{0, -time.Second} {
			clock := newFakeCacheClock(time.Now())
			c := newMemoryCacheWithClock(defaultExpiration, 0, clock, nil)
			c.Set("k", "v", 0)
			clock.Advance(24 * time.Hour)

			if _, ok := c.Get("k"); !ok {
				t.Fatalf("value expired with constructor default %v", defaultExpiration)
			}
		}
	})

	t.Run("negative per-entry TTL does not expire", func(t *testing.T) {
		clock := newFakeCacheClock(time.Now())
		var c Cache = newMemoryCacheWithClock(time.Millisecond, 0, clock, nil)
		c.Set("k", "v", -time.Second)
		clock.Advance(24 * time.Hour)

		if _, ok := c.Get("k"); !ok {
			t.Fatal("value with a negative TTL expired")
		}
	})

	t.Run("expired Get is a miss with cleanup disabled", func(t *testing.T) {
		clock := newFakeCacheClock(time.Now())
		var c Cache = newMemoryCacheWithClock(time.Minute, 0, clock, nil)
		c.Set("k", "v", 20*time.Millisecond)
		clock.Advance(21 * time.Millisecond)

		if _, ok := c.Get("k"); ok {
			t.Fatal("expired Get returned a value")
		}
	})

	t.Run("Set restarts the TTL", func(t *testing.T) {
		clock := newFakeCacheClock(time.Now())
		var c Cache = newMemoryCacheWithClock(time.Minute, 0, clock, nil)
		c.Set("k", "first", 80*time.Millisecond)
		clock.Advance(50 * time.Millisecond)
		c.Set("k", "second", 80*time.Millisecond)
		clock.Advance(50 * time.Millisecond)

		got, ok := c.Get("k")
		if !ok || got != "second" {
			t.Fatalf("Get after overwrite = %v, %v; want second, true", got, ok)
		}
		clock.Advance(31 * time.Millisecond)
		if _, ok := c.Get("k"); ok {
			t.Fatal("overwritten value survived past its restarted TTL")
		}
	})

	t.Run("storage is not size capped", func(t *testing.T) {
		var c Cache = NewMemoryCache(time.Minute, 0)
		for i := range 1_000 {
			c.Set(fmt.Sprintf("key-%d", i), i, time.Minute)
		}
		for i := range 1_000 {
			got, ok := c.Get(fmt.Sprintf("key-%d", i))
			if !ok || got != i {
				t.Fatalf("entry %d = %v, %v; want %d, true", i, got, ok, i)
			}
		}
	})
}

func TestNoOpCache(t *testing.T) {
	var c Cache = NewNoOpCache()
	c.Set("k", "v", time.Minute)
	if _, ok := c.Get("k"); ok {
		t.Error("no-op cache returned a stored value")
	}
	c.Delete("k")
	c.Clear()
}

func TestMemoryCacheCleanupLifecycle(t *testing.T) {
	clock := newFakeCacheClock(time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC))
	cleaned := make(chan struct{}, 1)
	c := newMemoryCacheWithClock(time.Minute, 5*time.Minute, clock, func() {
		cleaned <- struct{}{}
	})

	select {
	case interval := <-clock.started:
		if interval != 5*time.Minute {
			t.Fatalf("cleanup ticker interval = %v, want 5m", interval)
		}
	case <-time.After(time.Second):
		t.Fatal("cleanup worker did not start")
	}

	c.Set("expired", "value", time.Minute)
	clock.Advance(2 * time.Minute)
	select {
	case <-cleaned:
	case <-time.After(time.Second):
		t.Fatal("cleanup worker did not process the tick")
	}

	c.mu.RLock()
	remaining := len(c.items)
	c.mu.RUnlock()
	if remaining != 0 {
		t.Fatalf("stored entries after cleanup = %d, want 0", remaining)
	}

	c.Close()
	c.Close()
	select {
	case <-clock.ticker.stopped:
	default:
		t.Fatal("Close returned before the cleanup worker stopped")
	}
}

func TestMemoryCacheNonPositiveCleanupIntervalStartsNoWorker(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Minute} {
		clock := newFakeCacheClock(time.Now())
		c := newMemoryCacheWithClock(time.Minute, interval, clock, nil)
		c.Close()

		if calls := clock.TickerCalls(); calls != 0 {
			t.Fatalf("NewTicker calls with cleanup interval %v = %d, want 0", interval, calls)
		}
	}
}

func TestMemoryCacheConcurrentOperations(t *testing.T) {
	c := NewMemoryCache(time.Minute, 0)
	var workers sync.WaitGroup
	for worker := range 8 {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			for i := range 1_000 {
				key := fmt.Sprintf("%d-%d", worker, i%16)
				c.Set(key, i, time.Minute)
				c.Get(key)
				if i%3 == 0 {
					c.Delete(key)
				}
				if i%101 == 0 {
					c.Clear()
				}
			}
		}(worker)
	}
	workers.Wait()
}
