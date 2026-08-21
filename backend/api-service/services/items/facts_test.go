package items

import (
	"context"
	"testing"

	"guardian-tracker/api-service/services/bungie"
)

const farmSource = "Source: Vault of Glass. This item cannot be reacquired."

func TestRepresentativeFarmOnly(t *testing.T) {
	tests := []struct {
		name          string
		cols          []bungie.CollectibleDefinition
		wantFarmOnly  bool
		wantDisagreed bool
	}{
		{
			name: "no collectibles",
		},
		{
			name:         "single farm-only collectible",
			cols:         []bungie.CollectibleDefinition{col(1, 1, farmSource)},
			wantFarmOnly: true,
		},
		{
			name: "single reacquirable collectible",
			cols: []bungie.CollectibleDefinition{col(1, 1, "Source: Trials of Osiris")},
		},
		{
			// The whole population verified against a real manifest: every
			// farm-only item has exactly one collectible, so multi-collectible
			// items agree by never being farm-only at all.
			name: "two reacquirable collectibles agree",
			cols: []bungie.CollectibleDefinition{
				col(90, 1, "Into the Light"),
				col(20, 1, "Season of the Wish"),
			},
		},
		{
			// The watched case. Representative is the LOWEST hash, so the
			// answer is deterministic even though the inputs conflict.
			name: "collectibles disagree, lowest hash wins",
			cols: []bungie.CollectibleDefinition{
				col(90, 1, farmSource),
				col(20, 1, "Season of the Wish"),
			},
			wantFarmOnly:  false,
			wantDisagreed: true,
		},
		{
			name: "collectibles disagree, lowest hash is the farm-only one",
			cols: []bungie.CollectibleDefinition{
				col(90, 1, "Season of the Wish"),
				col(20, 1, farmSource),
			},
			wantFarmOnly:  true,
			wantDisagreed: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			farmOnly, disagreed := representativeFarmOnly(tc.cols)
			if farmOnly != tc.wantFarmOnly {
				t.Errorf("farmOnly = %v, want %v", farmOnly, tc.wantFarmOnly)
			}
			if disagreed != tc.wantDisagreed {
				t.Errorf("disagreed = %v, want %v", disagreed, tc.wantDisagreed)
			}
		})
	}
}

// Representative choice must not depend on the order the rows arrive in — that
// dependence is exactly what the unordered manifest SELECT used to introduce.
func TestRepresentativeFarmOnly_IndependentOfInputOrder(t *testing.T) {
	a := col(20, 1, farmSource)
	b := col(90, 1, "Season of the Wish")

	forward, _ := representativeFarmOnly([]bungie.CollectibleDefinition{a, b})
	reverse, _ := representativeFarmOnly([]bungie.CollectibleDefinition{b, a})
	if forward != reverse {
		t.Errorf("farmOnly depends on row order: %v vs %v", forward, reverse)
	}
	if !forward {
		t.Error("lowest-hash representative should be the farm-only one")
	}
}

// A disagreement must not fail the request. Serving Collections a deterministic
// answer beats refusing to serve it over a manifest data anomaly; the condition
// is surfaced by logging instead.
func TestLookup_DisagreementStillServesADeterministicAnswer(t *testing.T) {
	svc := NewService(&factsRepo{
		items: map[uint32]*bungie.InventoryItemDefinition{1: weapon(1, "Contested")},
		cols: map[uint32][]bungie.CollectibleDefinition{1: {
			col(90, 1, farmSource),
			col(20, 1, "Season of the Wish"),
		}},
	})

	got, err := svc.Lookup(context.Background(), []uint32{1})
	if err != nil {
		t.Fatalf("Lookup returned an error for a manifest anomaly: %v", err)
	}
	if got[1].FarmOnly {
		t.Error("FarmOnly = true, want the lowest-hash representative's false")
	}
}

func TestBuildFacts_ProjectsCanonicalFields(t *testing.T) {
	item := weapon(7, "Gjallarhorn")
	item.DisplayProperties.Description = "desc"
	item.DisplayProperties.Icon = "/icon.png"

	f := buildFacts(item, []bungie.CollectibleDefinition{col(3, 7, "Raid")})

	if f.ItemHash != 7 || f.Name != "Gjallarhorn" || f.Description != "desc" || f.Icon != "/icon.png" {
		t.Errorf("facts = %+v", f)
	}
	if !f.IsExotic {
		t.Error("IsExotic = false, want true for an exotic tier")
	}
	if f.Rarity == "" {
		t.Error("Rarity is empty")
	}
	if f.Category == "" {
		t.Error("Category is empty, want the collection rollup bucket")
	}
}
