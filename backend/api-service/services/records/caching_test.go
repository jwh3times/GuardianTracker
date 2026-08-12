package records

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	manifestrepo "guardian-tracker/api-service/services/manifest"
)

// countingManifest reports how often each manifest projection was read, and can
// start out empty — which is what a manifest read looks like while the download
// is still in flight.
type countingManifest struct {
	fakeRecordsManifest
	weaponTypeCalls int
	exoticCalls     int
	catalystCalls   int
}

func (m *countingManifest) GetWeaponTypesByName() (map[string]string, error) {
	m.weaponTypeCalls++
	return m.weaponTypes, nil
}

func (m *countingManifest) GetExoticWeaponsByName() (map[string]manifestrepo.ExoticWeapon, error) {
	m.exoticCalls++
	return m.exoticWeapons, nil
}

func (m *countingManifest) GetCatalystLinks() ([]manifestrepo.CatalystLink, error) {
	m.catalystCalls++
	return m.catalystLinks, nil
}

// These three projections are read once an hour and shared by every user, so an
// empty read cached for that hour degrades every catalyst and crafting entry to
// its fallback label. The rule was written as a comment at each site and
// asserted nowhere; a mutation that dropped the predicate passed the whole
// suite.
func TestManifestProjections_DoNotCacheAnEmptyRead(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		// read runs the memoized accessor and reports whether it produced data.
		read     func(*Service) bool
		calls    func(*countingManifest) int
		populate func(*countingManifest)
	}{
		{
			name:  "weapon types",
			read:  func(s *Service) bool { return len(s.weaponTypesByName(ctx)) > 0 },
			calls: func(m *countingManifest) int { return m.weaponTypeCalls },
			populate: func(m *countingManifest) {
				m.weaponTypes = map[string]string{"gjallarhorn": "Rocket Launcher"}
			},
		},
		{
			name:  "exotic weapons",
			read:  func(s *Service) bool { return len(s.exoticWeaponsByName(ctx)) > 0 },
			calls: func(m *countingManifest) int { return m.exoticCalls },
			populate: func(m *countingManifest) {
				m.exoticWeapons = map[string]manifestrepo.ExoticWeapon{"gjallarhorn": {Type: "Rocket Launcher"}}
			},
		},
		{
			name:  "catalyst links",
			read:  func(s *Service) bool { return len(s.catalystLinks(ctx)) > 0 },
			calls: func(m *countingManifest) int { return m.catalystCalls },
			populate: func(m *countingManifest) {
				m.catalystLinks = []manifestrepo.CatalystLink{{WeaponName: "Gjallarhorn"}}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &countingManifest{}
			s := NewService(nil, m, cache.NewMemoryCache(time.Minute, 0), time.Minute)

			// The manifest is not ready yet: the read comes back empty.
			if tc.read(s) {
				t.Fatal("expected an empty first read")
			}
			if got := tc.calls(m); got != 1 {
				t.Fatalf("manifest read %d times, want 1", got)
			}

			// The manifest lands. The empty read must not be standing in for it.
			tc.populate(m)
			if !tc.read(s) {
				t.Error("second read was empty — the empty result was cached for the hour")
			}
			if got := tc.calls(m); got != 2 {
				t.Errorf("manifest read %d times, want 2", got)
			}

			// The populated read is cached, so a third read does not hit the manifest.
			if !tc.read(s) {
				t.Error("third read was empty")
			}
			if got := tc.calls(m); got != 2 {
				t.Errorf("manifest read %d times, want 2 — the non-empty result should be cached", got)
			}
		})
	}
}

// TestCacheKeysAreDistinct guards a risk this refactor introduced: the core
// settings key used to be a constant local to its one function and is now a
// package-level const beside three others. Two of them holding the same string
// would have each feature reading the other's value — which now degrades to a
// logged wrong-type miss rather than silently succeeding, but still means one of
// them never caches anything.
func TestCacheKeysAreDistinct(t *testing.T) {
	keys := map[string]string{
		"weaponTypes":   weaponTypesCacheKey,
		"exoticWeapons": exoticWeaponsCacheKey,
		"catalystLinks": catalystLinksCacheKey,
		"coreSettings":  coreSettingsCacheKey,
		"profile":       recordsCacheKey(3, "member-1"),
	}
	seen := map[string]string{}
	for name, key := range keys {
		if other, clash := seen[key]; clash {
			t.Errorf("%s and %s share the cache key %q", name, other, key)
		}
		seen[key] = name
	}
}

// TestGetProfileRecords_UsesTheServiceTTL pins which TTL the profile cache is
// wired with. The generic honours whatever it is given (cache.TestLoad_HonoursTheTTL);
// what only this site can say is that it hands over the configured
// CACHE_TTL_RECORDS rather than one of the day-long manifest TTLs beside it.
func TestGetProfileRecords_UsesTheServiceTTL(t *testing.T) {
	fetches := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches++
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{}}}}}`)
	}))
	t.Cleanup(srv.Close)

	const ttl = 40 * time.Millisecond
	s := NewService(bungie.NewClient("k", srv.URL, 100, 100), &fakeRecordsManifest{},
		cache.NewMemoryCache(time.Minute, 0), ttl)

	if _, _, err := s.getProfileRecords(context.Background(), 3, "member-1", "token"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.getProfileRecords(context.Background(), 3, "member-1", "token"); err != nil {
		t.Fatal(err)
	}
	if fetches != 1 {
		t.Fatalf("fetches = %d, want 1 — the second call should have been cached", fetches)
	}

	time.Sleep(3 * ttl)
	if _, _, err := s.getProfileRecords(context.Background(), 3, "member-1", "token"); err != nil {
		t.Fatal(err)
	}
	if fetches != 2 {
		t.Errorf("fetches = %d, want 2 — the entry should have expired at the service TTL", fetches)
	}
}
