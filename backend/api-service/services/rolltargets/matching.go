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
// MatchedColumns and TargetColumns are the authoritative score; MatchedPerks
// is the display evidence identifying which perk names contributed.
type NearMiss struct {
	Roll           ownedrolls.OwnedRoll
	MatchedPerks   []string
	MatchedColumns int
	TargetColumns  int
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
	// with the best partial copy when some, but not all, unique target perk
	// columns are satisfied.
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
			if containsAll(roll.Perks, target.Perks) {
				matched = true
				m := Match{Target: target, Roll: roll}
				if target.Wanted {
					report.Wanted = append(report.Wanted, m)
				} else {
					report.Unwanted = append(report.Unwanted, m)
				}
				continue
			}

			matchedPerks, matchedColumns, targetColumns := scoreTargetColumns(roll.Columns, target.Perks)
			// A near miss has some, but not all, target columns. Strictly greater
			// preserves the owned-copy order for equal scores; the reader orders
			// copies by item hash then instance id.
			if matchedColumns > 0 && matchedColumns < targetColumns &&
				(best == nil || matchedColumns > best.MatchedColumns) {
				best = &NearMiss{
					Roll:           roll,
					MatchedPerks:   matchedPerks,
					MatchedColumns: matchedColumns,
					TargetColumns:  targetColumns,
				}
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

// scoreTargetColumns counts the owned weapon's unique perk columns that carry
// at least one target name, and which of those columns currently seat a target
// perk. Two distinct target names from one socket therefore contribute one
// target column, never two.
func scoreTargetColumns(columns []ownedrolls.OwnedPerkColumn, wanted []string) ([]string, int, int) {
	wantedSet := make(map[string]struct{}, len(wanted))
	for _, perk := range wanted {
		wantedSet[perk] = struct{}{}
	}

	matchedNames := map[string]struct{}{}
	seenSocketIndexes := map[int]struct{}{}
	matchedColumns := 0
	targetColumns := 0
	for _, column := range columns {
		if _, seen := seenSocketIndexes[column.SocketIndex]; seen {
			continue
		}
		seenSocketIndexes[column.SocketIndex] = struct{}{}
		isTargetColumn := false
		for _, possible := range column.PossiblePerks {
			if _, wanted := wantedSet[possible]; wanted {
				isTargetColumn = true
				break
			}
		}
		if !isTargetColumn {
			continue
		}
		targetColumns++
		if _, matched := wantedSet[column.Perk]; matched {
			matchedColumns++
			matchedNames[column.Perk] = struct{}{}
		}
	}

	matchedPerks := make([]string, 0, len(matchedNames))
	for _, perk := range wanted {
		if _, matched := matchedNames[perk]; matched {
			matchedPerks = append(matchedPerks, perk)
		}
	}
	return matchedPerks, matchedColumns, targetColumns
}

func containsAll(owned, wanted []string) bool {
	if len(wanted) == 0 {
		return false
	}
	for _, perk := range wanted {
		if !contains(owned, perk) {
			return false
		}
	}
	return true
}

func contains(sorted []string, want string) bool {
	i := sort.SearchStrings(sorted, want)
	return i < len(sorted) && sorted[i] == want
}
