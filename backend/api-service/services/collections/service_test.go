package collections

import (
	"context"
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
