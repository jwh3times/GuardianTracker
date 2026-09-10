package membershipstate

import (
	"context"
	"errors"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
)

func newCache(t *testing.T) cache.Cache {
	t.Helper()
	c := cache.NewMemoryCache(time.Minute, 0)
	t.Cleanup(c.Close)
	return c
}

func fenced(t *testing.T) (*Publication, cache.Cache) {
	t.Helper()
	c := newCache(t)
	return New(func(membershipType int, membershipID string) { c.Delete("k") }), c
}

func TestLoad_CachesWhatItLoaded(t *testing.T) {
	p, c := fenced(t)
	calls := 0
	load := func() (string, error) { calls++; return "value", nil }

	first, err := Load(context.Background(), p, c, 3, "member-1", "k", time.Minute, load)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	second, err := Load(context.Background(), p, c, 3, "member-1", "k", time.Minute, load)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if first != "value" || second != "value" {
		t.Errorf("values = %q/%q, want value/value", first, second)
	}
	if calls != 1 {
		t.Errorf("load called %d times, want 1 — the second read was not served from the cache", calls)
	}
}

// The deterministic form of the race this exists to close: a load is in
// flight, the membership is refreshed mid-load, and the older result must be
// returned to its own caller but must not be left behind for the next one.
func TestLoad_RefreshDuringALoadPreventsPublication(t *testing.T) {
	p, c := fenced(t)

	got, err := Load(context.Background(), p, c, 3, "member-1", "k", time.Minute,
		func() (string, error) {
			// The refresh lands while this load is running.
			p.Advance(3, "member-1")
			return "pre-refresh", nil
		})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got != "pre-refresh" {
		t.Errorf("value = %q; the initiating request must still get its own result", got)
	}
	if cached, found := c.Get("k"); found {
		t.Errorf("pre-refresh value %v was published after the refresh returned", cached)
	}
}

// A refresh for somebody else must not stop this load from publishing.
func TestLoad_AnotherMembershipsRefreshDoesNotBlockPublication(t *testing.T) {
	p, c := fenced(t)

	if _, err := Load(context.Background(), p, c, 3, "member-1", "k", time.Minute,
		func() (string, error) {
			p.Advance(3, "member-2")
			return "value", nil
		}); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if _, found := c.Get("k"); !found {
		t.Error("another membership's refresh prevented this one from publishing")
	}
}

func TestLoad_DoesNotCacheAnError(t *testing.T) {
	p, c := fenced(t)
	boom := errors.New("upstream is down")

	if _, err := Load(context.Background(), p, c, 3, "member-1", "k", time.Minute,
		func() (string, error) { return "", boom }); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}

	if _, found := c.Get("k"); found {
		t.Error("a failed load was cached")
	}
}

// A cached value of the wrong type is a miss, not a panic — the same rule
// cache.Load established.
func TestLoad_WrongCachedTypeIsAMiss(t *testing.T) {
	p, c := fenced(t)
	c.Set("k", 42, time.Minute)

	got, err := Load(context.Background(), p, c, 3, "member-1", "k", time.Minute,
		func() (string, error) { return "value", nil })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != "value" {
		t.Errorf("value = %q, want the freshly loaded value", got)
	}
}

// A nil publication would silently drop the fence, so it is refused rather
// than treated as "no fencing required".
func TestLoad_NilPublicationFailsLoudly(t *testing.T) {
	c := newCache(t)
	loaded := false

	_, err := Load(context.Background(), nil, c, 3, "member-1", "k", time.Minute,
		func() (string, error) { loaded = true; return "value", nil })

	if !errors.Is(err, ErrNilPublication) {
		t.Errorf("err = %v, want ErrNilPublication", err)
	}
	if loaded {
		t.Error("load ran without a fence")
	}
}

// A nil cache means "load every time", which is a supported degraded mode
// rather than a fencing failure.
func TestLoad_NilCacheLoadsEveryTime(t *testing.T) {
	p := New(nil)
	calls := 0

	for range 2 {
		got, err := Load(context.Background(), p, nil, 3, "member-1", "k", time.Minute,
			func() (string, error) { calls++; return "value", nil })
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got != "value" {
			t.Errorf("value = %q, want value", got)
		}
	}
	if calls != 2 {
		t.Errorf("load called %d times, want 2", calls)
	}
}
