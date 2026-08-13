package collections

import (
	"testing"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
	"guardian-tracker/api-service/services/sources"
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

// anchorFixture mirrors the real-manifest shape: a dominant "Items" root that
// holds the category subtrees, a separate small noise root ("Badges"), and a
// nameless node under Items that must be dropped entirely.
//
//	Items(1)
//	  ├─ Weapons(10) ─ Hand Cannons(11) ─ [col 1000, 1001]
//	  ├─ Armor(20) ─ [col 2000]
//	  └─ ""(30) ─ [col 3000]        (nameless node: skipped, subtree dropped)
//	Badges(90) ─ [col 9000, 9001]  (noise root: not selected as dominant)
func anchorFixture() (map[uint32]*manifest.PresentationNodeDef, []manifest.CollectibleWithItem) {
	nodes := map[uint32]*manifest.PresentationNodeDef{
		1:  node(1, "Items", []uint32{10, 20, 30}, nil),
		10: node(10, "Weapons", []uint32{11}, nil),
		11: node(11, "Hand Cannons", nil, []uint32{1000, 1001}),
		20: node(20, "Armor", nil, []uint32{2000}),
		30: node(30, "", nil, []uint32{3000}), // nameless node -> dropped
		90: node(90, "Badges", nil, []uint32{9000, 9001}),
	}
	cols := []manifest.CollectibleWithItem{
		col(1000, 100, "Fatebringer"),
		col(1001, 101, "The Palindrome"),
		col(2000, 200, "Helm of Saint-14"),
		col(3000, 300, "Hidden Item"),
		col(9000, 900, "Badge A"),
		col(9001, 901, "Badge B"),
	}
	return nodes, cols
}

// TestBuildTreeStructure_AnchorsAtDominantRoot is the core behavior: ts.Roots are
// the dominant root's NAMED children (the in-game categories), name-sorted; the
// dominant root itself ("Items") and the noise root ("Badges") are NOT roots, and
// the nameless node under Items is dropped along with its subtree.
func TestBuildTreeStructure_AnchorsAtDominantRoot(t *testing.T) {
	nodes, cols := anchorFixture()

	ts := buildTreeStructure(nodes, cols)

	// Roots are the dominant root's named children, name-sorted: Armor, Weapons.
	if len(ts.Roots) != 2 {
		t.Fatalf("roots = %d (%+v), want 2 (Armor, Weapons)", len(ts.Roots), rootNames(ts.Roots))
	}
	if ts.Roots[0].Name != "Armor" || ts.Roots[0].Hash != 20 {
		t.Fatalf("roots[0] = %q (hash %d), want Armor (20)", ts.Roots[0].Name, ts.Roots[0].Hash)
	}
	if ts.Roots[1].Name != "Weapons" || ts.Roots[1].Hash != 10 {
		t.Fatalf("roots[1] = %q (hash %d), want Weapons (10)", ts.Roots[1].Name, ts.Roots[1].Hash)
	}

	// "Items" (the dominant root) must NOT appear as a top-level root.
	// "Badges" (the noise root) must NOT appear.
	// "" (the nameless node) must NOT appear.
	for _, r := range ts.Roots {
		if r.Hash == 1 || r.Name == "Items" {
			t.Fatalf("dominant root itself leaked into Roots: %+v", rootNames(ts.Roots))
		}
		if r.Hash == 90 || r.Name == "Badges" {
			t.Fatalf("noise root Badges leaked into Roots: %+v", rootNames(ts.Roots))
		}
		if r.Hash == 30 || r.Name == "" {
			t.Fatalf("nameless node leaked into Roots: %+v", rootNames(ts.Roots))
		}
	}
}

// TestBuildTreeStructure_PreservesNestingUnderCategory: Weapons -> Hand Cannons -> leaves.
func TestBuildTreeStructure_PreservesNestingUnderCategory(t *testing.T) {
	nodes, cols := anchorFixture()
	ts := buildTreeStructure(nodes, cols)

	weapons := findRoot(ts.Roots, "Weapons")
	if weapons == nil {
		t.Fatalf("no Weapons root; roots=%+v", rootNames(ts.Roots))
	}
	if len(weapons.Children) != 1 || weapons.Children[0].Hash != 11 || weapons.Children[0].Name != "Hand Cannons" {
		t.Fatalf("Weapons children = %+v, want single Hand Cannons (11)", weapons.Children)
	}
	hc := weapons.Children[0]
	if len(hc.Leaves) != 2 {
		t.Fatalf("Hand Cannons leaves = %+v, want 2", hc.Leaves)
	}
	gotItems := map[string]bool{}
	for _, lf := range hc.Leaves {
		gotItems[lf.ItemHash] = true
	}
	if !gotItems["100"] || !gotItems["101"] {
		t.Fatalf("Hand Cannons leaf items = %v, want 100 & 101", gotItems)
	}
}

// TestBuildTreeStructure_DropsNamelessNodeSubtree: the nameless node 30 and its
// collectible 3000 (item 300) must not be present anywhere in the tree, though the
// item map still indexes valid collectibles regardless of placement.
func TestBuildTreeStructure_DropsNamelessNodeSubtree(t *testing.T) {
	nodes, cols := anchorFixture()
	ts := buildTreeStructure(nodes, cols)

	// Item 300's collectible lives only under the nameless node -> never placed as a leaf.
	for _, r := range ts.Roots {
		assertNoLeafItem(t, r, "300")
		assertNoNode(t, r, 30)
	}
}

func TestOverlay_RollsUpCounts(t *testing.T) {
	nodes, cols := anchorFixture()
	ts := buildTreeStructure(nodes, cols)

	// Owned (by ITEM hash): Fatebringer(100) + Helm(200).
	got := ts.overlay(map[uint32]bool{100: true, 200: true})

	if len(got) != 2 {
		t.Fatalf("roots = %d, want 2", len(got))
	}

	weapons := findCounted(got, "Weapons")
	if weapons == nil {
		t.Fatalf("no Weapons in overlay; got=%+v", got)
	}
	if weapons.Total != 2 || weapons.Collected != 1 {
		t.Fatalf("Weapons counts = %d/%d, want 1/2", weapons.Collected, weapons.Total)
	}
	if weapons.Hash != "10" || weapons.Name != "Weapons" {
		t.Fatalf("Weapons meta = %+v", weapons)
	}
	hc := weapons.Children[0]
	if hc.Total != 2 || hc.Collected != 1 {
		t.Fatalf("Hand Cannons counts = %d/%d, want 1/2", hc.Collected, hc.Total)
	}
	if len(hc.Items) != 2 {
		t.Fatalf("Hand Cannons items = %+v, want 2", hc.Items)
	}

	armor := findCounted(got, "Armor")
	if armor == nil {
		t.Fatalf("no Armor in overlay; got=%+v", got)
	}
	if armor.Total != 1 || armor.Collected != 1 {
		t.Fatalf("Armor counts = %d/%d, want 1/1", armor.Collected, armor.Total)
	}
	if len(armor.Items) != 1 || armor.Items[0] != "200" {
		t.Fatalf("Armor items = %+v, want [200]", armor.Items)
	}
}

func TestBuildTreeStructure_SkipsNamelessAndUnknownCollectibles(t *testing.T) {
	// Dominant root "Items" -> "Weapons" holds a real collectible, a nameless one
	// (skipped), and an unknown hash (absent from the collectible set).
	nodes := map[uint32]*manifest.PresentationNodeDef{
		1:  node(1, "Items", []uint32{10}, nil),
		10: node(10, "Weapons", nil, []uint32{1000, 1001, 9999}),
	}
	nameless := col(1001, 101, "")                                           // empty name -> skipped
	cols := []manifest.CollectibleWithItem{col(1000, 100, "Real"), nameless} // 9999 absent entirely
	ts := buildTreeStructure(nodes, cols)
	if len(ts.Roots) != 1 {
		t.Fatalf("roots = %+v, want single Weapons root", rootNames(ts.Roots))
	}
	w := ts.Roots[0]
	if w.Name != "Weapons" {
		t.Fatalf("root = %q, want Weapons", w.Name)
	}
	if len(w.Leaves) != 1 || w.Leaves[0].ItemHash != "100" {
		t.Fatalf("leaves = %+v (only the valid collectible should remain)", w.Leaves)
	}
}

// TestBuildTreeStructure_SkipsNamelessNodes: nameless presentation nodes are dropped
// at every depth, taking their subtrees with them.
func TestBuildTreeStructure_SkipsNamelessNodes(t *testing.T) {
	// Items(1) -> Weapons(10) -> [ ""(12) -> col 1200 ] + col 1000
	// The nameless node 12 (and item 120) must vanish; the named Weapons leaf stays.
	nodes := map[uint32]*manifest.PresentationNodeDef{
		1:  node(1, "Items", []uint32{10}, nil),
		10: node(10, "Weapons", []uint32{12}, []uint32{1000}),
		12: node(12, "", nil, []uint32{1200}), // nameless node -> dropped
	}
	cols := []manifest.CollectibleWithItem{
		col(1000, 100, "Real Weapon"),
		col(1200, 120, "Hidden"),
	}
	ts := buildTreeStructure(nodes, cols)

	if len(ts.Roots) != 1 || ts.Roots[0].Name != "Weapons" {
		t.Fatalf("roots = %+v, want single Weapons root", rootNames(ts.Roots))
	}
	w := ts.Roots[0]
	// The nameless child 12 must not be present.
	for _, ch := range w.Children {
		if ch.Hash == 12 || ch.Name == "" {
			t.Fatalf("nameless node 12 leaked under Weapons: %+v", w.Children)
		}
	}
	assertNoNode(t, w, 12)
	assertNoLeafItem(t, w, "120")
	// The real leaf survives.
	if len(w.Leaves) != 1 || w.Leaves[0].ItemHash != "100" {
		t.Fatalf("Weapons leaves = %+v, want only item 100", w.Leaves)
	}
}

// TestOverlay_DedupesDuplicateItemHashWithinNode reproduces the re-issued-item bug:
// the manifest carries two DestinyCollectibleDefinition rows for one itemHash under
// the same presentation node (e.g. Choir of One's two collectible hashes under one
// node). Only one of the two collectible rows is acquired, but ownership is
// itemHash-level, so the node must count this as a single owned leaf, not two leaves
// with only one collected.
func TestOverlay_DedupesDuplicateItemHashWithinNode(t *testing.T) {
	nodes := map[uint32]*manifest.PresentationNodeDef{
		1:  node(1, "Items", []uint32{20}, nil),
		20: node(20, "Armor", nil, []uint32{2000, 2001}),
	}
	cols := []manifest.CollectibleWithItem{
		col(2000, 200, "Choir of One"),
		col(2001, 200, "Choir of One"),
	}
	ts := buildTreeStructure(nodes, cols)

	// Owned (by ITEM hash): item 200 is owned, regardless of which of its two
	// collectible rows the profile response happened to mark acquired.
	got := ts.overlay(map[uint32]bool{200: true})

	armor := findCounted(got, "Armor")
	if armor == nil {
		t.Fatalf("no Armor in overlay; got=%+v", got)
	}
	if armor.Total != 1 || armor.Collected != 1 {
		t.Fatalf("Armor counts = %d/%d, want 1/1 (duplicate itemHash under one node must collapse to a single leaf)", armor.Collected, armor.Total)
	}
	if len(armor.Items) != 1 || armor.Items[0] != "200" {
		t.Fatalf("Armor items = %+v, want [200]", armor.Items)
	}
}

func TestBuildTreeStructure_UnionsLinkedCollectibleAcquisitionSources(t *testing.T) {
	nodes := map[uint32]*manifest.PresentationNodeDef{
		1:  node(1, "Items", []uint32{20}, nil),
		20: node(20, "Weapons", nil, []uint32{2000, 2001, 2002}),
	}
	raid := col(2000, 200, "Fatebringer")
	raid.Collectible.SourceString = "Vault of Glass raid"
	kiosk := col(2001, 200, "Fatebringer")
	kiosk.Collectible.SourceString = "Monument to Lost Lights"
	duplicate := col(2002, 200, "Fatebringer")
	duplicate.Collectible.SourceString = "Vault of Glass raid"

	item := buildTreeStructure(nodes, []manifest.CollectibleWithItem{raid, kiosk, duplicate}).Items["200"]
	want := []sources.AcquisitionSource{
		{Text: "Monument to Lost Lights", Difficulty: sources.Easy},
		{Text: "Vault of Glass raid", Difficulty: sources.Challenging, RaidDungeon: true},
	}
	if len(item.AcquisitionSources) != len(want) {
		t.Fatalf("acquisitionSources = %+v, want %+v", item.AcquisitionSources, want)
	}
	for i := range want {
		if item.AcquisitionSources[i] != want[i] {
			t.Errorf("acquisitionSources[%d] = %+v, want %+v", i, item.AcquisitionSources[i], want[i])
		}
	}
}

// TestOverlay_DedupesDuplicateItemHash_NeitherAcquired covers the un-owned duplicate
// case: two collectible rows for one itemHash under a node, neither acquired, still
// yields exactly one un-collected leaf (not two).
func TestOverlay_DedupesDuplicateItemHash_NeitherAcquired(t *testing.T) {
	nodes := map[uint32]*manifest.PresentationNodeDef{
		1:  node(1, "Items", []uint32{20}, nil),
		20: node(20, "Armor", nil, []uint32{2000, 2001}),
	}
	cols := []manifest.CollectibleWithItem{
		col(2000, 200, "Choir of One"),
		col(2001, 200, "Choir of One"),
	}
	ts := buildTreeStructure(nodes, cols)

	got := ts.overlay(map[uint32]bool{}) // nothing owned

	armor := findCounted(got, "Armor")
	if armor == nil {
		t.Fatalf("no Armor in overlay; got=%+v", got)
	}
	if armor.Total != 1 || armor.Collected != 0 {
		t.Fatalf("Armor counts = %d/%d, want 0/1 (duplicate itemHash under one node must collapse to a single leaf)", armor.Collected, armor.Total)
	}
}

func TestBuildTreeStructure_CycleGuard(t *testing.T) {
	// Dominant root Items(1) -> A(10) -> B(11) -> A(10) (cycle); B holds the collectible.
	// Must terminate and still yield a non-empty forest anchored under Items.
	nodes := map[uint32]*manifest.PresentationNodeDef{
		1:  node(1, "Items", []uint32{10}, nil),
		10: node(10, "A", []uint32{11}, nil),
		11: node(11, "B", []uint32{10}, []uint32{1000}),
	}
	cols := []manifest.CollectibleWithItem{col(1000, 100, "X")}
	ts := buildTreeStructure(nodes, cols) // must terminate
	if len(ts.Roots) == 0 {
		t.Fatalf("expected at least one root, got none")
	}
}

// --- helpers ---

func rootNames(roots []structNode) []string {
	out := make([]string, 0, len(roots))
	for _, r := range roots {
		out = append(out, r.Name)
	}
	return out
}

func findRoot(roots []structNode, name string) *structNode {
	for i := range roots {
		if roots[i].Name == name {
			return &roots[i]
		}
	}
	return nil
}

func findCounted(roots []CollectionNode, name string) *CollectionNode {
	for i := range roots {
		if roots[i].Name == name {
			return &roots[i]
		}
	}
	return nil
}

func assertNoLeafItem(t *testing.T, n structNode, itemHash string) {
	t.Helper()
	for _, lf := range n.Leaves {
		if lf.ItemHash == itemHash {
			t.Fatalf("found forbidden leaf item %q under node %q (hash %d)", itemHash, n.Name, n.Hash)
		}
	}
	for _, ch := range n.Children {
		assertNoLeafItem(t, ch, itemHash)
	}
}

func assertNoNode(t *testing.T, n structNode, hash uint32) {
	t.Helper()
	for _, ch := range n.Children {
		if ch.Hash == hash {
			t.Fatalf("found forbidden node hash %d under %q", hash, n.Name)
		}
		assertNoNode(t, ch, hash)
	}
}
