package collections

import (
	"context"
	"testing"
	"time"

	"guardian-tracker/api-service/services/items"
)

// cachedAnalysis installs an analysis for one membership, stamped under the
// current manifest generation as a completed load would have left it.
func cachedAnalysis(t *testing.T, m *MembershipAnalysis, membershipType int, membershipID string) *analysis {
	t.Helper()
	catalog := fixtureCatalog()
	collected := map[uint32]bool{1000: true}
	return m.cached(membershipType, membershipID, &analysis{
		catalog:   catalog,
		collected: collected,
		owned:     deriveOwnedItems(catalog, collected),
		tree:      buildTreeStructure(fixtureNodes(), catalog),
		fetchedAt: time.Unix(1000, 0).UTC(),
	})
}

// The deterministic form of the race the fence exists to close, driven through
// the public refresh: an analysis is rebuilding, the user's refresh lands, and
// the pre-refresh work must not refill the entry the refresh just cleared.
//
// A rebuild is used rather than a cold load because it is the reachable
// publication site that does not need a live manifest — and it is the more
// dangerous one, since it republishes profile data the refresh meant to retire.
func TestRefreshMembership_ARebuildInFlightIsNotReused(t *testing.T) {
	catalog := &fakeCatalog{facts: fixtureCatalog()}
	m := newAnalysis(t, catalog, &fakeNodes{nodes: fixtureNodes()})
	svc := NewService(m, &fakeVendors{}, &fakeParticipant{}, &fakeParticipant{})

	cachedAnalysis(t, m, 3, "member-1")
	swapped(t, m, "v2") // the cached analysis now needs its manifest half rebuilt

	// The refresh lands while the rebuild is reading the catalog.
	catalog.onRead = func() {
		if err := svc.RefreshMembership(context.Background(), Membership{MembershipType: 3, MembershipID: "member-1"}); err != nil {
			t.Errorf("RefreshMembership: %v", err)
		}
	}

	got, err := m.getAnalysis(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("the initiating request must still get its own result: %v", err)
	}
	if len(got.catalog) != 2 {
		t.Fatalf("catalog = %d entries, want the coherent rebuild it loaded", len(got.catalog))
	}

	if cached, found := m.cache.Get(analysisCacheKey(3, "member-1")); found {
		t.Fatalf("a pre-refresh analysis %v was published after the refresh returned", cached)
	}
}

// One user's refresh must not discard another user's rebuild.
func TestRefreshMembership_DoesNotRetireAnotherMembershipsRebuild(t *testing.T) {
	catalog := &fakeCatalog{facts: fixtureCatalog()}
	m := newAnalysis(t, catalog, &fakeNodes{nodes: fixtureNodes()})
	svc := NewService(m, &fakeVendors{}, &fakeParticipant{}, &fakeParticipant{})

	cachedAnalysis(t, m, 3, "member-1")
	swapped(t, m, "v2")

	catalog.onRead = func() {
		if err := svc.RefreshMembership(context.Background(), Membership{MembershipType: 3, MembershipID: "member-2"}); err != nil {
			t.Errorf("RefreshMembership: %v", err)
		}
	}

	if _, err := m.getAnalysis(context.Background(), 3, "member-1", "token"); err != nil {
		t.Fatalf("getAnalysis: %v", err)
	}

	if _, found := m.cache.Get(analysisCacheKey(3, "member-1")); !found {
		t.Error("another membership's refresh discarded this membership's rebuilt analysis")
	}
}

// Both fences have to agree before an analysis is installed. This is the other
// axis: a manifest swap landing mid-rebuild still retires the work, exactly as
// it did before the refresh fence was added beside it.
func TestPublishAnalysis_EitherFenceCanRefuse(t *testing.T) {
	cases := []struct {
		name          string
		retire        func(m *MembershipAnalysis)
		wantPublished bool
	}{
		{"neither fence moved", func(*MembershipAnalysis) {}, true},
		{"a manifest swap landed", func(m *MembershipAnalysis) { _ = m.OnVersionChanged("v-next") }, false},
		{"a membership refresh landed", func(m *MembershipAnalysis) { m.InvalidateCache(3, "member-1") }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newAnalysis(t, &fakeCatalog{facts: fixtureCatalog()}, &fakeNodes{nodes: fixtureNodes()})

			sinceSwap := m.publication.Begin()
			sinceRefresh := m.refresh.Begin(3, "member-1")
			tc.retire(m)

			committed := false
			published := publishAnalysis(sinceSwap, sinceRefresh, func() { committed = true })

			if published != tc.wantPublished {
				t.Errorf("publishAnalysis = %v, want %v", published, tc.wantPublished)
			}
			if committed != tc.wantPublished {
				t.Errorf("commit ran = %v, want %v", committed, tc.wantPublished)
			}
		})
	}
}

// A refresh evicts the entry as well as advancing the generation, so a read
// that follows it reloads from Bungie rather than serving what was there.
func TestInvalidateCache_EvictsTheEntryItRetires(t *testing.T) {
	m := newAnalysis(t, &fakeCatalog{facts: fixtureCatalog()}, &fakeNodes{nodes: fixtureNodes()})
	cachedAnalysis(t, m, 3, "member-1")

	if _, found := m.cache.Get(analysisCacheKey(3, "member-1")); !found {
		t.Fatal("the fixture must start with a cached analysis")
	}

	m.InvalidateCache(3, "member-1")

	if _, found := m.cache.Get(analysisCacheKey(3, "member-1")); found {
		t.Error("the generation advanced but the entry was left in place")
	}
}

// A manifest swap must not evict the per-membership analysis: its expensive
// half is a rate-limited Bungie profile fetch, and the swap only invalidates
// the manifest-derived half. Only a refresh evicts.
func TestOnVersionChanged_DoesNotEvictTheMembershipAnalysis(t *testing.T) {
	m := newAnalysis(t, &fakeCatalog{facts: []items.AcquisitionFacts{}}, &fakeNodes{nodes: fixtureNodes()})
	cachedAnalysis(t, m, 3, "member-1")

	swapped(t, m, "v2")

	if _, found := m.cache.Get(analysisCacheKey(3, "member-1")); !found {
		t.Error("a manifest swap evicted the membership analysis; only a refresh should")
	}
}
