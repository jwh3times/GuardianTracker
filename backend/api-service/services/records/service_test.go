package records

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	manifestrepo "guardian-tracker/api-service/services/manifest"
)

// fakeRecordsManifest satisfies records.ManifestRepo with JSON-built fixtures.
type fakeRecordsManifest struct {
	nodes       map[uint32]*manifestrepo.PresentationNodeDef
	records     map[uint32]*manifestrepo.RecordDef
	weaponTypes map[string]string
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
			1001: recordFromJSON(t, `{"hash":1001,"displayProperties":{"name":"Sunshot Catalyst","icon":"/icons/sunshot.png"}}`),
			1002: recordFromJSON(t, `{"hash":1002,"displayProperties":{"name":"Riskrunner Catalyst"}}`),
			1003: recordFromJSON(t, `{"hash":1003,"displayProperties":{"name":"Thunderlord Catalyst"}}`),
			1004: recordFromJSON(t, `{"hash":1004,"displayProperties":{"name":"Mystery Catalyst"}}`),
		},
		weaponTypes: map[string]string{
			"sunshot":     "Hand Cannon",
			"riskrunner":  "Submachine Gun",
			"thunderlord": "Machine Gun",
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

// TestGetCatalysts_WeaponTypeAndIcon is the B10 regression: catalyst entries
// resolve their weapon type from the manifest (with a graceful fallback) and
// carry the record icon.
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
	if got := byName["Sunshot Catalyst"]; got.Icon != "/icons/sunshot.png" {
		t.Errorf("Sunshot icon = %q", got.Icon)
	}
	// "Mystery" isn't a known weapon — type stays empty, never wrong.
	if got := byName["Mystery Catalyst"]; got.Type != "" {
		t.Errorf("unknown weapon type = %q, want empty fallback", got.Type)
	}
}

func TestGetCrafting_TypeResolutionAndProgress(t *testing.T) {
	m := &fakeRecordsManifest{
		nodes: map[uint32]*manifestrepo.PresentationNodeDef{
			9100: nodeFromJSON(t, `{"hash":9100,
				"displayProperties":{"name":"Crafting Patterns"},
				"children":{"records":[{"recordHash":2001},{"recordHash":2002}]}}`),
		},
		records: map[uint32]*manifestrepo.RecordDef{
			2001: recordFromJSON(t, `{"hash":2001,"displayProperties":{"name":"Bold Endings"}}`),
			2002: recordFromJSON(t, `{"hash":2002,"displayProperties":{"name":"Unknownium"}}`),
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
