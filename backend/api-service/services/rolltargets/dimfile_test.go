package rolltargets

import "testing"

func kindsOf(f DIMFile) []DIMLineKind {
	out := make([]DIMLineKind, len(f.Lines))
	for i, l := range f.Lines {
		out[i] = l.Kind
	}
	return out
}

func TestParseDIMFile_MetadataAndComments(t *testing.T) {
	f := ParseDIMFile("title:My rolls\n@description: what it covers \n// a comment\n\n")
	if f.Title != "My rolls" {
		t.Errorf("title = %q", f.Title)
	}
	// The leading @ is optional and the value is trimmed.
	if f.Description != "what it covers" {
		t.Errorf("description = %q", f.Description)
	}
	for _, k := range kindsOf(f) {
		if k != DIMLineComment {
			t.Fatalf("kinds = %v, want all comment", kindsOf(f))
		}
	}
}

func TestParseDIMFile_RollWithPerksAndNotes(t *testing.T) {
	f := ParseDIMFile(`dimwishlist:item=1234&perks=111,222#notes:pvp roll`)
	if len(f.Lines) != 1 || f.Lines[0].Kind != DIMLineRoll {
		t.Fatalf("lines = %+v", f.Lines)
	}
	l := f.Lines[0]
	if l.ItemHash != 1234 || len(l.PerkHashes) != 2 || l.PerkHashes[0] != 111 || l.PerkHashes[1] != 222 {
		t.Errorf("roll = %+v", l)
	}
	if l.Notes != "pvp roll" {
		t.Errorf("notes = %q", l.Notes)
	}
	if l.Number != 1 {
		t.Errorf("line number = %d, want 1", l.Number)
	}
}

// Block notes apply to every following roll until replaced, and a line's own
// notes win over them.
func TestParseDIMFile_BlockNotesApplyUntilReplaced(t *testing.T) {
	f := ParseDIMFile("//notes:pve set\n" +
		"dimwishlist:item=1&perks=10\n" +
		"dimwishlist:item=2&perks=20#notes:own note\n" +
		"//notes:pvp set\n" +
		"dimwishlist:item=3&perks=30\n")
	rolls := f.Rolls()
	if len(rolls) != 3 {
		t.Fatalf("rolls = %d, want 3", len(rolls))
	}
	want := []string{"pve set", "own note", "pvp set"}
	for i, w := range want {
		if rolls[i].Notes != w {
			t.Errorf("roll %d notes = %q, want %q", i, rolls[i].Notes, w)
		}
	}
}

// "//notes:" also begins with "//", so it must be tested before the comment
// rule — otherwise every block note is swallowed and its rolls lose their text.
func TestParseDIMFile_BlockNotesAreNotSwallowedByTheCommentRule(t *testing.T) {
	f := ParseDIMFile("//notes:carried\ndimwishlist:item=1&perks=10\n")
	if got := f.Rolls()[0].Notes; got != "carried" {
		t.Errorf("notes = %q, want \"carried\"", got)
	}
}

func TestParseDIMFile_ItemHashSemantics(t *testing.T) {
	cases := []struct {
		name   string
		line   string
		kind   DIMLineKind
		reason string
	}{
		{"no perks", "dimwishlist:item=1234", DIMLineUnsupported, ReasonNoPerks},
		{"undesirable without perks", "dimwishlist:item=-1234#notes:nope", DIMLineUnsupported, ReasonNoPerks},
		{"beyond uint32", "dimwishlist:item=4294967296&perks=1", DIMLineMalformed, ReasonItemHashRange},
		{"not dim syntax", "just some text", DIMLineMalformed, ReasonUnrecognized},
		{"legacy banshee", "https://banshee-44.com/?weapon=123&socketEntries=1,2", DIMLineUnsupported, ReasonLegacyURL},
		{"legacy dtr", "https://destinytracker.com/destiny-2/db/items/123?perks=1,2", DIMLineUnsupported, ReasonLegacyURL},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := ParseDIMFile(tc.line)
			l := f.Lines[0]
			if l.Kind != tc.kind || l.Reason != tc.reason {
				t.Errorf("kind/reason = %q/%q, want %q/%q", l.Kind, l.Reason, tc.kind, tc.reason)
			}
		})
	}
}

// DIM's own perk handling: comma-separated, and anything that is not a positive
// number is dropped. The line regex admits pipes, so a pipe-bearing entry
// reaches the splitter intact and must be discarded rather than mis-parsed.
func TestParseDIMFile_PerkListFollowsDIMsOwnRules(t *testing.T) {
	f := ParseDIMFile("dimwishlist:item=1&perks=111,0,222,333|444")
	l := f.Lines[0]
	if l.Kind != DIMLineRoll {
		t.Fatalf("kind = %q", l.Kind)
	}
	if len(l.PerkHashes) != 2 || l.PerkHashes[0] != 111 || l.PerkHashes[1] != 222 {
		t.Errorf("perks = %v, want [111 222] — zero and the pipe entry dropped", l.PerkHashes)
	}
}

// Notes stop at a pipe, and DIM's escape for a line break is expanded.
func TestParseDIMFile_NotesStopAtPipeAndExpandEscapes(t *testing.T) {
	f := ParseDIMFile(`dimwishlist:item=1&perks=111#notes:first\nsecond|trailing`)
	if got := f.Lines[0].Notes; got != "first\nsecond" {
		t.Errorf("notes = %q, want \"first\\nsecond\"", got)
	}
}

// Every line of a file yields exactly one outcome, in order — a line is never
// dropped, because a dropped line is indistinguishable from one that imported.
func TestParseDIMFile_EveryLineYieldsExactlyOneOutcome(t *testing.T) {
	text := "title:t\n" +
		"// comment\n" +
		"dimwishlist:item=1&perks=10\n" + // a wanted roll
		"dimwishlist:item=-2&perks=20\n" + // an unwanted roll
		"dimwishlist:item=3\n" + // unsupported: names no perks
		"garbage\n" +
		"\n"
	f := ParseDIMFile(text)
	if len(f.Lines) != 7 {
		t.Fatalf("outcomes = %d, want 7", len(f.Lines))
	}
	want := []DIMLineKind{
		DIMLineComment, DIMLineComment, DIMLineRoll, DIMLineRoll,
		DIMLineUnsupported, DIMLineMalformed, DIMLineComment,
	}
	for i, w := range want {
		if f.Lines[i].Kind != w {
			t.Errorf("line %d kind = %q, want %q", i+1, f.Lines[i].Kind, w)
		}
		if f.Lines[i].Number != i+1 {
			t.Errorf("line %d reported number %d", i+1, f.Lines[i].Number)
		}
	}
}

func TestParseDIMFile_HandlesCRLF(t *testing.T) {
	f := ParseDIMFile("title:t\r\ndimwishlist:item=1&perks=10\r\n")
	if f.Title != "t" {
		t.Errorf("title = %q", f.Title)
	}
	if len(f.Rolls()) != 1 || f.Rolls()[0].PerkHashes[0] != 10 {
		t.Errorf("rolls = %+v", f.Rolls())
	}
}

// DIM writes an unwanted roll as a negative item hash and the any-item wildcard
// as the literal -69420. The wildcard is negative too, so it must be recognised
// before the sign is read as "unwanted".
func TestParseDIMFile_StanceAndWildcard(t *testing.T) {
	cases := []struct {
		name      string
		line      string
		wanted    bool
		anyWeapon bool
		itemHash  uint32
	}{
		{"wanted roll", "dimwishlist:item=1234&perks=111", true, false, 1234},
		{"unwanted roll", "dimwishlist:item=-1234&perks=111", false, false, 1234},
		{"wildcard", "dimwishlist:item=-69420&perks=111", true, true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := ParseDIMFile(tc.line).Lines[0]
			if l.Kind != DIMLineRoll {
				t.Fatalf("kind = %q, want roll", l.Kind)
			}
			if l.Wanted != tc.wanted || l.AnyWeapon != tc.anyWeapon || l.ItemHash != tc.itemHash {
				t.Errorf("wanted/anyWeapon/hash = %v/%v/%d, want %v/%v/%d",
					l.Wanted, l.AnyWeapon, l.ItemHash, tc.wanted, tc.anyWeapon, tc.itemHash)
			}
		})
	}
}

// The absolute value of an unwanted roll still has to fit an item hash.
func TestParseDIMFile_UnwantedRollRespectsHashRange(t *testing.T) {
	l := ParseDIMFile("dimwishlist:item=-4294967296&perks=1").Lines[0]
	if l.Kind != DIMLineMalformed || l.Reason != ReasonItemHashRange {
		t.Errorf("kind/reason = %q/%q, want malformed/%q", l.Kind, l.Reason, ReasonItemHashRange)
	}
}
