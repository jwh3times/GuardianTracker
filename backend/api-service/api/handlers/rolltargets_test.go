package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
	"guardian-tracker/api-service/services/ownedrolls"
	"guardian-tracker/api-service/services/rolltargets"

	"github.com/gin-gonic/gin"
)

// --- stub roll target service ---
//
// The handler owns no roll-target rules, so these tests drive the seam: what
// reaches the service, and what each typed outcome renders as. The rules
// themselves are tested in services/rolltargets.

type stubRollTargets struct {
	targets []rolltargets.StoredTarget
	target  rolltargets.StoredTarget
	report  rolltargets.ImportReport
	err     error

	matches           rolltargets.MatchReport
	gotMembership     string
	gotMembershipType int
	gotAdd            rolltargets.AddCommand
	gotUpdateID       rolltargets.TargetID
	gotUpdate         rolltargets.UpdateCommand
	gotRemoveID       rolltargets.TargetID
	gotImportText     string

	bulkResult      rolltargets.BulkResult
	deleteAllResult int
	gotBulkIDs      []rolltargets.TargetID
	deleteAllCalled bool
}

func (s *stubRollTargets) List(_ context.Context, m string) ([]rolltargets.StoredTarget, error) {
	s.gotMembership = m
	return s.targets, s.err
}

func (s *stubRollTargets) Add(_ context.Context, m string, cmd rolltargets.AddCommand) (rolltargets.StoredTarget, error) {
	s.gotMembership, s.gotAdd = m, cmd
	return s.target, s.err
}

func (s *stubRollTargets) Update(_ context.Context, m string, id rolltargets.TargetID, patch rolltargets.UpdateCommand) (rolltargets.StoredTarget, error) {
	s.gotMembership, s.gotUpdateID, s.gotUpdate = m, id, patch
	return s.target, s.err
}

func (s *stubRollTargets) Remove(_ context.Context, m string, id rolltargets.TargetID) error {
	s.gotMembership, s.gotRemoveID = m, id
	return s.err
}

func (s *stubRollTargets) DeleteMany(_ context.Context, m string, ids []rolltargets.TargetID) (rolltargets.BulkResult, error) {
	s.gotMembership, s.gotBulkIDs = m, ids
	return s.bulkResult, s.err
}

func (s *stubRollTargets) DeleteAll(_ context.Context, m string) (int, error) {
	s.gotMembership, s.deleteAllCalled = m, true
	return s.deleteAllResult, s.err
}

func (s *stubRollTargets) ImportDIM(_ context.Context, m, text string) (rolltargets.ImportReport, error) {
	s.gotMembership, s.gotImportText = m, text
	return s.report, s.err
}

func (s *stubRollTargets) Matches(_ context.Context, mt int, m string) (rolltargets.MatchReport, error) {
	s.gotMembership, s.gotMembershipType = m, mt
	return s.matches, s.err
}

func newRollTargetRouter(h *RollTargetsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("membership_id", "test-member-123")
		c.Set("membership_type", 3)
		c.Next()
	})
	r.GET("/api/rolltargets", h.GetRollTargets)
	r.POST("/api/rolltargets", h.AddRollTarget)
	r.PATCH("/api/rolltargets/:id", h.UpdateRollTarget)
	r.DELETE("/api/rolltargets/:id", h.RemoveRollTarget)
	r.POST("/api/rolltargets/import", h.ImportRollTargets)
	r.POST("/api/rolltargets/bulk", h.BulkDeleteRollTargets)
	r.GET("/api/rolltargets/matches", h.GetRollTargetMatches)
	return r
}

func rollTargetHash(h uint32) *uint32 { return &h }

func TestGetRollTargets_SerializesWeaponAndAnyWeaponTargets(t *testing.T) {
	stub := &stubRollTargets{targets: []rolltargets.StoredTarget{
		{ID: 1, ItemHash: rollTargetHash(1234), Wanted: true, Perks: []string{"Outlaw"}, Notes: "pvp", CreatedAt: time.Unix(0, 0)},
		{ID: 2, ItemHash: nil, Wanted: false, Perks: []string{"Firefly"}, CreatedAt: time.Unix(0, 0)},
	}}
	w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodGet, "/api/rolltargets", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp []rollTargetResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("targets = %d, want 2", len(resp))
	}
	if resp[0].ID != "1" || resp[0].ItemHash == nil || *resp[0].ItemHash != 1234 || resp[0].AnyWeapon || !resp[0].Wanted {
		t.Errorf("weapon target = %+v", resp[0])
	}
	// An any-weapon target must serialise as an explicit null, never as item 0.
	if resp[1].ItemHash != nil || !resp[1].AnyWeapon || resp[1].Wanted {
		t.Errorf("any-weapon target = %+v", resp[1])
	}
	if !strings.Contains(w.Body.String(), `"itemHash":null`) {
		t.Errorf("any-weapon target did not serialise itemHash as null: %s", w.Body.String())
	}
}

func TestGetRollTargets_EmptyListSerializesAsAnArray(t *testing.T) {
	w := send(newRollTargetRouter(NewRollTargetsHandler(&stubRollTargets{})), http.MethodGet, "/api/rolltargets", "")
	if body := strings.TrimSpace(w.Body.String()); body != "[]" {
		t.Errorf("body = %s, want []", body)
	}
}

// Roll targets are addressed by the JWT alone — never by anything the client
// supplies, in the body, the query string or the path.
func TestRollTargets_UseTheJWTMembership(t *testing.T) {
	const fromJWT = "test-member-123"

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"plain read", http.MethodGet, "/api/rolltargets", ""},
		{"membership in the body", http.MethodPost, "/api/rolltargets",
			`{"itemHash":1,"perks":["Outlaw"],"membershipId":"someone-else"}`},
		{"membership in the query string", http.MethodGet,
			"/api/rolltargets?membershipId=someone-else&membership_id=someone-else", ""},
		{"membership in the query string on a write", http.MethodPost,
			"/api/rolltargets?membershipId=someone-else", `{"itemHash":1,"perks":["Outlaw"]}`},
		{"membership on the import route", http.MethodPost,
			"/api/rolltargets/import?membershipId=someone-else", "dimwishlist:item=1&perks=2"},
		{"membership in the bulk delete body", http.MethodPost, "/api/rolltargets/bulk",
			`{"action":"delete","ids":["1"],"membershipId":"someone-else"}`},
		{"membership on the bulk route's query string", http.MethodPost,
			"/api/rolltargets/bulk?membershipId=someone-else", `{"action":"delete_all"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubRollTargets{}
			send(newRollTargetRouter(NewRollTargetsHandler(stub)), tc.method, tc.path, tc.body)
			if stub.gotMembership != fromJWT {
				t.Errorf("membership = %q, want %q — client input reached the identity",
					stub.gotMembership, fromJWT)
			}
		})
	}
}

// Omitting `wanted` must take the default of true. Read as false it would
// silently invert what the caller asked for.
func TestAddRollTarget_DefaultsToWanted(t *testing.T) {
	stub := &stubRollTargets{}
	w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodPost, "/api/rolltargets",
		`{"itemHash":1234,"perks":["Outlaw"]}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}
	if !stub.gotAdd.Wanted {
		t.Error("omitted wanted was read as false")
	}
	if stub.gotAdd.ItemHash == nil || *stub.gotAdd.ItemHash != 1234 {
		t.Errorf("item hash = %v", stub.gotAdd.ItemHash)
	}

	stub.gotAdd = rolltargets.AddCommand{}
	send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodPost, "/api/rolltargets",
		`{"itemHash":1234,"wanted":false,"perks":["Outlaw"]}`)
	if stub.gotAdd.Wanted {
		t.Error("explicit wanted:false was not carried")
	}
}

// Omitting itemHash is meaningful: it saves an any-weapon target.
func TestAddRollTarget_OmittedItemHashIsAnAnyWeaponTarget(t *testing.T) {
	stub := &stubRollTargets{}
	send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodPost, "/api/rolltargets",
		`{"perks":["Outlaw"]}`)
	if stub.gotAdd.ItemHash != nil {
		t.Errorf("item hash = %v, want nil", stub.gotAdd.ItemHash)
	}
}

// A nil patch field means "leave it alone" and must not reach the service as an
// empty value.
func TestUpdateRollTarget_OmittedFieldsStayNil(t *testing.T) {
	stub := &stubRollTargets{}
	w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodPatch, "/api/rolltargets/7",
		`{"notes":"changed"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if stub.gotUpdateID != 7 {
		t.Errorf("id = %d, want 7", stub.gotUpdateID)
	}
	if stub.gotUpdate.Perks != nil {
		t.Error("an omitted perks field reached the service as a value")
	}
	if stub.gotUpdate.Notes == nil || *stub.gotUpdate.Notes != "changed" {
		t.Errorf("notes = %v", stub.gotUpdate.Notes)
	}
}

func TestRemoveRollTarget_NoContentOnSuccess(t *testing.T) {
	stub := &stubRollTargets{}
	w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodDelete, "/api/rolltargets/9", "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if stub.gotRemoveID != 9 {
		t.Errorf("id = %d, want 9", stub.gotRemoveID)
	}
}

// The "delete" action carries its ids through to the service as TargetIDs and
// reports back whatever the service says it deleted and skipped.
func TestBulkDeleteRollTargets_DeleteCarriesIDsAndReportsCounts(t *testing.T) {
	stub := &stubRollTargets{bulkResult: rolltargets.BulkResult{Deleted: 2, Skipped: 1}}
	w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodPost, "/api/rolltargets/bulk",
		`{"action":"delete","ids":["1","2","3"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if len(stub.gotBulkIDs) != 3 || stub.gotBulkIDs[0] != 1 || stub.gotBulkIDs[2] != 3 {
		t.Errorf("service saw ids %v, want [1 2 3]", stub.gotBulkIDs)
	}
	var resp map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["deleted"] != 2 || resp["skipped"] != 1 {
		t.Errorf("body = %v, want {deleted:2 skipped:1}", resp)
	}
}

// "delete_all" carries no ids and always reports skipped:0 — it names
// nothing that could be missing or foreign.
func TestBulkDeleteRollTargets_DeleteAllReportsZeroSkipped(t *testing.T) {
	stub := &stubRollTargets{deleteAllResult: 7}
	w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodPost, "/api/rolltargets/bulk",
		`{"action":"delete_all"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if !stub.deleteAllCalled {
		t.Fatal("DeleteAll was not called")
	}
	var resp map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["deleted"] != 7 || resp["skipped"] != 0 {
		t.Errorf("body = %v, want {deleted:7 skipped:0}", resp)
	}
}

func TestBulkDeleteRollTargets_RejectsAnUnknownAction(t *testing.T) {
	r := newRollTargetRouter(NewRollTargetsHandler(&stubRollTargets{}))
	if w := send(r, http.MethodPost, "/api/rolltargets/bulk", `{"action":"nuke","ids":["1"]}`); w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestBulkDeleteRollTargets_RejectsAnIDThatIsNotOne(t *testing.T) {
	r := newRollTargetRouter(NewRollTargetsHandler(&stubRollTargets{}))
	if w := send(r, http.MethodPost, "/api/rolltargets/bulk", `{"action":"delete","ids":["abc"]}`); w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// The domain's own validation refusals (empty/oversized id lists) surface as
// 400, matching every other validation error this handler maps.
func TestBulkDeleteRollTargets_ValidationErrorsAre400(t *testing.T) {
	for _, err := range []error{rolltargets.ErrNoTargets, rolltargets.ErrTooManyTargets} {
		stub := &stubRollTargets{err: err}
		w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodPost, "/api/rolltargets/bulk",
			`{"action":"delete","ids":["1"]}`)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%v status = %d, want 400: %s", err, w.Code, w.Body.String())
		}
	}
}

func TestRollTargets_RejectsAnIDThatIsNotOne(t *testing.T) {
	r := newRollTargetRouter(NewRollTargetsHandler(&stubRollTargets{}))
	for _, path := range []string{"/api/rolltargets/abc", "/api/rolltargets/0", "/api/rolltargets/-1"} {
		if w := send(r, http.MethodDelete, path, ""); w.Code != http.StatusBadRequest {
			t.Errorf("DELETE %s status = %d, want 400", path, w.Code)
		}
	}
}

// Each typed outcome has one status. A missing target and a foreign one are
// both 404: telling them apart would confirm another user's target ids.
func TestRollTargets_ErrorStatusMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
		code string
	}{
		{"no database", rolltargets.ErrUnavailable, http.StatusServiceUnavailable, "DB_UNAVAILABLE"},
		{"missing or foreign", rolltargets.ErrNotFound, http.StatusNotFound, ""},
		{"already saved", rolltargets.ErrDuplicate, http.StatusConflict, ""},
		{"manifest warming", rolltargets.ErrPerksUnavailable, http.StatusServiceUnavailable, "MANIFEST_NOT_READY"},
		{"no perks", rolltargets.ErrNoPerks, http.StatusBadRequest, ""},
		{"too many perks", rolltargets.ErrTooManyPerks, http.StatusBadRequest, ""},
		{"duplicate perk", rolltargets.ErrDuplicatePerk, http.StatusBadRequest, ""},
		{"notes too long", rolltargets.ErrNotesTooLong, http.StatusBadRequest, ""},
		{"not a weapon", rolltargets.ErrNotAWeapon, http.StatusBadRequest, ""},
		{"unknown perk", rolltargets.ErrUnknownPerk, http.StatusBadRequest, ""},
		{"unknown perk name", rolltargets.ErrUnknownPerkName, http.StatusBadRequest, ""},
		{"anything else", errors.New("boom"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubRollTargets{err: tc.err}
			w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodGet, "/api/rolltargets", "")
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", w.Code, tc.want, w.Body.String())
			}
			if tc.code != "" && !strings.Contains(w.Body.String(), tc.code) {
				t.Errorf("body = %s, want code %s", w.Body.String(), tc.code)
			}
			// A refusal must never leak the domain's package prefix.
			if strings.Contains(w.Body.String(), "rolltargets:") {
				t.Errorf("wire carried the package prefix: %s", w.Body.String())
			}
		})
	}
}

// The import body is the file itself, and every line comes back — including the
// ones that did not import.
func TestImportRollTargets_ReportsEveryLine(t *testing.T) {
	hash := uint32(1000)
	stub := &stubRollTargets{report: rolltargets.ImportReport{
		Title: "mine",
		Lines: []rolltargets.ImportLine{
			{Number: 1, Outcome: rolltargets.OutcomeImported, ItemHash: &hash, Wanted: true, Perks: []string{"Outlaw"}},
			{Number: 2, Outcome: rolltargets.OutcomeMalformed, Detail: "not a recognized DIM wish list line"},
		},
	}}
	body := "dimwishlist:item=1000&perks=111\ngarbage"
	w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodPost, "/api/rolltargets/import", body)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	// The file reaches the service byte for byte.
	if stub.gotImportText != body {
		t.Errorf("service saw %q, want the raw body", stub.gotImportText)
	}
	var resp importReportResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Title != "mine" || resp.Imported != 1 {
		t.Errorf("report = %+v", resp)
	}
	if len(resp.Lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(resp.Lines))
	}
	if resp.Lines[1].Outcome != string(rolltargets.OutcomeMalformed) || resp.Lines[1].Detail == "" {
		t.Errorf("the failing line lost its reason: %+v", resp.Lines[1])
	}
	if resp.Counts[string(rolltargets.OutcomeMalformed)] != 1 {
		t.Errorf("counts = %v", resp.Counts)
	}
}

// The structured unresolved-perk detail transcribes straight onto the wire,
// and PerkName is omitted when the domain did not have one to give.
func TestImportRollTargets_TranscribesUnresolvedPerkDetail(t *testing.T) {
	stub := &stubRollTargets{report: rolltargets.ImportReport{
		Lines: []rolltargets.ImportLine{
			{
				Number: 1, Outcome: rolltargets.OutcomeUnresolvedPerk,
				Detail:     `perk "Drop Mag" resolves to more than one plug`,
				Unresolved: &rolltargets.UnresolvedPerk{Hash: 333, Name: "Drop Mag", Reason: rolltargets.UnresolvedAmbiguous},
			},
			{
				Number: 2, Outcome: rolltargets.OutcomeUnresolvedPerk,
				Detail:     "no perk matches hash 999",
				Unresolved: &rolltargets.UnresolvedPerk{Hash: 999, Reason: rolltargets.UnresolvedNotInPool},
			},
		},
	}}
	w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodPost, "/api/rolltargets/import",
		"dimwishlist:item=1000&perks=333")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp importReportResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(resp.Lines))
	}
	u1 := resp.Lines[0].Unresolved
	if u1 == nil || u1.PerkHash != 333 || u1.PerkName != "Drop Mag" || u1.Reason != "ambiguous" {
		t.Errorf("line 1 unresolved = %+v", u1)
	}
	// A no-name hash must omit perkName entirely, not send an empty string.
	if !strings.Contains(w.Body.String(), `"perkHash":999`) {
		t.Errorf("body missing perkHash 999: %s", w.Body.String())
	}
	u2 := resp.Lines[1].Unresolved
	if u2 == nil || u2.PerkName != "" || u2.Reason != "not-in-pool" {
		t.Errorf("line 2 unresolved = %+v", u2)
	}
	var raw map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	line2 := raw["lines"].([]any)[1].(map[string]any)
	unresolved2 := line2["unresolved"].(map[string]any)
	if _, present := unresolved2["perkName"]; present {
		t.Errorf("perkName present when the domain gave no name: %v", unresolved2)
	}
}

// --- fakes for the real-service, no-package-prefix invariant below ---
//
// These drive the actual *rolltargets.Service — not the stub — so the
// assertion covers the whole stack a client actually sees rather than a
// handler test's own expectations of what the service would say.

type fakeBulkImportRepo struct {
	targets []rolltargets.StoredTarget
	nextID  rolltargets.TargetID
}

func (r *fakeBulkImportRepo) List(context.Context, string) ([]rolltargets.StoredTarget, error) {
	return r.targets, nil
}

func (r *fakeBulkImportRepo) Add(_ context.Context, _ string, cmd rolltargets.AddCommand) (rolltargets.StoredTarget, error) {
	r.nextID++
	t := rolltargets.StoredTarget{ID: r.nextID, ItemHash: cmd.ItemHash, Wanted: cmd.Wanted, Perks: cmd.Perks, Notes: cmd.Notes}
	r.targets = append(r.targets, t)
	return t, nil
}

func (r *fakeBulkImportRepo) Update(context.Context, string, rolltargets.TargetID, rolltargets.UpdateCommand) (rolltargets.StoredTarget, error) {
	return rolltargets.StoredTarget{}, nil
}
func (r *fakeBulkImportRepo) Remove(context.Context, string, rolltargets.TargetID) error { return nil }
func (r *fakeBulkImportRepo) RemoveMany(context.Context, string, []rolltargets.TargetID) (int, error) {
	return 0, nil
}
func (r *fakeBulkImportRepo) RemoveAll(context.Context, string) (int, error) { return 0, nil }

// fakeImportPerkPool serves one weapon (hash 1000) with a paired base/enhanced
// perk, a base-only perk, and an ambiguous one — enough to drive every
// unresolved-perk reason through one file.
type fakeImportPerkPool struct{}

func (fakeImportPerkPool) GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error) {
	if itemHash != 1000 {
		return nil, nil
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

func (fakeImportPerkPool) WeaponPerkNames() (map[string]string, error) {
	return map[string]string{"outlaw": "Outlaw", "firefly": "Firefly", "drop mag": "Drop Mag"}, nil
}

func (fakeImportPerkPool) PlugNames(hashes []uint32) (map[uint32]string, error) {
	names := map[uint32]string{111: "Outlaw", 911: "Outlaw", 222: "Firefly", 333: "Drop Mag", 444: "Minor Spec"}
	out := map[uint32]string{}
	for _, h := range hashes {
		if n, ok := names[h]; ok {
			out[h] = n
		}
	}
	return out, nil
}

// No detail the real service produces, across every outcome one file can
// carry, may reach the wire with this package's log-oriented "rolltargets:"
// prefix. This drives the actual service through the actual handler route —
// not a stub — so it is the whole stack a client would see, not just what a
// handler test expects the service to say.
func TestImportRollTargets_RealServiceNeverLeaksThePackagePrefix(t *testing.T) {
	real := rolltargets.NewService(&fakeBulkImportRepo{
		targets: []rolltargets.StoredTarget{{ID: 1, ItemHash: rollTargetHash(1000), Wanted: true, Perks: []string{"Outlaw"}}},
	}, fakeImportPerkPool{}, nil)

	text := "dimwishlist:item=1000&perks=111,222\n" + // imports
		"dimwishlist:item=4242&perks=111\n" + // unknown weapon
		"dimwishlist:item=1000&perks=111\n" + // already saved
		"dimwishlist:item=1000&perks=999\n" + // unresolved: not in pool
		"dimwishlist:item=1000&perks=333\n" + // unresolved: ambiguous
		"dimwishlist:item=-69420&perks=444\n" + // unresolved: not a weapon perk
		"dimwishlist:item=1000\n" + // no perks named -> unsupported
		"nonsense\n" // malformed
	w := send(newRollTargetRouter(NewRollTargetsHandler(real)), http.MethodPost, "/api/rolltargets/import", text)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "rolltargets:") {
		t.Errorf("wire carried the package prefix: %s", w.Body.String())
	}

	var resp importReportResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Lines) != 8 {
		t.Fatalf("lines = %d, want 8", len(resp.Lines))
	}
	if got := resp.Counts[string(rolltargets.OutcomeUnresolvedPerk)]; got != 3 {
		t.Errorf("unresolved perk count = %d, want 3", got)
	}
}

func TestImportRollTargets_RejectsEmptyAndOversizedBodies(t *testing.T) {
	r := newRollTargetRouter(NewRollTargetsHandler(&stubRollTargets{}))
	if w := send(r, http.MethodPost, "/api/rolltargets/import", ""); w.Code != http.StatusBadRequest {
		t.Errorf("empty body status = %d, want 400", w.Code)
	}
	huge := strings.Repeat("x", maxImportBytes+1)
	if w := send(r, http.MethodPost, "/api/rolltargets/import", huge); w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body status = %d, want 413", w.Code)
	}
}

func TestImportRollTargets_PropagatesServiceFailure(t *testing.T) {
	stub := &stubRollTargets{err: rolltargets.ErrUnavailable}
	w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodPost, "/api/rolltargets/import", "dimwishlist:item=1&perks=2")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", w.Code, w.Body.String())
	}
}

func TestGetRollTargetMatches_SeparatesWantedUnwantedAndUnmatched(t *testing.T) {
	hash := uint32(1000)
	stub := &stubRollTargets{matches: rolltargets.MatchReport{
		Wanted: []rolltargets.Match{{
			Target: rolltargets.StoredTarget{ID: 1, ItemHash: &hash, Wanted: true, Perks: []string{"Outlaw"}, Notes: "pvp"},
			Roll:   ownedrolls.OwnedRoll{ItemHash: 1000, InstanceID: "abc", Perks: []string{"Firefly", "Outlaw"}},
		}},
		Unwanted: []rolltargets.Match{{
			Target: rolltargets.StoredTarget{ID: 2, ItemHash: &hash, Perks: []string{"Firefly"}},
			Roll:   ownedrolls.OwnedRoll{ItemHash: 1000, InstanceID: "def", Perks: []string{"Firefly"}},
		}},
		UnmatchedTargets: []rolltargets.UnmatchedTarget{
			{
				Target: rolltargets.StoredTarget{ID: 3, ItemHash: nil, Wanted: true, Perks: []string{"Rampage", "Outlaw"}},
				NearMiss: &rolltargets.NearMiss{
					Roll:         ownedrolls.OwnedRoll{ItemHash: 2000, InstanceID: "near", Perks: []string{"Firefly", "Rampage"}},
					MatchedPerks: []string{"Rampage"},
				},
			},
		},
	}}
	w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodGet, "/api/rolltargets/matches", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp matchReportResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Wanted) != 1 || resp.Wanted[0].InstanceID != "abc" || resp.Wanted[0].TargetID != "1" {
		t.Errorf("wanted = %+v", resp.Wanted)
	}
	if len(resp.Unwanted) != 1 || resp.Unwanted[0].InstanceID != "def" {
		t.Errorf("unwanted = %+v", resp.Unwanted)
	}
	// The roll still worth chasing has to survive onto the wire.
	if len(resp.UnmatchedTargets) != 1 || resp.UnmatchedTargets[0].ID != "3" || !resp.UnmatchedTargets[0].AnyWeapon {
		t.Errorf("unmatched = %+v", resp.UnmatchedTargets)
	}
	if best := resp.UnmatchedTargets[0].BestCopy; best == nil || best.InstanceID != "near" || len(best.MatchedPerks) != 1 || best.MatchedPerks[0] != "Rampage" {
		t.Errorf("best copy = %+v, want serialized near miss", best)
	}
}

// Matching reads inventory per membership pair, so the platform has to travel
// with the id.
func TestGetRollTargetMatches_PassesTheMembershipPair(t *testing.T) {
	stub := &stubRollTargets{}
	send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodGet, "/api/rolltargets/matches", "")
	if stub.gotMembershipType != 3 || stub.gotMembership != "test-member-123" {
		t.Errorf("service saw %d/%q, want 3/test-member-123", stub.gotMembershipType, stub.gotMembership)
	}
}

// An inventory that could not be read must not render as "nothing matches".
func TestGetRollTargetMatches_InventoryFailuresHaveTheirOwnStatus(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
		code string
	}{
		{"no reader", rolltargets.ErrOwnedRollsUnavailable, http.StatusServiceUnavailable, "OWNED_ROLLS_UNAVAILABLE"},
		{"components unreadable", ownedrolls.ErrInventoryUnavailable, http.StatusServiceUnavailable, "OWNED_ROLLS_UNAVAILABLE"},
		{"bungie authorization gone", ownedrolls.ErrNoCredential, http.StatusUnauthorized, "BUNGIE_REAUTH_REQUIRED"},
		{"manifest not ready", ownedrolls.ErrPerksUnavailable, http.StatusServiceUnavailable, "MANIFEST_NOT_READY"},
		{"bungie rate limited", &bungie.BungieError{ErrorCode: 36, ThrottleSeconds: 5}, http.StatusTooManyRequests, "RATE_LIMITED"},
		{"bungie failed", &bungie.BungieError{ErrorCode: 1618}, http.StatusBadGateway, "BUNGIE_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubRollTargets{err: tc.err}
			w := send(newRollTargetRouter(NewRollTargetsHandler(stub)), http.MethodGet, "/api/rolltargets/matches", "")
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", w.Code, tc.want, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.code) {
				t.Errorf("body = %s, want code %s", w.Body.String(), tc.code)
			}
			if strings.Contains(w.Body.String(), "ownedrolls:") {
				t.Errorf("wire carried the package prefix: %s", w.Body.String())
			}
		})
	}
}
