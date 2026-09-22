package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

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

	gotMembership string
	gotAdd        rolltargets.AddCommand
	gotUpdateID   rolltargets.TargetID
	gotUpdate     rolltargets.UpdateCommand
	gotRemoveID   rolltargets.TargetID
	gotImportText string
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

func (s *stubRollTargets) ImportDIM(_ context.Context, m, text string) (rolltargets.ImportReport, error) {
	s.gotMembership, s.gotImportText = m, text
	return s.report, s.err
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
