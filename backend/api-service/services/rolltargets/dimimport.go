package rolltargets

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"guardian-tracker/api-service/services/manifest"
)

// ImportOutcome is what became of one line of a DIM-format file. Every line of
// the file produces exactly one, in file order.
type ImportOutcome string

const (
	// OutcomeImported means the line became a stored roll target.
	OutcomeImported ImportOutcome = "imported"

	// OutcomeSkipped means the line was a comment or metadata — nothing to
	// import and nothing wrong.
	OutcomeSkipped ImportOutcome = "skipped"

	// OutcomeAlreadySaved means this membership has already saved this exact
	// roll. Several different rolls on one weapon are ordinary and all import;
	// only an identical one is skipped, which is what makes re-importing the
	// same file harmless.
	OutcomeAlreadySaved ImportOutcome = "already saved"

	// OutcomeUnknownWeapon means the item hash resolves to no weapon with perk
	// columns.
	OutcomeUnknownWeapon ImportOutcome = "unknown weapon"

	// OutcomeUnresolvedPerk means a perk hash on the line could not be resolved
	// against the weapon's own pool.
	OutcomeUnresolvedPerk ImportOutcome = "unresolved perk"

	// OutcomeUnsupported means DIM accepts the line but the stored model cannot
	// carry it — a perkless roll, or a legacy URL line.
	OutcomeUnsupported ImportOutcome = "unsupported"

	// OutcomeMalformed means the line is not DIM syntax.
	OutcomeMalformed ImportOutcome = "malformed"

	// OutcomeFailed means the line was valid but could not be stored.
	OutcomeFailed ImportOutcome = "failed"
)

// UnresolvedPerkReason classifies why one DIM line's perk hash did not become
// a stored roll target's perk name.
type UnresolvedPerkReason string

const (
	// UnresolvedNotInPool means a weapon-bound line's hash is not in that
	// weapon's own pool, or a wildcard line's hash is unknown to the manifest
	// entirely (PerkPool.PlugNames has no entry for it). The two collapse into
	// one reason because they are the same fact from two callers: "this hash
	// names no perk this line could use."
	UnresolvedNotInPool UnresolvedPerkReason = "not-in-pool"

	// UnresolvedAmbiguous means the hash resolved to a plug the manifest could
	// not link to a single base/enhanced pair (manifest.PerkPlug.Ambiguous).
	UnresolvedAmbiguous UnresolvedPerkReason = "ambiguous"

	// UnresolvedNotAWeaponPerk means a wildcard line's hash resolved to a real
	// plug name, but not one any weapon's own perk columns carry — a mod or
	// other non-perk plug, which could never match a saved roll target.
	UnresolvedNotAWeaponPerk UnresolvedPerkReason = "not-a-weapon-perk"
)

// UnresolvedPerk is the structured reason one DIM line's perk hash did not
// resolve into a stored roll target's perk name. Name is empty when the
// hash's own display name could not be resolved either — UnresolvedNotInPool
// is the only reason that can leave it empty.
type UnresolvedPerk struct {
	Hash   uint32
	Name   string
	Reason UnresolvedPerkReason
}

// Error is a plain-language message with no package prefix, safe to put on
// the wire as an import line's Detail.
func (u *UnresolvedPerk) Error() string {
	switch u.Reason {
	case UnresolvedAmbiguous:
		return fmt.Sprintf("perk %q resolves to more than one plug", u.Name)
	case UnresolvedNotAWeaponPerk:
		return fmt.Sprintf("perk %q is not one any weapon can roll", u.Name)
	default:
		if u.Name != "" {
			return fmt.Sprintf("this weapon cannot roll %q", u.Name)
		}
		// A wildcard line lands here too, where there is no weapon to name.
		return fmt.Sprintf("no perk matches hash %d", u.Hash)
	}
}

// ImportLine reports one line's fate, with enough detail to point the reader at
// the file and say why.
type ImportLine struct {
	Number  int
	Outcome ImportOutcome
	Detail  string

	// Unresolved is set only for OutcomeUnresolvedPerk lines whose failure
	// traces to one specific perk hash — never for a naming problem with no
	// single hash to blame (an empty or oversized perk set).
	Unresolved *UnresolvedPerk

	// ItemHash is nil for an any-weapon roll.
	ItemHash *uint32
	Wanted   bool
	Perks    []string
}

// ImportReport is the result of importing a DIM-format file: the file's own
// metadata, and one outcome per line.
//
// Nothing is summarised away. A line that did not import is reported with the
// reason, because a line that vanishes is indistinguishable from one that
// worked — which is the failure mode CONTEXT.md names.
type ImportReport struct {
	Title       string
	Description string
	Lines       []ImportLine
}

// Counts tallies the report by outcome.
func (r ImportReport) Counts() map[ImportOutcome]int {
	out := map[ImportOutcome]int{}
	for _, l := range r.Lines {
		out[l.Outcome]++
	}
	return out
}

// Imported returns how many lines became stored targets.
func (r ImportReport) Imported() int { return r.Counts()[OutcomeImported] }

// ImportDIM parses a DIM-format file and stores every roll it can represent.
//
// Import never overwrites: an identical roll is reported and left as it was. A
// DIM file is a suggestion, and a hand-authored target is not something a
// suggestion should silently replace.
//
// A storage failure on one line does not abandon the rest — the remaining lines
// are still attempted and reported — except when persistence is unavailable
// entirely, which is returned as an error rather than repeated on every line.
//
// Duplicate suppression is the store's uniqueness constraint rather than a
// pre-read, so two identical lines in one file behave exactly like a re-import.
func (s *Service) ImportDIM(ctx context.Context, membershipID, text string) (ImportReport, error) {
	file := ParseDIMFile(text)
	report := ImportReport{Title: file.Title, Description: file.Description}

	for _, line := range file.Lines {
		out := ImportLine{Number: line.Number, Outcome: OutcomeSkipped}
		switch line.Kind {
		case DIMLineComment:
			report.Lines = append(report.Lines, out)
			continue
		case DIMLineUnsupported:
			out.Outcome, out.Detail = OutcomeUnsupported, line.Reason
			report.Lines = append(report.Lines, out)
			continue
		case DIMLineMalformed:
			out.Outcome, out.Detail = OutcomeMalformed, line.Reason
			report.Lines = append(report.Lines, out)
			continue
		}

		out.Wanted = line.Wanted
		var hash *uint32
		if !line.AnyWeapon {
			h := line.ItemHash
			hash = &h
			out.ItemHash = &h
		}

		perks, err := s.resolveDIMPerks(hash, line.PerkHashes)
		if err != nil {
			switch {
			case errors.Is(err, ErrNotAWeapon):
				out.Outcome, out.Detail = OutcomeUnknownWeapon, "item hash does not resolve to a weapon with perk columns"
			case errors.Is(err, ErrPerksUnavailable):
				return ImportReport{}, err
			default:
				out.Outcome = OutcomeUnresolvedPerk
				var unresolved *UnresolvedPerk
				switch {
				case errors.As(err, &unresolved):
					out.Unresolved, out.Detail = unresolved, unresolved.Error()
				case errors.Is(err, ErrNoPerks):
					out.Detail = "names no perk this line could use"
				case errors.Is(err, ErrTooManyPerks):
					out.Detail = "names more perks than one roll target can hold"
				default:
					out.Detail = "could not resolve this line's perks"
				}
			}
			report.Lines = append(report.Lines, out)
			continue
		}
		out.Perks = perks

		_, err = s.Add(ctx, membershipID, AddCommand{
			ItemHash: hash, Wanted: line.Wanted, Perks: perks, Notes: line.Notes,
		})
		switch {
		case err == nil:
			out.Outcome = OutcomeImported
		case errors.Is(err, ErrUnavailable):
			return ImportReport{}, err
		case errors.Is(err, ErrDuplicate):
			out.Outcome = OutcomeAlreadySaved
		case errors.Is(err, ErrPerksUnavailable):
			return ImportReport{}, err
		case errors.Is(err, ErrUnknownPerkName):
			// Defensive fallback: resolveDIMPerks already checks a wildcard
			// line's names against WeaponPerkNames before this call is ever
			// made, so this path should not be reachable in practice. Kept
			// because Add performs its own check regardless of caller, and a
			// manifest swap between the two reads is a real (if rare) window.
			out.Outcome, out.Detail = OutcomeUnresolvedPerk, "not one any weapon can roll"
		default:
			out.Outcome, out.Detail = OutcomeFailed, plainDetail(err)
		}
		report.Lines = append(report.Lines, out)
	}
	return report, nil
}

// resolveDIMPerks turns a DIM line's perk hashes into this weapon's perk names.
//
// DIM files carry *base* perk hashes and rely on the reader to match the
// enhanced variant too — its own documentation says "select the regular version
// and Enhanced versions will be matched automatically". That is exactly what
// PerkColumn.Plugs provides, so a hash is looked up against both Base and
// Enhanced and resolves to the one name they share.
//
// A plug the manifest could not resolve to a single hash per variant is marked
// Ambiguous, and an ambiguous plug refuses to resolve rather than guessing
// which of several same-named perks the file meant.
//
// A wildcard line's names are additionally checked against WeaponPerkNames
// here, before the caller ever tries to store them — duplicating the check
// Add itself performs (see rolltargets.go's ErrUnknownPerkName), but only
// here is the offending hash still in scope, which is what lets an import
// report say which one failed instead of just that one did.
func (s *Service) resolveDIMPerks(itemHash *uint32, hashes []uint32) ([]string, error) {
	byHash, err := s.dimPerkIndex(itemHash, hashes)
	if err != nil {
		return nil, err
	}

	var weaponNames map[string]string
	if itemHash == nil {
		weaponNames, err = s.perks.WeaponPerkNames()
		if err != nil {
			return nil, ErrPerksUnavailable
		}
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(hashes))
	for _, h := range hashes {
		plug, ok := byHash[h]
		if !ok {
			// Not in this weapon's own pool, or (wildcard) unknown to the
			// manifest entirely — PlugNames simply omits a hash it cannot
			// name, so there is no name to report here either.
			return nil, &UnresolvedPerk{Hash: h, Reason: UnresolvedNotInPool}
		}
		if plug.Ambiguous {
			return nil, &UnresolvedPerk{Hash: h, Name: plug.Name, Reason: UnresolvedAmbiguous}
		}
		if weaponNames != nil {
			if _, ok := weaponNames[strings.ToLower(plug.Name)]; !ok {
				return nil, &UnresolvedPerk{Hash: h, Name: plug.Name, Reason: UnresolvedNotAWeaponPerk}
			}
		}
		// A file may name both variants of one perk; they are one wanted perk.
		if _, dup := seen[plug.Name]; dup {
			continue
		}
		seen[plug.Name] = struct{}{}
		out = append(out, plug.Name)
	}
	if len(out) == 0 {
		return nil, ErrNoPerks
	}
	if len(out) > MaxPerks {
		return nil, ErrTooManyPerks
	}
	return out, nil
}

// plainDetail turns any domain error into wire-safe text: this package's
// sentinels all carry a "rolltargets: " log prefix, which a caller reading an
// import report must never see.
func plainDetail(err error) string {
	return strings.TrimPrefix(err.Error(), "rolltargets: ")
}

// dimPerkIndex builds the hash-to-perk index a line resolves against.
//
// A line naming a weapon resolves against that weapon's own pool, which is the
// only scope where a perk name is unambiguous. A wildcard line names no weapon,
// so there is no pool — its hashes are resolved straight to plug names instead.
// That is sound in this direction only: many plugs share a name, but each hash
// has exactly one, and a perk's two variants share theirs, so base and enhanced
// both normalise to the same answer.
func (s *Service) dimPerkIndex(itemHash *uint32, hashes []uint32) (map[uint32]manifest.PerkPlug, error) {
	if itemHash == nil {
		names, err := s.perks.PlugNames(hashes)
		if err != nil {
			return nil, ErrPerksUnavailable
		}
		out := make(map[uint32]manifest.PerkPlug, len(names))
		for h, name := range names {
			out[h] = manifest.PerkPlug{Name: name, Base: h}
		}
		return out, nil
	}

	cols, err := s.perks.GetWeaponPerks(*itemHash)
	if err != nil {
		return nil, ErrPerksUnavailable
	}
	if len(cols) == 0 {
		return nil, ErrNotAWeapon
	}
	out := map[uint32]manifest.PerkPlug{}
	for _, c := range cols {
		for _, p := range c.Plugs {
			if p.Base != 0 {
				out[p.Base] = p
			}
			if p.Enhanced != 0 {
				out[p.Enhanced] = p
			}
		}
	}
	return out, nil
}
