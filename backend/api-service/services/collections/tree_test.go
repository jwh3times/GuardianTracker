package collections

import (
	"testing"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
)

// node builds a PresentationNodeDef with child nodes and collectibles.
func node(hash uint32, name string, childNodes []uint32, childCols []uint32) *manifest.PresentationNodeDef {
	d := &manifest.PresentationNodeDef{Hash: hash}
	d.DisplayProperties = bungie.DisplayProperties{Name: name, Icon: "/i/" + name + ".png"}
	for _, c := range childNodes {
		d.Children.PresentationNodes = append(d.Children.PresentationNodes, struct {
			PresentationNodeHash uint32 `json:"presentationNodeHash"`
		}{c})
	}
	for _, c := range childCols {
		d.Children.Collectibles = append(d.Children.Collectibles, struct {
			CollectibleHash uint32 `json:"collectibleHash"`
		}{c})
	}
	return d
}

// col builds a CollectibleWithItem (collectibleHash, itemHash, name).
func col(colHash, itemHash uint32, name string) manifest.CollectibleWithItem {
	item := &bungie.InventoryItemDefinition{Hash: itemHash, ItemType: bungie.ItemTypeWeapon}
	item.DisplayProperties = bungie.DisplayProperties{Name: name}
	item.Inventory.TierType = bungie.TierTypeLegendary
	return manifest.CollectibleWithItem{
		Collectible: bungie.CollectibleDefinition{Hash: colHash, ItemHash: itemHash,
			DisplayProperties: bungie.DisplayProperties{Name: name}},
		Item: item,
	}
}

func TestBuildTreeStructure_NestsAndDiscoversRoots(t *testing.T) {
	// Weapons(10) -> HandCannons(11) -> [col 1000]; plus an unrelated Records-only node(99).
	nodes := map[uint32]*manifest.PresentationNodeDef{
		10: node(10, "Weapons", []uint32{11}, nil),
		11: node(11, "Hand Cannons", nil, []uint32{1000}),
		99: node(99, "Triumphs", nil, nil), // no collectibles anywhere below -> excluded
	}
	cols := []manifest.CollectibleWithItem{col(1000, 100, "Fatebringer")}

	ts := buildTreeStructure(nodes, cols)

	if len(ts.Roots) != 1 || ts.Roots[0].Hash != 10 {
		t.Fatalf("roots = %+v, want single root 10", ts.Roots)
	}
	if len(ts.Roots[0].Children) != 1 || ts.Roots[0].Children[0].Hash != 11 {
		t.Fatalf("root children = %+v", ts.Roots[0].Children)
	}
	leaves := ts.Roots[0].Children[0].Leaves
	if len(leaves) != 1 || leaves[0].CollectibleHash != 1000 || leaves[0].ItemHash != "100" {
		t.Fatalf("leaves = %+v", leaves)
	}
	if _, ok := ts.Items["100"]; !ok {
		t.Fatalf("items missing item hash 100: %+v", ts.Items)
	}
}

func TestOverlay_RollsUpCounts(t *testing.T) {
	nodes := map[uint32]*manifest.PresentationNodeDef{
		10: node(10, "Weapons", []uint32{11}, nil),
		11: node(11, "Hand Cannons", nil, []uint32{1000, 2000}),
	}
	cols := []manifest.CollectibleWithItem{col(1000, 100, "A"), col(2000, 200, "B")}
	ts := buildTreeStructure(nodes, cols)

	got := ts.overlay(map[uint32]bool{1000: true}) // collected by COLLECTIBLE hash

	if len(got) != 1 {
		t.Fatalf("roots = %d", len(got))
	}
	root := got[0]
	if root.Total != 2 || root.Collected != 1 {
		t.Fatalf("root counts = %d/%d, want 1/2", root.Collected, root.Total)
	}
	if root.Hash != "10" || root.Name != "Weapons" {
		t.Fatalf("root meta = %+v", root)
	}
	leaf := root.Children[0]
	if leaf.Total != 2 || leaf.Collected != 1 {
		t.Fatalf("leaf counts = %d/%d", leaf.Collected, leaf.Total)
	}
	if len(leaf.Items) != 2 || leaf.Items[0] != "100" {
		t.Fatalf("leaf items = %+v", leaf.Items)
	}
}

func TestBuildTreeStructure_SkipsNamelessAndUnknownCollectibles(t *testing.T) {
	nodes := map[uint32]*manifest.PresentationNodeDef{
		10: node(10, "Weapons", nil, []uint32{1000, 1001, 9999}),
	}
	nameless := col(1001, 101, "")  // empty name -> skipped
	cols := []manifest.CollectibleWithItem{col(1000, 100, "Real"), nameless} // 9999 absent entirely
	ts := buildTreeStructure(nodes, cols)
	if len(ts.Roots) != 1 || len(ts.Roots[0].Leaves) != 1 || ts.Roots[0].Leaves[0].ItemHash != "100" {
		t.Fatalf("leaves = %+v (only the valid collectible should remain)", ts.Roots[0].Leaves)
	}
}

func TestBuildTreeStructure_CycleGuard(t *testing.T) {
	// 10 -> 11 -> 10 (cycle); 11 holds the collectible.
	nodes := map[uint32]*manifest.PresentationNodeDef{
		10: node(10, "A", []uint32{11}, nil),
		11: node(11, "B", []uint32{10}, []uint32{1000}),
	}
	cols := []manifest.CollectibleWithItem{col(1000, 100, "X")}
	ts := buildTreeStructure(nodes, cols) // must terminate
	if len(ts.Roots) == 0 {
		t.Fatalf("expected at least one root, got none")
	}
}
