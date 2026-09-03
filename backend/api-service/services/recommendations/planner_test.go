package recommendations

import (
	"reflect"
	"testing"

	"guardian-tracker/api-service/services/efficiency"
	"guardian-tracker/api-service/services/sources"
)

type fakeRanker struct {
	result efficiency.RankResult
	input  efficiency.RankInput
	mutate bool
}

func (f *fakeRanker) Rank(input efficiency.RankInput) efficiency.RankResult {
	f.input = input
	if f.mutate {
		delete(input.MissingItemHashes, 1)
		input.WishlistItemHashes[99] = struct{}{}
		input.LiveAvailability[99] = "mutated"
		input.ActiveMilestoneNames[0] = "mutated"
	}
	return f.result
}

func TestRecommendOwnsRankedWordingCoherenceAndEmphasis(t *testing.T) {
	ranker := &fakeRanker{result: efficiency.RankResult{
		State: efficiency.RankReady,
		Candidates: []efficiency.RankedCandidate{
			{SourceHash: 7, Label: "Vault of Glass", SourceText: "Vault of Glass raid", Kind: sources.KindActivity, MissingCount: 2, WishlistCount: 1, AvailableNow: true, Featured: true},
			{SourceHash: 8, Label: "Banshee-44", SourceText: "Banshee-44 gunsmith", Kind: sources.KindVendor, MissingCount: 1, WishlistCount: 1},
		},
	}}
	got := NewPlanner(ranker).Recommend(Input{MissingItemHashes: map[uint32]struct{}{1: {}}})

	want := []Recommendation{
		{ID: "eff-7", Action: "Run Vault of Glass", Explanation: "Fills 2 missing items, 1 on your wishlist — available now", Kind: KindActivity, SourceText: "Vault of Glass raid", Difficulty: sources.Challenging, Emphasis: EmphasisAvailableNow},
		{ID: "eff-8", Action: "Visit Banshee-44", Explanation: "Fills 1 missing item, 1 on your wishlist", Kind: KindVendor, SourceText: "Banshee-44 gunsmith", Difficulty: sources.Easy, Emphasis: EmphasisWishlist},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Recommend() = %+v, want %+v", got, want)
	}
}

func TestRecommendPreservesRankOrderAndIgnoresUnsupportedCandidates(t *testing.T) {
	ranker := &fakeRanker{result: efficiency.RankResult{State: efficiency.RankReady, Candidates: []efficiency.RankedCandidate{
		{SourceHash: 9, Label: "Vendor", SourceText: "vendor", Kind: sources.KindVendor, MissingCount: 1},
		{SourceHash: 1, Label: "Excluded", Kind: sources.KindExcluded, MissingCount: 1},
		{SourceHash: 4, Label: "Crucible", SourceText: "complete crucible", Kind: sources.KindActivity, MissingCount: 1},
	}}}
	got := NewPlanner(ranker).Recommend(Input{})
	if len(got) != 2 || got[0].ID != "eff-9" || got[1].ID != "eff-4" {
		t.Fatalf("rank order = %+v", got)
	}
	if got[0].Emphasis != EmphasisVendor || got[1].Emphasis != EmphasisActivity {
		t.Errorf("default emphases = %+v", got)
	}
}

func TestRecommendXurFallbackPreservesOrderPrecedenceAndCap(t *testing.T) {
	items := make([]XurItem, 7)
	missing := make(map[uint32]struct{})
	wishlist := map[uint32]struct{}{1: {}}
	for i := range items {
		hash := uint32(i + 1)
		items[i] = XurItem{Hash: hash, Name: "Item", Type: "Weapon"}
		missing[hash] = struct{}{}
	}
	got := NewPlanner(&fakeRanker{result: efficiency.RankResult{State: efficiency.RankCold, Candidates: make([]efficiency.RankedCandidate, 0)}}).Recommend(Input{
		MissingItemHashes: missing, WishlistItemHashes: wishlist, XurPresent: true, XurItems: items,
	})

	if len(got) != 5 {
		t.Fatalf("Xûr fallback count = %d, want 5", len(got))
	}
	if got[0].ID != "r-wl-0" || got[0].Emphasis != EmphasisWishlist {
		t.Errorf("missing+wishlisted precedence = %+v", got[0])
	}
	if got[1].ID != "r-xur-1" || got[1].Emphasis != EmphasisXur {
		t.Errorf("missing fallback = %+v", got[1])
	}
	for _, outcome := range got {
		if outcome.Kind != KindVendor || outcome.SourceText != "Xûr" || outcome.Difficulty != sources.Easy || outcome.TimeEstimate != "5 min" {
			t.Errorf("incoherent Xûr outcome: %+v", outcome)
		}
	}
}

func TestRecommendGenericWeeklyFallback(t *testing.T) {
	got := NewPlanner(nil).Recommend(Input{})
	if len(got) != 1 {
		t.Fatalf("fallback count = %d, want 1", len(got))
	}
	want := Recommendation{
		ID: "r-milestones", Action: "Complete weekly milestones before reset",
		Explanation: "Earn pinnacle gear and XP before Tuesday 17:00 UTC",
		Kind:        KindWeekly, Difficulty: sources.Moderate, Emphasis: EmphasisWeekly, TimeEstimate: "2-3 hrs",
	}
	if got[0] != want {
		t.Fatalf("fallback = %+v, want %+v", got[0], want)
	}
}

func TestRecommendDoesNotMutateInput(t *testing.T) {
	input := Input{
		MissingItemHashes:    map[uint32]struct{}{1: {}},
		WishlistItemHashes:   map[uint32]struct{}{2: {}},
		LiveAvailability:     map[uint32]string{3: "Xûr"},
		ActiveMilestoneNames: []string{"Vault of Glass"},
		XurPresent:           true,
		XurItems:             []XurItem{{Hash: 1, Name: "Item", Type: "Weapon"}},
	}
	want := cloneInput(input)
	ranker := &fakeRanker{mutate: true, result: efficiency.RankResult{State: efficiency.RankReady, Candidates: make([]efficiency.RankedCandidate, 0)}}
	got := NewPlanner(ranker).Recommend(input)
	if !reflect.DeepEqual(input, want) {
		t.Fatalf("input mutated: got %+v, want %+v", input, want)
	}
	if len(got) != 1 || got[0].ID != "r-xur-0" {
		t.Fatalf("ranker mutation leaked into fallback: %+v", got)
	}
}
