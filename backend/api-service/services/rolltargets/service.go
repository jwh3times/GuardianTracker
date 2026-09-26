package rolltargets

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Service is the handler-facing owner of roll-target reads and mutations. It
// validates a target against the weapon's own perk pool before persisting it,
// so a saved target is one that can actually match.
type Service struct {
	repo  Repository
	perks PerkPool
	owned OwnedRollReader
}

// NewService builds the roll-target service over its persistence port, the
// manifest perk pool it validates against, and the source of the weapons the
// player currently owns.
//
// owned may be nil, which leaves every stored and imported capability intact
// and makes matching report ErrOwnedRollsUnavailable rather than an empty
// result — a deployment without it has no matches to show, not no matches.
func NewService(repo Repository, perks PerkPool, owned OwnedRollReader) *Service {
	return &Service{repo: repo, perks: perks, owned: owned}
}

// List returns the membership's saved roll targets in repository order.
func (s *Service) List(ctx context.Context, membershipID string) ([]StoredTarget, error) {
	return s.repo.List(ctx, membershipID)
}

// Add validates and saves one weapon's wanted roll.
func (s *Service) Add(ctx context.Context, membershipID string, cmd AddCommand) (StoredTarget, error) {
	cmd.ImportID, cmd.ImportTitle = "", ""
	return s.add(ctx, membershipID, cmd)
}

func (s *Service) add(ctx context.Context, membershipID string, cmd AddCommand) (StoredTarget, error) {
	perks, err := s.validatePerks(cmd.ItemHash, cmd.Perks)
	if err != nil {
		return StoredTarget{}, err
	}
	if err := validateNotes(cmd.Notes); err != nil {
		return StoredTarget{}, err
	}
	cmd.Perks = perks
	return s.repo.Add(ctx, membershipID, cmd)
}

// Update validates and applies a patch to one target the membership owns.
//
// Validating replacement perks needs the weapon they belong to, and the patch
// does not carry it — so the target is read first. That read is also what turns
// a patch against a missing or foreign target into ErrNotFound before anything
// is written.
func (s *Service) Update(ctx context.Context, membershipID string, id TargetID, patch UpdateCommand) (StoredTarget, error) {
	if patch.Perks == nil && patch.Notes == nil {
		// Nothing to apply. Reading the target back is the honest answer, and
		// it still reports ErrNotFound for a target that is not theirs.
		return s.find(ctx, membershipID, id)
	}
	if patch.Notes != nil {
		if err := validateNotes(*patch.Notes); err != nil {
			return StoredTarget{}, err
		}
	}
	if patch.Perks != nil {
		current, err := s.find(ctx, membershipID, id)
		if err != nil {
			return StoredTarget{}, err
		}
		perks, err := s.validatePerks(current.ItemHash, *patch.Perks)
		if err != nil {
			return StoredTarget{}, err
		}
		patch.Perks = &perks
	}
	return s.repo.Update(ctx, membershipID, id, patch)
}

// Remove deletes one target the membership owns.
func (s *Service) Remove(ctx context.Context, membershipID string, id TargetID) error {
	return s.repo.Remove(ctx, membershipID, id)
}

// DeleteMany removes every listed target the membership owns. Missing and
// foreign ids are skipped and counted rather than failing the command.
func (s *Service) DeleteMany(ctx context.Context, membershipID string, ids []TargetID) (BulkResult, error) {
	unique, err := bulkTargetIDs(ids)
	if err != nil {
		return BulkResult{}, err
	}
	removed, err := s.repo.RemoveMany(ctx, membershipID, unique)
	if err != nil {
		return BulkResult{}, err
	}
	return bulkTargetResult(unique, removed), nil
}

// DeleteAll removes every target the membership owns.
func (s *Service) DeleteAll(ctx context.Context, membershipID string) (int, error) {
	return s.repo.RemoveAll(ctx, membershipID)
}

// DeleteImport removes targets newly saved by one import for this membership.
func (s *Service) DeleteImport(ctx context.Context, membershipID string, id ImportID) (int, error) {
	parsed, err := uuid.Parse(string(id))
	if err != nil || parsed == uuid.Nil {
		return 0, ErrInvalidImportID
	}
	return s.repo.RemoveImport(ctx, membershipID, ImportID(parsed.String()))
}

// find resolves one target the membership owns, or ErrNotFound.
//
// It reads the list rather than a single row because the Repository port has no
// single-target read: adding one would be a second way to ask the same
// question, and a membership's saved targets are a short list.
func (s *Service) find(ctx context.Context, membershipID string, id TargetID) (StoredTarget, error) {
	targets, err := s.repo.List(ctx, membershipID)
	if err != nil {
		return StoredTarget{}, err
	}
	for _, t := range targets {
		if t.ID == id {
			return t, nil
		}
	}
	return StoredTarget{}, ErrNotFound
}

// validatePerks checks a target's perks and returns them canonicalised and
// sorted.
//
// A target naming a weapon is checked against that weapon's own pool. An
// any-weapon target has no pool, so its names are checked against the
// manifest's plugs instead — a weaker check, but the alternative is storing a
// name nothing can ever match.
//
// Matching is case-insensitive on the display name but the stored value keeps
// the manifest's spelling, so a target written "kill clip" persists as
// "Kill Clip" and reads back the way the game writes it. The result is sorted
// because the perks are AND-ed: order carries no meaning, and normalising it is
// what lets the same roll be recognised as already saved.
func (s *Service) validatePerks(itemHash *uint32, perks []string) ([]string, error) {
	cleaned := make([]string, 0, len(perks))
	for _, p := range perks {
		if t := strings.TrimSpace(p); t != "" {
			cleaned = append(cleaned, t)
		}
	}
	switch {
	case len(cleaned) == 0:
		return nil, ErrNoPerks
	case len(cleaned) > MaxPerks:
		return nil, ErrTooManyPerks
	}

	if itemHash == nil {
		return s.validateAnyWeaponPerks(cleaned)
	}

	pool, err := s.pool(*itemHash)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(cleaned))
	out := make([]string, 0, len(cleaned))
	for _, p := range cleaned {
		canonical, ok := pool[strings.ToLower(p)]
		if !ok {
			return nil, ErrUnknownPerk
		}
		if _, dup := seen[canonical]; dup {
			return nil, ErrDuplicatePerk
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	sort.Strings(out)
	return out, nil
}

// validateAnyWeaponPerks checks an any-weapon target's names against every name
// a weapon perk column carries. There is no single pool to check against, but
// the union of all of them is exactly the set a weapon-bound target could
// name, so a name outside it could never match. A plug hash resolving to a
// name is not enough: mods and other non-perk plugs have names too.
func (s *Service) validateAnyWeaponPerks(perks []string) ([]string, error) {
	names, err := s.perks.WeaponPerkNames()
	if err != nil {
		return nil, ErrPerksUnavailable
	}
	seen := make(map[string]struct{}, len(perks))
	out := make([]string, 0, len(perks))
	for _, p := range perks {
		canonical, ok := names[strings.ToLower(p)]
		if !ok {
			return nil, ErrUnknownPerkName
		}
		if _, dup := seen[canonical]; dup {
			return nil, ErrDuplicatePerk
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	sort.Strings(out)
	return out, nil
}

// pool reads the weapon's rollable perk names, keyed by lowercase name.
func (s *Service) pool(itemHash uint32) (map[string]string, error) {
	cols, err := s.perks.GetWeaponPerks(itemHash)
	if err != nil {
		// Every read failure is the same answer here — a warming manifest and a
		// broken one both mean "we cannot tell you what this weapon rolls". What
		// must not collapse into it is a *successful* read of nothing, handled
		// below, which is a verdict about the weapon rather than about the read.
		return nil, ErrPerksUnavailable
	}
	if len(cols) == 0 {
		// A successful read of no columns is a verdict: the hash is not a
		// weapon, or is one with nothing to choose. Distinct from a failed
		// read, which never reaches here.
		return nil, ErrNotAWeapon
	}
	names := map[string]string{}
	for _, c := range cols {
		for _, p := range c.Perks {
			names[strings.ToLower(p)] = p
		}
	}
	if len(names) == 0 {
		return nil, ErrNotAWeapon
	}
	return names, nil
}

func validateNotes(notes string) error {
	if utf8.RuneCountInString(notes) > MaxNoteRunes {
		return ErrNotesTooLong
	}
	return nil
}

// bulkTargetIDs removes duplicate ids before the size rules apply, so a
// client that sends the same target twice is asking about one target — not
// two, and not a larger command than it thinks. Mirrors wishlist's bulkIDs.
func bulkTargetIDs(ids []TargetID) ([]TargetID, error) {
	seen := make(map[TargetID]struct{}, len(ids))
	unique := make([]TargetID, 0, len(ids))
	for _, id := range ids {
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return nil, ErrNoTargets
	}
	if len(unique) > MaxBulkTargets {
		return nil, fmt.Errorf("%w: at most %d", ErrTooManyTargets, MaxBulkTargets)
	}
	return unique, nil
}

// bulkTargetResult reports what the storage layer actually touched. Targets
// that were missing or belong to another membership are skipped rather than
// failing the command, so one stale id in a selection does not discard the
// rest.
func bulkTargetResult(requested []TargetID, affected int) BulkResult {
	if affected < 0 {
		affected = 0
	}
	skipped := len(requested) - affected
	if skipped < 0 {
		skipped = 0
	}
	return BulkResult{Deleted: affected, Skipped: skipped}
}
