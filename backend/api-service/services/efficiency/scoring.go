package efficiency

import (
	"sort"
	"strings"

	"guardian-tracker/api-service/services/sources"
)

const (
	weightExotic    = 3.0
	weightLegendary = 1.0
	weightOther     = 0.5
	wishlistBoost   = 5.0

	multVendorNow = 3.0
	multFeatured  = 1.5
	multBase      = 1.0

	maxCandidates = 6
)

// RankState distinguishes an unavailable index from a complete snapshot with
// no matching candidates.
type RankState string

const (
	RankCold  RankState = "cold"
	RankReady RankState = "ready"
)

// RankInput contains the resolved player and weekly facts used for scoring.
type RankInput struct {
	MissingItemHashes    map[uint32]struct{}
	WishlistItemHashes   map[uint32]struct{}
	LiveAvailability     map[uint32]string
	ActiveMilestoneNames []string
}

// RankedCandidate is a ranked source-bucket fact. Presentation policy and the
// internal score deliberately do not cross this seam.
type RankedCandidate struct {
	SourceHash    uint32
	Label         string
	SourceText    string
	Kind          sources.Kind
	MissingCount  int
	WishlistCount int
	AvailableNow  bool
	Featured      bool
}

// RankResult always carries an allocated Candidates slice.
type RankResult struct {
	State      RankState
	Candidates []RankedCandidate
}

type scoredCandidate struct {
	candidate RankedCandidate
	score     float64
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
	for _, milestone := range activeMilestones {
		if strings.Contains(strings.ToLower(milestone), l) {
			return true
		}
	}
	return false
}

// Rank scores the current complete snapshot. It never waits for an index build:
// cold means no snapshot has ever published, while an older complete snapshot
// remains ready during replacement.
func (e *Engine) Rank(input RankInput) RankResult {
	e.ensureIndex()
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := RankResult{State: RankCold, Candidates: make([]RankedCandidate, 0)}
	if !e.ready {
		return result
	}
	result.State = RankReady
	if len(input.MissingItemHashes) == 0 {
		return result
	}

	scored := make([]scoredCandidate, 0)
	for _, bucket := range e.buckets {
		if bucket.Kind == sources.KindExcluded {
			continue
		}

		candidate := RankedCandidate{
			SourceHash: bucket.SourceHash,
			Label:      bucket.Label,
			SourceText: bucket.SourceString,
			Kind:       bucket.Kind,
			Featured:   labelIsFeatured(bucket.Label, input.ActiveMilestoneNames),
		}
		for _, item := range bucket.Items {
			if _, missing := input.MissingItemHashes[item.ItemHash]; !missing {
				continue
			}
			if _, available := input.LiveAvailability[item.ItemHash]; available {
				candidate.AvailableNow = true
			}
		}

		multiplier := bucketMultiplier(candidate.AvailableNow, candidate.Featured)
		var score float64
		for _, item := range bucket.Items {
			if _, missing := input.MissingItemHashes[item.ItemHash]; !missing {
				continue
			}
			candidate.MissingCount++
			weight := rarityWeight(item.Rarity)
			if _, wished := input.WishlistItemHashes[item.ItemHash]; wished {
				weight *= wishlistBoost
				candidate.WishlistCount++
			}
			score += weight * multiplier
		}
		if candidate.MissingCount != 0 {
			scored = append(scored, scoredCandidate{candidate: candidate, score: score})
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].candidate.SourceHash < scored[j].candidate.SourceHash
	})
	if len(scored) > maxCandidates {
		scored = scored[:maxCandidates]
	}
	result.Candidates = make([]RankedCandidate, len(scored))
	for i := range scored {
		result.Candidates[i] = scored[i].candidate
	}
	return result
}
