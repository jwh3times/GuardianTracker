package efficiency

import (
	"testing"

	"guardian-tracker/api-service/services/sources"
)

func mkEngine(buckets map[uint32]*Bucket) *Engine {
	return &Engine{buckets: buckets, ready: true}
}

func TestRankDistinguishesColdFromReadyEmpty(t *testing.T) {
	cold := (&Engine{}).Rank(RankInput{})
	if cold.State != RankCold || cold.Candidates == nil || len(cold.Candidates) != 0 {
		t.Fatalf("cold result = %+v, want cold with allocated empty candidates", cold)
	}

	ready := mkEngine(map[uint32]*Bucket{}).Rank(RankInput{})
	if ready.State != RankReady || ready.Candidates == nil || len(ready.Candidates) != 0 {
		t.Fatalf("ready result = %+v, want ready with allocated empty candidates", ready)
	}
}

func TestRankReturnsOrderedFactsWithoutPresentationPolicy(t *testing.T) {
	buckets := map[uint32]*Bucket{
		1: {SourceHash: 1, Label: "Vault of Glass", SourceString: "Vault of Glass raid", Kind: sources.KindActivity, Items: []BucketItem{
			{ItemHash: 101, Rarity: "Legendary"},
			{ItemHash: 102, Rarity: "Exotic"},
		}},
		2: {SourceHash: 2, Label: "Crucible", SourceString: "Complete Crucible", Kind: sources.KindActivity, Items: []BucketItem{
			{ItemHash: 201, Rarity: "Legendary"},
		}},
		3: {SourceHash: 3, Label: "Eververse", SourceString: "Eververse", Kind: sources.KindExcluded, Items: []BucketItem{
			{ItemHash: 301, Rarity: "Legendary"},
		}},
	}
	result := mkEngine(buckets).Rank(RankInput{
		MissingItemHashes: map[uint32]struct{}{101: {}, 102: {}, 201: {}, 301: {}},
	})

	if result.State != RankReady || len(result.Candidates) != 2 {
		t.Fatalf("result = %+v, want two ready candidates", result)
	}
	got := result.Candidates[0]
	if got.SourceHash != 1 || got.Label != "Vault of Glass" || got.SourceText != "Vault of Glass raid" || got.Kind != sources.KindActivity {
		t.Errorf("top candidate identity = %+v", got)
	}
	if got.MissingCount != 2 || got.WishlistCount != 0 || got.AvailableNow || got.Featured {
		t.Errorf("top candidate facts = %+v", got)
	}
}

func TestRankWishlistAvailabilityFeaturedAndTieBreak(t *testing.T) {
	buckets := map[uint32]*Bucket{
		20: {SourceHash: 20, Label: "Trials", SourceString: "Trials", Kind: sources.KindActivity, Items: []BucketItem{{ItemHash: 101, Rarity: "Legendary"}}},
		10: {SourceHash: 10, Label: "Xûr", SourceString: "Xûr", Kind: sources.KindVendor, Items: []BucketItem{{ItemHash: 201, Rarity: "Legendary"}}},
		30: {SourceHash: 30, Label: "Vault of Glass", SourceString: "Vault of Glass raid", Kind: sources.KindActivity, Items: []BucketItem{{ItemHash: 301, Rarity: "Legendary"}}},
	}
	result := mkEngine(buckets).Rank(RankInput{
		MissingItemHashes:    map[uint32]struct{}{101: {}, 201: {}, 301: {}},
		WishlistItemHashes:   map[uint32]struct{}{101: {}},
		LiveAvailability:     map[uint32]string{201: "Xûr"},
		ActiveMilestoneNames: []string{"Featured Vault of Glass"},
	})

	if got := result.Candidates[0]; got.SourceHash != 20 || got.WishlistCount != 1 {
		t.Errorf("wishlist candidate = %+v, want source 20 first", got)
	}
	if got := result.Candidates[1]; got.SourceHash != 10 || !got.AvailableNow {
		t.Errorf("available candidate = %+v, want source 10 second", got)
	}
	if got := result.Candidates[2]; got.SourceHash != 30 || !got.Featured {
		t.Errorf("featured candidate = %+v, want source 30 third", got)
	}

	tied := mkEngine(map[uint32]*Bucket{
		2: {SourceHash: 2, Kind: sources.KindActivity, Items: []BucketItem{{ItemHash: 2, Rarity: "Legendary"}}},
		1: {SourceHash: 1, Kind: sources.KindActivity, Items: []BucketItem{{ItemHash: 1, Rarity: "Legendary"}}},
	}).Rank(RankInput{MissingItemHashes: map[uint32]struct{}{1: {}, 2: {}}})
	if tied.Candidates[0].SourceHash != 1 || tied.Candidates[1].SourceHash != 2 {
		t.Errorf("tie order = %+v, want ascending source hash", tied.Candidates)
	}
}

func TestRankCapsAtSix(t *testing.T) {
	buckets := make(map[uint32]*Bucket)
	missing := make(map[uint32]struct{})
	for hash := uint32(1); hash <= 8; hash++ {
		buckets[hash] = &Bucket{SourceHash: hash, Kind: sources.KindActivity, Items: []BucketItem{{ItemHash: hash, Rarity: "Legendary"}}}
		missing[hash] = struct{}{}
	}
	result := mkEngine(buckets).Rank(RankInput{MissingItemHashes: missing})
	if len(result.Candidates) != 6 {
		t.Fatalf("candidate count = %d, want 6", len(result.Candidates))
	}
	if result.Candidates[5].SourceHash != 6 {
		t.Errorf("last candidate source = %d, want tie-break source 6", result.Candidates[5].SourceHash)
	}
}
