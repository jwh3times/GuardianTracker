package collections

import (
	"context"
	"encoding/json"
	"slices"
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

func TestLightweight_StripsItems(t *testing.T) {
	full := MembershipCollections{
		Tree: []CollectionNode{{
			Hash: "10", Total: 1, Items: []string{"100"},
			Children: []CollectionNode{{Hash: "11", Items: []string{"100"}}},
		}},
		Items:           map[string]DestinyItem{"100": {ItemHash: "100"}},
		CollectedHashes: []string{"100"},
		AvailableNow:    map[string]string{"100": "Xûr"},
		Summary:         CategorySummary{Weapons: CategoryCount{Total: 1}},
	}

	lw := full.Lightweight()

	if lw.Items != nil {
		t.Errorf("Items map not stripped")
	}
	if lw.CollectedHashes != nil {
		t.Errorf("CollectedHashes not stripped")
	}
	if lw.AvailableNow != nil {
		t.Errorf("AvailableNow not stripped")
	}
	if lw.Tree[0].Items != nil || lw.Tree[0].Children[0].Items != nil {
		t.Errorf("node Items not stripped recursively")
	}
	if lw.Tree[0].Total != 1 || lw.Summary.Weapons.Total != 1 {
		t.Errorf("counts/summary must survive Lightweight")
	}
	// Original must be untouched (value-copy contract).
	if full.Items == nil || full.Tree[0].Items == nil || full.CollectedHashes == nil || full.AvailableNow == nil {
		t.Errorf("Lightweight mutated the source")
	}
}

func TestGetMembershipCollections_Projects(t *testing.T) {
	// Build a tree from a minimal two-category fixture: Weapons (node 10) ->
	// Hand Cannons (node 11) holds collectible 1000/item 100 (collected) and
	// 1001/item 101 (missing). Armor (node 20) holds 2000/item 200 (missing).
	fixtureNodes := map[uint32]*manifest.PresentationNodeDef{
		1:  node(1, "Items", []uint32{10, 20}, nil),
		10: node(10, "Weapons", []uint32{11}, nil),
		11: node(11, "Hand Cannons", nil, []uint32{1000, 1001}),
		20: node(20, "Armor", nil, []uint32{2000}),
	}
	catalog := []items.AcquisitionFacts{
		weapon(100, "Fatebringer", 1000),     // weapon, collected
		weapon(101, "The Palindrome", 1001),  // weapon, missing
		armor(200, "Helm of Saint-14", 2000), // armor, missing
	}
	m := newAnalysis(t, &fakeCatalog{facts: catalog}, &fakeNodes{nodes: fixtureNodes})
	m.cached(3, "proj-member", &analysis{
		catalog:   catalog,
		collected: map[uint32]bool{1000: true}, // only Fatebringer's collectible
		owned:     deriveOwnedItems(catalog, map[uint32]bool{1000: true}),
		tree:      buildTreeStructure(fixtureNodes, catalog),
		fetchedAt: time.Now(),
	})

	result, err := m.GetMembershipCollections(context.Background(), 3, "proj-member", "token")
	if err != nil {
		t.Fatalf("GetMembershipCollections: %v", err)
	}

	// Tree must be non-empty and contain the expected top-level categories
	// (Armor, Weapons — name-sorted) from the dominant "Items" root's children.
	if len(result.Tree) == 0 {
		t.Fatalf("Tree is empty")
	}
	if len(result.Tree) < 2 || result.Tree[0].Name != "Armor" || result.Tree[1].Name != "Weapons" {
		catNames := make([]string, 0, len(result.Tree))
		for _, n := range result.Tree {
			catNames = append(catNames, n.Name)
		}
		t.Errorf("top-level categories = %v, want [Armor Weapons]", catNames)
	}

	// Summary: 2 weapons total, 1 collected; 1 armor total, 0 collected.
	if result.Summary.Weapons.Total != 2 || result.Summary.Weapons.Collected != 1 {
		t.Errorf("summary.weapons = %d/%d, want 1/2", result.Summary.Weapons.Collected, result.Summary.Weapons.Total)
	}
	if result.Summary.Armor.Total != 1 || result.Summary.Armor.Collected != 0 {
		t.Errorf("summary.armor = %d/%d, want 0/1", result.Summary.Armor.Collected, result.Summary.Armor.Total)
	}

	// CollectedHashes must include the owned item's hash (item 100 = "100").
	if !slices.Contains(result.CollectedHashes, "100") {
		t.Errorf("CollectedHashes = %v; expected item hash \"100\" for Fatebringer", result.CollectedHashes)
	}
	// The missing item (item 101) must NOT appear in CollectedHashes.
	if slices.Contains(result.CollectedHashes, "101") {
		t.Errorf("CollectedHashes contains uncollected item 101")
	}

	// Items map must be populated (from the tree's shared item-detail map).
	if len(result.Items) == 0 {
		t.Errorf("Items map is empty; expected item detail entries")
	}
	if _, ok := result.Items["100"]; !ok {
		t.Errorf("Items[\"100\"] missing; map keys = %v", func() []string {
			ks := make([]string, 0, len(result.Items))
			for k := range result.Items {
				ks = append(ks, k)
			}
			return ks
		}())
	}
}

func TestGetMembershipCollections_CollectedHashesAreDeterministicOwnedItemSet(t *testing.T) {
	// Item 300 is a re-issued item with two acquired collectible rows.
	// collectedHashes is an item set, so it appears once and the result is
	// numerically sorted.
	catalog := []items.AcquisitionFacts{
		weapon(300, "Reissued Weapon", 1, 2),
		weapon(100, "Other Weapon", 3),
	}
	collected := map[uint32]bool{1: true, 2: true, 3: true}
	m := newAnalysis(t, &fakeCatalog{facts: catalog}, &fakeNodes{})
	m.cached(3, "member-collected-set", &analysis{
		catalog:   catalog,
		collected: collected,
		owned:     deriveOwnedItems(catalog, collected),
		tree:      &TreeStructure{},
		fetchedAt: time.Now(),
	})

	result, err := m.GetMembershipCollections(context.Background(), 3, "member-collected-set", "token")
	if err != nil {
		t.Fatalf("GetMembershipCollections: %v", err)
	}
	if want := []string{"100", "300"}; !slices.Equal(result.CollectedHashes, want) {
		t.Errorf("CollectedHashes = %v, want deterministic owned-item set %v", result.CollectedHashes, want)
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
