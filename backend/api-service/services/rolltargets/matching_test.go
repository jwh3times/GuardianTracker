package rolltargets

import (
	"context"
	"errors"
	"sort"
	"testing"

	"guardian-tracker/api-service/services/ownedrolls"
)

type fakeOwned struct {
	rolls []ownedrolls.OwnedRoll
	err   error

	gotType int
	gotID   string
}

func (f *fakeOwned) Read(_ context.Context, membershipType int, membershipID string) ([]ownedrolls.OwnedRoll, error) {
	f.gotType, f.gotID = membershipType, membershipID
	return f.rolls, f.err
}

// owned perks arrive sorted from ownedrolls; targets arrive sorted from the
// service. Both invariants are relied on here.
func roll(hash uint32, instance string, perks ...string) ownedrolls.OwnedRoll {
	return ownedrolls.OwnedRoll{ItemHash: hash, InstanceID: instance, Perks: perks}
}

func perkColumn(index int, current string, possible ...string) ownedrolls.OwnedPerkColumn {
	return ownedrolls.OwnedPerkColumn{SocketIndex: index, Perk: current, PossiblePerks: possible}
}

func rollWithColumns(hash uint32, instance string, columns ...ownedrolls.OwnedPerkColumn) ownedrolls.OwnedRoll {
	perks := make([]string, 0, len(columns))
	for _, column := range columns {
		perks = append(perks, column.Perk)
	}
	sort.Strings(perks)
	return ownedrolls.OwnedRoll{ItemHash: hash, InstanceID: instance, Perks: perks, Columns: columns}
}

func target(id TargetID, hash *uint32, wanted bool, perks ...string) StoredTarget {
	return StoredTarget{ID: id, ItemHash: hash, Wanted: wanted, Perks: perks}
}

// owned is taken as the interface, not as *fakeOwned: passing a typed nil
// pointer through an interface parameter yields a non-nil interface, so a test
// written the other way would exercise a panic rather than the nil path.
func matchSvc(targets []StoredTarget, owned OwnedRollReader) *Service {
	return NewService(&fakeRepo{targets: targets}, testPool(), owned)
}

// A target matches when every perk it names is present. Extra perks on the
// weapon do not prevent it: a target says what must be there, not what must not.
func TestMatches_RequiresEveryNamedPerkAndToleratesExtras(t *testing.T) {
	owned := &fakeOwned{rolls: []ownedrolls.OwnedRoll{
		roll(1000, "a", "Firefly", "Kill Clip", "Outlaw"), // superset
		roll(1000, "b", "Outlaw"),                         // missing Kill Clip
	}}
	report, err := matchSvc([]StoredTarget{
		target(1, weapon(1000), true, "Kill Clip", "Outlaw"),
	}, owned).Matches(context.Background(), 3, "m1")
	if err != nil {
		t.Fatalf("Matches: %v", err)
	}
	if len(report.Wanted) != 1 {
		t.Fatalf("wanted matches = %d, want 1 (%+v)", len(report.Wanted), report)
	}
	if report.Wanted[0].Roll.InstanceID != "a" {
		t.Errorf("matched instance = %q, want the superset roll", report.Wanted[0].Roll.InstanceID)
	}
	if len(report.UnmatchedTargets) != 0 {
		t.Errorf("target reported unmatched despite a match: %+v", report.UnmatchedTargets)
	}
}

// A weapon of a different hash never matches a target that names a weapon.
func TestMatches_RespectsTheNamedWeapon(t *testing.T) {
	owned := &fakeOwned{rolls: []ownedrolls.OwnedRoll{roll(2000, "a", "Kill Clip", "Outlaw")}}
	report, _ := matchSvc([]StoredTarget{
		target(1, weapon(1000), true, "Kill Clip", "Outlaw"),
	}, owned).Matches(context.Background(), 3, "m1")
	if len(report.Wanted) != 0 {
		t.Errorf("a different weapon matched: %+v", report.Wanted)
	}
	if len(report.UnmatchedTargets) != 1 {
		t.Errorf("unmatched = %d, want 1", len(report.UnmatchedTargets))
	}
}

// An any-weapon target is tested against every owned weapon — that is what
// makes it a wildcard rather than a target with a missing field.
func TestMatches_AnyWeaponTargetSpansEveryWeapon(t *testing.T) {
	owned := &fakeOwned{rolls: []ownedrolls.OwnedRoll{
		roll(1000, "a", "Outlaw"),
		roll(2000, "b", "Outlaw"),
		roll(3000, "c", "Firefly"),
	}}
	report, _ := matchSvc([]StoredTarget{
		target(1, nil, true, "Outlaw"),
	}, owned).Matches(context.Background(), 3, "m1")
	if len(report.Wanted) != 2 {
		t.Fatalf("any-weapon matches = %d, want 2 (%+v)", len(report.Wanted), report.Wanted)
	}
}

// An unwanted roll is reported separately: these are copies to dismantle, not
// copies to celebrate.
func TestMatches_SeparatesWantedFromUnwanted(t *testing.T) {
	owned := &fakeOwned{rolls: []ownedrolls.OwnedRoll{roll(1000, "a", "Kill Clip", "Outlaw")}}
	report, _ := matchSvc([]StoredTarget{
		target(1, weapon(1000), true, "Outlaw"),
		target(2, weapon(1000), false, "Kill Clip"),
	}, owned).Matches(context.Background(), 3, "m1")
	if len(report.Wanted) != 1 || report.Wanted[0].Target.ID != 1 {
		t.Errorf("wanted = %+v", report.Wanted)
	}
	if len(report.Unwanted) != 1 || report.Unwanted[0].Target.ID != 2 {
		t.Errorf("unwanted = %+v", report.Unwanted)
	}
}

// A saved roll nothing satisfies is the most useful thing the feature has to
// say. It must survive into the report.
func TestMatches_CarriesUnmatchedTargets(t *testing.T) {
	owned := &fakeOwned{rolls: []ownedrolls.OwnedRoll{roll(1000, "a", "Firefly")}}
	report, _ := matchSvc([]StoredTarget{
		target(1, weapon(1000), true, "Kill Clip", "Outlaw"),
	}, owned).Matches(context.Background(), 3, "m1")
	if len(report.UnmatchedTargets) != 1 || report.UnmatchedTargets[0].Target.ID != 1 {
		t.Errorf("unmatched = %+v", report.UnmatchedTargets)
	}
}

// A still-chasing target carries its highest-scoring partial copy. Scores are
// matched target columns, not all perks on the weapon; an inapplicable weapon
// is ignored and an equal score keeps the first copy in the reader's stable
// order.
func TestMatches_CarriesTheBestNearMissForAnUnmatchedTarget(t *testing.T) {
	owned := &fakeOwned{rolls: []ownedrolls.OwnedRoll{
		rollWithColumns(1000, "a",
			perkColumn(0, "Outlaw", "Outlaw", "Vorpal Weapon"),
			perkColumn(1, "Explosive Payload", "Explosive Payload", "Firefly"),
			perkColumn(2, "Rampage", "Kill Clip", "Rampage")),
		rollWithColumns(1000, "b",
			perkColumn(0, "Outlaw", "Outlaw", "Vorpal Weapon"),
			perkColumn(1, "Firefly", "Explosive Payload", "Firefly"),
			perkColumn(2, "Rampage", "Kill Clip", "Rampage")),
		rollWithColumns(1000, "c",
			perkColumn(0, "Vorpal Weapon", "Outlaw", "Vorpal Weapon"),
			perkColumn(1, "Firefly", "Explosive Payload", "Firefly"),
			perkColumn(2, "Kill Clip", "Kill Clip", "Rampage")),
		rollWithColumns(2000, "d",
			perkColumn(0, "Outlaw", "Outlaw"),
			perkColumn(1, "Firefly", "Firefly"),
			perkColumn(2, "Kill Clip", "Kill Clip")),
	}}
	report, err := matchSvc([]StoredTarget{
		target(1, weapon(1000), true, "Firefly", "Kill Clip", "Outlaw"),
	}, owned).Matches(context.Background(), 3, "m1")
	if err != nil {
		t.Fatalf("Matches: %v", err)
	}
	if len(report.UnmatchedTargets) != 1 {
		t.Fatalf("unmatched = %+v, want one", report.UnmatchedTargets)
	}
	near := report.UnmatchedTargets[0].NearMiss
	if near == nil {
		t.Fatal("near miss = nil, want the best partial copy")
	}
	if near.Roll.InstanceID != "b" {
		t.Errorf("best copy = %q, want first two-perk tie b", near.Roll.InstanceID)
	}
	if got := near.MatchedPerks; len(got) != 2 || got[0] != "Firefly" || got[1] != "Outlaw" {
		t.Errorf("matched perks = %v, want [Firefly Outlaw]", got)
	}
	if near.MatchedColumns != 2 || near.TargetColumns != 3 {
		t.Errorf("column score = %d/%d, want 2/3", near.MatchedColumns, near.TargetColumns)
	}
}

// Two target names from one socket are one target column. Seating one of those
// mutually exclusive names is not a near miss: it matches all of the target's
// columns, even though the impossible name-level target remains unmatched.
func TestMatches_CountsDistinctTargetColumnsNotTargetNames(t *testing.T) {
	owned := &fakeOwned{rolls: []ownedrolls.OwnedRoll{
		rollWithColumns(1000, "a", perkColumn(0, "Firefly", "Firefly", "Kill Clip")),
	}}
	report, err := matchSvc([]StoredTarget{
		target(1, weapon(1000), true, "Firefly", "Kill Clip"),
	}, owned).Matches(context.Background(), 3, "m1")
	if err != nil {
		t.Fatalf("Matches: %v", err)
	}
	if len(report.UnmatchedTargets) != 1 {
		t.Fatalf("unmatched = %+v, want one", report.UnmatchedTargets)
	}
	if report.UnmatchedTargets[0].NearMiss != nil {
		t.Errorf("near miss = %+v, want nil for 1 of 1 matched target columns", report.UnmatchedTargets[0].NearMiss)
	}
}

// A socket referenced by more than one Manifest category is still one physical
// perk column and must contribute only once to the score.
func TestMatches_CountsEachSocketIndexOnce(t *testing.T) {
	duplicate := perkColumn(0, "Firefly", "Firefly")
	owned := &fakeOwned{rolls: []ownedrolls.OwnedRoll{
		rollWithColumns(1000, "a",
			duplicate,
			duplicate,
			perkColumn(1, "Rampage", "Kill Clip", "Rampage")),
	}}
	report, err := matchSvc([]StoredTarget{
		target(1, weapon(1000), true, "Firefly", "Kill Clip"),
	}, owned).Matches(context.Background(), 3, "m1")
	if err != nil {
		t.Fatalf("Matches: %v", err)
	}
	near := report.UnmatchedTargets[0].NearMiss
	if near == nil {
		t.Fatal("near miss = nil, want the Firefly copy")
	}
	if near.MatchedColumns != 1 || near.TargetColumns != 2 {
		t.Errorf("column score = %d/%d, want 1/2", near.MatchedColumns, near.TargetColumns)
	}
}

func TestMatches_OmitsNearMissWhenNoTargetPerkMatches(t *testing.T) {
	owned := &fakeOwned{rolls: []ownedrolls.OwnedRoll{roll(1000, "a", "Firefly")}}
	report, _ := matchSvc([]StoredTarget{
		target(1, weapon(1000), true, "Kill Clip", "Outlaw"),
	}, owned).Matches(context.Background(), 3, "m1")
	if report.UnmatchedTargets[0].NearMiss != nil {
		t.Errorf("near miss = %+v, want nil for a zero-perk copy", report.UnmatchedTargets[0].NearMiss)
	}
}

func TestMatches_FullMatchWinsOverAnEarlierNearMiss(t *testing.T) {
	owned := &fakeOwned{rolls: []ownedrolls.OwnedRoll{
		roll(1000, "a", "Outlaw"),
		roll(1000, "b", "Kill Clip", "Outlaw"),
	}}
	report, _ := matchSvc([]StoredTarget{
		target(1, weapon(1000), true, "Kill Clip", "Outlaw"),
	}, owned).Matches(context.Background(), 3, "m1")
	if len(report.Wanted) != 1 || len(report.UnmatchedTargets) != 0 {
		t.Errorf("report = %+v, want one full match and no unmatched target", report)
	}
}

// An inventory that could not be read is not an empty inventory. The failure
// must reach the caller rather than rendering as "nothing you own matches".
func TestMatches_InventoryFailurePassesThrough(t *testing.T) {
	owned := &fakeOwned{err: ownedrolls.ErrInventoryUnavailable}
	_, err := matchSvc([]StoredTarget{
		target(1, weapon(1000), true, "Outlaw"),
	}, owned).Matches(context.Background(), 3, "m1")
	if !errors.Is(err, ownedrolls.ErrInventoryUnavailable) {
		t.Fatalf("err = %v, want ErrInventoryUnavailable", err)
	}
}

// A deployment with no way to read the inventory reports that, rather than an
// empty match report.
func TestMatches_NoReaderIsNotAnEmptyResult(t *testing.T) {
	_, err := matchSvc([]StoredTarget{
		target(1, weapon(1000), true, "Outlaw"),
	}, nil).Matches(context.Background(), 3, "m1")
	if !errors.Is(err, ErrOwnedRollsUnavailable) {
		t.Fatalf("err = %v, want ErrOwnedRollsUnavailable", err)
	}
}

// With nothing saved there is nothing to match, and no reason to read the
// player's whole inventory to discover that.
func TestMatches_NoTargetsReadsNoInventory(t *testing.T) {
	owned := &fakeOwned{}
	report, err := matchSvc(nil, owned).Matches(context.Background(), 3, "m1")
	if err != nil {
		t.Fatalf("Matches: %v", err)
	}
	if len(report.Wanted)+len(report.Unwanted)+len(report.UnmatchedTargets) != 0 {
		t.Errorf("report = %+v, want empty", report)
	}
	if owned.gotID != "" {
		t.Error("the inventory was read despite nothing being saved")
	}
}

// Matching needs the platform as well as the id: inventory is read per
// membership pair.
func TestMatches_PassesTheWholeMembershipPair(t *testing.T) {
	owned := &fakeOwned{}
	matchSvc([]StoredTarget{target(1, weapon(1000), true, "Outlaw")}, owned).
		Matches(context.Background(), 3, "m1")
	if owned.gotType != 3 || owned.gotID != "m1" {
		t.Errorf("reader saw %d/%q, want 3/m1", owned.gotType, owned.gotID)
	}
}
