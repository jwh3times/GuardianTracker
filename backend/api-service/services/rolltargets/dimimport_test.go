package rolltargets

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"guardian-tracker/api-service/services/manifest"
)

// importRepo is a minimal in-memory Repository: import is about what reaches
// storage, so the fake stores rather than records.
type importRepo struct {
	Repository

	targets []StoredTarget
	nextID  TargetID
	addErr  error
	listErr error
}

func (r *importRepo) List(context.Context, string) ([]StoredTarget, error) {
	return r.targets, r.listErr
}

func (r *importRepo) RemoveImport(_ context.Context, _ string, id ImportID) (int, error) {
	kept := make([]StoredTarget, 0, len(r.targets))
	deleted := 0
	for _, target := range r.targets {
		if target.ImportID == id {
			deleted++
		} else {
			kept = append(kept, target)
		}
	}
	r.targets = kept
	return deleted, nil
}

func TestDeleteImport_RemovesOnlyItsNewTargetsAndAllowsReimport(t *testing.T) {
	svc := importSvc(&importRepo{})
	ctx := context.Background()
	manual, err := svc.Add(ctx, "m1", AddCommand{Perks: []string{"Outlaw"}, Wanted: true, ImportID: "forged", ImportTitle: "forged"})
	if err != nil {
		t.Fatal(err)
	}
	if manual.ImportID != "" || manual.ImportTitle != "" {
		t.Fatalf("manual provenance = %+v", manual)
	}
	text := "dimwishlist:item=-69420&perks=111\ndimwishlist:item=1000&perks=222"
	first, err := svc.ImportDIM(ctx, "m1", text)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteImport(ctx, "m1", ""); !errors.Is(err, ErrInvalidImportID) {
		t.Fatalf("empty ID error = %v", err)
	}
	if _, err := svc.DeleteImport(ctx, "m1", "not-an-id"); !errors.Is(err, ErrInvalidImportID) {
		t.Fatalf("invalid ID error = %v", err)
	}
	deleted, err := svc.DeleteImport(ctx, "m1", ImportID(strings.ToUpper(string(first.ImportID))))
	if err != nil || deleted != 1 {
		t.Fatalf("delete = %d, %v", deleted, err)
	}
	targets, err := svc.List(ctx, "m1")
	if err != nil || len(targets) != 1 || targets[0].ID != manual.ID {
		t.Fatalf("remaining = %+v, %v", targets, err)
	}
	deleted, err = svc.DeleteImport(ctx, "m1", first.ImportID)
	if err != nil || deleted != 0 {
		t.Fatalf("repeat delete = %d, %v", deleted, err)
	}
	second, err := svc.ImportDIM(ctx, "m1", text)
	if err != nil || second.Imported() != 1 || second.ImportID == first.ImportID {
		t.Fatalf("reimport = %+v, %v", second, err)
	}
}

func (r *importRepo) Add(_ context.Context, _ string, cmd AddCommand) (StoredTarget, error) {
	if r.addErr != nil {
		return StoredTarget{}, r.addErr
	}
	// The store's uniqueness is the whole roll — weapon, stance and perks —
	// not the weapon alone, so the fake keys the same way.
	key := rollKey(cmd.ItemHash, cmd.Wanted, cmd.Perks)
	for _, t := range r.targets {
		if rollKey(t.ItemHash, t.Wanted, t.Perks) == key {
			return StoredTarget{}, ErrDuplicate
		}
	}
	r.nextID++
	t := StoredTarget{
		ImportID: cmd.ImportID, ImportTitle: cmd.ImportTitle,
		ID: r.nextID, ItemHash: cmd.ItemHash, Wanted: cmd.Wanted,
		Perks: cmd.Perks, Notes: cmd.Notes,
	}
	r.targets = append(r.targets, t)
	return t, nil
}

func rollKey(hash *uint32, wanted bool, perks []string) string {
	weapon := "any"
	if hash != nil {
		weapon = strconv.FormatUint(uint64(*hash), 10)
	}
	return fmt.Sprintf("%s|%t|%s", weapon, wanted, strings.Join(perks, ","))
}

// importPool serves one weapon (hash 1000) whose trait column holds a paired
// base/enhanced perk, a base-only perk, and an ambiguous one.
type importPool struct{ err error }

// PlugNames serves the wildcard path, which has no weapon pool to resolve
// against. Only the perks this fixture's weapon knows are resolvable, so an
// unknown hash is absent — exactly as the manifest would report it.
func (p importPool) PlugNames(hashes []uint32) (map[uint32]string, error) {
	if p.err != nil {
		return nil, p.err
	}
	// 444 is a real plug that sits in no weapon's perk column — a mod.
	names := map[uint32]string{111: "Outlaw", 911: "Outlaw", 222: "Firefly", 333: "Drop Mag", 444: "Minor Spec"}
	out := map[uint32]string{}
	for _, h := range hashes {
		if n, ok := names[h]; ok {
			out[h] = n
		}
	}
	return out, nil
}

func (p importPool) GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error) {
	if p.err != nil {
		return nil, p.err
	}
	if itemHash != 1000 {
		return nil, nil // a successful read of nothing: not a weapon
	}
	return []manifest.PerkColumn{{
		Role: "trait", Label: "Trait 1",
		Perks: []string{"Outlaw", "Firefly", "Drop Mag"},
		Plugs: []manifest.PerkPlug{
			{Name: "Outlaw", Base: 111, Enhanced: 911},
			{Name: "Firefly", Base: 222},
			{Name: "Drop Mag", Base: 333, Ambiguous: true},
		},
	}}, nil
}

// WeaponPerkNames: weapon 1000 is the whole manifest's weapon set here.
func (p importPool) WeaponPerkNames() (map[string]string, error) {
	if p.err != nil {
		return nil, p.err
	}
	return map[string]string{"outlaw": "Outlaw", "firefly": "Firefly", "drop mag": "Drop Mag"}, nil
}

func importSvc(repo Repository) *Service { return NewService(repo, importPool{}, nil) }

func TestImportDIM_TitleLimitIsUnicodeAndCheckedBeforeWriting(t *testing.T) {
	for _, length := range []int{500, 501} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			svc := importSvc(&importRepo{})
			ctx := context.Background()
			_, err := svc.ImportDIM(ctx, "m1", "dimwishlist:item=1000&perks=111\ntitle:"+strings.Repeat("界", length))
			if length == 501 && !errors.Is(err, ErrImportTitleTooLong) {
				t.Fatalf("error = %v", err)
			}
			if length == 500 && err != nil {
				t.Fatal(err)
			}
			targets, err := svc.List(ctx, "m1")
			if err != nil {
				t.Fatal(err)
			}
			want := 1
			if length == 501 {
				want = 0
			}
			if len(targets) != want {
				t.Fatalf("saved %d targets, want %d", len(targets), want)
			}
		})
	}
}

func TestImportDIM_NewTargetsShareImmutableImportProvenance(t *testing.T) {
	svc := importSvc(&importRepo{})
	ctx := context.Background()
	first, err := svc.ImportDIM(ctx, "m1", "title:My DIM rolls\ndimwishlist:item=1000&perks=111\ndimwishlist:item=-69420&perks=222")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(string(first.ImportID)); err != nil {
		t.Fatalf("import ID = %q: %v", first.ImportID, err)
	}
	second, err := svc.ImportDIM(ctx, "m1", "title:My DIM rolls\ndimwishlist:item=1000&perks=111\ndimwishlist:item=1000&perks=222")
	if err != nil {
		t.Fatal(err)
	}
	if second.ImportID == "" || second.ImportID == first.ImportID {
		t.Fatalf("second import ID = %q", second.ImportID)
	}
	targets, err := svc.List(ctx, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 3 {
		t.Fatalf("targets = %+v", targets)
	}
	for i, target := range targets {
		wantID := first.ImportID
		if i == 2 {
			wantID = second.ImportID
		}
		if target.ImportID != wantID || target.ImportTitle != "My DIM rolls" {
			t.Errorf("target = %+v, want import %s and original title", target, wantID)
		}
	}
	repeated, err := svc.ImportDIM(ctx, "m1", "dimwishlist:item=1000&perks=111")
	if err != nil || repeated.ImportID != "" {
		t.Fatalf("duplicate import = %+v, %v", repeated, err)
	}
}

// A wildcard line resolves its hashes straight to plug names, and a plug can
// have a name without being a weapon perk. Such a line could never match, so it
// reports as unresolved rather than being stored or failing anonymously.
func TestImportDIM_WildcardNamingANonWeaponPerkIsUnresolved(t *testing.T) {
	repo := &importRepo{}
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"dimwishlist:item=-69420&perks=444\n")
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	if got := outcomes(report); len(got) != 1 || got[0] != OutcomeUnresolvedPerk {
		t.Fatalf("outcomes = %v, want [%q]", got, OutcomeUnresolvedPerk)
	}
	if len(repo.targets) != 0 {
		t.Errorf("stored %d targets, want none", len(repo.targets))
	}
	u := report.Lines[0].Unresolved
	if u == nil || u.Hash != 444 || u.Name != "Minor Spec" || u.Reason != UnresolvedNotAWeaponPerk {
		t.Errorf("unresolved = %+v, want {Hash:444 Name:\"Minor Spec\" Reason:not-a-weapon-perk}", u)
	}
}

// A weapon-bound line naming a hash outside that weapon's own pool has no
// name to report — the pool lookup never resolved one — but the hash itself
// still is, so a reader can look it up themselves.
func TestImportDIM_WeaponBoundHashNotInPoolReportsTheHashWithNoName(t *testing.T) {
	repo := &importRepo{}
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"dimwishlist:item=1000&perks=999\n")
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	u := report.Lines[0].Unresolved
	if u == nil || u.Hash != 999 || u.Name != "" || u.Reason != UnresolvedNotInPool {
		t.Errorf("unresolved = %+v, want {Hash:999 Name:\"\" Reason:not-in-pool}", u)
	}
}

// An ambiguous plug's name is already known — it just cannot say which of two
// hashes the file meant.
func TestImportDIM_AmbiguousPlugReportsItsHashAndName(t *testing.T) {
	repo := &importRepo{}
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"dimwishlist:item=1000&perks=333")
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	u := report.Lines[0].Unresolved
	if u == nil || u.Hash != 333 || u.Name != "Drop Mag" || u.Reason != UnresolvedAmbiguous {
		t.Errorf("unresolved = %+v, want {Hash:333 Name:\"Drop Mag\" Reason:ambiguous}", u)
	}
}

// A wildcard line's hash the manifest cannot name at all is "not-in-pool" too
// — PlugNames omits a hash it does not know, the same as a weapon's own pool
// omitting one it does not carry.
func TestImportDIM_WildcardHashUnknownToTheManifestReportsNoName(t *testing.T) {
	repo := &importRepo{}
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"dimwishlist:item=-69420&perks=555\n")
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	u := report.Lines[0].Unresolved
	if u == nil || u.Hash != 555 || u.Name != "" || u.Reason != UnresolvedNotInPool {
		t.Errorf("unresolved = %+v, want {Hash:555 Name:\"\" Reason:not-in-pool}", u)
	}
}

// A line that fails to store for a reason other than duplication or perk
// resolution — an overlong note, say — reports OutcomeFailed with a detail
// that has had this package's log prefix stripped, the same as every other
// outcome.
func TestImportDIM_FailedLineDetailHasNoPackagePrefix(t *testing.T) {
	repo := &importRepo{}
	longNote := strings.Repeat("x", 501)
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"dimwishlist:item=1000&perks=111#notes:"+longNote)
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	if got := report.Lines[0].Outcome; got != OutcomeFailed {
		t.Fatalf("outcome = %q, want %q", got, OutcomeFailed)
	}
	if strings.Contains(report.Lines[0].Detail, "rolltargets:") {
		t.Errorf("detail carried the package prefix: %q", report.Lines[0].Detail)
	}
	if report.Lines[0].Detail == "" {
		t.Error("the failing line lost its reason")
	}
}

// No detail on the wire may carry this package's log-oriented "rolltargets:"
// prefix, across every outcome a file can produce in one pass — including the
// ones that resolve to a Go sentinel error rather than a *UnresolvedPerk.
func TestImportDIM_NoDetailCarriesThePackagePrefix(t *testing.T) {
	repo := &importRepo{targets: []StoredTarget{
		{ID: 1, ItemHash: weapon(1000), Wanted: true, Perks: []string{"Outlaw"}},
	}}
	text := "dimwishlist:item=1000&perks=111\n" + // imported
		"dimwishlist:item=4242&perks=111\n" + // unknown weapon
		"dimwishlist:item=1000&perks=111\n" + // already saved
		"dimwishlist:item=1000&perks=999\n" + // unresolved: not in pool
		"dimwishlist:item=1000&perks=333\n" + // unresolved: ambiguous
		"dimwishlist:item=-69420&perks=444\n" + // unresolved: not a weapon perk
		"nonsense\n" // malformed
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1", text)
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	for _, l := range report.Lines {
		if strings.Contains(l.Detail, "rolltargets:") {
			t.Errorf("line %d (%s) detail carried the package prefix: %q", l.Number, l.Outcome, l.Detail)
		}
	}
}

func outcomes(r ImportReport) []ImportOutcome {
	out := make([]ImportOutcome, len(r.Lines))
	for i, l := range r.Lines {
		out[i] = l.Outcome
	}
	return out
}

// The acceptance case: a file mixing every kind of line yields one outcome per
// line, in order, with nothing dropped.
func TestImportDIM_OneOutcomePerLine(t *testing.T) {
	repo := &importRepo{}
	text := "title:mine\n" +
		"dimwishlist:item=1000&perks=111,222\n" + // imports
		"dimwishlist:item=4242&perks=111\n" + // unknown weapon
		"dimwishlist:item=1000&perks=111,222\n" + // the identical roll again
		"dimwishlist:item=1000&perks=999\n" + // a perk this weapon cannot roll
		"dimwishlist:item=1000\n" + // no perks named
		"nonsense\n" // malformed
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1", text)
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	want := []ImportOutcome{
		OutcomeSkipped, OutcomeImported, OutcomeUnknownWeapon, OutcomeAlreadySaved,
		OutcomeUnresolvedPerk, OutcomeUnsupported, OutcomeMalformed,
	}
	got := outcomes(report)
	if len(got) != len(want) {
		t.Fatalf("outcomes = %v, want %d entries", got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d outcome = %q, want %q", i+1, got[i], want[i])
		}
	}
	if report.Title != "mine" {
		t.Errorf("title = %q", report.Title)
	}
	if report.Imported() != 1 {
		t.Errorf("imported = %d, want 1", report.Imported())
	}
}

// DIM files carry base perk hashes and expect the reader to match the enhanced
// variant too. Both spellings must land on the same stored perk.
func TestImportDIM_ResolvesBaseAndEnhancedToOneName(t *testing.T) {
	for _, hash := range []string{"111", "911"} {
		repo := &importRepo{}
		_, err := importSvc(repo).ImportDIM(context.Background(), "m1",
			"dimwishlist:item=1000&perks="+hash)
		if err != nil {
			t.Fatalf("ImportDIM(%s): %v", hash, err)
		}
		if len(repo.targets) != 1 || len(repo.targets[0].Perks) != 1 || repo.targets[0].Perks[0] != "Outlaw" {
			t.Errorf("perks=%s stored %+v, want [Outlaw]", hash, repo.targets)
		}
	}
}

// A file naming both variants of one perk is naming one wanted perk, not two.
func TestImportDIM_CollapsesBothVariantsOfOnePerk(t *testing.T) {
	repo := &importRepo{}
	if _, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"dimwishlist:item=1000&perks=111,911,222"); err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	// Sorted, because the perks are AND-ed and their order carries no meaning.
	got := repo.targets[0].Perks
	if len(got) != 2 || got[0] != "Firefly" || got[1] != "Outlaw" {
		t.Errorf("perks = %v, want [Firefly Outlaw]", got)
	}
}

// An ambiguous plug names more than one hash, so the file's intent is unknown.
// Refusing beats guessing.
func TestImportDIM_RefusesAnAmbiguousPlug(t *testing.T) {
	repo := &importRepo{}
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"dimwishlist:item=1000&perks=333")
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	if got := report.Lines[0].Outcome; got != OutcomeUnresolvedPerk {
		t.Fatalf("outcome = %q, want %q", got, OutcomeUnresolvedPerk)
	}
	if len(repo.targets) != 0 {
		t.Error("an ambiguous perk was stored")
	}
}

// An identical roll is skipped and left as it was, which is what makes
// re-importing the same file harmless.
func TestImportDIM_SkipsARollAlreadySaved(t *testing.T) {
	repo := &importRepo{targets: []StoredTarget{
		{ID: 1, ItemHash: weapon(1000), Wanted: true, Perks: []string{"Outlaw"}, Notes: "mine"},
	}}
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"dimwishlist:item=1000&perks=111")
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	if got := report.Lines[0].Outcome; got != OutcomeAlreadySaved {
		t.Errorf("outcome = %q, want %q", got, OutcomeAlreadySaved)
	}
	if len(repo.targets) != 1 || repo.targets[0].Notes != "mine" {
		t.Errorf("existing target was modified: %+v", repo.targets)
	}
}

// A different roll on a weapon the player already targets is ordinary: a wish
// list normally lists several acceptable rolls for one gun.
func TestImportDIM_AddsASecondRollForTheSameWeapon(t *testing.T) {
	repo := &importRepo{targets: []StoredTarget{
		{ID: 1, ItemHash: weapon(1000), Wanted: true, Perks: []string{"Outlaw"}},
	}}
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"dimwishlist:item=1000&perks=222")
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	if got := report.Lines[0].Outcome; got != OutcomeImported {
		t.Fatalf("outcome = %q, want %q", got, OutcomeImported)
	}
	if len(repo.targets) != 2 {
		t.Errorf("targets = %d, want 2", len(repo.targets))
	}
}

// DIM's unwanted roll and any-item wildcard both import now, each keeping what
// distinguishes it.
func TestImportDIM_StoresUnwantedRollsAndWildcards(t *testing.T) {
	repo := &importRepo{}
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"dimwishlist:item=-1000&perks=111\ndimwishlist:item=-69420&perks=222\n")
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	for i, l := range report.Lines {
		if l.Outcome != OutcomeImported {
			t.Fatalf("line %d outcome = %q (%s), want imported", i+1, l.Outcome, l.Detail)
		}
	}
	if len(repo.targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(repo.targets))
	}
	unwanted := repo.targets[0]
	if unwanted.Wanted || unwanted.ItemHash == nil || *unwanted.ItemHash != 1000 {
		t.Errorf("unwanted roll = %+v, want wanted=false on weapon 1000", unwanted)
	}
	wildcard := repo.targets[1]
	if !wildcard.Wanted || wildcard.ItemHash != nil {
		t.Errorf("wildcard = %+v, want wanted=true with no weapon", wildcard)
	}
	if len(wildcard.Perks) != 1 || wildcard.Perks[0] != "Firefly" {
		t.Errorf("wildcard perks = %v, want [Firefly]", wildcard.Perks)
	}
}

// Notes ride along, including block notes that apply to a following roll.
func TestImportDIM_CarriesNotesOntoTheTarget(t *testing.T) {
	repo := &importRepo{}
	if _, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"//notes:pvp set\ndimwishlist:item=1000&perks=111"); err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	if repo.targets[0].Notes != "pvp set" {
		t.Errorf("notes = %q, want \"pvp set\"", repo.targets[0].Notes)
	}
}

// Persistence being absent is one condition about the whole import, not a
// verdict repeated on every line.
func TestImportDIM_UnavailablePersistenceFailsTheImport(t *testing.T) {
	repo := &importRepo{addErr: ErrUnavailable}
	_, err := importSvc(repo).ImportDIM(context.Background(), "m1", "dimwishlist:item=1000&perks=111")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

// An unreadable perk pool is not a verdict about any weapon, so it stops the
// import rather than marking every line "unknown weapon".
func TestImportDIM_UnreadablePoolStopsTheImport(t *testing.T) {
	svc := NewService(&importRepo{}, importPool{err: errors.New("manifest warming")}, nil)
	_, err := svc.ImportDIM(context.Background(), "m1", "dimwishlist:item=1000&perks=111")
	if !errors.Is(err, ErrPerksUnavailable) {
		t.Fatalf("err = %v, want ErrPerksUnavailable", err)
	}
}

// One bad line does not abandon the rest of the file.
func TestImportDIM_ContinuesPastABadLine(t *testing.T) {
	repo := &importRepo{}
	report, err := importSvc(repo).ImportDIM(context.Background(), "m1",
		"garbage\ndimwishlist:item=1000&perks=222\n")
	if err != nil {
		t.Fatalf("ImportDIM: %v", err)
	}
	if report.Imported() != 1 {
		t.Errorf("imported = %d, want 1; outcomes %v", report.Imported(), outcomes(report))
	}
}
