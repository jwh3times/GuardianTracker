// Package recommendations owns complete, player-facing acquisition
// recommendation outcomes. It is pure policy over facts resolved by callers.
package recommendations

import (
	"fmt"

	"guardian-tracker/api-service/services/efficiency"
	"guardian-tracker/api-service/services/sources"
)

// Kind describes the action a recommendation asks the player to take.
type Kind string

const (
	KindActivity Kind = "activity"
	KindVendor   Kind = "vendor"
	KindWeekly   Kind = "weekly"
)

// Emphasis is the presentation choice attached to a complete recommendation.
type Emphasis string

const (
	EmphasisActivity     Emphasis = "Activity"
	EmphasisVendor       Emphasis = "Vendor"
	EmphasisAvailableNow Emphasis = "Available now"
	EmphasisWishlist     Emphasis = "Wishlist"
	EmphasisXur          Emphasis = "Xur"
	EmphasisWeekly       Emphasis = "Weekly"
)

// XurItem is the ordered subset of Xûr inventory needed by fallback policy.
type XurItem struct {
	Hash uint32
	Name string
	Type string
}

// Input contains resolved, read-only facts. Recommend copies every collection
// before handing any ranking subset to its dependency.
type Input struct {
	MissingItemHashes    map[uint32]struct{}
	WishlistItemHashes   map[uint32]struct{}
	LiveAvailability     map[uint32]string
	ActiveMilestoneNames []string
	XurPresent           bool
	XurItems             []XurItem
}

// Recommendation is a complete action outcome; Weekly only adapts these fields
// to its stable JSON shape.
type Recommendation struct {
	ID           string
	Action       string
	Explanation  string
	Kind         Kind
	SourceText   string
	Difficulty   sources.DifficultyTier
	Emphasis     Emphasis
	TimeEstimate string
}

// Ranker is the pure ranking subset consumed by Planner.
type Ranker interface {
	Rank(input efficiency.RankInput) efficiency.RankResult
}

// Planner turns ranked facts into complete outcomes and owns both fallbacks.
type Planner struct {
	ranker Ranker
}

func NewPlanner(ranker Ranker) *Planner {
	return &Planner{ranker: ranker}
}

// Recommend is pure, never errors, and always returns an allocated non-empty
// result. Ranked order is preserved exactly.
func (p *Planner) Recommend(input Input) []Recommendation {
	owned := cloneInput(input)
	if p != nil && p.ranker != nil {
		ranked := p.ranker.Rank(efficiency.RankInput{
			MissingItemHashes:    cloneSet(owned.MissingItemHashes),
			WishlistItemHashes:   cloneSet(owned.WishlistItemHashes),
			LiveAvailability:     cloneAvailability(owned.LiveAvailability),
			ActiveMilestoneNames: append([]string(nil), owned.ActiveMilestoneNames...),
		})
		if ranked.State == efficiency.RankReady {
			if outcomes := rankedRecommendations(ranked.Candidates); len(outcomes) != 0 {
				return outcomes
			}
		}
	}
	if outcomes := xurFallback(owned); len(outcomes) != 0 {
		return outcomes
	}
	return []Recommendation{{
		ID:           "r-milestones",
		Action:       "Complete weekly milestones before reset",
		Explanation:  "Earn pinnacle gear and XP before Tuesday 17:00 UTC",
		Kind:         KindWeekly,
		Difficulty:   sources.Moderate,
		Emphasis:     EmphasisWeekly,
		TimeEstimate: "2-3 hrs",
	}}
}

func rankedRecommendations(candidates []efficiency.RankedCandidate) []Recommendation {
	out := make([]Recommendation, 0, len(candidates))
	for _, candidate := range candidates {
		kind, verb, emphasis, ok := presentation(candidate.Kind)
		if !ok {
			continue
		}
		if candidate.AvailableNow {
			emphasis = EmphasisAvailableNow
		} else if candidate.WishlistCount > 0 {
			emphasis = EmphasisWishlist
		}
		out = append(out, Recommendation{
			ID:          fmt.Sprintf("eff-%d", candidate.SourceHash),
			Action:      verb + " " + candidate.Label,
			Explanation: explanation(candidate.MissingCount, candidate.WishlistCount, candidate.AvailableNow),
			Kind:        kind,
			SourceText:  candidate.SourceText,
			Difficulty:  sources.Difficulty(candidate.SourceText),
			Emphasis:    emphasis,
		})
	}
	return out
}

func presentation(kind sources.Kind) (Kind, string, Emphasis, bool) {
	switch kind {
	case sources.KindActivity:
		return KindActivity, "Run", EmphasisActivity, true
	case sources.KindVendor:
		return KindVendor, "Visit", EmphasisVendor, true
	default:
		return "", "", "", false
	}
}

func explanation(missingCount, wishlistCount int, availableNow bool) string {
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

func xurFallback(input Input) []Recommendation {
	if !input.XurPresent {
		return make([]Recommendation, 0)
	}
	out := make([]Recommendation, 0, 5)
	for i, item := range input.XurItems {
		if len(out) == 5 {
			break
		}
		if _, wished := input.WishlistItemHashes[item.Hash]; wished {
			out = append(out, Recommendation{
				ID:           fmt.Sprintf("r-wl-%d", i),
				Action:       fmt.Sprintf("Buy %s from Xûr", item.Name),
				Explanation:  fmt.Sprintf("%s — on your wishlist and available now", item.Type),
				Kind:         KindVendor,
				SourceText:   "Xûr",
				Difficulty:   sources.Easy,
				Emphasis:     EmphasisWishlist,
				TimeEstimate: "5 min",
			})
		} else if _, missing := input.MissingItemHashes[item.Hash]; missing {
			out = append(out, Recommendation{
				ID:           fmt.Sprintf("r-xur-%d", i),
				Action:       fmt.Sprintf("Buy %s from Xûr", item.Name),
				Explanation:  fmt.Sprintf("%s — missing from your collection", item.Type),
				Kind:         KindVendor,
				SourceText:   "Xûr",
				Difficulty:   sources.Easy,
				Emphasis:     EmphasisXur,
				TimeEstimate: "5 min",
			})
		}
	}
	return out
}

func cloneInput(input Input) Input {
	return Input{
		MissingItemHashes:    cloneSet(input.MissingItemHashes),
		WishlistItemHashes:   cloneSet(input.WishlistItemHashes),
		LiveAvailability:     cloneAvailability(input.LiveAvailability),
		ActiveMilestoneNames: append([]string(nil), input.ActiveMilestoneNames...),
		XurPresent:           input.XurPresent,
		XurItems:             append([]XurItem(nil), input.XurItems...),
	}
}

func cloneSet(input map[uint32]struct{}) map[uint32]struct{} {
	out := make(map[uint32]struct{}, len(input))
	for hash := range input {
		out[hash] = struct{}{}
	}
	return out
}

func cloneAvailability(input map[uint32]string) map[uint32]string {
	out := make(map[uint32]string, len(input))
	for hash, vendor := range input {
		out[hash] = vendor
	}
	return out
}
