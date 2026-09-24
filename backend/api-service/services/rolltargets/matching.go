package rolltargets

import (
	"context"
	"sort"

	"guardian-tracker/api-service/services/ownedrolls"
)

// Match is one weapon the player owns that satisfies one of their saved rolls.
type Match struct {
	Target StoredTarget
	Roll   ownedrolls.OwnedRoll
}

// NearMiss is the best owned copy for a target that nothing fully satisfies.
// MatchedPerks is the subset of the target's perks present on that copy. It is
// carried explicitly so consumers never have to recreate the scoring rule.
type NearMiss struct {
	Roll         ownedrolls.OwnedRoll
	MatchedPerks []string
}

// UnmatchedTarget is a saved target that no owned copy fully satisfies. A
// target with no partially matching copy still appears, with NearMiss nil.
type UnmatchedTarget struct {
	Target   StoredTarget
	NearMiss *NearMiss
}

// MatchReport is the complete answer to "which of my weapons match what I
// saved".
//
// Unmatched targets are carried, not dropped. A saved roll that nothing
// satisfies is the most useful thing the feature has to say — it is the roll
// still worth chasing — and reporting only matches would hide it.
type MatchReport struct {
	// Wanted are owned weapons satisfying a roll the player wants.
	Wanted []Match

	// Unwanted are owned weapons satisfying a roll the player marked unwanted:
	// copies they can dismantle.
	Unwanted []Match

	// UnmatchedTargets are saved rolls that nothing owned satisfies, together
	// with the best partial copy when at least one target perk is present.
	UnmatchedTargets []UnmatchedTarget
}

// OwnedRollReader supplies the weapons the membership currently holds.
// Satisfied by *ownedrolls.Service.
//
// Its failures pass through rather than being flattened: an inventory that
// could not be read is not an empty inventory, and must not render as "nothing
// you own matches".
type OwnedRollReader interface {
	Read(ctx context.Context, membershipType int, membershipID string) ([]ownedrolls.OwnedRoll, error)
}

// Matches joins the membership's saved roll targets to the weapons it owns.
//
// A target matches an owned weapon when **every** perk the target names is
// currently in that weapon's perk columns. Extra perks on the weapon do not
// prevent a match — a target says what must be there, not what must not.
//
// An any-weapon target is tested against every owned weapon, which is what
// makes it a wildcard rather than a target with a missing field.
func (s *Service) Matches(ctx context.Context, membershipType int, membershipID string) (MatchReport, error) {
	targets, err := s.repo.List(ctx, membershipID)
	if err != nil {
		return MatchReport{}, err
	}
	if len(targets) == 0 {
		return MatchReport{}, nil
	}
	if s.owned == nil {
		return MatchReport{}, ErrOwnedRollsUnavailable
	}
	rolls, err := s.owned.Read(ctx, membershipType, membershipID)
	if err != nil {
		return MatchReport{}, err
	}

	report := MatchReport{}
	for _, target := range targets {
		matched := false
		var best *NearMiss
		for _, roll := range rolls {
			if !targetApplies(target, roll) {
				continue
			}
			matchedPerks := matchingPerks(roll.Perks, target.Perks)
			if len(target.Perks) == 0 || len(matchedPerks) != len(target.Perks) {
				// Strictly greater preserves the owned-copy order for ties. The
				// reader guarantees that order by item hash then instance id.
				if len(matchedPerks) > 0 && (best == nil || len(matchedPerks) > len(best.MatchedPerks)) {
					best = &NearMiss{Roll: roll, MatchedPerks: matchedPerks}
				}
				continue
			}
			matched = true
			m := Match{Target: target, Roll: roll}
			if target.Wanted {
				report.Wanted = append(report.Wanted, m)
			} else {
				report.Unwanted = append(report.Unwanted, m)
			}
		}
		if !matched {
			report.UnmatchedTargets = append(report.UnmatchedTargets, UnmatchedTarget{
				Target:   target,
				NearMiss: best,
			})
		}
	}
	return report, nil
}

// targetApplies reports whether a target is about this weapon at all.
func targetApplies(target StoredTarget, roll ownedrolls.OwnedRoll) bool {
	if target.AnyWeapon() {
		return true
	}
	return *target.ItemHash == roll.ItemHash
}

// matchingPerks returns the target perks present on an owned copy. Both inputs
// use canonical Manifest spelling and sorted order; the result follows the
// target's order so it is stable on the wire.
func matchingPerks(owned, wanted []string) []string {
	var matched []string
	for _, perk := range wanted {
		if contains(owned, perk) {
			matched = append(matched, perk)
		}
	}
	return matched
}

func contains(sorted []string, want string) bool {
	i := sort.SearchStrings(sorted, want)
	return i < len(sorted) && sorted[i] == want
}
