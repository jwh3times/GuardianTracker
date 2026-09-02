package records

import (
	"context"
	"sync"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	manifestrepo "guardian-tracker/api-service/services/manifest"
)

type blockingProjectionManifest struct {
	fakeRecordsManifest
	projection string
	started    chan struct{}
	release    chan struct{}
	once       sync.Once
	calls      int
}

func (m *blockingProjectionManifest) block(projection string) {
	if m.projection != projection {
		return
	}
	m.once.Do(func() {
		close(m.started)
		<-m.release
	})
}

func (m *blockingProjectionManifest) GetWeaponTypesByName() (map[string]string, error) {
	m.calls++
	value := m.weaponTypes
	m.block("weapon types")
	return value, nil
}

func (m *blockingProjectionManifest) GetExoticWeaponsByName() (map[string]manifestrepo.ExoticWeapon, error) {
	m.calls++
	value := m.exoticWeapons
	m.block("exotic weapons")
	return value, nil
}

func (m *blockingProjectionManifest) GetCatalystLinks() ([]manifestrepo.CatalystLink, error) {
	m.calls++
	value := m.catalystLinks
	m.block("catalyst links")
	return value, nil
}

// A projection load that began against the retired manifest is still useful to
// its initiating request, but must not repopulate the shared cache after the
// version observer has invalidated it.
func TestManifestProjectionLoad_DoesNotPublishAcrossVersionChange(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		seed func(*blockingProjectionManifest, string)
		read func(*Service) string
	}{
		{
			name: "weapon types",
			seed: func(m *blockingProjectionManifest, value string) {
				m.weaponTypes = map[string]string{"weapon": value}
			},
			read: func(s *Service) string { return s.weaponTypesByName(ctx)["weapon"] },
		},
		{
			name: "exotic weapons",
			seed: func(m *blockingProjectionManifest, value string) {
				m.exoticWeapons = map[string]manifestrepo.ExoticWeapon{"weapon": {Type: value}}
			},
			read: func(s *Service) string { return s.exoticWeaponsByName(ctx)["weapon"].Type },
		},
		{
			name: "catalyst links",
			seed: func(m *blockingProjectionManifest, value string) {
				m.catalystLinks = []manifestrepo.CatalystLink{{WeaponName: value}}
			},
			read: func(s *Service) string { return s.catalystLinks(ctx)[0].WeaponName },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &blockingProjectionManifest{
				projection: tc.name,
				started:    make(chan struct{}),
				release:    make(chan struct{}),
			}
			tc.seed(m, "old")
			s := NewService(nil, m, cache.NewMemoryCache(time.Minute, 0), time.Minute)

			result := make(chan string, 1)
			go func() { result <- tc.read(s) }()
			<-m.started

			if err := s.OnVersionChanged("v2"); err != nil {
				t.Fatalf("OnVersionChanged: %v", err)
			}
			tc.seed(m, "new")
			close(m.release)

			if got := <-result; got != "old" {
				t.Fatalf("initiating read = %q, want old", got)
			}
			if got := tc.read(s); got != "new" {
				t.Fatalf("read after swap = %q, want new", got)
			}
			if got := tc.read(s); got != "new" {
				t.Fatalf("cached read = %q, want new", got)
			}
			if m.calls != 2 {
				t.Fatalf("manifest reads = %d, want 2", m.calls)
			}
		})
	}
}

// The three manifest-derived lookup tables must go; the per-user Bungie profile
// records in the same cache must not — a manifest swap does not invalidate them,
// and dropping them would force a refetch for every active user.
func TestOnVersionChanged_DropsManifestTablesOnly(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, 0)
	s := NewService(nil, nil, c, time.Minute)

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
		t.Error("per-membership profile records were evicted; they are Bungie data, not manifest data")
	}
	if _, ok := c.Get("settings:core"); !ok {
		t.Error("Bungie common settings were evicted; they are not manifest-derived")
	}
}
