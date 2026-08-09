// Package sources owns the Destiny source-string vocabulary: the keywords that
// say what an item's acquisition source is, and every classification derived
// from them.
//
// It exists because that vocabulary used to be transcribed into four
// independent tables across three packages — collections (difficulty tiers),
// efficiency (raid/dungeon join, action kinds) and weekly (milestone
// categories). The tables overlapped heavily but nothing kept them in step, so
// shipping a new dungeon meant editing three files and forgetting one produced
// silent wrongness: the raid milestone's missing-count badge simply never
// appeared, with no error and no log.
//
// One faceted table replaces the parallel lists. Each keyword carries its own
// facets, so a fact stated once cannot disagree with itself.
package sources

import "strings"

// Difficulty tiers. Every non-Unrated result is a positive keyword match;
// anything unmatched is honestly Unrated rather than a misleading default.
const (
	Challenging = "Challenging"
	Moderate    = "Moderate"
	Easy        = "Easy"
	Unrated     = "Unrated"
)

// Kind classifies a source as an actionable thing to go and do.
type Kind string

const (
	// KindExcluded is paid, passive, or otherwise not actionable. Items from
	// these sources still count in Collections; they are never recommended.
	KindExcluded Kind = "excluded"
	KindVendor   Kind = "vendor"
	KindActivity Kind = "activity"
)

// entry is one source keyword and everything true about it. Facets live
// together so a keyword cannot be tiered Challenging-because-raid in one place
// while failing the raid/dungeon test in another.
type entry struct {
	keyword string
	tier    string
	// raidDungeon marks loot that drops from a named raid or dungeon. The
	// per-milestone missing-count join is restricted to these, so a milestone
	// name cannot loosely match a short or generic bucket label.
	raidDungeon bool
}

// difficultyTable is scanned in order, first hit wins, so "Grandmaster
// Nightfall" scores Challenging before the Moderate "nightfall" rule. Verified
// against the real manifest source-string histogram (2026-06-30). Extend freely
// — the consistency test in this package guards the facets against drift.
var difficultyTable = []entry{
	// --- Challenging: raids ---
	{"raid", Challenging, true},
	{"vault of glass", Challenging, true},
	{"king's fall", Challenging, true},
	{"root of nightmares", Challenging, true},
	{"crota", Challenging, true},
	{"deep stone", Challenging, true},
	{"garden of salvation", Challenging, true},
	{"last wish", Challenging, true},
	{"vow of the disciple", Challenging, true},
	{"salvation's edge", Challenging, true},
	{"desert perpetual", Challenging, true},
	// --- Challenging: high-end PvP and endgame ---
	{"trials", Challenging, false},
	{"competitive", Challenging, false},
	{"iron banner", Challenging, false},
	{"glory", Challenging, false},
	{"grandmaster", Challenging, false},
	{"pantheon", Challenging, false},
	// --- Moderate: dungeons ---
	{"dungeon", Moderate, true},
	{"prophecy", Moderate, true},
	{"grasp of avarice", Moderate, true},
	{"duality", Moderate, true},
	{"spire of the watcher", Moderate, true},
	{"shattered throne", Moderate, true},
	{"pit of heresy", Moderate, true},
	{"ghosts of the deep", Moderate, true},
	{"warlord", Moderate, true},
	{"vesper", Moderate, true},
	{"sundered doctrine", Moderate, true},
	// --- Moderate: everything else ---
	{"nightfall", Moderate, false},
	{"lost sector", Moderate, false},
	{"exotic quest", Moderate, false},
	{"exotic mission", Moderate, false},
	{"dreaming city", Moderate, false},
	{"season of", Moderate, false},
	{"seasonal", Moderate, false},
	{"episode", Moderate, false},
	{"into the light", Moderate, false},
	{"solstice", Moderate, false},
	{"exploring", Moderate, false},
	{"kepler", Moderate, false},
	{"wellspring", Moderate, false},
	{"dares", Moderate, false},
	{"black armory", Moderate, false},
	{"sparrow racing", Moderate, false},
	// --- Easy ---
	{"rank-up package", Easy, false},
	{"rank up package", Easy, false},
	{"earned while leveling", Easy, false},
	{"engram", Easy, false},
	{"season pass", Easy, false},
	{"eververse", Easy, false},
	{"bright dust", Easy, false},
	{"focusing", Easy, false},
	{"world drop", Easy, false},
	{"complete crucible", Easy, false},
	{"complete gambit", Easy, false},
	{"complete strikes", Easy, false},
	{"complete vanguard", Easy, false},
	{"rank-up", Easy, false},
	{"vendor", Easy, false},
	{"banshee", Easy, false},
	{"ada-1", Easy, false},
	{"xûr", Easy, false},
	{"xur", Easy, false},
	{"tower", Easy, false},
	{"monument", Easy, false},
	{"deluxe edition", Easy, false},
	{"pre-order", Easy, false},
	{"preorder", Easy, false},
	{"charity", Easy, false},
	{"special offer", Easy, false},
	{"rewards pass", Easy, false},
	{"new monarchy", Easy, false},
	{"dead orbit", Easy, false},
	{"future war cult", Easy, false},
	{"saint-14", Easy, false},
}

// unreacquirable short-circuits the tiers: a source that cannot be reacquired
// has no meaningful difficulty. Checked first because reacquirable raid and
// dungeon collectibles never carry this string (verified against the manifest),
// so it cannot steal a real rating.
const unreacquirable = "cannot be reacquired"

// Difficulty infers an acquisition-difficulty estimate from a source string.
func Difficulty(source string) string {
	s := strings.ToLower(strings.TrimSpace(source))
	if s == "" || strings.Contains(s, unreacquirable) {
		return Unrated
	}
	for _, e := range difficultyTable {
		if strings.Contains(s, e.keyword) {
			return e.tier
		}
	}
	return Unrated
}

// IsRaidOrDungeon reports whether a source is named raid or dungeon loot.
//
// Unlike Difficulty this scans the whole table rather than stopping at the
// first hit: "Grandmaster Nightfall" in a dungeon tiers Challenging on
// "grandmaster" but is still dungeon loot.
func IsRaidOrDungeon(source string) bool {
	s := strings.ToLower(source)
	for _, e := range difficultyTable {
		if e.raidDungeon && strings.Contains(s, e.keyword) {
			return true
		}
	}
	return false
}

// excludedSourceHashes are paid/passive/non-actionable sources verified against
// the real manifest (2026-06-23).
var excludedSourceHashes = map[uint32]struct{}{
	860688654:  {}, // Eververse
	4036739795: {}, // Bright Engrams
	1593696611: {}, // Season Pass Reward
	1358645302: {}, // Unlocked by a special offer
	2892963218: {}, // Earned while leveling
	1658014144: {}, // Monument of Triumph (re-acquisition kiosk)
	333761108:  {}, // Rewards Pass
	1118966764: {}, // Shader dismantle
	2387628034: {}, // Random Perks: cannot be reacquired
}

// excludeKeywords catch unmapped paid/passive sources via the source string.
var excludeKeywords = []string{
	"eververse", "bright engram", "season pass", "special offer", "while leveling",
	"monument", "rewards pass", "dismantle", unreacquirable, "silver",
}

// vendorKeywords mark a source as a vendor "buy/visit" action.
var vendorKeywords = []string{"xûr", "xur", "banshee", "ada-1", "gunsmith", "rank-up packages"}

// activityKeywords mark a source as a runnable activity.
var activityKeywords = []string{
	"raid", "dungeon", "trials", "iron banner", "nightfall", "strike", "crucible",
	"gambit", "vault of glass", "exotic", "lost sector", "campaign", "quest",
}

// ActionKind classifies a source bucket as something to do. Exclusion is checked
// first by hash, then by keyword; a source matching nothing is conservatively
// excluded rather than guessed at.
//
// Callers own the wording of the resulting action — this reports the kind only.
func ActionKind(sourceHash uint32, source string) Kind {
	if _, ok := excludedSourceHashes[sourceHash]; ok {
		return KindExcluded
	}
	s := strings.ToLower(source)
	if matchesAny(s, excludeKeywords) {
		return KindExcluded
	}
	if matchesAny(s, vendorKeywords) {
		return KindVendor
	}
	if matchesAny(s, activityKeywords) {
		return KindActivity
	}
	return KindExcluded
}

// MilestoneCategory buckets a weekly milestone by its display name.
func MilestoneCategory(name string) string {
	s := strings.ToLower(name)
	switch {
	case strings.Contains(s, "nightfall") || strings.Contains(s, "ordeal"):
		return "Nightfall"
	case strings.Contains(s, "raid"):
		return "Featured Raid"
	case strings.Contains(s, "dungeon"):
		return "Featured Dungeon"
	case strings.Contains(s, "crucible") || strings.Contains(s, "iron banner") || strings.Contains(s, "trials"):
		return "Crucible"
	case strings.Contains(s, "gambit"):
		return "Gambit"
	case strings.Contains(s, "strike") || strings.Contains(s, "vanguard"):
		return "Strikes"
	default:
		return "Weekly Activity"
	}
}

func matchesAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
