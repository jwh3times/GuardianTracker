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

// fabricate builds a CollectibleWithItem for buildSummary tests.
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

func TestBuildSummary(t *testing.T) {
	s := &Service{}
	items := []manifest.CollectibleWithItem{
		fabricate(1, 101, "Owned Gun", "World drops", bungie.TierTypeLegendary),
		fabricate(2, 102, "Missing Gun", "Found in the raid", bungie.TierTypeLegendary),
		fabricate(3, 103, "Missing Exotic", "", bungie.TierTypeExotic),
	}
	collected := map[uint32]bool{1: true}

	sum := s.buildSummary(items, collected)

	if sum.Total != 3 {
		t.Errorf("Total = %d, want 3", sum.Total)
	}
	if sum.Collected != 1 {
		t.Errorf("Collected = %d, want 1", sum.Collected)
	}
	if len(sum.Missing) != 2 {
		t.Fatalf("len(Missing) = %d, want 2", len(sum.Missing))
	}
	if len(sum.CollectedItems) != 1 {
		t.Fatalf("len(CollectedItems) = %d, want 1", len(sum.CollectedItems))
	}
	if sum.CollectedItems[0].Name != "Owned Gun" {
		t.Errorf("CollectedItems[0].Name = %q", sum.CollectedItems[0].Name)
	}
	if sum.Missing[0].Name != "Missing Gun" || sum.Missing[0].Difficulty != "Challenging" {
		t.Errorf("Missing[0] = %+v", sum.Missing[0])
	}
	if !sum.Missing[1].IsExotic || sum.Missing[1].Rarity != "Exotic" {
		t.Errorf("Missing[1] = %+v, want exotic", sum.Missing[1])
	}
}

func TestGetMissingItemHashes_FromCachedCollections(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, time.Minute)
	s := &Service{cache: c}

	cached := &UserCollections{
		Weapons: CollectionSummary{Missing: []DestinyItem{{ItemHash: "100"}, {ItemHash: "200"}}},
		Armor:   CollectionSummary{Missing: []DestinyItem{{ItemHash: "300"}}},
		Exotics: CollectionSummary{Missing: []DestinyItem{{ItemHash: "400"}}},
	}
	c.Set("collections:3:member-1", cached, time.Minute)

	got, err := s.GetMissingItemHashes(context.Background(), 3, "member-1", "token")
	if err != nil {
		t.Fatalf("GetMissingItemHashes: %v", err)
	}
	for _, want := range []uint32{100, 200, 300, 400} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing hash %d not in result", want)
		}
	}
	if len(got) != 4 {
		t.Errorf("len = %d, want 4", len(got))
	}
}
