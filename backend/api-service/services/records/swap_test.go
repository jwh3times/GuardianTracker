package records

import (
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
)

// The three manifest-derived lookup tables must go; the per-user Bungie profile
// records in the same cache must not — a manifest swap does not invalidate them,
// and dropping them would force a refetch for every active user.
func TestOnVersionChanged_DropsManifestTablesOnly(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, 0)
	s := &Service{cache: c}

	c.Set(weaponTypesCacheKey, map[string]string{"a": "Hand Cannon"}, time.Minute)
	c.Set(exoticWeaponsCacheKey, map[string]string{"b": "Exotic"}, time.Minute)
	c.Set(catalystLinksCacheKey, []string{"link"}, time.Minute)
	c.Set(recordsCacheKey(3, "member-1"), "bungie-profile-records", time.Minute)
	c.Set("settings:core", "bungie-settings", time.Minute)

	if err := s.OnVersionChanged("v2"); err != nil {
		t.Fatalf("OnVersionChanged: %v", err)
	}

	for _, key := range []string{weaponTypesCacheKey, exoticWeaponsCacheKey, catalystLinksCacheKey} {
		if _, ok := c.Get(key); ok {
			t.Errorf("%s survived a manifest swap", key)
		}
	}
	if _, ok := c.Get(recordsCacheKey(3, "member-1")); !ok {
		t.Error("per-user profile records were evicted; they are Bungie data, not manifest data")
	}
	if _, ok := c.Get("settings:core"); !ok {
		t.Error("Bungie common settings were evicted; they are not manifest-derived")
	}
}
