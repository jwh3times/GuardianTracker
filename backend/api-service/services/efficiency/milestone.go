package efficiency

import "strings"

// raidDungeonKeywords mark a source bucket as raid/dungeon loot. The per-milestone
// missing-count join is restricted to these specific, named sources so a milestone
// name cannot loosely match a short/generic bucket label. Verified against the real
// manifest (2026-06-30): the ~10 weekly raid milestones map 1:1 to these buckets.
var raidDungeonKeywords = []string{
	"raid", "dungeon", "vault of glass", "king's fall", "root of nightmares",
	"crota", "deep stone", "garden of salvation", "last wish", "vow of the disciple",
	"salvation's edge", "desert perpetual", "prophecy", "grasp of avarice", "duality",
	"spire of the watcher", "shattered throne", "pit of heresy", "ghosts of the deep",
	"warlord", "sundered doctrine", "vesper",
}

func bucketIsRaidDungeon(sourceString string) bool {
	s := strings.ToLower(sourceString)
	for _, kw := range raidDungeonKeywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

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
		if b.Kind != "activity" || b.Label == "" || !bucketIsRaidDungeon(b.SourceString) {
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
