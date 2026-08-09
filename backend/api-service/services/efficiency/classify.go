package efficiency

import (
	"regexp"
	"strings"

	"guardian-tracker/api-service/services/sources"
)

var labelStrip = regexp.MustCompile(`(?i)^source:\s*|["""]|(\s+raid\.?$)|(\s+quest$)|(\s+campaign$)|(\s+dungeon\.?$)`)

func cleanLabel(sourceString string) string {
	s := labelStrip.ReplaceAllString(sourceString, "")
	return strings.TrimSpace(strings.Trim(s, ". "))
}

// classifyBucket turns a source into an action kind plus the wording shown to
// the player. Classifying the source is services/sources' job — it owns the
// vocabulary; phrasing the action is ours.
func classifyBucket(sourceHash uint32, sourceString, label string) (kind, text string) {
	switch k := sources.ActionKind(sourceHash, sourceString); k {
	case sources.KindVendor:
		return string(k), "Visit " + label
	case sources.KindActivity:
		return string(k), "Run " + label
	default:
		return string(sources.KindExcluded), ""
	}
}
