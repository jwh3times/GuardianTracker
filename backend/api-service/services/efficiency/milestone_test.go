package efficiency

import "testing"

func raidEngine() *Engine {
	return &Engine{buckets: map[uint32]*Bucket{
		1: {SourceHash: 1, Label: "Vault of Glass", SourceString: "Vault of Glass Raid",
			Kind: "activity", Items: []BucketItem{{ItemHash: 100}, {ItemHash: 101}, {ItemHash: 102}}},
		2: {SourceHash: 2, Label: "Banshee-44", SourceString: "Banshee-44 gunsmith",
			Kind: "vendor", Items: []BucketItem{{ItemHash: 200}}},
		3: {SourceHash: 3, Label: "Vault of Glass", SourceString: "Vault of Glass Raid legacy attribution",
			Kind: "activity", Items: []BucketItem{{ItemHash: 100}}},
	}}
}

func TestMissingForMilestone(t *testing.T) {
	e := raidEngine()
	missing := map[uint32]struct{}{100: {}, 102: {}, 200: {}}

	if c, m := e.MissingForMilestone("Vault of Glass", missing); !m || c != 2 {
		t.Errorf("raid match = (%d,%v), want (2,true)", c, m)
	}
	if _, m := e.MissingForMilestone("Banshee-44 Inventory", missing); m {
		t.Error("vendor bucket should not match a milestone count")
	}
	if c, m := e.MissingForMilestone("Vault of Glass", map[uint32]struct{}{}); !m || c != 0 {
		t.Errorf("fully collected = (%d,%v), want (0,true)", c, m)
	}
	if _, m := e.MissingForMilestone("", missing); m {
		t.Error("empty milestone name should not match")
	}
	cold := &Engine{}
	if _, m := cold.MissingForMilestone("Vault of Glass", missing); m {
		t.Error("cold index should not match")
	}
}
