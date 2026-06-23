package efficiency

import (
	"fmt"
	"sort"
	"strings"
)

// Scoring weights — all tunable in one place. No ML, no opaque scoring.
const (
	weightExotic    = 3.0
	weightLegendary = 1.0
	weightOther     = 0.5
	wishlistBoost   = 5.0

	multVendorNow = 3.0 // item buyable right now (time-sensitive)
	multFeatured  = 1.5 // source matches an active milestone this week
	multBase      = 1.0

	maxActions = 6
)

// ScoredAction is one ranked recommendation produced by Rank.
type ScoredAction struct {
	ID            string
	SourceHash    uint32
	Label         string
	SourceString  string
	Kind          string // "activity" | "vendor"
	Text          string // "Run Vault of Glass"
	Why           string // "Fills 12 missing items, 2 on your wishlist"
	MissingCount  int
	WishlistCount int
	AvailableNow  bool
	Featured      bool
	Score         float64
}

func rarityWeight(rarity string) float64 {
	switch rarity {
	case "Exotic":
		return weightExotic
	case "Legendary":
		return weightLegendary
	default:
		return weightOther
	}
}

func bucketMultiplier(availableNow, featured bool) float64 {
	switch {
	case availableNow:
		return multVendorNow
	case featured:
		return multFeatured
	default:
		return multBase
	}
}

func labelIsFeatured(label string, activeMilestones []string) bool {
	l := strings.ToLower(strings.TrimSpace(label))
	if l == "" {
		return false
	}
	for _, m := range activeMilestones {
		if strings.Contains(strings.ToLower(m), l) {
			return true
		}
	}
	return false
}

// Rank scores every recommendable bucket against the player's missing set and
// returns the top actions, highest score first. Pure over the cached index; never
// fetches. Returns nil when the index is empty (caller falls back).
func (e *Engine) Rank(missing, wishlist map[uint32]struct{}, liveVendors map[uint32]string, activeMilestones []string) []ScoredAction {
	e.ensureIndex()
	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.buckets) == 0 || len(missing) == 0 {
		return nil
	}

	var actions []ScoredAction
	for _, b := range e.buckets {
		if b.Kind == "excluded" {
			continue
		}
		featured := labelIsFeatured(b.Label, activeMilestones)

		var missingCount, wishlistCount int
		var availableNow bool
		// First pass: determine availability for the whole bucket's multiplier.
		for _, it := range b.Items {
			if _, ok := missing[it.ItemHash]; !ok {
				continue
			}
			if liveVendors != nil {
				if _, ok := liveVendors[it.ItemHash]; ok {
					availableNow = true
				}
			}
		}
		mult := bucketMultiplier(availableNow, featured)

		var score float64
		for _, it := range b.Items {
			if _, ok := missing[it.ItemHash]; !ok {
				continue
			}
			missingCount++
			w := rarityWeight(it.Rarity)
			if wishlist != nil {
				if _, ok := wishlist[it.ItemHash]; ok {
					w *= wishlistBoost
					wishlistCount++
				}
			}
			score += w * mult
		}
		if missingCount == 0 {
			continue
		}
		actions = append(actions, ScoredAction{
			ID:            fmt.Sprintf("eff-%d", b.SourceHash),
			SourceHash:    b.SourceHash,
			Label:         b.Label,
			SourceString:  b.SourceString,
			Kind:          b.Kind,
			Text:          b.Text,
			Why:           buildWhy(missingCount, wishlistCount, availableNow),
			MissingCount:  missingCount,
			WishlistCount: wishlistCount,
			AvailableNow:  availableNow,
			Featured:      featured,
			Score:         score,
		})
	}

	sort.SliceStable(actions, func(i, j int) bool {
		if actions[i].Score != actions[j].Score {
			return actions[i].Score > actions[j].Score
		}
		return actions[i].SourceHash < actions[j].SourceHash // stable tiebreak
	})
	if len(actions) > maxActions {
		actions = actions[:maxActions]
	}
	return actions
}

func buildWhy(missingCount, wishlistCount int, availableNow bool) string {
	plural := "s"
	if missingCount == 1 {
		plural = ""
	}
	why := fmt.Sprintf("Fills %d missing item%s", missingCount, plural)
	if wishlistCount > 0 {
		why += fmt.Sprintf(", %d on your wishlist", wishlistCount)
	}
	if availableNow {
		why += " — available now"
	}
	return why
}
