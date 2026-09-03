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

// classifyBucket delegates the source vocabulary to services/sources. Visible
// action wording belongs to services/recommendations.
func classifyBucket(sourceHash uint32, sourceString string) sources.Kind {
	return sources.ActionKind(sourceHash, sourceString)
}
