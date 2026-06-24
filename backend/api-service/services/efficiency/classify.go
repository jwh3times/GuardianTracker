package efficiency

import (
	"regexp"
	"strings"
)

// excludedSourceHashes are paid/passive/non-actionable sources verified against the
// real manifest (2026-06-23). Items from these still count in Collections; they are
// never recommended as "go do this" actions.
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
	"monument", "rewards pass", "dismantle", "cannot be reacquired", "silver",
}

// vendorKeywords mark a source as a vendor "buy/visit" action.
var vendorKeywords = []string{"xûr", "xur", "banshee", "ada-1", "gunsmith", "rank-up packages"}

// activityKeywords mark a source as a runnable activity.
var activityKeywords = []string{
	"raid", "dungeon", "trials", "iron banner", "nightfall", "strike", "crucible",
	"gambit", "vault of glass", "exotic", "lost sector", "campaign", "quest",
}

var labelStrip = regexp.MustCompile(`(?i)^source:\s*|["""]|(\s+raid\.?$)|(\s+quest$)|(\s+campaign$)|(\s+dungeon\.?$)`)

func cleanLabel(sourceString string) string {
	s := labelStrip.ReplaceAllString(sourceString, "")
	return strings.TrimSpace(strings.Trim(s, ". "))
}

func classifyBucket(sourceHash uint32, sourceString, label string) (kind, text string) {
	if _, ok := excludedSourceHashes[sourceHash]; ok {
		return "excluded", ""
	}
	ls := strings.ToLower(sourceString)
	for _, kw := range excludeKeywords {
		if strings.Contains(ls, kw) {
			return "excluded", ""
		}
	}
	for _, kw := range vendorKeywords {
		if strings.Contains(ls, kw) {
			return "vendor", "Visit " + label
		}
	}
	for _, kw := range activityKeywords {
		if strings.Contains(ls, kw) {
			return "activity", "Run " + label
		}
	}
	// Conservative default: not a clearly actionable source.
	return "excluded", ""
}
