package efficiency

import (
	"strings"

	"guardian-tracker/api-service/services/sources"
)

// MissingForMilestone returns how many of the player's missing items drop from the
// raid/dungeon source bucket(s) this milestone covers, and whether any bucket matched.
// The join is the inverse of labelIsFeatured: a milestone "owns" a bucket when the
// milestone name contains the bucket's cleaned label (case-insensitive). Pure over the
// cached index; never fetches. Returns (0,false) when the index is cold or the name is
// empty. A matched-but-fully-collected milestone returns (0,true).
func (e *Engine) MissingForMilestone(milestoneName string, missing map[uint32]struct{}) (int, bool) {
	e.ensureIndex()
	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.buckets) == 0 || strings.TrimSpace(milestoneName) == "" {
		return 0, false
	}
	name := strings.ToLower(milestoneName)
	count := 0
	matched := false
	for _, b := range e.buckets {
		if b.Kind != "activity" || b.Label == "" || !sources.IsRaidOrDungeon(b.SourceString) {
			continue
		}
		if !strings.Contains(name, strings.ToLower(b.Label)) {
			continue
		}
		matched = true
		for _, it := range b.Items {
			if _, ok := missing[it.ItemHash]; ok {
				count++
			}
		}
	}
	return count, matched
}
