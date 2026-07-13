package records

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	manifestrepo "guardian-tracker/api-service/services/manifest"
)

// fakeRecordsManifest satisfies records.ManifestRepo with JSON-built fixtures.
type fakeRecordsManifest struct {
	nodes         map[uint32]*manifestrepo.PresentationNodeDef
	records       map[uint32]*manifestrepo.RecordDef
	weaponTypes   map[string]string
	exoticWeapons map[string]manifestrepo.ExoticWeapon
	catalystLinks []manifestrepo.CatalystLink
}

func (f *fakeRecordsManifest) GetPresentationNodeDefinitions(hashes []uint32) (map[uint32]*manifestrepo.PresentationNodeDef, error) {
	out := map[uint32]*manifestrepo.PresentationNodeDef{}
	for _, h := range hashes {
		if n, ok := f.nodes[h]; ok {
			out[h] = n
		}
	}
	return out, nil
}

func (f *fakeRecordsManifest) GetRecordDefinitions(hashes []uint32) (map[uint32]*manifestrepo.RecordDef, error) {
	out := map[uint32]*manifestrepo.RecordDef{}
	for _, h := range hashes {
		if r, ok := f.records[h]; ok {
			out[h] = r
		}
	}
	return out, nil
}

func (f *fakeRecordsManifest) GetWeaponTypesByName() (map[string]string, error) {
	if f.weaponTypes == nil {
		return map[string]string{}, nil
	}
	return f.weaponTypes, nil
}

func (f *fakeRecordsManifest) GetExoticWeaponsByName() (map[string]manifestrepo.ExoticWeapon, error) {
	if f.exoticWeapons == nil {
		return map[string]manifestrepo.ExoticWeapon{}, nil
	}
	return f.exoticWeapons, nil
}

func (f *fakeRecordsManifest) GetCatalystLinks() ([]manifestrepo.CatalystLink, error) {
	return f.catalystLinks, nil
}

func nodeFromJSON(t *testing.T, blob string) *manifestrepo.PresentationNodeDef {
	t.Helper()
	var def manifestrepo.PresentationNodeDef
	if err := json.Unmarshal([]byte(blob), &def); err != nil {
		t.Fatalf("nodeFromJSON: %v", err)
	}
	return &def
}

func recordFromJSON(t *testing.T, blob string) *manifestrepo.RecordDef {
	t.Helper()
	var def manifestrepo.RecordDef
	if err := json.Unmarshal([]byte(blob), &def); err != nil {
		t.Fatalf("recordFromJSON: %v", err)
	}
	return &def
}

// newBungieServer fakes the two Bungie endpoints the records service calls:
// /Settings/ (core settings) and /Destiny2/.../Profile/... (records component).
func newBungieServer(t *testing.T, profileJSON string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/Settings"):
			fmt.Fprint(w, `{"ErrorCode":1,"Response":{"destiny2CoreSettings":{
				"exoticCatalystsRootNodeHash":9000,
				"craftingRootNodeHash":9100,
				"activeSealsRootNodeHash":9200,
				"legacySealsRootNodeHash":0}}}`)
		case strings.Contains(r.URL.Path, "/Profile/"):
			fmt.Fprint(w, profileJSON)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newRecordsService(t *testing.T, srv *httptest.Server, m ManifestRepo) *Service {
	t.Helper()
	client := bungie.NewClient("test-key", srv.URL, 100, 100)
	return NewService(client, m, cache.NewMemoryCache(time.Minute, time.Minute), time.Minute)
}

// catalystFixture: root node 9000 → records 1001-1004 named like real catalysts.
// The record icons are the generic catalyst glyph (record_*.png); the weapon
// picture comes from the exotic-weapon map (weapon_*.png), proving the service
// uses the latter.
func catalystFixture(t *testing.T) *fakeRecordsManifest {
	t.Helper()
	return &fakeRecordsManifest{
		nodes: map[uint32]*manifestrepo.PresentationNodeDef{
			9000: nodeFromJSON(t, `{"hash":9000,
				"displayProperties":{"name":"Exotic Catalysts"},
				"children":{"records":[
					{"recordHash":1001},{"recordHash":1002},{"recordHash":1003},{"recordHash":1004}]}}`),
		},
		records: map[uint32]*manifestrepo.RecordDef{
			1001: recordFromJSON(t, `{"hash":1001,"recordTypeName":"Exotic Catalysts","stateInfo":{"obscuredDescription":"Found in strikes and the Crucible."},"displayProperties":{"name":"Sunshot Catalyst","description":"Defeat targets with Sunshot.","icon":"/icons/record_sunshot.png"}}`),
			1002: recordFromJSON(t, `{"hash":1002,"recordTypeName":"Exotic Catalysts","displayProperties":{"name":"Riskrunner Catalyst","description":"Chain lightning to defeat targets."}}`),
			1003: recordFromJSON(t, `{"hash":1003,"recordTypeName":"Exotic Catalysts","stateInfo":{"obscuredDescription":"Found in Nightfall: The Ordeal."},"displayProperties":{"name":"Thunderlord Catalyst"}}`),
			1004: recordFromJSON(t, `{"hash":1004,"recordTypeName":"Exotic Catalysts","displayProperties":{"name":"Mystery Catalyst"}}`),
		},
		exoticWeapons: map[string]manifestrepo.ExoticWeapon{
			"sunshot":     {Type: "Hand Cannon", Icon: "/icons/weapon_sunshot.png"},
			"riskrunner":  {Type: "Submachine Gun", Icon: "/icons/weapon_riskrunner.png"},
			"thunderlord": {Type: "Machine Gun", Icon: "/icons/weapon_thunderlord.png"},
		},
	}
}

// TestGetCatalysts_StatusMapping covers the record-state → status truth table:
// no record → missing, Obscured → missing, Redeemed → complete,
// otherwise in-progress with the first incomplete objective.
func TestGetCatalysts_StatusMapping(t *testing.T) {
	// 1001: no record at all. 1002: obscured. 1003: redeemed.
	// 1004: in progress, first objective complete, second incomplete.
	profile := `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{
		"1002":{"state":8},
		"1003":{"state":1},
		"1004":{"state":4,"objectives":[
			{"progress":500,"completionValue":500,"complete":true,"progressDescription":"Done part"},
			{"progress":250,"completionValue":700,"complete":false,"progressDescription":"Targets defeated"}]}
	}}}}}`
	s := newRecordsService(t, newBungieServer(t, profile), catalystFixture(t))

	cats, fetchedAt, err := s.GetCatalysts(context.Background(), 3, "4611686018467260757", "token")
	if err != nil {
		t.Fatalf("GetCatalysts: %v", err)
	}
	if fetchedAt.IsZero() {
		t.Error("fetchedAt is zero")
	}
	if len(cats) != 4 {
		t.Fatalf("len(cats) = %d, want 4", len(cats))
	}

	byName := map[string]Catalyst{}
	for _, c := range cats {
		byName[c.Name] = c
	}

	if got := byName["Sunshot Catalyst"]; got.Status != "missing" {
		t.Errorf("no-record status = %q, want missing", got.Status)
	}
	if got := byName["Riskrunner Catalyst"]; got.Status != "missing" {
		t.Errorf("obscured status = %q, want missing", got.Status)
	}
	if got := byName["Thunderlord Catalyst"]; got.Status != "complete" {
		t.Errorf("redeemed status = %q, want complete", got.Status)
	}
	prog := byName["Mystery Catalyst"]
	if prog.Status != "in-progress" {
		t.Fatalf("in-progress status = %q", prog.Status)
	}
	if prog.Obj == nil || prog.Obj.Cur != 250 || prog.Obj.Max != 700 || prog.Obj.Label != "Targets defeated" {
		t.Errorf("in-progress obj = %+v, want first *incomplete* objective", prog.Obj)
	}
}

// TestGetCatalysts_WeaponTypeAndIcon: catalyst entries resolve their weapon type
// AND weapon picture from the exotic-weapon map (not the near-transparent record
// glyph), with a graceful empty fallback for unknown weapons.
func TestGetCatalysts_WeaponTypeAndIcon(t *testing.T) {
	profile := `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{}}}}}`
	s := newRecordsService(t, newBungieServer(t, profile), catalystFixture(t))

	cats, _, err := s.GetCatalysts(context.Background(), 3, "4611686018467260757", "token")
	if err != nil {
		t.Fatalf("GetCatalysts: %v", err)
	}
	byName := map[string]Catalyst{}
	for _, c := range cats {
		byName[c.Name] = c
	}
	if got := byName["Sunshot Catalyst"]; got.Type != "Hand Cannon" {
		t.Errorf("Sunshot type = %q, want Hand Cannon", got.Type)
	}
	// The weapon picture, not the record's own catalyst-glyph icon.
	if got := byName["Sunshot Catalyst"]; got.Icon != "/icons/weapon_sunshot.png" {
		t.Errorf("Sunshot icon = %q, want the weapon icon", got.Icon)
	}
	// "Mystery" isn't a known weapon — type and icon stay empty, never wrong.
	if got := byName["Mystery Catalyst"]; got.Type != "" || got.Icon != "" {
		t.Errorf("unknown weapon = {type %q icon %q}, want empty fallback", got.Type, got.Icon)
	}
}

func TestGetCrafting_TypeResolutionAndProgress(t *testing.T) {
	// Patterns live under the combined "Patterns & Catalysts" node (9000), not
	// craftingRootNodeHash — see GetCrafting / combinedFixture.
	m := &fakeRecordsManifest{
		nodes: map[uint32]*manifestrepo.PresentationNodeDef{
			9000: nodeFromJSON(t, `{"hash":9000,
				"displayProperties":{"name":"Patterns & Catalysts"},
				"children":{"records":[{"recordHash":2001},{"recordHash":2002}]}}`),
		},
		records: map[uint32]*manifestrepo.RecordDef{
			2001: recordFromJSON(t, `{"hash":2001,"recordTypeName":"Weapon Pattern","displayProperties":{"name":"Bold Endings","icon":"/icons/bold_endings.jpg"}}`),
			2002: recordFromJSON(t, `{"hash":2002,"recordTypeName":"Weapon Pattern","displayProperties":{"name":"Unknownium"}}`),
		},
		weaponTypes: map[string]string{"bold endings": "Hand Cannon"},
	}
	profile := `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{
		"2001":{"state":4,"objectives":[{"progress":3,"completionValue":5,"complete":false}]},
		"2002":{"state":1,"objectives":[{"progress":5,"completionValue":5,"complete":true}]}
	}}}}}`
	s := newRecordsService(t, newBungieServer(t, profile), m)

	patterns, _, err := s.GetCrafting(context.Background(), 3, "4611686018467260757", "token")
	if err != nil {
		t.Fatalf("GetCrafting: %v", err)
	}
	if len(patterns) != 2 {
		t.Fatalf("len(patterns) = %d, want 2", len(patterns))
	}
	byName := map[string]CraftPattern{}
	for _, p := range patterns {
		byName[p.Name] = p
	}
	bold := byName["Bold Endings"]
	if bold.Type != "Hand Cannon" {
		t.Errorf("Bold Endings type = %q, want Hand Cannon (B10)", bold.Type)
	}
	// The pattern record's icon is the weapon picture and is passed through.
	if bold.Icon != "/icons/bold_endings.jpg" {
		t.Errorf("Bold Endings icon = %q, want the weapon icon", bold.Icon)
	}
	if bold.Patterns.Cur != 3 || bold.Patterns.Max != 5 {
		t.Errorf("Bold Endings progress = %+v", bold.Patterns)
	}
	unknown := byName["Unknownium"]
	if unknown.Type != "Weapon" {
		t.Errorf("unmatched name type = %q, want generic Weapon fallback", unknown.Type)
	}
	if unknown.Note != "Pattern unlocked" {
		t.Errorf("redeemed pattern note = %q", unknown.Note)
	}
}

// combinedFixture mirrors the real manifest: exoticCatalystsRootNodeHash (9000)
// is a combined "Patterns & Catalysts" node whose subtree holds BOTH exotic
// catalyst records and weapon-pattern records. The two are told apart only by
// recordTypeName. (craftingRootNodeHash points at a record-less "Shape" nav node.)
func combinedFixture(t *testing.T) *fakeRecordsManifest {
	t.Helper()
	return &fakeRecordsManifest{
		nodes: map[uint32]*manifestrepo.PresentationNodeDef{
			9000: nodeFromJSON(t, `{"hash":9000,"displayProperties":{"name":"Patterns & Catalysts"},
				"children":{"presentationNodes":[{"presentationNodeHash":9001},{"presentationNodeHash":9002}]}}`),
			9001: nodeFromJSON(t, `{"hash":9001,"displayProperties":{"name":"Exotic Catalysts"},
				"children":{"records":[{"recordHash":1001},{"recordHash":1002}]}}`),
			9002: nodeFromJSON(t, `{"hash":9002,"displayProperties":{"name":"Weapon Patterns"},
				"children":{"records":[{"recordHash":2001},{"recordHash":2002}]}}`),
		},
		records: map[uint32]*manifestrepo.RecordDef{
			1001: recordFromJSON(t, `{"hash":1001,"recordTypeName":"Exotic Catalysts","displayProperties":{"name":"Sunshot Catalyst"}}`),
			1002: recordFromJSON(t, `{"hash":1002,"recordTypeName":"Exotic Catalysts","displayProperties":{"name":"Riskrunner Catalyst"}}`),
			2001: recordFromJSON(t, `{"hash":2001,"recordTypeName":"Weapon Pattern","displayProperties":{"name":"Bold Endings"}}`),
			2002: recordFromJSON(t, `{"hash":2002,"recordTypeName":"Weapon Pattern","displayProperties":{"name":"Aberrant Action"}}`),
		},
	}
}

// TestCatalystsAndCraftingPartition is the regression for the bug where the
// Catalysts tab showed legendary crafting patterns and the Crafting tab was
// empty: both endpoints draw from the same "Patterns & Catalysts" node and must
// partition records by recordTypeName.
func TestCatalystsAndCraftingPartition(t *testing.T) {
	profile := `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{}}}}}`
	srv := newBungieServer(t, profile)
	s := newRecordsService(t, srv, combinedFixture(t))

	cats, _, err := s.GetCatalysts(context.Background(), 3, "4611686018467260757", "token")
	if err != nil {
		t.Fatalf("GetCatalysts: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("GetCatalysts returned %d items, want 2 (catalysts only): %+v", len(cats), cats)
	}
	for _, c := range cats {
		if !strings.HasSuffix(c.Name, "Catalyst") {
			t.Errorf("GetCatalysts leaked a non-catalyst record: %q", c.Name)
		}
	}

	patterns, _, err := s.GetCrafting(context.Background(), 3, "4611686018467260757", "token")
	if err != nil {
		t.Fatalf("GetCrafting: %v", err)
	}
	if len(patterns) != 2 {
		t.Fatalf("GetCrafting returned %d items, want 2 (patterns only): %+v", len(patterns), patterns)
	}
	for _, p := range patterns {
		if strings.HasSuffix(p.Name, "Catalyst") {
			t.Errorf("GetCrafting leaked a catalyst record: %q", p.Name)
		}
	}
}

// TestResolveCatalystWeapon covers the three resolution strategies against the
// real-world name shapes: exact stripped name, a unique substring match for
// abbreviated names ("Khvostov" → "Khvostov 7G-0X"), and a description scan for
// names with no relation to the weapon ("Immovable Refit" → Vexcalibur).
func TestResolveCatalystWeapon(t *testing.T) {
	exotics := map[string]manifestrepo.ExoticWeapon{
		"sunshot":             {Type: "Hand Cannon", Icon: "/w/sunshot.jpg"},
		"khvostov 7g-0x":      {Type: "Auto Rifle", Icon: "/w/khvostov.jpg"},
		"whisper of the worm": {Type: "Sniper Rifle", Icon: "/w/whisper.jpg"},
		"vexcalibur":          {Type: "Glaive", Icon: "/w/vexcalibur.jpg"},
	}
	namesByLen := make([]string, 0, len(exotics))
	for n := range exotics {
		namesByLen = append(namesByLen, n)
	}
	sort.Slice(namesByLen, func(i, j int) bool { return len(namesByLen[i]) > len(namesByLen[j]) })

	cases := []struct {
		name, desc, wantIcon string
	}{
		{"Sunshot Catalyst", "Defeat enemies with Sunshot.", "/w/sunshot.jpg"},                        // exact
		{"Khvostov Catalyst", "Defeat enemies with Khvostov.", "/w/khvostov.jpg"},                     // substring
		{"Whisper Catalyst", "Defeat enemies with Whisper of the Worm.", "/w/whisper.jpg"},            // substring
		{"Immovable Refit", "Apply any catalyst to Vexcalibur through shaping.", "/w/vexcalibur.jpg"}, // description
		{"Nonexistent Catalyst", "Defeat enemies with some weapon.", ""},                              // miss
	}
	for _, c := range cases {
		got := resolveCatalystWeapon(c.name, c.desc, exotics, namesByLen)
		if got.Icon != c.wantIcon {
			t.Errorf("resolve(%q) icon = %q, want %q", c.name, got.Icon, c.wantIcon)
		}
	}
}

// TestGetCatalysts_SourceByStatus verifies the hybrid source line: a missing or
// complete catalyst shows its acquisition source (the record's obscured
// description), an in-progress one shows the completion requirement (the record
// description), and a record with neither falls back to its category.
func TestGetCatalysts_SourceByStatus(t *testing.T) {
	// 1001 Sunshot: missing → obscured (acquisition) source. 1002 Riskrunner:
	// in-progress → completion requirement. 1003 Thunderlord: complete → obscured
	// source. 1004 Mystery: missing, no obscured text → record category fallback.
	profile := `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{
		"1002":{"state":4,"objectives":[{"progress":1,"completionValue":10,"complete":false}]},
		"1003":{"state":1}
	}}}}}`
	s := newRecordsService(t, newBungieServer(t, profile), catalystFixture(t))

	cats, _, err := s.GetCatalysts(context.Background(), 3, "4611686018467260757", "token")
	if err != nil {
		t.Fatalf("GetCatalysts: %v", err)
	}
	byName := map[string]Catalyst{}
	for _, c := range cats {
		byName[c.Name] = c
	}
	if got := byName["Sunshot Catalyst"]; got.Status != "missing" || got.Source != "Found in strikes and the Crucible." {
		t.Errorf("missing source = {%q %q}, want {missing, obscured description}", got.Status, got.Source)
	}
	if got := byName["Riskrunner Catalyst"]; got.Status != "in-progress" || got.Source != "Chain lightning to defeat targets." {
		t.Errorf("in-progress source = {%q %q}, want the completion requirement", got.Status, got.Source)
	}
	if got := byName["Thunderlord Catalyst"]; got.Status != "complete" || got.Source != "Found in Nightfall: The Ordeal." {
		t.Errorf("complete source = {%q %q}, want {complete, obscured description}", got.Status, got.Source)
	}
	// No obscured text → record category ("Exotic Catalysts").
	if got := byName["Mystery Catalyst"]; got.Source != "Exotic Catalysts" {
		t.Errorf("fallback source = %q, want record category", got.Source)
	}
}

func TestRecords_NilManifestReturnsNotReady(t *testing.T) {
	s := NewService(nil, nil, cache.NewMemoryCache(time.Minute, time.Minute), time.Minute)

	// Without a manifest the endpoints must report manifest-not-ready (→ 503), not
	// a 200 with an empty list — otherwise the UI can't tell "still downloading"
	// apart from "you have no catalysts" (B-review #4).
	if cats, _, err := s.GetCatalysts(context.Background(), 3, "x", "t"); !errors.Is(err, manifestrepo.ErrNotReady) || len(cats) != 0 {
		t.Errorf("GetCatalysts = %v items, err %v; want empty, ErrNotReady", len(cats), err)
	}
	if patterns, _, err := s.GetCrafting(context.Background(), 3, "x", "t"); !errors.Is(err, manifestrepo.ErrNotReady) || len(patterns) != 0 {
		t.Errorf("GetCrafting = %v items, err %v; want empty, ErrNotReady", len(patterns), err)
	}
	if seals, _, err := s.GetSeals(context.Background(), 3, "x", "t"); !errors.Is(err, manifestrepo.ErrNotReady) || len(seals) != 0 {
		t.Errorf("GetSeals = %v items, err %v; want empty, ErrNotReady", len(seals), err)
	}
}

func TestGetSeals_CompletionAndGilding(t *testing.T) {
	m := &fakeRecordsManifest{
		nodes: map[uint32]*manifestrepo.PresentationNodeDef{
			9200: nodeFromJSON(t, `{"hash":9200,
				"displayProperties":{"name":"Seals"},
				"children":{"presentationNodes":[{"presentationNodeHash":9300}]}}`),
			9300: nodeFromJSON(t, `{"hash":9300,
				"displayProperties":{"name":"Dredgen"},
				"completionRecordHash":4000,
				"children":{"records":[{"recordHash":3001},{"recordHash":3002}]}}`),
		},
		records: map[uint32]*manifestrepo.RecordDef{
			3001: recordFromJSON(t, `{"hash":3001,"displayProperties":{"name":"Triumph One"}}`),
			3002: recordFromJSON(t, `{"hash":3002,"displayProperties":{"name":"Triumph Two"}}`),
		},
	}
	profile := `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{
		"3001":{"state":1,"objectives":[{"progress":10,"completionValue":10,"complete":true}]},
		"3002":{"state":4,"objectives":[{"progress":2,"completionValue":10,"complete":false}]},
		"4000":{"state":1,"intervalObjectives":[
			{"progress":1,"completionValue":1,"complete":true},
			{"progress":0,"completionValue":1,"complete":false}]}
	}}}}}`
	s := newRecordsService(t, newBungieServer(t, profile), m)

	seals, _, err := s.GetSeals(context.Background(), 3, "4611686018467260757", "token")
	if err != nil {
		t.Fatalf("GetSeals: %v", err)
	}
	if len(seals) != 1 {
		t.Fatalf("len(seals) = %d, want 1", len(seals))
	}
	seal := seals[0]
	if seal.Name != "Dredgen" {
		t.Errorf("seal name = %q", seal.Name)
	}
	if seal.Pct != 50 {
		t.Errorf("pct = %d, want 50 (1 of 2 triumphs)", seal.Pct)
	}
	if seal.Gilded != 1 {
		t.Errorf("gilded = %d, want 1 (one complete interval objective)", seal.Gilded)
	}
	if seal.Left != "1 triumph left" {
		t.Errorf("left = %q", seal.Left)
	}
}

// TestGetSeals_ObjectivesBreakdown covers the per-objective drill-down data:
// single vs. multiple objectives, explicit-hidden exclusion, absent-visible
// inclusion (the *bool regression), blank-label numbering over included
// objectives only, redeemed-record authority over stale objective payloads,
// omission of the objectives key entirely when none survive, and zero
// completionValue normalization.
func TestGetSeals_ObjectivesBreakdown(t *testing.T) {
	m := &fakeRecordsManifest{
		nodes: map[uint32]*manifestrepo.PresentationNodeDef{
			9200: nodeFromJSON(t, `{"hash":9200,
				"displayProperties":{"name":"Seals"},
				"children":{"presentationNodes":[{"presentationNodeHash":9300}]}}`),
			9300: nodeFromJSON(t, `{"hash":9300,
				"displayProperties":{"name":"Dredgen"},
				"children":{"records":[
					{"recordHash":3001},{"recordHash":3002},{"recordHash":3003},
					{"recordHash":3004},{"recordHash":3005}]}}`),
		},
		records: map[uint32]*manifestrepo.RecordDef{
			3001: recordFromJSON(t, `{"hash":3001,"displayProperties":{"name":"Single Objective"}}`),
			3002: recordFromJSON(t, `{"hash":3002,"displayProperties":{"name":"Mixed Visibility"}}`),
			3003: recordFromJSON(t, `{"hash":3003,"displayProperties":{"name":"Stale Complete"}}`),
			3004: recordFromJSON(t, `{"hash":3004,"displayProperties":{"name":"No Objectives"}}`),
			3005: recordFromJSON(t, `{"hash":3005,"displayProperties":{"name":"Zero Max"}}`),
		},
	}
	profile := `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{
		"3001":{"state":4,"objectives":[
			{"progress":3,"completionValue":5,"complete":false,"visible":true,"progressDescription":"Kills"}
		]},
		"3002":{"state":4,"objectives":[
			{"progress":1,"completionValue":2,"complete":false,"progressDescription":""},
			{"progress":2,"completionValue":2,"complete":true,"visible":false,"progressDescription":"Hidden"},
			{"progress":7,"completionValue":5,"complete":false,"visible":true,"progressDescription":"   "}
		]},
		"3003":{"state":1,"objectives":[
			{"progress":0,"completionValue":1,"complete":false,"visible":true,"progressDescription":"Stale"}
		]},
		"3004":{"state":4},
		"3005":{"state":4,"objectives":[
			{"progress":0,"completionValue":0,"complete":false,"visible":true,"progressDescription":"Zero Progress"}
		]}
	}}}}}`
	s := newRecordsService(t, newBungieServer(t, profile), m)

	seals, _, err := s.GetSeals(context.Background(), 3, "4611686018467260757", "token")
	if err != nil {
		t.Fatalf("GetSeals: %v", err)
	}
	if len(seals) != 1 {
		t.Fatalf("len(seals) = %d, want 1", len(seals))
	}
	byLabel := map[string]Triumph{}
	for _, tr := range seals[0].Triumphs {
		byLabel[tr.Label] = tr
	}

	// Single objective.
	single := byLabel["Single Objective"]
	if len(single.Objectives) != 1 {
		t.Fatalf("single objective count = %d, want 1", len(single.Objectives))
	}
	if got := single.Objectives[0]; got.Label != "Kills" || got.Done || got.Cur != 3 || got.Max != 5 {
		t.Errorf("single objective = %+v, want {Kills false 3 5}", got)
	}

	// Multiple objectives: the explicitly-hidden one is excluded, the
	// visible-absent one is included, blank labels are numbered over the two
	// surviving (included) objectives only, and a stale overshoot on an
	// incomplete objective is clamped to Max.
	mixed := byLabel["Mixed Visibility"]
	if len(mixed.Objectives) != 2 {
		t.Fatalf("mixed objective count = %d, want 2 (explicitly-hidden one excluded)", len(mixed.Objectives))
	}
	if got := mixed.Objectives[0]; got.Label != "Objective 1" || got.Done || got.Cur != 1 || got.Max != 2 {
		t.Errorf("mixed objective 0 = %+v, want {Objective 1 false 1 2}", got)
	}
	if got := mixed.Objectives[1]; got.Label != "Objective 2" || got.Done || got.Cur != 5 || got.Max != 5 {
		t.Errorf("mixed objective 1 = %+v, want {Objective 2 false 5 5} (Cur clamped to Max)", got)
	}

	// Redeemed record: stale incomplete objective payload must still report
	// Done=true and Cur == Max.
	stale := byLabel["Stale Complete"]
	if len(stale.Objectives) != 1 {
		t.Fatalf("stale objective count = %d, want 1", len(stale.Objectives))
	}
	if got := stale.Objectives[0]; !got.Done || got.Cur != got.Max {
		t.Errorf("stale objective = %+v, want Done=true and Cur == Max", got)
	}

	// No objectives at all: the field must be entirely absent from the JSON
	// (nil slice + omitempty), not an empty array.
	none := byLabel["No Objectives"]
	if none.Objectives != nil {
		t.Errorf("no-objectives Objectives = %+v, want nil", none.Objectives)
	}
	blob, err := json.Marshal(none)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(blob, &fields); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if _, ok := fields["objectives"]; ok {
		t.Errorf("json = %s, must omit the \"objectives\" key entirely", blob)
	}

	// Zero completionValue normalizes to Max = 1.
	zero := byLabel["Zero Max"]
	if len(zero.Objectives) != 1 {
		t.Fatalf("zero-max objective count = %d, want 1", len(zero.Objectives))
	}
	if got := zero.Objectives[0]; got.Max != 1 {
		t.Errorf("zero-max objective Max = %d, want 1", got.Max)
	}
}

// TestGetCatalysts_EffectLinkage covers the hash-first linkage chain: objective
// overlap first, then stripped-name match, then catalyst-plug-name match, then
// falling back to the record's own description when nothing links. It also
// covers the record-side ambiguity guard — an objective hash claimed by more
// than one record must never be used to link, even if it's unambiguous on the
// weapon side.
func TestGetCatalysts_EffectLinkage(t *testing.T) {
	m := &fakeRecordsManifest{
		nodes: map[uint32]*manifestrepo.PresentationNodeDef{
			9000: nodeFromJSON(t, `{"hash":9000,
				"displayProperties":{"name":"Exotic Catalysts"},
				"children":{"records":[
					{"recordHash":1001},{"recordHash":1002},{"recordHash":1003},
					{"recordHash":1004},{"recordHash":1005},{"recordHash":1006}]}}`),
		},
		records: map[uint32]*manifestrepo.RecordDef{
			// Objective-hash overlap with the "Sunshot" link (3107076149).
			1001: recordFromJSON(t, `{"hash":1001,"recordTypeName":"Exotic Catalysts","objectiveHashes":[999,3107076149],"displayProperties":{"name":"Sunshot Catalyst","description":"Record fallback text one."}}`),
			// No objective overlap; stripped name "Riskrunner" matches the
			// "Riskrunner" link by weapon name.
			1002: recordFromJSON(t, `{"hash":1002,"recordTypeName":"Exotic Catalysts","displayProperties":{"name":"Riskrunner Catalyst","description":"Record fallback text two."}}`),
			// Name doesn't strip-match any weapon ("Thunderlord" != "Thunderlord
			// Prime"), but matches a catalyst-plug name on that link.
			1003: recordFromJSON(t, `{"hash":1003,"recordTypeName":"Exotic Catalysts","displayProperties":{"name":"Thunderlord Catalyst","description":"Record fallback text three."}}`),
			// No link matches at all — falls back to the record's own description.
			1004: recordFromJSON(t, `{"hash":1004,"recordTypeName":"Exotic Catalysts","displayProperties":{"name":"Mystery Catalyst","description":"Record fallback text four."}}`),
			// Objective hash 777 is claimed by BOTH 1005 and 1006 — ambiguous on
			// the record side even though only one weapon link (unrelated,
			// non-name-matching "Whatever") declares it, so it must not be used.
			1005: recordFromJSON(t, `{"hash":1005,"recordTypeName":"Exotic Catalysts","objectiveHashes":[777],"displayProperties":{"name":"Ambiguous Catalyst","description":"Record fallback text five."}}`),
			1006: recordFromJSON(t, `{"hash":1006,"recordTypeName":"Exotic Catalysts","objectiveHashes":[777],"displayProperties":{"name":"Also Ambiguous Catalyst","description":"Record fallback text six."}}`),
		},
		catalystLinks: []manifestrepo.CatalystLink{
			{WeaponName: "Sunshot", ObjectiveHashes: []uint32{3107076149},
				Catalysts: []manifestrepo.WeaponCatalyst{{Name: "Sunshot Catalyst", Description: "Loose Change text."}}},
			{WeaponName: "Riskrunner",
				Catalysts: []manifestrepo.WeaponCatalyst{{Name: "Arc Conductor", Description: "Chain lightning text."}}},
			{WeaponName: "Thunderlord Prime",
				Catalysts: []manifestrepo.WeaponCatalyst{{Name: "Thunderlord Catalyst", Description: "Machine gun text."}}},
			{WeaponName: "Whatever", ObjectiveHashes: []uint32{777}},
		},
	}
	profile := `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{}}}}}`
	s := newRecordsService(t, newBungieServer(t, profile), m)

	cats, _, err := s.GetCatalysts(context.Background(), 3, "4611686018467260757", "token")
	if err != nil {
		t.Fatalf("GetCatalysts: %v", err)
	}
	byName := map[string]Catalyst{}
	for _, c := range cats {
		byName[c.Name] = c
	}

	if got := byName["Sunshot Catalyst"].Effect; got != "Loose Change text." {
		t.Errorf("objective-overlap effect = %q, want %q", got, "Loose Change text.")
	}
	if got := byName["Riskrunner Catalyst"].Effect; got != "Chain lightning text." {
		t.Errorf("name-strip effect = %q, want %q", got, "Chain lightning text.")
	}
	if got := byName["Thunderlord Catalyst"].Effect; got != "Machine gun text." {
		t.Errorf("plug-name effect = %q, want %q", got, "Machine gun text.")
	}
	if got := byName["Mystery Catalyst"].Effect; got != "Record fallback text four." {
		t.Errorf("no-link effect = %q, want the record's own description", got)
	}
	if got := byName["Ambiguous Catalyst"].Effect; got != "Record fallback text five." {
		t.Errorf("record-ambiguous effect = %q, want the record's own description (hash must not link)", got)
	}
	if got := byName["Also Ambiguous Catalyst"].Effect; got != "Record fallback text six." {
		t.Errorf("record-ambiguous effect = %q, want the record's own description (hash must not link)", got)
	}
}
