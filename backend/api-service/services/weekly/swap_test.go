package weekly

import (
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
)

// OnVersionChanged must drop the manifest-resolved global payload and nothing
// else — the per-user Bungie data in the same cache is still valid.
func TestOnVersionChanged_DropsPublicWeeklyOnly(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, 0)
	s := &Service{cache: c, version: fakeVersioner{"v1"}}

	c.Set(publicWeeklyCacheKey, &publicWeeklyCache{}, time.Minute)
	c.Set(rosterCacheKey(3, "member-1"), []string{"char-1"}, time.Minute)
	c.Set(characterVendorsCacheKey(3, "member-1", "char-1"), "raw-bungie-response", time.Minute)

	if err := s.OnVersionChanged("v2"); err != nil {
		t.Fatalf("OnVersionChanged: %v", err)
	}

	if _, ok := c.Get(publicWeeklyCacheKey); ok {
		t.Error("weekly:public survived a manifest swap; its milestone labels are manifest-resolved")
	}
	if _, ok := c.Get(rosterCacheKey(3, "member-1")); !ok {
		t.Error("weekly:roster was evicted; it holds Bungie character data, not manifest data")
	}
	if _, ok := c.Get(characterVendorsCacheKey(3, "member-1", "char-1")); !ok {
		t.Error("the raw vendor response was evicted; a swap must not force a Bungie refetch")
	}
}

// The daily-vendor rows carry manifest-resolved item names, so their key is
// scoped by manifest version: a swap orphans the old entry instead of needing an
// eviction sweep the cache has no API for.
func TestDailyVendorsCacheKey_ScopedByManifestVersion(t *testing.T) {
	before := (&Service{version: fakeVersioner{"v1"}}).dailyVendorsCacheKey(3, "member-1", "char-1")
	after := (&Service{version: fakeVersioner{"v2"}}).dailyVendorsCacheKey(3, "member-1", "char-1")

	if before == after {
		t.Fatalf("key did not change across manifest versions: %q", before)
	}
	// Still scoped per character — a version scope must not collapse the
	// per-character isolation that component 402 requires.
	other := (&Service{version: fakeVersioner{"v1"}}).dailyVendorsCacheKey(3, "member-1", "char-2")
	if before == other {
		t.Error("key is no longer per-character")
	}
	if !strings.Contains(before, "v1") || !strings.Contains(after, "v2") {
		t.Errorf("keys do not carry the manifest version: %q / %q", before, after)
	}
}

// A service with no version source must still produce a usable, stable key
// rather than panicking or collapsing every version into one bucket silently.
func TestDailyVendorsCacheKey_NoVersionSource(t *testing.T) {
	s := &Service{}
	k := s.dailyVendorsCacheKey(3, "member-1", "char-1")
	if !strings.Contains(k, "none") {
		t.Errorf("key without a version source = %q, want a 'none' scope", k)
	}
	if k != s.dailyVendorsCacheKey(3, "member-1", "char-1") {
		t.Error("key is not stable across calls")
	}
}
