package collections

import (
	"context"
	"slices"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
)

func TestClassifyDifficulty(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		isExotic bool
		want     string
	}{
		{"raid keyword", "Found in the raid \"Last Wish\"", false, "Challenging"},
		{"vault of glass", "Vault of Glass raid", false, "Challenging"},
		{"kings fall", "King's Fall", false, "Challenging"},
		{"root of nightmares", "Root of Nightmares completion", false, "Challenging"},
		{"crota", "Crota's End", false, "Challenging"},
		{"deep stone", "Deep Stone Crypt", false, "Challenging"},
		{"garden", "Garden of Salvation", false, "Challenging"},
		{"dungeon", "Found in the dungeon", false, "Moderate"},
		{"prophecy", "Prophecy dungeon reward", false, "Moderate"},
		{"grasp", "Grasp of Avarice", false, "Moderate"},
		{"duality", "Duality dungeon", false, "Moderate"},
		{"spire", "Spire of the Watcher", false, "Moderate"},
		{"shattered throne", "The Shattered Throne", false, "Moderate"},
		{"pit of heresy", "Pit of Heresy", false, "Moderate"},
		{"trials", "Trials of Osiris reward", false, "Challenging"},
		{"competitive", "Competitive Crucible", false, "Challenging"},
		{"iron banner", "Iron Banner engrams", false, "Challenging"},
		{"glory", "Glory rank rewards", false, "Challenging"},
		{"nightfall", "Nightfall: The Ordeal", false, "Moderate"},
		{"exotic quest", "Complete the exotic quest", true, "Moderate"},
		{"exotic no source", "", true, "Moderate"},
		{"plain world drop", "World drops", false, "Easy"},
		{"empty non-exotic", "", false, "Easy"},
		{"case insensitive", "FOUND IN THE RAID", false, "Challenging"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyDifficulty(tc.source, tc.isExotic); got != tc.want {
				t.Errorf("classifyDifficulty(%q, %v) = %q, want %q", tc.source, tc.isExotic, got, tc.want)
			}
		})
	}
}

// fabricate builds a CollectibleWithItem (a weapon collectible) for service tests.
func fabricate(colHash, itemHash uint32, name, source string, tier int) manifest.CollectibleWithItem {
	item := &bungie.InventoryItemDefinition{
		Hash:              itemHash,
		DisplayProperties: bungie.DisplayProperties{Name: name},
		ItemType:          bungie.ItemTypeWeapon,
	}
	item.Inventory.TierType = tier
	return manifest.CollectibleWithItem{
		Collectible: bungie.CollectibleDefinition{
			Hash:              colHash,
			ItemHash:          itemHash,
			SourceString:      source,
			DisplayProperties: bungie.DisplayProperties{Name: name},
		},
		Item: item,
	}
}

func TestBuildCategorySummary(t *testing.T) {
	items := []manifest.CollectibleWithItem{
		fabricate(1, 101, "Owned Gun", "World drops", bungie.TierTypeLegendary),
		fabricate(2, 102, "Missing Gun", "Found in the raid", bungie.TierTypeLegendary),
		fabricate(3, 103, "Missing Exotic", "", bungie.TierTypeExotic),
	}
	collected := map[uint32]bool{1: true}

	sum := buildCategorySummary(items, collected)

	if sum.Weapons.Total != 2 || sum.Weapons.Collected != 1 {
		t.Errorf("weapons = %d/%d, want 1/2", sum.Weapons.Collected, sum.Weapons.Total)
	}
	if sum.Exotics.Total != 1 || sum.Exotics.Collected != 0 {
		t.Errorf("exotics = %d/%d, want 0/1", sum.Exotics.Collected, sum.Exotics.Total)
	}
}

func TestLightweight_StripsItems(t *testing.T) {
	full := UserCollections{
		Tree: []CollectionNode{{
			Hash: "10", Total: 1, Items: []string{"100"},
			Children: []CollectionNode{{Hash: "11", Items: []string{"100"}}},
		}},
		Items:           map[string]DestinyItem{"100": {ItemHash: "100"}},
		CollectedHashes: []string{"100"},
		Summary:         CategorySummary{Weapons: CategoryCount{Total: 1}},
	}

	lw := full.Lightweight()

	if lw.Items != nil {
		t.Errorf("Items map not stripped")
	}
	if lw.CollectedHashes != nil {
		t.Errorf("CollectedHashes not stripped")
	}
	if lw.Tree[0].Items != nil || lw.Tree[0].Children[0].Items != nil {
		t.Errorf("node Items not stripped recursively")
	}
	if lw.Tree[0].Total != 1 || lw.Summary.Weapons.Total != 1 {
		t.Errorf("counts/summary must survive Lightweight")
	}
	// Original must be untouched (value-copy contract).
	if full.Items == nil || full.Tree[0].Items == nil || full.CollectedHashes == nil {
		t.Errorf("Lightweight mutated the source")
	}
}

// fakeManifest implements ManifestRepo for service tests.
type fakeManifest struct {
	cols  []manifest.CollectibleWithItem
	nodes map[uint32]*manifest.PresentationNodeDef
}

func (f *fakeManifest) GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error) {
	return f.cols, nil
}
func (f *fakeManifest) GetAllPresentationNodes() (map[uint32]*manifest.PresentationNodeDef, error) {
	return f.nodes, nil
}

// armorCol builds a CollectibleWithItem for a legendary armor piece.
func armorCol(colHash, itemHash uint32, name string) manifest.CollectibleWithItem {
	item := &bungie.InventoryItemDefinition{Hash: itemHash, ItemType: bungie.ItemTypeArmor}
	item.DisplayProperties = bungie.DisplayProperties{Name: name}
	item.Inventory.TierType = bungie.TierTypeLegendary
	return manifest.CollectibleWithItem{
		Collectible: bungie.CollectibleDefinition{Hash: colHash, ItemHash: itemHash,
			DisplayProperties: bungie.DisplayProperties{Name: name}},
		Item: item,
	}
}

func TestGetUserCollections_Projects(t *testing.T) {
	// Build a tree from a minimal two-category fixture: Weapons (node 10) ->
	// Hand Cannons (node 11) holds collectible 1000/item 100 (collected) and
	// 1001/item 101 (missing). Armor (node 20) holds 2000/item 200 (missing).
	fixtureNodes := map[uint32]*manifest.PresentationNodeDef{
		1:  node(1, "Items", []uint32{10, 20}, nil),
		10: node(10, "Weapons", []uint32{11}, nil),
		11: node(11, "Hand Cannons", nil, []uint32{1000, 1001}),
		20: node(20, "Armor", nil, []uint32{2000}),
	}
	fixtureCols := []manifest.CollectibleWithItem{
		col(1000, 100, "Fatebringer"),           // weapon, collected
		col(1001, 101, "The Palindrome"),        // weapon, missing
		armorCol(2000, 200, "Helm of Saint-14"), // armor, missing
	}
	tree := buildTreeStructure(fixtureNodes, fixtureCols)

	// Only Fatebringer (collectible hash 1000) is collected.
	a := &analysis{
		collectibles: fixtureCols,
		collected:    map[uint32]bool{1000: true},
		tree:         tree,
		fetchedAt:    time.Now(),
	}
	c := cache.NewMemoryCache(time.Minute, time.Minute)
	c.Set("collections:3:proj-member", a, time.Minute)
	s := &Service{cache: c}

	result, err := s.GetUserCollections(context.Background(), 3, "proj-member", "token")
	if err != nil {
		t.Fatalf("GetUserCollections: %v", err)
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

func TestGetMissingItemHashes_ExcludesCosmetics(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, time.Minute)
	s := &Service{cache: c}

	weapon := fabricate(1, 100, "Gun", "World drops", bungie.TierTypeLegendary) // item 100
	cosmetic := manifest.CollectibleWithItem{
		Collectible: bungie.CollectibleDefinition{Hash: 2, ItemHash: 200,
			DisplayProperties: bungie.DisplayProperties{Name: "Ship"}},
		Item: &bungie.InventoryItemDefinition{Hash: 200, ItemType: 21,
			DisplayProperties: bungie.DisplayProperties{Name: "Ship"}},
	}
	a := &analysis{
		collectibles: []manifest.CollectibleWithItem{weapon, cosmetic},
		collected:    map[uint32]bool{}, // nothing collected
		fetchedAt:    time.Now(),
	}
	c.Set("collections:3:member-1", a, time.Minute)

	got, err := s.GetMissingItemHashes(context.Background(), 3, "member-1", "token")
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
