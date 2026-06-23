package efficiency

import "testing"

func mkEngine(buckets map[uint32]*Bucket) *Engine {
	return &Engine{buckets: buckets}
}

func TestRankOrdersByScoreAndFiltersEmpty(t *testing.T) {
	buckets := map[uint32]*Bucket{
		1: {SourceHash: 1, Label: "Vault of Glass", Kind: "activity", Text: "Run Vault of Glass", Items: []BucketItem{
			{ItemHash: 101, Rarity: "Legendary"},
			{ItemHash: 102, Rarity: "Exotic"},
		}},
		2: {SourceHash: 2, Label: "Crucible", Kind: "activity", Text: "Run Crucible", Items: []BucketItem{
			{ItemHash: 201, Rarity: "Legendary"},
		}},
		3: {SourceHash: 3, Label: "Eververse", Kind: "excluded", Items: []BucketItem{
			{ItemHash: 301, Rarity: "Legendary"},
		}},
	}
	e := mkEngine(buckets)
	missing := map[uint32]struct{}{101: {}, 102: {}, 201: {}, 301: {}}

	got := e.Rank(missing, nil, nil, nil)

	if len(got) != 2 {
		t.Fatalf("got %d actions, want 2 (excluded bucket dropped)", len(got))
	}
	if got[0].Label != "Vault of Glass" {
		t.Errorf("top action = %q, want Vault of Glass (higher score)", got[0].Label)
	}
	if got[0].MissingCount != 2 {
		t.Errorf("VoG missing count = %d, want 2", got[0].MissingCount)
	}
	if got[1].Label != "Crucible" {
		t.Errorf("second action = %q, want Crucible", got[1].Label)
	}
	if got[0].Why != "Fills 2 missing items" {
		t.Errorf("VoG why = %q, want \"Fills 2 missing items\"", got[0].Why)
	}
}

func TestRankWishlistAndVendorBoost(t *testing.T) {
	buckets := map[uint32]*Bucket{
		1: {SourceHash: 1, Label: "Trials", Kind: "activity", Text: "Run Trials", Items: []BucketItem{{ItemHash: 101, Rarity: "Legendary"}}},
		2: {SourceHash: 2, Label: "Xûr", Kind: "vendor", Text: "Visit Xûr", Items: []BucketItem{{ItemHash: 201, Rarity: "Legendary"}}},
	}
	e := mkEngine(buckets)
	missing := map[uint32]struct{}{101: {}, 201: {}}
	wishlist := map[uint32]struct{}{101: {}}
	liveVendors := map[uint32]string{201: "Xûr"}

	got := e.Rank(missing, wishlist, liveVendors, nil)
	// 101: legendary(1) * wishlist(5) * base(1) = 5
	// 201: legendary(1) * 1 * vendorNow(3) = 3
	if got[0].Label != "Trials" {
		t.Errorf("top = %q, want Trials (wishlist boost)", got[0].Label)
	}
	if !got[1].AvailableNow || got[1].WishlistCount != 0 {
		t.Errorf("Xûr action availability/wishlist wrong: %+v", got[1])
	}
	if got[0].WishlistCount != 1 {
		t.Errorf("Trials wishlist count = %d, want 1", got[0].WishlistCount)
	}
	if got[0].Why != "Fills 1 missing item, 1 on your wishlist" {
		t.Errorf("Trials why = %q", got[0].Why)
	}
	if got[1].Why != "Fills 1 missing item — available now" {
		t.Errorf("Xûr why = %q", got[1].Why)
	}
}
