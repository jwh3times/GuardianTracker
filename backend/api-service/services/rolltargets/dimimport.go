package rolltargets

import (
	"context"
	"errors"
	"fmt"

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

// ImportLine reports one line's fate, with enough detail to point the reader at
// the file and say why.
type ImportLine struct {
	Number  int
	Outcome ImportOutcome
	Detail  string

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
				out.Outcome, out.Detail = OutcomeUnknownWeapon, ErrNotAWeapon.Error()
			case errors.Is(err, ErrPerksUnavailable):
				return ImportReport{}, err
			default:
				out.Outcome, out.Detail = OutcomeUnresolvedPerk, err.Error()
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
			// A wildcard line's hashes resolved to names, but not to names any
			// weapon perk carries (a mod, say). It could never match.
			out.Outcome, out.Detail = OutcomeUnresolvedPerk, err.Error()
		default:
			out.Outcome, out.Detail = OutcomeFailed, err.Error()
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
func (s *Service) resolveDIMPerks(itemHash *uint32, hashes []uint32) ([]string, error) {
	byHash, err := s.dimPerkIndex(itemHash, hashes)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(hashes))
	for _, h := range hashes {
		plug, ok := byHash[h]
		if !ok {
			return nil, fmt.Errorf("%w: %d", ErrUnknownPerk, h)
		}
		if plug.Ambiguous {
			return nil, fmt.Errorf("%w: %d names %q, which resolves to more than one plug",
				ErrUnknownPerk, h, plug.Name)
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
