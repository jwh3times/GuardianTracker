package collections

import (
	"context"
	"slices"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
	"guardian-tracker/api-service/services/sources"
)

// The keyword vocabulary is owned and exhaustively tested by services/sources.
// What matters here is that this package's seam delegates to it rather than
// carrying a second copy that can drift.
func TestClassifyDifficulty_DelegatesToSources(t *testing.T) {
	for _, source := range []string{
		`Source: "Vault of Glass" Raid`,
		"Source: Complete strikes",
		"Grandmaster Nightfall",
		"Random Perks: This item cannot be reacquired from Collection",
		"",
	} {
		if got, want := ClassifyDifficulty(source, false), sources.Difficulty(source); got != want {
			t.Errorf("ClassifyDifficulty(%q) = %q, want sources.Difficulty = %q", source, got, want)
		}
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
	owned := map[uint32]bool{101: true}

	sum := buildCategorySummary(items, owned)

	if sum.Weapons.Total != 2 || sum.Weapons.Collected != 1 {
		t.Errorf("weapons = %d/%d, want 1/2", sum.Weapons.Collected, sum.Weapons.Total)
	}
	if sum.Exotics.Total != 1 || sum.Exotics.Collected != 0 {
		t.Errorf("exotics = %d/%d, want 0/1", sum.Exotics.Collected, sum.Exotics.Total)
	}
}

// TestBuildCategorySummary_DedupesDuplicateItemHash reproduces the re-issued-item
// bug: two collectible rows share one itemHash (e.g. Choir of One's two collectible
// hashes). Only one row is acquired, but ownership is itemHash-level and Total must
// count the distinct item once, not once per collectible row.
func TestBuildCategorySummary_DedupesDuplicateItemHash(t *testing.T) {
	items := []manifest.CollectibleWithItem{
		fabricate(1, 101, "Choir of One", "Found in the raid", bungie.TierTypeExotic),
		fabricate(2, 101, "Choir of One", "Found in the raid", bungie.TierTypeExotic),
	}
	owned := map[uint32]bool{101: true}

	sum := buildCategorySummary(items, owned)

	if sum.Exotics.Total != 1 || sum.Exotics.Collected != 1 {
		t.Errorf("exotics = %d/%d, want 1/1", sum.Exotics.Collected, sum.Exotics.Total)
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
	fixtureCols := []manifest.CollectibleWithItem{
		col(1000, 100, "Fatebringer"),           // weapon, collected
		col(1001, 101, "The Palindrome"),        // weapon, missing
		armorCol(2000, 200, "Helm of Saint-14"), // armor, missing
	}
	tree := buildTreeStructure(fixtureNodes, fixtureCols)

	// Only Fatebringer (collectible hash 1000, item hash 100) is collected.
	a := &analysis{
		collectibles: fixtureCols,
		collected:    map[uint32]bool{1000: true},
		owned:        map[uint32]bool{100: true},
		tree:         tree,
		fetchedAt:    time.Now(),
	}
	c := cache.NewMemoryCache(time.Minute, 0)
	c.Set("collections:3:proj-member", a, time.Minute)
	s := &Service{cache: c}

	result, err := s.GetMembershipCollections(context.Background(), 3, "proj-member", "token")
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
	// Two acquired collectible rows point to item 300. collectedHashes is an
	// item set, so item 300 appears once and the result is numerically sorted.
	duplicateA := fabricate(1, 300, "Reissued Weapon", "World drops", bungie.TierTypeLegendary)
	duplicateB := fabricate(2, 300, "Reissued Weapon", "World drops", bungie.TierTypeLegendary)
	other := fabricate(3, 100, "Other Weapon", "World drops", bungie.TierTypeLegendary)
	a := &analysis{
		collectibles: []manifest.CollectibleWithItem{duplicateA, duplicateB, other},
		collected:    map[uint32]bool{1: true, 2: true, 3: true},
		owned:        map[uint32]bool{300: true, 100: true},
		tree:         &TreeStructure{},
		fetchedAt:    time.Now(),
	}
	c := cache.NewMemoryCache(time.Minute, 0)
	c.Set("collections:3:member-collected-set", a, time.Minute)
	s := &Service{cache: c}

	result, err := s.GetMembershipCollections(context.Background(), 3, "member-collected-set", "token")
	if err != nil {
		t.Fatalf("GetMembershipCollections: %v", err)
	}
	if want := []string{"100", "300"}; !slices.Equal(result.CollectedHashes, want) {
		t.Errorf("CollectedHashes = %v, want deterministic owned-item set %v", result.CollectedHashes, want)
	}
}

func TestToDestinyItem_FarmOnly(t *testing.T) {
	item := &bungie.InventoryItemDefinition{Hash: 1, ItemType: bungie.ItemTypeWeapon, ItemSubType: 9}
	item.Inventory.TierType = bungie.TierTypeLegendary
	item.DisplayProperties.Name = "Test Cannon"

	cwi := &manifest.CollectibleWithItem{
		Collectible: bungie.CollectibleDefinition{
			ItemHash:     1,
			SourceString: "Random Perks: This item cannot be reacquired from Collection",
		},
		Item: item,
	}
	di := toDestinyItem(cwi)
	if !di.FarmOnly {
		t.Error("FarmOnly = false, want true for a 'cannot be reacquired' source")
	}
	if di.Difficulty != "Unrated" {
		t.Errorf("Difficulty = %q, want Unrated", di.Difficulty)
	}

	cwi.Collectible.SourceString = "Source: World drops"
	if toDestinyItem(cwi).FarmOnly {
		t.Error("FarmOnly = true, want false for a normal source")
	}
}

func TestToDestinyItem_ItemType(t *testing.T) {
	mkCWI := func(itemType, itemSubType int) *manifest.CollectibleWithItem {
		item := &bungie.InventoryItemDefinition{Hash: 1, ItemType: itemType, ItemSubType: itemSubType}
		item.Inventory.TierType = bungie.TierTypeLegendary
		item.DisplayProperties.Name = "Test Item"
		return &manifest.CollectibleWithItem{
			Collectible: bungie.CollectibleDefinition{Hash: 1, ItemHash: 1},
			Item:        item,
		}
	}

	cases := []struct {
		name        string
		itemType    int
		itemSubType int
		wantType    string
	}{
		{"shader (19,20)", bungie.ItemTypeMod, bungie.ItemSubTypeShader, "Shader"},
		{"ornament (19,21)", bungie.ItemTypeMod, bungie.ItemSubTypeOrnament, "Ornament"},
		{"finisher (29)", bungie.ItemTypeFinisher, 0, "Finisher"},
		{"ghost (24)", 24, 0, "Ghost"},
		{"ship (21)", 21, 0, "Ship"},
		{"emblem (14)", 14, 0, "Emblem"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			di := toDestinyItem(mkCWI(tc.itemType, tc.itemSubType))
			if di.ItemType != tc.wantType {
				t.Errorf("toDestinyItem ItemType = %q, want %q", di.ItemType, tc.wantType)
			}
		})
	}
}

func TestGetMissingItemHashes_ExcludesCosmetics(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, 0)
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
		owned:        map[uint32]bool{},
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

// TestGetMissingItemHashes_DedupesDuplicateItemHash reproduces the re-issued-item
// bug: two collectible rows share one itemHash. When either is acquired, ownership
// is itemHash-level, so the item must NOT show up as missing. When neither is
// acquired, the item hash appears in the missing set exactly once (map semantics).
func TestGetMissingItemHashes_DedupesDuplicateItemHash(t *testing.T) {
	weaponA := fabricate(1, 100, "Choir of One", "Found in the raid", bungie.TierTypeExotic)
	weaponB := fabricate(2, 100, "Choir of One", "Found in the raid", bungie.TierTypeExotic)

	t.Run("one acquired", func(t *testing.T) {
		c := cache.NewMemoryCache(time.Minute, 0)
		s := &Service{cache: c}
		a := &analysis{
			collectibles: []manifest.CollectibleWithItem{weaponA, weaponB},
			collected:    map[uint32]bool{1: true}, // only collectible 1 acquired
			owned:        map[uint32]bool{100: true},
			fetchedAt:    time.Now(),
		}
		c.Set("collections:3:member-dup-1", a, time.Minute)

		got, err := s.GetMissingItemHashes(context.Background(), 3, "member-dup-1", "token")
		if err != nil {
			t.Fatalf("GetMissingItemHashes: %v", err)
		}
		if _, ok := got[100]; ok {
			t.Errorf("item 100 owned via one of its duplicate collectibles must not be missing")
		}
	})

	t.Run("neither acquired", func(t *testing.T) {
		c := cache.NewMemoryCache(time.Minute, 0)
		s := &Service{cache: c}
		a := &analysis{
			collectibles: []manifest.CollectibleWithItem{weaponA, weaponB},
			collected:    map[uint32]bool{},
			owned:        map[uint32]bool{},
			fetchedAt:    time.Now(),
		}
		c.Set("collections:3:member-dup-2", a, time.Minute)

		got, err := s.GetMissingItemHashes(context.Background(), 3, "member-dup-2", "token")
		if err != nil {
			t.Fatalf("GetMissingItemHashes: %v", err)
		}
		if _, ok := got[100]; !ok {
			t.Errorf("item 100 should be missing")
		}
		if len(got) != 1 {
			t.Errorf("len = %d, want 1 (duplicate collectible rows for one itemHash must not double-count)", len(got))
		}
	})
}
