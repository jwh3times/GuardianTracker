package sources

import (
	"encoding/json"
	"testing"
)

func TestDescribeBuildsCanonicalAcquisitionSource(t *testing.T) {
	got := Describe(`  Source: "Vault of Glass" Raid  `)
	want := AcquisitionSource{
		Text:        `Source: "Vault of Glass" Raid`,
		Difficulty:  Challenging,
		RaidDungeon: true,
	}
	if got != want {
		t.Fatalf("Describe() = %+v, want %+v", got, want)
	}

	blob, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(blob) != `{"text":"Source: \"Vault of Glass\" Raid","difficulty":"Challenging","raidDungeon":true}` {
		t.Fatalf("JSON = %s", blob)
	}
}

func TestDescribeAllReturnsDeterministicDistinctUnion(t *testing.T) {
	got := DescribeAll([]string{
		"World drops",
		`Source: "Vault of Glass" Raid`,
		"  Exotic quest  ",
		"World drops",
		"",
	})
	want := []AcquisitionSource{
		{Text: "Exotic quest", Difficulty: Moderate},
		{Text: `Source: "Vault of Glass" Raid`, Difficulty: Challenging, RaidDungeon: true},
		{Text: "World drops", Difficulty: Easy},
	}
	if len(got) != len(want) {
		t.Fatalf("DescribeAll() = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("DescribeAll()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// The test that could not exist while the vocabulary lived in four separate
// tables: every keyword tiered as a raid or a dungeon must also answer true to
// IsRaidOrDungeon, and nothing else may.
//
// This is the regression guard for the failure the split caused. Shipping a new
// dungeon used to mean editing three files; forgetting the efficiency one left
// the raid milestone's missing-count badge silently absent — no error, no log.
func TestFacetsAgree_RaidDungeonMatchesTier(t *testing.T) {
	for _, e := range difficultyTable {
		if !e.raidDungeon {
			continue
		}
		// A raid or dungeon is never Easy — if one is tiered that way, either
		// the tier or the facet is wrong.
		if e.tier != Challenging && e.tier != Moderate {
			t.Errorf("keyword %q is raid/dungeon but tiered %q", e.keyword, e.tier)
		}
		if !IsRaidOrDungeon(e.keyword) {
			t.Errorf("keyword %q is flagged raid/dungeon but IsRaidOrDungeon says no", e.keyword)
		}
		if got := Difficulty(e.keyword); got != e.tier {
			t.Errorf("Difficulty(%q) = %q, want its own tier %q", e.keyword, got, e.tier)
		}
	}
}

// Guards the table's ordering contract: first hit wins, so every Challenging
// entry must precede every Moderate one, and every Moderate every Easy.
// "Grandmaster Nightfall" depends on this.
func TestTableOrdering_TiersAreGrouped(t *testing.T) {
	rank := map[string]int{Challenging: 0, Moderate: 1, Easy: 2}
	last := -1
	for _, e := range difficultyTable {
		r, ok := rank[e.tier]
		if !ok {
			t.Fatalf("keyword %q has unknown tier %q", e.keyword, e.tier)
		}
		if r < last {
			t.Fatalf("keyword %q (%s) appears after a lower tier; first-hit-wins is broken", e.keyword, e.tier)
		}
		last = r
	}
}

func TestNoDuplicateKeywords(t *testing.T) {
	seen := map[string]string{}
	for _, e := range difficultyTable {
		if prev, dup := seen[e.keyword]; dup {
			t.Errorf("keyword %q listed twice (%s and %s)", e.keyword, prev, e.tier)
		}
		seen[e.keyword] = e.tier
	}
}

func TestDifficulty(t *testing.T) {
	cases := []struct {
		name, source, want string
	}{
		{"empty", "", Unrated},
		{"unreacquirable short-circuits", "Random Perks: cannot be reacquired", Unrated},
		{"raid keyword", `Found in the raid "Last Wish"`, Challenging},
		{"named raid", "Vault of Glass raid", Challenging},
		{"named dungeon", "Prophecy dungeon reward", Moderate},
		{"trials", "Trials of Osiris reward", Challenging},
		{"nightfall", "Nightfall: The Ordeal", Moderate},
		// Ordering: grandmaster (Challenging) must beat nightfall (Moderate).
		{"grandmaster beats nightfall", "Grandmaster Nightfall", Challenging},
		{"easy vendor", "Rank-up package from Banshee-44", Easy},
		{"unmatched is honest", "Some entirely novel source", Unrated},
	}
	for _, tc := range cases {
		if got := Difficulty(tc.source); got != tc.want {
			t.Errorf("%s: Difficulty(%q) = %q, want %q", tc.name, tc.source, got, tc.want)
		}
	}
}

// IsRaidOrDungeon scans the whole table rather than stopping at the first tier
// hit, so a source that tiers on a non-raid keyword is still recognised.
func TestIsRaidOrDungeon_ScansPastTheFirstTierHit(t *testing.T) {
	if !IsRaidOrDungeon("Grandmaster run of the Duality dungeon") {
		t.Error("a source tiering on 'grandmaster' must still register as dungeon loot")
	}
	if IsRaidOrDungeon("Iron Banner rank-up package") {
		t.Error("Iron Banner is Challenging but is not raid/dungeon loot")
	}
	if IsRaidOrDungeon("") {
		t.Error("empty source is not raid/dungeon loot")
	}
}

func TestActionKind(t *testing.T) {
	cases := []struct {
		name   string
		hash   uint32
		source string
		want   Kind
	}{
		{"excluded by hash", 860688654, "anything at all", KindExcluded},
		{"excluded by keyword", 0, "Purchased from Eververse", KindExcluded},
		{"unreacquirable is excluded", 0, "cannot be reacquired", KindExcluded},
		{"vendor", 0, "Sold by Banshee-44", KindVendor},
		{"activity", 0, "Complete the Vault of Glass raid", KindActivity},
		{"unknown is conservatively excluded", 0, "mystery", KindExcluded},
		// Exclusion wins over activity: a raid item sold in the kiosk is not an
		// action to go and do.
		{"exclusion beats activity", 0, "Monument of Triumph raid rewards", KindExcluded},
	}
	for _, tc := range cases {
		if got := ActionKind(tc.hash, tc.source); got != tc.want {
			t.Errorf("%s: ActionKind(%d, %q) = %q, want %q", tc.name, tc.hash, tc.source, got, tc.want)
		}
	}
}

func TestMilestoneCategory(t *testing.T) {
	cases := map[string]string{
		"Nightfall: The Ordeal": "Nightfall",
		"Vanguard Ordeal":       "Nightfall",
		"Featured Raid: Crota":  "Featured Raid",
		"Weekly Dungeon":        "Featured Dungeon",
		"Crucible Playlist":     "Crucible",
		"Iron Banner":           "Crucible",
		"Trials of Osiris":      "Crucible",
		"Gambit Playlist":       "Gambit",
		"Vanguard Ops":          "Strikes",
		"Weekly Strike":         "Strikes",
		"Clan Rewards":          "Weekly Activity",
	}
	for name, want := range cases {
		if got := MilestoneCategory(name); got != want {
			t.Errorf("MilestoneCategory(%q) = %q, want %q", name, got, want)
		}
	}
}

// A milestone the category function calls a raid must be one IsRaidOrDungeon
// agrees about. These two ran off independent switches before this package, so
// a milestone could be labelled "Featured Raid" while its bucket failed the
// raid/dungeon join — the exact mismatch that dropped the badge.
func TestMilestoneCategoryAgreesWithRaidDungeon(t *testing.T) {
	for _, name := range []string{
		"Featured Raid: Vault of Glass",
		"Weekly Dungeon: Prophecy",
		"Crota's End",
	} {
		cat := MilestoneCategory(name)
		isRD := IsRaidOrDungeon(name)
		if (cat == "Featured Raid" || cat == "Featured Dungeon") && !isRD {
			t.Errorf("%q categorised %q but IsRaidOrDungeon says no", name, cat)
		}
	}
}
