package efficiency

import (
	"testing"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
)

type fakeSource struct {
	rows []manifest.CollectibleWithItem
}

func (f fakeSource) GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error) {
	return f.rows, nil
}

type fakeVersion struct{ v string }

func (f fakeVersion) Version() string { return f.v }

func item(hash int, tier int) *bungie.InventoryItemDefinition {
	d := &bungie.InventoryItemDefinition{Hash: uint32(hash)}
	d.Inventory.TierType = tier
	return d
}

func TestBuildIndexGroupsBySourceHash(t *testing.T) {
	src := fakeSource{rows: []manifest.CollectibleWithItem{
		{Collectible: bungie.CollectibleDefinition{Hash: 1, ItemHash: 101, SourceHash: 2065138144, SourceString: `Source: "Vault of Glass" Raid`}, Item: item(101, 5)},
		{Collectible: bungie.CollectibleDefinition{Hash: 2, ItemHash: 102, SourceHash: 2065138144, SourceString: `Source: "Vault of Glass" Raid`}, Item: item(102, 6)},
		{Collectible: bungie.CollectibleDefinition{Hash: 4, ItemHash: 101, SourceHash: 2065138144, SourceString: `Source: "Vault of Glass" Raid`}, Item: item(101, 5)},
		{Collectible: bungie.CollectibleDefinition{Hash: 3, ItemHash: 103, SourceHash: 860688654, SourceString: "Source: Eververse"}, Item: item(103, 5)},
	}}
	e := NewEngine(src, fakeVersion{v: "v1"})
	e.BuildIndex()

	if !e.IsReady() {
		t.Fatal("index not ready after build")
	}
	vog := e.buckets[2065138144]
	if vog == nil || len(vog.Items) != 2 {
		t.Fatalf("VoG bucket = %+v, want 2 distinct item hashes", vog)
	}
	if vog.Kind != "activity" || vog.Label != "Vault of Glass" {
		t.Errorf("VoG bucket kind/label = %q/%q", vog.Kind, vog.Label)
	}
	if vog.Items[1].Rarity != "Exotic" {
		t.Errorf("item 102 rarity = %q, want Exotic", vog.Items[1].Rarity)
	}
	if ev := e.buckets[860688654]; ev == nil || ev.Kind != "excluded" {
		t.Errorf("Eververse bucket should exist and be excluded, got %+v", ev)
	}
}

func TestBuildIndexPublishesCompleteEmptySnapshot(t *testing.T) {
	e := NewEngine(fakeSource{}, fakeVersion{v: "v1"})
	e.BuildIndex()
	if !e.IsReady() {
		t.Fatal("complete empty index must be ready, not cold")
	}
	result := e.Rank(RankInput{MissingItemHashes: map[uint32]struct{}{1: {}}})
	if result.State != RankReady || result.Candidates == nil || len(result.Candidates) != 0 {
		t.Fatalf("empty snapshot result = %+v", result)
	}
}
