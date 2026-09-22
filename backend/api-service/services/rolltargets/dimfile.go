package rolltargets

import (
	"regexp"
	"strconv"
	"strings"
)

// DIM-format parsing.
//
// The grammar below is taken from DIM's own parser
// (DestinyItemManager/DIM, src/app/wishlists/wishlist-file.ts), not from its
// prose documentation — an owner-supplied file was written against the
// implementation, so the implementation is the contract. Verified 2026-09-22;
// the evidence is recorded on the issue this landed under.

// dimRollLine is DIM's own line regex, transcribed to Go.
//
//	/^dimwishlist:item=(?<itemHash>-?\d+)(?:&perks=)?(?<itemPerks>[\d|,]*)(?:#notes:)?(?<wishListNotes>[^|]*)/
//
// Note what is optional: `&perks=` may be absent entirely, and the perk charset
// admits pipes as well as digits and commas.
var dimRollLine = regexp.MustCompile(`^dimwishlist:item=(-?\d+)(?:&perks=)?([\d|,]*)(?:#notes:)?([^|]*)`)

var (
	dimTitleLine       = regexp.MustCompile(`^@?title:(.+)$`)
	dimDescriptionLine = regexp.MustCompile(`^@?description:(.+)$`)
	dimLegacyDTR       = regexp.MustCompile(`^https://destinytracker\.com/destiny-2/db/items/\d+`)
	dimLegacyBanshee   = regexp.MustCompile(`^https://banshee-44\.com/\?weapon=`)
)

const (
	dimBlockNotesPrefix = "//notes:"
	dimCommentPrefix    = "//"

	// dimWildcardItemID is DIM's WildcardItemId: a roll that applies to every
	// item rather than to one weapon. Its literal value is part of the format.
	dimWildcardItemID = -69420
)

// DIMLineKind classifies one line of a DIM-format file. Every line of a parsed
// file gets exactly one kind — a line is never silently dropped, because a
// dropped line is indistinguishable from one that imported.
type DIMLineKind string

const (
	// DIMLineRoll is a roll this model can represent: at least one perk, on a
	// named weapon or on DIM's any-item wildcard, wanted or unwanted.
	DIMLineRoll DIMLineKind = "roll"

	// DIMLineComment is a blank line, a `//` comment, block notes, or the
	// file's title/description. Nothing to import and nothing wrong.
	DIMLineComment DIMLineKind = "comment"

	// DIMLineUnsupported is a line DIM accepts that Guardian Tracker's stored
	// model cannot carry. Reason says which.
	DIMLineUnsupported DIMLineKind = "unsupported"

	// DIMLineMalformed is a line that is not DIM syntax at all.
	DIMLineMalformed DIMLineKind = "malformed"
)

// Reasons attached to DIMLineUnsupported and DIMLineMalformed.
const (
	ReasonNoPerks       = "names no perks, so it would match every copy of the weapon"
	ReasonLegacyURL     = "a legacy banshee-44 or destinytracker line, which is not supported"
	ReasonUnrecognized  = "not a recognized DIM wish list line"
	ReasonItemHashRange = "item hash is outside the range of a Destiny item hash"
)

// DIMLine is one parsed line of a DIM-format file, in file order.
type DIMLine struct {
	// Number is the 1-based line number, so a report can point at the file.
	Number int
	Kind   DIMLineKind
	Reason string

	// ItemHash, AnyWeapon, Wanted and PerkHashes are set only for DIMLineRoll.
	//
	// AnyWeapon marks DIM's wildcard item: perks without a weapon. Wanted is
	// false for DIM's undesirable ("trash") roll, which it writes as a negative
	// item hash.
	ItemHash   uint32
	AnyWeapon  bool
	Wanted     bool
	PerkHashes []uint32

	// Notes is the line's own `#notes:` text, or the block notes in force when
	// the line has none of its own.
	Notes string
}

// DIMFile is a parsed DIM-format file: its metadata and one outcome per line.
type DIMFile struct {
	Title       string
	Description string
	Lines       []DIMLine
}

// Rolls returns only the lines this model can import, in file order.
func (f DIMFile) Rolls() []DIMLine {
	var out []DIMLine
	for _, l := range f.Lines {
		if l.Kind == DIMLineRoll {
			out = append(out, l)
		}
	}
	return out
}

// ParseDIMFile parses DIM wish-list text into one outcome per line.
//
// It is pure syntax: it resolves no hashes and consults no manifest, so a line
// reported as a roll here may still turn out to name a weapon or a perk that
// does not exist. That verdict belongs to the service, which can read the
// manifest.
func ParseDIMFile(text string) DIMFile {
	var file DIMFile
	blockNotes := ""

	// A file that ends in a newline has no extra line after it. Splitting the
	// raw text would invent one, and a phantom line in a per-line report points
	// the reader at something their file does not contain.
	text = strings.TrimSuffix(strings.TrimSuffix(text, "\n"), "\r")

	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, "\r")
		num := i + 1

		// Block notes must be tested before the generic comment rule, since
		// they also begin with "//" — the order DIM itself uses.
		if strings.HasPrefix(line, dimBlockNotesPrefix) {
			blockNotes = dimNotes(strings.TrimPrefix(line, dimBlockNotesPrefix))
			file.Lines = append(file.Lines, DIMLine{Number: num, Kind: DIMLineComment})
			continue
		}
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, dimCommentPrefix) {
			file.Lines = append(file.Lines, DIMLine{Number: num, Kind: DIMLineComment})
			continue
		}
		if m := dimTitleLine.FindStringSubmatch(line); m != nil {
			file.Title = strings.TrimSpace(m[1])
			file.Lines = append(file.Lines, DIMLine{Number: num, Kind: DIMLineComment})
			continue
		}
		if m := dimDescriptionLine.FindStringSubmatch(line); m != nil {
			file.Description = strings.TrimSpace(m[1])
			file.Lines = append(file.Lines, DIMLine{Number: num, Kind: DIMLineComment})
			continue
		}
		if dimLegacyDTR.MatchString(line) || dimLegacyBanshee.MatchString(line) {
			file.Lines = append(file.Lines, DIMLine{Number: num, Kind: DIMLineUnsupported, Reason: ReasonLegacyURL})
			continue
		}
		file.Lines = append(file.Lines, parseDIMRoll(line, num, blockNotes))
	}
	return file
}

func parseDIMRoll(line string, num int, blockNotes string) DIMLine {
	m := dimRollLine.FindStringSubmatch(line)
	if m == nil {
		return DIMLine{Number: num, Kind: DIMLineMalformed, Reason: ReasonUnrecognized}
	}

	notes := dimNotes(m[3])
	if notes == "" {
		notes = blockNotes
	}
	out := DIMLine{Number: num, Notes: notes}

	// The regex admits any signed integer, including ones no item hash can
	// hold, so the range check is part of parsing rather than an assumption.
	raw, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return DIMLine{Number: num, Kind: DIMLineMalformed, Reason: ReasonUnrecognized}
	}
	out.Wanted = true
	switch {
	case raw == dimWildcardItemID:
		// The wildcard is checked before the sign test: it is negative, but it
		// means "every item", not "unwanted".
		out.AnyWeapon = true
	case raw < 0:
		// DIM writes an undesirable roll as a negative hash and takes the
		// absolute value itself.
		out.Wanted = false
		raw = -raw
		if raw > 4294967295 {
			out.Kind, out.Reason = DIMLineMalformed, ReasonItemHashRange
			return out
		}
		out.ItemHash = uint32(raw)
	case raw > 4294967295:
		out.Kind, out.Reason = DIMLineMalformed, ReasonItemHashRange
		return out
	default:
		out.ItemHash = uint32(raw)
	}

	out.PerkHashes = dimPerkHashes(m[2])
	if len(out.PerkHashes) == 0 {
		out.Kind, out.Reason = DIMLineUnsupported, ReasonNoPerks
		return out
	}
	out.Kind = DIMLineRoll
	return out
}

// dimPerkHashes splits DIM's perk list the way DIM does: comma-separated, and
// anything that is not a positive number is dropped.
//
// The charset the line regex admits includes pipes, so an entry like "123|456"
// reaches here intact and is discarded — which is what DIM does with it too,
// since Number("123|456") is NaN.
func dimPerkHashes(field string) []uint32 {
	var out []uint32
	for _, part := range strings.Split(field, ",") {
		n, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
		if err != nil || n == 0 {
			continue
		}
		out = append(out, uint32(n))
	}
	return out
}

// dimNotes expands the escape sequence DIM writes for a line break.
func dimNotes(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, `\n`, "\n"))
}
