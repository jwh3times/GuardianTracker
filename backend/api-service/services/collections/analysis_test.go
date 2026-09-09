package collections

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/manifest"
	"guardian-tracker/api-service/services/sources"
)

// fakeCatalog stands in for Items. reads counts catalog loads so a test can
// prove a rebuild actually happened rather than silently reusing stale state;
// onRead runs before the result is returned, which is how the fencing tests
// land a manifest swap in the middle of one.
type fakeCatalog struct {
	facts  []items.AcquisitionFacts
	err    error
	reads  int
	onRead func()
}

func (f *fakeCatalog) Catalog(context.Context) ([]items.AcquisitionFacts, error) {
	f.reads++
	if f.onRead != nil {
		f.onRead()
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.facts, nil
}

// fakeNodes stands in for the manifest's presentation-node reader.
type fakeNodes struct {
	nodes map[uint32]*manifest.PresentationNodeDef
	err   error
	reads int
}

func (f *fakeNodes) GetAllPresentationNodes() (map[uint32]*manifest.PresentationNodeDef, error) {
	f.reads++
	if f.err != nil {
		return nil, f.err
	}
	return f.nodes, nil
}

// newAnalysis builds a MembershipAnalysis over the given fakes. The Bungie
// client and manifest service are nil: every test here works from a cached
// analysis or the manifest-derived half, never the cold profile fetch.
func newAnalysis(t *testing.T, catalog CatalogReader, nodes PresentationNodeReader) *MembershipAnalysis {
	t.Helper()
	c := cache.NewMemoryCache(time.Minute, 0)
	t.Cleanup(c.Close)
	return NewMembershipAnalysis(nil, nil, catalog, nodes, c, time.Minute)
}

// cached installs a per-membership analysis stamped with the current
// generation, as a completed load would have left it.
func (m *MembershipAnalysis) cached(membershipType int, membershipID string, a *analysis) *analysis {
	a.builtUnder = m.publication.Begin()
	m.cache.Set(analysisCacheKey(membershipType, membershipID), a, m.cacheTTL)
	return a
}

func TestDestinyItemJSONHasSourceScopedDifficultyOnly(t *testing.T) {
	blob, err := json.Marshal(DestinyItem{
		ItemHash: "100",
		AcquisitionSources: []sources.AcquisitionSource{
			{Text: "Vault of Glass raid", Difficulty: sources.Challenging, RaidDungeon: true},
		},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(blob, &wire); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, exists := wire["difficulty"]; exists {
		t.Fatalf("item JSON must not invent aggregate difficulty: %s", blob)
	}
	if _, exists := wire["sources"]; exists {
		t.Fatalf("legacy text-only sources must not shadow acquisitionSources: %s", blob)
	}
	if _, exists := wire["acquisitionSources"]; !exists {
		t.Fatalf("item JSON is missing acquisitionSources: %s", blob)
	}
}

func TestBuildCategorySummary(t *testing.T) {
	exotic := weapon(103, "Missing Exotic", 3)
	exotic.Category = "exotics"
	catalog := []items.AcquisitionFacts{
		weapon(101, "Owned Gun", 1),
		weapon(102, "Missing Gun", 2),
		exotic,
	}
	owned := map[uint32]bool{101: true}

	sum := buildCategorySummary(catalog, owned)

	if sum.Weapons.Total != 2 || sum.Weapons.Collected != 1 {
		t.Errorf("weapons = %d/%d, want 1/2", sum.Weapons.Collected, sum.Weapons.Total)
	}
	if sum.Exotics.Total != 1 || sum.Exotics.Collected != 0 {
		t.Errorf("exotics = %d/%d, want 0/1", sum.Exotics.Collected, sum.Exotics.Total)
	}
}

// A re-issued item is one catalog entry carrying both of its collectible hashes
// (e.g. Choir of One), so the summary counts the item once — never once per
// collectible row — however many collectibles link to it.
func TestBuildCategorySummary_CountsAReissuedItemOnce(t *testing.T) {
	reissued := weapon(101, "Choir of One", 1, 2)
	reissued.Category = "exotics"

	sum := buildCategorySummary([]items.AcquisitionFacts{reissued}, map[uint32]bool{101: true})

	if sum.Exotics.Total != 1 || sum.Exotics.Collected != 1 {
		t.Errorf("exotics = %d/%d, want 1/1", sum.Exotics.Collected, sum.Exotics.Total)
	}
}

// Ownership is item-level: the profile response marks only the collectible row
// the player actually earned, never every duplicate, so any linked collectible
// being acquired owns the item.
func TestDeriveOwnedItems_AnyLinkedCollectibleOwnsTheItem(t *testing.T) {
	catalog := []items.AcquisitionFacts{
		weapon(100, "Choir of One", 1000, 1001),
		weapon(200, "Unowned", 2000),
	}

	owned := deriveOwnedItems(catalog, map[uint32]bool{1001: true})

	if !owned[100] {
		t.Error("item 100 must be owned via its second linked collectible")
	}
	if owned[200] {
		t.Error("item 200 has no acquired collectible and must not be owned")
	}
}

func TestGetMissingItemHashes_ExcludesCosmetics(t *testing.T) {
	ship := weapon(200, "Ship", 2)
	ship.Category = "cosmetics"
	catalog := []items.AcquisitionFacts{weapon(100, "Gun", 1), ship}

	m := newAnalysis(t, &fakeCatalog{facts: catalog}, &fakeNodes{})
	m.cached(3, "member-1", &analysis{
		catalog:   catalog,
		collected: map[uint32]bool{}, // nothing collected
		owned:     map[uint32]bool{},
		fetchedAt: time.Now(),
	})

	got, err := m.GetMissingItemHashes(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("GetMissingItemHashes: %v", err)
	}
	if _, ok := got[100]; !ok {
		t.Errorf("weapon item 100 should be missing")
	}
	if _, ok := got[200]; ok {
		t.Errorf("cosmetic item 200 must be excluded")
	}
	if len(got) != 1 {
		t.Errorf("len = %d, want 1", len(got))
	}
}

// A re-issued item carries several collectible hashes. Acquiring either one
// owns the item, so it must not show up as missing; acquiring neither leaves it
// missing exactly once.
func TestGetMissingItemHashes_ReissuedItemFollowsItemLevelOwnership(t *testing.T) {
	catalog := []items.AcquisitionFacts{weapon(100, "Choir of One", 1, 2)}

	cases := []struct {
		name        string
		collected   map[uint32]bool
		wantMissing bool
	}{
		{"one collectible acquired", map[uint32]bool{1: true}, false},
		{"neither acquired", map[uint32]bool{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newAnalysis(t, &fakeCatalog{facts: catalog}, &fakeNodes{})
			m.cached(3, "member-dup", &analysis{
				catalog:   catalog,
				collected: tc.collected,
				owned:     deriveOwnedItems(catalog, tc.collected),
				fetchedAt: time.Now(),
			})

			got, err := m.GetMissingItemHashes(context.Background(), 3, "member-dup", "token")
			if err != nil {
				t.Fatalf("GetMissingItemHashes: %v", err)
			}
			if _, missing := got[100]; missing != tc.wantMissing {
				t.Errorf("item 100 missing = %v, want %v", missing, tc.wantMissing)
			}
			if len(got) > 1 {
				t.Errorf("missing set = %v; duplicate collectible rows must not double-count", got)
			}
		})
	}
}
