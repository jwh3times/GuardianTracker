package rolltargets

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/services/manifest"
)

// fakeRepo records what reached persistence and returns what the test tells it
// to. It embeds Repository so an unstubbed method panics rather than quietly
// succeeding.
type fakeRepo struct {
	Repository

	targets   []StoredTarget
	listErr   error
	addErr    error
	updateErr error

	gotAdd    AddCommand
	gotPatch  UpdateCommand
	gotID     TargetID
	addCalled bool
	updCalled bool
}

func (f *fakeRepo) List(context.Context, string) ([]StoredTarget, error) {
	return f.targets, f.listErr
}

func (f *fakeRepo) Add(_ context.Context, _ string, cmd AddCommand) (StoredTarget, error) {
	f.addCalled = true
	f.gotAdd = cmd
	if f.addErr != nil {
		return StoredTarget{}, f.addErr
	}
	return StoredTarget{ID: 1, ItemHash: cmd.ItemHash, Perks: cmd.Perks, Notes: cmd.Notes, CreatedAt: time.Unix(0, 0)}, nil
}

func (f *fakeRepo) Update(_ context.Context, _ string, id TargetID, patch UpdateCommand) (StoredTarget, error) {
	f.updCalled = true
	f.gotID, f.gotPatch = id, patch
	if f.updateErr != nil {
		return StoredTarget{}, f.updateErr
	}
	return StoredTarget{ID: id}, nil
}

// fakePool serves one weapon's perk columns.
type fakePool struct {
	cols []manifest.PerkColumn
	err  error
}

func (f fakePool) GetWeaponPerks(uint32) ([]manifest.PerkColumn, error) { return f.cols, f.err }

// WeaponPerkNames treats the one fixture weapon as the whole manifest, so the
// names an any-weapon target may use are exactly the names it can roll.
func (f fakePool) WeaponPerkNames() (map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := map[string]string{}
	for _, c := range f.cols {
		for _, p := range c.Perks {
			out[strings.ToLower(p)] = p
		}
	}
	return out, nil
}

func (f fakePool) PlugNames(hashes []uint32) (map[uint32]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := map[uint32]string{}
	for _, h := range hashes {
		out[h] = "Plug " + strconv.FormatUint(uint64(h), 10)
	}
	return out, nil
}

func testPool() fakePool {
	return fakePool{cols: []manifest.PerkColumn{
		{Role: "barrel", Label: "Barrel", Perks: []string{"Smallbore", "Full Bore"}},
		{Role: "trait", Label: "Trait 1", Perks: []string{"Outlaw", "Rapid Hit"}},
		{Role: "trait", Label: "Trait 2", Perks: []string{"Kill Clip", "Frenzy"}},
	}}
}

func svc(repo Repository, pool PerkPool) *Service { return NewService(repo, pool, nil) }

// weapon is shorthand for a target that names a weapon.
func weapon(h uint32) *uint32 { return &h }

func TestAdd_PersistsCanonicalPerkSpelling(t *testing.T) {
	repo := &fakeRepo{}
	// The user types the perk in lower case; the manifest's spelling is what is
	// stored, so a target reads back the way the game writes it.
	got, err := svc(repo, testPool()).Add(context.Background(), "m1", AddCommand{
		ItemHash: weapon(1000), Perks: []string{"  kill clip ", "OUTLAW"},
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Canonical manifest spelling, and sorted: the perks are AND-ed, so their
	// order carries no meaning and normalising it is what makes one roll
	// recognisable as already saved.
	want := []string{"Kill Clip", "Outlaw"}
	if len(got.Perks) != 2 || got.Perks[0] != want[0] || got.Perks[1] != want[1] {
		t.Errorf("stored perks = %v, want %v", got.Perks, want)
	}
	if len(repo.gotAdd.Perks) != 2 || repo.gotAdd.Perks[0] != "Kill Clip" {
		t.Errorf("repo received %v", repo.gotAdd.Perks)
	}
}

func TestAdd_RejectsPerkTheWeaponCannotRoll(t *testing.T) {
	repo := &fakeRepo{}
	_, err := svc(repo, testPool()).Add(context.Background(), "m1", AddCommand{
		ItemHash: weapon(1000), Perks: []string{"Outlaw", "Recombination"},
	})
	if !errors.Is(err, ErrUnknownPerk) {
		t.Fatalf("err = %v, want ErrUnknownPerk", err)
	}
	// A target that can never match must not reach storage.
	if repo.addCalled {
		t.Error("unmatchable target was persisted")
	}
}

func TestAdd_RejectsEmptyAndOversizedPerkSets(t *testing.T) {
	cases := []struct {
		name  string
		perks []string
		want  error
	}{
		{"none", nil, ErrNoPerks},
		{"only blanks", []string{"   ", ""}, ErrNoPerks},
		{"over the cap", make([]string, MaxPerks+1), ErrNoPerks}, // blanks trim to none
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc(&fakeRepo{}, testPool()).Add(context.Background(), "m1",
				AddCommand{ItemHash: weapon(1000), Perks: tc.perks})
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
	// A genuinely oversized set of real perks reports the cap, not emptiness.
	big := make([]string, MaxPerks+1)
	for i := range big {
		big[i] = "Outlaw"
	}
	_, err := svc(&fakeRepo{}, testPool()).Add(context.Background(), "m1",
		AddCommand{ItemHash: weapon(1000), Perks: big})
	if !errors.Is(err, ErrTooManyPerks) {
		t.Errorf("oversized err = %v, want ErrTooManyPerks", err)
	}
}

func TestAdd_RejectsTheSamePerkTwice(t *testing.T) {
	_, err := svc(&fakeRepo{}, testPool()).Add(context.Background(), "m1", AddCommand{
		ItemHash: weapon(1000), Perks: []string{"Outlaw", "outlaw"},
	})
	if !errors.Is(err, ErrDuplicatePerk) {
		t.Fatalf("err = %v, want ErrDuplicatePerk", err)
	}
}

func TestAdd_RejectsOverlongNotes(t *testing.T) {
	// Counted in code points: 500 multi-byte runes are legal, 501 are not.
	legal := strings.Repeat("é", MaxNoteRunes)
	if _, err := svc(&fakeRepo{}, testPool()).Add(context.Background(), "m1",
		AddCommand{ItemHash: weapon(1000), Perks: []string{"Outlaw"}, Notes: legal}); err != nil {
		t.Fatalf("500 runes rejected: %v", err)
	}
	_, err := svc(&fakeRepo{}, testPool()).Add(context.Background(), "m1",
		AddCommand{ItemHash: weapon(1000), Perks: []string{"Outlaw"}, Notes: legal + "é"})
	if !errors.Is(err, ErrNotesTooLong) {
		t.Errorf("err = %v, want ErrNotesTooLong", err)
	}
}

// A successful read of no columns is a verdict about the item; a failed read is
// not. They must never collapse into one another.
func TestAdd_SeparatesNotAWeaponFromUnreadablePool(t *testing.T) {
	_, err := svc(&fakeRepo{}, fakePool{cols: nil}).Add(context.Background(), "m1",
		AddCommand{ItemHash: weapon(42), Perks: []string{"Outlaw"}})
	if !errors.Is(err, ErrNotAWeapon) {
		t.Errorf("empty pool err = %v, want ErrNotAWeapon", err)
	}

	_, err = svc(&fakeRepo{}, fakePool{err: errors.New("manifest warming")}).Add(
		context.Background(), "m1", AddCommand{ItemHash: weapon(1000), Perks: []string{"Outlaw"}})
	if !errors.Is(err, ErrPerksUnavailable) {
		t.Errorf("failed read err = %v, want ErrPerksUnavailable", err)
	}
}

// An any-weapon target has no weapon pool, but its names still have to be
// perks some weapon can roll — otherwise it is saved and never matches. A name
// no weapon perk carries is refused before storage; a real one is stored in
// the manifest's spelling, exactly as a weapon-bound target is.
func TestAdd_AnyWeaponTargetNamesAreCheckedAgainstTheManifest(t *testing.T) {
	repo := &fakeRepo{}
	_, err := svc(repo, testPool()).Add(context.Background(), "m1",
		AddCommand{Perks: []string{"Outlaw", "Not A Perk"}})
	if !errors.Is(err, ErrUnknownPerkName) {
		t.Fatalf("err = %v, want ErrUnknownPerkName", err)
	}
	if repo.addCalled {
		t.Error("unmatchable any-weapon target was persisted")
	}

	got, err := svc(repo, testPool()).Add(context.Background(), "m1",
		AddCommand{Perks: []string{" kill clip", "OUTLAW"}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(got.Perks) != 2 || got.Perks[0] != "Kill Clip" || got.Perks[1] != "Outlaw" {
		t.Errorf("stored perks = %v, want [Kill Clip Outlaw]", got.Perks)
	}
}

// With the name set unreadable there is no answer, and "unknown perk" would be
// a false verdict about the name.
func TestAdd_AnyWeaponTargetWithUnreadableManifestIsUnavailable(t *testing.T) {
	_, err := svc(&fakeRepo{}, fakePool{err: errors.New("manifest warming")}).Add(
		context.Background(), "m1", AddCommand{Perks: []string{"Outlaw"}})
	if !errors.Is(err, ErrPerksUnavailable) {
		t.Fatalf("err = %v, want ErrPerksUnavailable", err)
	}
}

func TestUpdate_ValidatesReplacementPerksAgainstTheStoredWeapon(t *testing.T) {
	repo := &fakeRepo{targets: []StoredTarget{{ID: 7, ItemHash: weapon(1000), Perks: []string{"Outlaw"}}}}
	perks := []string{"Kill Clip"}
	if _, err := svc(repo, testPool()).Update(context.Background(), "m1", 7,
		UpdateCommand{Perks: &perks}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !repo.updCalled || repo.gotID != 7 {
		t.Fatalf("repo.Update not called for id 7 (called=%v id=%d)", repo.updCalled, repo.gotID)
	}

	bad := []string{"Recombination"}
	_, err := svc(repo, testPool()).Update(context.Background(), "m1", 7, UpdateCommand{Perks: &bad})
	if !errors.Is(err, ErrUnknownPerk) {
		t.Errorf("err = %v, want ErrUnknownPerk", err)
	}
}

func TestUpdate_ForeignTargetIsNotFoundBeforeAnyWrite(t *testing.T) {
	repo := &fakeRepo{targets: []StoredTarget{{ID: 7, ItemHash: weapon(1000)}}}
	perks := []string{"Outlaw"}
	_, err := svc(repo, testPool()).Update(context.Background(), "m1", 99, UpdateCommand{Perks: &perks})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if repo.updCalled {
		t.Error("wrote against a target the membership does not own")
	}
}

// Persistence being absent must not read as "you have saved nothing".
func TestList_UnavailableIsNotEmpty(t *testing.T) {
	repo := &fakeRepo{listErr: ErrUnavailable}
	got, err := svc(repo, testPool()).List(context.Background(), "m1")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
	if got != nil {
		t.Errorf("targets = %v, want nil", got)
	}
}
