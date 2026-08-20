package manifeststate

import (
	"context"
	"errors"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
)

func newCache(t *testing.T) *cache.MemoryCache {
	t.Helper()
	c := cache.NewMemoryCache(time.Minute, 0)
	t.Cleanup(c.Close)
	return c
}

func TestLoad_MissLoadsAndCaches(t *testing.T) {
	p := New(nil)
	c := newCache(t)

	calls := 0
	got, err := Load(context.Background(), p, c, "k", time.Minute, func() (int, error) {
		calls++
		return 42, nil
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != 42 {
		t.Errorf("value = %d, want 42", got)
	}
	if calls != 1 {
		t.Errorf("load calls = %d, want 1", calls)
	}
	if cached, ok := c.Get("k"); !ok || cached.(int) != 42 {
		t.Errorf("cache = (%v, %v), want (42, true)", cached, ok)
	}
}

func TestLoad_HitSkipsLoad(t *testing.T) {
	p := New(nil)
	c := newCache(t)
	c.Set("k", 7, time.Minute)

	got, err := Load(context.Background(), p, c, "k", time.Minute, func() (int, error) {
		t.Fatal("load ran on a cache hit")
		return 0, nil
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != 7 {
		t.Errorf("value = %d, want 7", got)
	}
}

func TestLoad_ErrorIsNotCached(t *testing.T) {
	p := New(nil)
	c := newCache(t)
	want := errors.New("boom")

	if _, err := Load(context.Background(), p, c, "k", time.Minute, func() (int, error) {
		return 0, want
	}); !errors.Is(err, want) {
		t.Fatalf("Load error = %v, want %v", err, want)
	}
	if _, ok := c.Get("k"); ok {
		t.Error("an error was cached")
	}
}

func TestLoad_WrongCachedTypeIsAMiss(t *testing.T) {
	p := New(nil)
	c := newCache(t)
	c.Set("k", "not an int", time.Minute)

	got, err := Load(context.Background(), p, c, "k", time.Minute, func() (int, error) {
		return 9, nil
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != 9 {
		t.Errorf("value = %d, want 9", got)
	}
	if cached, ok := c.Get("k"); !ok || cached.(int) != 9 {
		t.Errorf("cache = (%v, %v), want (9, true) — the mistyped entry should have been replaced", cached, ok)
	}
}

// ADR 0014's required shape: block a load, advance the manifest, release the
// old work, and prove the result reached its caller but was not installed.
func TestLoad_ValueLoadedUnderARetiredGenerationIsNotCached(t *testing.T) {
	p := New(nil)
	c := newCache(t)

	got, err := Load(context.Background(), p, c, "k", time.Minute, func() (int, error) {
		// Stands in for "a new manifest landed while this load was in flight".
		if err := p.Advance("v2"); err != nil {
			return 0, err
		}
		return 42, nil
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != 42 {
		t.Errorf("value = %d, want 42 — the initiating caller still gets its coherent result", got)
	}
	if _, ok := c.Get("k"); ok {
		t.Error("a value loaded under a retired generation was installed as reusable state")
	}
}

func TestLoad_StillCachesWhenTheVersionMerelyRepeats(t *testing.T) {
	p := New(nil)
	if err := p.Advance("v1"); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	c := newCache(t)

	if _, err := Load(context.Background(), p, c, "k", time.Minute, func() (int, error) {
		// Same version: an idempotent no-op, so this load is still current.
		if err := p.Advance("v1"); err != nil {
			return 0, err
		}
		return 42, nil
	}); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := c.Get("k"); !ok {
		t.Error("repeating the current version wrongly discarded a current load")
	}
}

func TestLoadIf_PredicateAndFenceBothApply(t *testing.T) {
	t.Run("predicate rejects", func(t *testing.T) {
		p := New(nil)
		c := newCache(t)

		got, err := LoadIf(context.Background(), p, c, "k", time.Minute,
			func() ([]int, error) { return nil, nil },
			cache.NonEmptySlice[int],
		)
		if err != nil {
			t.Fatalf("LoadIf: %v", err)
		}
		if got != nil {
			t.Errorf("value = %v, want nil", got)
		}
		if _, ok := c.Get("k"); ok {
			t.Error("an empty slice was cached despite the predicate")
		}
	})

	t.Run("predicate accepts and generation is current", func(t *testing.T) {
		p := New(nil)
		c := newCache(t)

		if _, err := LoadIf(context.Background(), p, c, "k", time.Minute,
			func() ([]int, error) { return []int{1}, nil },
			cache.NonEmptySlice[int],
		); err != nil {
			t.Fatalf("LoadIf: %v", err)
		}
		if _, ok := c.Get("k"); !ok {
			t.Error("a non-empty slice on the current generation was not cached")
		}
	})
}

func TestLoad_NilCacheLoadsEveryTime(t *testing.T) {
	p := New(nil)
	calls := 0
	for i := 0; i < 2; i++ {
		if _, err := Load(context.Background(), p, nil, "k", time.Minute, func() (int, error) {
			calls++
			return 1, nil
		}); err != nil {
			t.Fatalf("Load: %v", err)
		}
	}
	if calls != 2 {
		t.Errorf("load calls = %d, want 2", calls)
	}
}

// A nil cache is a supported configuration; a nil publication is a wiring
// mistake that would silently remove the fence, so it fails loudly.
func TestLoad_NilPublicationIsAnError(t *testing.T) {
	c := newCache(t)
	loaded := false

	_, err := Load(context.Background(), nil, c, "k", time.Minute, func() (int, error) {
		loaded = true
		return 1, nil
	})
	if !errors.Is(err, ErrNilPublication) {
		t.Fatalf("Load error = %v, want ErrNilPublication", err)
	}
	if loaded {
		t.Error("load ran without a publication")
	}
}
