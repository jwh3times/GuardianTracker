package efficiency

import (
	"testing"

	"guardian-tracker/api-service/services/sources"
)

func TestCleanLabel(t *testing.T) {
	cases := map[string]string{
		`Source: "Vault of Glass" Raid`:                "Vault of Glass",
		`Source: Vesper's Host`:                        "Vesper's Host",
		`Source: Last Wish raid.`:                      "Last Wish",
		`Source: Complete Trials of Osiris Challenges`: "Complete Trials of Osiris Challenges",
	}
	for in, want := range cases {
		if got := cleanLabel(in); got != want {
			t.Errorf("cleanLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClassifyBucket(t *testing.T) {
	// Excluded by stable sourceHash (Eververse).
	if k := classifyBucket(860688654, "Source: Eververse"); k != sources.KindExcluded {
		t.Errorf("Eververse kind = %q, want excluded", k)
	}
	// Excluded by keyword fallback (season pass — unmapped hash).
	if k := classifyBucket(99999, "Source: Season Pass Reward"); k != sources.KindExcluded {
		t.Errorf("season pass kind = %q, want excluded", k)
	}
	// Activity by keyword.
	if k := classifyBucket(2065138144, `Source: "Vault of Glass" Raid`); k != sources.KindActivity {
		t.Errorf("VoG kind = %q, want activity", k)
	}
	// Vendor by keyword.
	kv := classifyBucket(1788267693, "Source: Earn rank-up packages from Banshee-44.")
	if kv != sources.KindVendor {
		t.Errorf("Banshee kind = %q, want vendor", kv)
	}
}
