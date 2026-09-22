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

	// UnmatchedTargets are saved rolls that nothing owned satisfies.
	UnmatchedTargets []StoredTarget
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
		for _, roll := range rolls {
			if !targetApplies(target, roll) {
				continue
			}
			if !containsAll(roll.Perks, target.Perks) {
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
			report.UnmatchedTargets = append(report.UnmatchedTargets, target)
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

// containsAll reports whether every wanted perk is present among the owned
// ones. Both slices are sorted and canonically spelled by their owners, so this
// is an exact comparison rather than a fuzzy one.
func containsAll(owned, wanted []string) bool {
	if len(wanted) == 0 {
		// A target with no perks would match every copy of the weapon. The
		// domain refuses to store one; refusing to match on one as well means a
		// row that predates that rule cannot quietly match everything.
		return false
	}
	for _, w := range wanted {
		if !contains(owned, w) {
			return false
		}
	}
	return true
}

func contains(sorted []string, want string) bool {
	i := sort.SearchStrings(sorted, want)
	return i < len(sorted) && sorted[i] == want
}
