package efficiency

import "testing"

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
	if k, _ := classifyBucket(860688654, "Source: Eververse", "Eververse"); k != "excluded" {
		t.Errorf("Eververse kind = %q, want excluded", k)
	}
	// Excluded by keyword fallback (season pass — unmapped hash).
	if k, _ := classifyBucket(99999, "Source: Season Pass Reward", "Season Pass Reward"); k != "excluded" {
		t.Errorf("season pass kind = %q, want excluded", k)
	}
	// Activity by keyword.
	k, text := classifyBucket(2065138144, `Source: "Vault of Glass" Raid`, "Vault of Glass")
	if k != "activity" || text != "Run Vault of Glass" {
		t.Errorf("VoG = (%q,%q), want (activity, Run Vault of Glass)", k, text)
	}
	// Vendor by keyword.
	kv, tv := classifyBucket(1788267693, "Source: Earn rank-up packages from Banshee-44.", "Earn rank-up packages from Banshee-44.")
	if kv != "vendor" {
		t.Errorf("Banshee kind = %q, want vendor", kv)
	}
	_ = tv
}
