package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/db"
)

// fakeFlagLister is an in-memory flagLister for resolver tests.
type fakeFlagLister struct {
	flags []db.FeatureFlag
	err   error
	calls int
}

func (f *fakeFlagLister) List(ctx context.Context) ([]db.FeatureFlag, error) {
	f.calls++
	return f.flags, f.err
}

func sampleFlags() []db.FeatureFlag {
	return []db.FeatureFlag{
		{Key: "global-search", Enabled: true, MinTier: 0},
		{Key: "god-roll", Enabled: false, MinTier: 2},
		{Key: "beta-thing", Enabled: true, MinTier: 1},
	}
}

func TestFlagResolver_ResolveKnownKeys(t *testing.T) {
	r := NewFlagResolver(&fakeFlagLister{flags: sampleFlags()}, cache.NewMemoryCache(time.Minute, time.Minute))
	cases := []struct {
		key         string
		wantEnabled bool
		wantMinTier int
		wantFound   bool
	}{
		{"global-search", true, 0, true},
		{"god-roll", false, 2, true},
		{"beta-thing", true, 1, true},
		{"no-such-key", false, 0, false},
	}
	for _, c := range cases {
		enabled, minTier, found, err := r.Resolve(context.Background(), c.key)
		if err != nil {
			t.Fatalf("Resolve(%q): unexpected err %v", c.key, err)
		}
		if enabled != c.wantEnabled || minTier != c.wantMinTier || found != c.wantFound {
			t.Errorf("Resolve(%q) = (%v,%d,%v), want (%v,%d,%v)",
				c.key, enabled, minTier, found, c.wantEnabled, c.wantMinTier, c.wantFound)
		}
	}
}

func TestFlagResolver_DegradedNilStore(t *testing.T) {
	r := NewFlagResolver(nil, cache.NewMemoryCache(time.Minute, time.Minute))
	list, err := r.List(context.Background())
	if err != nil || list != nil {
		t.Fatalf("degraded List = (%v, %v), want (nil, nil)", list, err)
	}
	_, _, found, err := r.Resolve(context.Background(), "global-search")
	if err != nil || found {
		t.Errorf("degraded Resolve found=%v err=%v, want found=false err=nil", found, err)
	}
}

func TestFlagResolver_CachesAcrossCalls(t *testing.T) {
	lister := &fakeFlagLister{flags: sampleFlags()}
	r := NewFlagResolver(lister, cache.NewMemoryCache(time.Minute, time.Minute))
	for i := 0; i < 3; i++ {
		if _, err := r.List(context.Background()); err != nil {
			t.Fatalf("List: %v", err)
		}
	}
	if lister.calls != 1 {
		t.Errorf("store hit %d times, want 1 (cached)", lister.calls)
	}
}

func TestFlagResolver_WrongTypedCacheFallsBackToStore(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, time.Minute)
	c.Set(flagsCacheKey, "not a flag slice", time.Minute) // poison the cache slot
	lister := &fakeFlagLister{flags: sampleFlags()}
	r := NewFlagResolver(lister, c)
	list, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 || lister.calls != 1 {
		t.Errorf("wrong-typed cache: got %d flags, %d store calls; want 3 flags, 1 call", len(list), lister.calls)
	}
}

func TestFlagResolver_StoreErrorPropagates(t *testing.T) {
	r := NewFlagResolver(&fakeFlagLister{err: errors.New("db down")}, cache.NewMemoryCache(time.Minute, time.Minute))
	if _, _, _, err := r.Resolve(context.Background(), "global-search"); err == nil {
		t.Error("Resolve: want error from store, got nil")
	}
}
