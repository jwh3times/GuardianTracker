package handlers

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

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/sources"
	"guardian-tracker/api-service/services/wishlist"

	"github.com/gin-gonic/gin"
)

// --- stub wish list core ---
//
// The handler owns no wish list rules any more, so these tests drive the seam
// rather than a fake database: what reaches the service, and what each typed
// outcome renders as. The rules themselves are tested in services/wishlist.

type stubEntries struct {
	entries []wishlist.StoredEntry
	stored  wishlist.StoredEntry
	bulk    wishlist.BulkResult
	err     error

	gotMembershipID string
	gotAdd          wishlist.AddCommand
	gotUpdateID     wishlist.EntryID
	gotUpdate       wishlist.UpdateCommand
	gotRemoveID     wishlist.EntryID
	gotBulkIDs      []wishlist.EntryID
	gotBulkPriority wishlist.Priority
	gotAction       string
}

func (s *stubEntries) List(_ context.Context, membershipID string) ([]wishlist.StoredEntry, error) {
	s.gotMembershipID = membershipID
	return s.entries, s.err
}

func (s *stubEntries) Add(_ context.Context, membershipID string, cmd wishlist.AddCommand) (wishlist.StoredEntry, error) {
	s.gotMembershipID, s.gotAdd = membershipID, cmd
	return s.stored, s.err
}

func (s *stubEntries) Update(_ context.Context, membershipID string, id wishlist.EntryID, patch wishlist.UpdateCommand) (wishlist.StoredEntry, error) {
	s.gotMembershipID, s.gotUpdateID, s.gotUpdate = membershipID, id, patch
	return s.stored, s.err
}

func (s *stubEntries) Remove(_ context.Context, membershipID string, id wishlist.EntryID) error {
	s.gotMembershipID, s.gotRemoveID = membershipID, id
	return s.err
}

func (s *stubEntries) RemoveMany(_ context.Context, membershipID string, ids []wishlist.EntryID) (wishlist.BulkResult, error) {
	s.gotMembershipID, s.gotBulkIDs, s.gotAction = membershipID, ids, "delete"
	return s.bulk, s.err
}

func (s *stubEntries) SetPriorityMany(_ context.Context, membershipID string, ids []wishlist.EntryID, priority wishlist.Priority) (wishlist.BulkResult, error) {
	s.gotMembershipID, s.gotBulkIDs, s.gotBulkPriority, s.gotAction = membershipID, ids, priority, "set_priority"
	return s.bulk, s.err
}

// --- mock manifest ---

type mockManifest struct {
	defs map[uint32]*bungie.InventoryItemDefinition
	cols map[uint32][]bungie.CollectibleDefinition
}

func (m *mockManifest) GetItemsByHashes(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error) {
	if m.defs == nil {
		return map[uint32]*bungie.InventoryItemDefinition{}, nil
	}
	out := make(map[uint32]*bungie.InventoryItemDefinition)
	for _, h := range hashes {
		if def, ok := m.defs[h]; ok {
			out[h] = def
		}
	}
	return out, nil
}

func (m *mockManifest) GetCollectiblesByItemHashes(hashes []uint32) (map[uint32][]bungie.CollectibleDefinition, error) {
	if m.cols == nil {
		return map[uint32][]bungie.CollectibleDefinition{}, nil
	}
	out := make(map[uint32][]bungie.CollectibleDefinition)
	for _, h := range hashes {
		if col, ok := m.cols[h]; ok {
			out[h] = col
		}
	}
	return out, nil
}

// --- mock live-vendor availability ---

type mockLiveVendors struct {
	hashes map[uint32]string
}

func (m *mockLiveVendors) LiveVendorItemHashes(_ context.Context, _ int, _, _ string) map[uint32]string {
	if m.hashes == nil {
		return map[uint32]string{}
	}
	return m.hashes
}

// --- mock token provider ---

type mockTokens struct {
	token string
	err   error
}

func (m *mockTokens) GetValidToken(_ string) (string, error) { return m.token, m.err }

// --- router setup helper ---

func newTestRouter(h *WishlistHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("membership_id", "test-member-123")
		c.Set("membership_type", 3)
		c.Next()
	})
	r.GET("/api/wishlist", h.GetWishlist)
	r.POST("/api/wishlist", h.AddToWishlist)
	r.PUT("/api/wishlist/:id", h.UpdateWishlistItem)
	r.DELETE("/api/wishlist/:id", h.RemoveFromWishlist)
	r.POST("/api/wishlist/bulk", h.BulkUpdate)
	return r
}

func send(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func savedEntry(id wishlist.EntryID, hash uint32, priority wishlist.Priority) wishlist.StoredEntry {
	return wishlist.StoredEntry{ID: id, ItemHash: hash, Priority: priority, CreatedAt: time.Now()}
}

// --- tests ---

func TestGetWishlist_SerializesStoredEntries(t *testing.T) {
	entries := &stubEntries{entries: []wishlist.StoredEntry{
		{ID: 1, ItemHash: 1234, Priority: wishlist.PriorityHigh, Notes: "nice roll", CreatedAt: time.Now()},
	}}
	w := send(newTestRouter(NewWishlistHandler(entries, nil, nil, nil)), http.MethodGet, "/api/wishlist", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp []wishlistResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("items = %d, want 1", len(resp))
	}
	if resp[0].ID != "1" || resp[0].ItemHash != 1234 || resp[0].Priority != "HIGH" || resp[0].Notes != "nice roll" {
		t.Errorf("item = %+v", resp[0])
	}
}

// An empty wish list is an empty JSON array, never null: the frontend maps over
// it directly.
func TestGetWishlist_EmptyListSerializesAsAnArray(t *testing.T) {
	w := send(newTestRouter(NewWishlistHandler(&stubEntries{}, nil, nil, nil)), http.MethodGet, "/api/wishlist", "")

	if body := strings.TrimSpace(w.Body.String()); body != "[]" {
		t.Errorf("body = %s, want []", body)
	}
}

func TestGetWishlist_UsesTheJWTMembership(t *testing.T) {
	entries := &stubEntries{}
	send(newTestRouter(NewWishlistHandler(entries, nil, nil, nil)), http.MethodGet, "/api/wishlist", "")

	if entries.gotMembershipID != "test-member-123" {
		t.Errorf("membership = %q, want the one the JWT carried", entries.gotMembershipID)
	}
}

// Every typed outcome the wish list can produce has exactly one wire meaning.
// Flattening any two of them would put domain meaning back in the handler.
func TestWishlistErrors_MapToTheExistingWire(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		status   int
		contains string
	}{
		{"unavailable", wishlist.ErrUnavailable, http.StatusServiceUnavailable, "DB_UNAVAILABLE"},
		{"not found", wishlist.ErrNotFound, http.StatusNotFound, "wishlist item not found"},
		{"duplicate", wishlist.ErrDuplicate, http.StatusConflict, "item already in wishlist"},
		{"items unavailable", wishlist.ErrItemsUnavailable, http.StatusServiceUnavailable, "MANIFEST_NOT_READY"},
		{"unknown item", wishlist.ErrUnknownItem, http.StatusBadRequest, "unknown item hash"},
		{"invalid priority", wishlist.ErrInvalidPriority, http.StatusBadRequest, "priority must be LOW, MEDIUM, HIGH, or URGENT"},
		{"notes too long", wishlist.ErrNotesTooLong, http.StatusBadRequest, "notes must be 500 characters or fewer"},
		{"unexpected", errors.New("connection reset"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRouter(NewWishlistHandler(&stubEntries{err: tc.err}, nil, nil, nil))
			w := send(r, http.MethodPost, "/api/wishlist", `{"itemHash":1234}`)

			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", w.Code, tc.status, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.contains) {
				t.Errorf("body = %s, want it to contain %q", w.Body.String(), tc.contains)
			}
		})
	}
}

func TestGetWishlist_UnavailablePersistenceReturns503(t *testing.T) {
	r := newTestRouter(NewWishlistHandler(&stubEntries{err: wishlist.ErrUnavailable}, nil, nil, nil))
	w := send(r, http.MethodGet, "/api/wishlist", "")

	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "DB_UNAVAILABLE") {
		t.Errorf("status = %d body = %s, want 503 DB_UNAVAILABLE", w.Code, w.Body.String())
	}
}

func TestAddToWishlist_PassesTheRequestThroughAsACommand(t *testing.T) {
	entries := &stubEntries{stored: savedEntry(9, 1234, wishlist.PriorityUrgent)}
	r := newTestRouter(NewWishlistHandler(entries, nil, nil, nil))

	w := send(r, http.MethodPost, "/api/wishlist", `{"itemHash":1234,"priority":"URGENT","notes":"soon"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}
	want := wishlist.AddCommand{ItemHash: 1234, Priority: wishlist.PriorityUrgent, Notes: "soon"}
	if entries.gotAdd != want {
		t.Errorf("command = %+v, want %+v", entries.gotAdd, want)
	}
}

// An omitted priority reaches the service empty. The default is the wish
// list's to choose, and the handler must not quietly pick one first.
func TestAddToWishlist_OmittedPriorityStaysUnsetAtTheSeam(t *testing.T) {
	entries := &stubEntries{stored: savedEntry(9, 1234, wishlist.PriorityMedium)}
	r := newTestRouter(NewWishlistHandler(entries, nil, nil, nil))

	send(r, http.MethodPost, "/api/wishlist", `{"itemHash":1234}`)

	if entries.gotAdd.Priority != "" {
		t.Errorf("priority = %q, want it left to the wish list", entries.gotAdd.Priority)
	}
}

func TestAddToWishlist_MissingItemHashIsRejectedBeforeTheSeam(t *testing.T) {
	entries := &stubEntries{}
	r := newTestRouter(NewWishlistHandler(entries, nil, nil, nil))

	w := send(r, http.MethodPost, "/api/wishlist", `{"priority":"HIGH"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if entries.gotAdd.ItemHash != 0 {
		t.Error("a request with no item hash reached the wish list")
	}
}

func TestUpdateWishlistItem_SendsAPartialPatch(t *testing.T) {
	entries := &stubEntries{stored: savedEntry(7, 1234, wishlist.PriorityLow)}
	r := newTestRouter(NewWishlistHandler(entries, nil, nil, nil))

	w := send(r, http.MethodPut, "/api/wishlist/7", `{"notes":"changed"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if entries.gotUpdateID != 7 {
		t.Errorf("entry id = %d, want 7", entries.gotUpdateID)
	}
	if entries.gotUpdate.Priority != nil {
		t.Error("an absent priority was sent as a change")
	}
	if entries.gotUpdate.Notes == nil || *entries.gotUpdate.Notes != "changed" {
		t.Errorf("notes patch = %v, want \"changed\"", entries.gotUpdate.Notes)
	}
}

func TestUpdateWishlistItem_NotFoundReturns404(t *testing.T) {
	r := newTestRouter(NewWishlistHandler(&stubEntries{err: wishlist.ErrNotFound}, nil, nil, nil))

	if w := send(r, http.MethodPut, "/api/wishlist/7", `{"notes":"x"}`); w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestWishlistItemID_MustBeNumeric(t *testing.T) {
	entries := &stubEntries{}
	r := newTestRouter(NewWishlistHandler(entries, nil, nil, nil))

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		w := send(r, method, "/api/wishlist/abc", `{}`)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s status = %d, want 400", method, w.Code)
		}
	}
	if entries.gotUpdateID != 0 || entries.gotRemoveID != 0 {
		t.Error("a malformed id reached the wish list")
	}
}

func TestRemoveFromWishlist_SuccessReturns204(t *testing.T) {
	entries := &stubEntries{}
	r := newTestRouter(NewWishlistHandler(entries, nil, nil, nil))

	w := send(r, http.MethodDelete, "/api/wishlist/5", "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if entries.gotRemoveID != 5 {
		t.Errorf("removed id = %d, want 5", entries.gotRemoveID)
	}
}

func TestRemoveFromWishlist_NotFoundReturns404(t *testing.T) {
	r := newTestRouter(NewWishlistHandler(&stubEntries{err: wishlist.ErrNotFound}, nil, nil, nil))

	if w := send(r, http.MethodDelete, "/api/wishlist/5", ""); w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestBulkUpdate_RoutesEachActionToItsCommand(t *testing.T) {
	t.Run("delete", func(t *testing.T) {
		entries := &stubEntries{bulk: wishlist.BulkResult{Updated: 2, Skipped: 1}}
		r := newTestRouter(NewWishlistHandler(entries, nil, nil, nil))

		w := send(r, http.MethodPost, "/api/wishlist/bulk", `{"action":"delete","ids":[1,2,999]}`)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		if entries.gotAction != "delete" || len(entries.gotBulkIDs) != 3 {
			t.Errorf("action = %q with ids %v", entries.gotAction, entries.gotBulkIDs)
		}
		var body struct{ Updated, Skipped int }
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Updated != 2 || body.Skipped != 1 {
			t.Errorf("body = %+v, want {2 1}", body)
		}
	})

	t.Run("set priority", func(t *testing.T) {
		entries := &stubEntries{bulk: wishlist.BulkResult{Updated: 1}}
		r := newTestRouter(NewWishlistHandler(entries, nil, nil, nil))

		w := send(r, http.MethodPost, "/api/wishlist/bulk", `{"action":"set_priority","ids":[4],"priority":"HIGH"}`)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		if entries.gotAction != "set_priority" || entries.gotBulkPriority != wishlist.PriorityHigh {
			t.Errorf("action = %q priority = %q", entries.gotAction, entries.gotBulkPriority)
		}
	})
}

// The action string is the handler's own vocabulary — it names which command to
// call, so an unknown one never reaches the wish list.
func TestBulkUpdate_UnknownActionIsRejectedBeforeTheSeam(t *testing.T) {
	entries := &stubEntries{}
	r := newTestRouter(NewWishlistHandler(entries, nil, nil, nil))

	w := send(r, http.MethodPost, "/api/wishlist/bulk", `{"action":"explode","ids":[1]}`)

	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "action must be") {
		t.Fatalf("status = %d body = %s, want 400", w.Code, w.Body.String())
	}
	if entries.gotAction != "" {
		t.Errorf("action %q reached the wish list", entries.gotAction)
	}
}

func TestBulkUpdate_ValidationRefusalsRenderAsTheExistingMessages(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		contains string
	}{
		{"empty", wishlist.ErrNoEntries, "ids must be a non-empty list"},
		{"too many", wishlist.ErrTooManyEntries, "at most 100 ids per request"},
		{"invalid priority", wishlist.ErrInvalidPriority, "priority must be LOW, MEDIUM, HIGH, or URGENT"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRouter(NewWishlistHandler(&stubEntries{err: tc.err}, nil, nil, nil))
			w := send(r, http.MethodPost, "/api/wishlist/bulk", `{"action":"delete","ids":[1]}`)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.contains) {
				t.Errorf("body = %s, want %q", w.Body.String(), tc.contains)
			}
		})
	}
}

func TestBulkUpdate_UnavailablePersistenceReturns503(t *testing.T) {
	r := newTestRouter(NewWishlistHandler(&stubEntries{err: wishlist.ErrUnavailable}, nil, nil, nil))
	w := send(r, http.MethodPost, "/api/wishlist/bulk", `{"action":"delete","ids":[1]}`)

	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "DB_UNAVAILABLE") {
		t.Errorf("status = %d body = %s, want 503 DB_UNAVAILABLE", w.Code, w.Body.String())
	}
}

// --- completion (still handler-owned until the complete wish list service) ---

func TestEnrichItems_WithManifest(t *testing.T) {
	entries := &stubEntries{entries: []wishlist.StoredEntry{
		{ID: 1, ItemHash: 5555, Priority: wishlist.PriorityUrgent, CreatedAt: time.Now()},
	}}
	manifest := &mockManifest{
		defs: map[uint32]*bungie.InventoryItemDefinition{
			5555: {
				Hash:              5555,
				DisplayProperties: bungie.DisplayProperties{Name: "Gjallarhorn"},
				ItemType:          bungie.ItemTypeWeapon,
				ItemSubType:       bungie.WeaponSubTypeRocketLauncher,
				Inventory: struct {
					TierType int `json:"tierType"`
				}{TierType: bungie.TierTypeExotic},
			},
		},
	}
	w := send(newTestRouter(NewWishlistHandler(entries, manifest, nil, nil)), http.MethodGet, "/api/wishlist", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp []wishlistResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("items = %d, want 1", len(resp))
	}
	if resp[0].Name != "Gjallarhorn" || resp[0].Rarity != "Exotic" || resp[0].ItemType != "Rocket Launcher" {
		t.Errorf("item = %+v", resp[0])
	}
	if resp[0].Priority != "URGENT" {
		t.Errorf("priority = %s, want URGENT", resp[0].Priority)
	}
}

// An item the manifest no longer carries keeps its user-authored metadata and
// the visible fallback projection, rather than vanishing from the list.
func TestEnrichItems_UnknownItemKeepsTheFallbackProjection(t *testing.T) {
	entries := &stubEntries{entries: []wishlist.StoredEntry{
		{ID: 1, ItemHash: 4242, Priority: wishlist.PriorityLow, Notes: "kept", CreatedAt: time.Now()},
	}}
	w := send(newTestRouter(NewWishlistHandler(entries, &mockManifest{}, nil, nil)), http.MethodGet, "/api/wishlist", "")

	var resp []wishlistResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("items = %d, want the stored entry retained", len(resp))
	}
	if resp[0].Name != "Unknown Item" || resp[0].ItemType != "Item" || resp[0].Rarity != "Common" || resp[0].Icon != "" {
		t.Errorf("fallback projection = %+v", resp[0])
	}
	if resp[0].Notes != "kept" {
		t.Errorf("notes = %q, want the user's metadata preserved", resp[0].Notes)
	}
}

// TestEnrichItems_AvailabilityAndAcquisitionSources: live availability stays
// separate from the deterministic union of collectible provenance.
func TestEnrichItems_AvailabilityAndAcquisitionSources(t *testing.T) {
	entries := &stubEntries{entries: []wishlist.StoredEntry{
		{ID: 1, ItemHash: 5555, Priority: wishlist.PriorityMedium, CreatedAt: time.Now()},
		{ID: 2, ItemHash: 6666, Priority: wishlist.PriorityMedium, CreatedAt: time.Now()},
	}}
	gjally := &bungie.InventoryItemDefinition{Hash: 5555}
	gjally.DisplayProperties.Name = "Gjallarhorn"
	gjally.DisplayProperties.Icon = "/icons/gjally.png"
	other := &bungie.InventoryItemDefinition{Hash: 6666}
	other.DisplayProperties.Name = "Fatebringer"
	manifest := &mockManifest{
		defs: map[uint32]*bungie.InventoryItemDefinition{5555: gjally, 6666: other},
		cols: map[uint32][]bungie.CollectibleDefinition{
			6666: {
				{ItemHash: 6666, SourceString: "Vault of Glass raid"},
				{ItemHash: 6666, SourceString: "Monument to Lost Lights"},
				{ItemHash: 6666, SourceString: "Vault of Glass raid"},
			},
		},
	}
	// 5555 sold by Banshee-44 (a non-Xûr vendor); 6666 not currently sold.
	live := &mockLiveVendors{hashes: map[uint32]string{5555: "Banshee-44"}}
	h := NewWishlistHandler(entries, manifest, live, &mockTokens{token: "tok"})

	w := send(newTestRouter(h), http.MethodGet, "/api/wishlist", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp []wishlistResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	byHash := map[uint32]wishlistResponse{}
	for _, it := range resp {
		byHash[it.ItemHash] = it
	}
	atVendor := byHash[5555]
	if !atVendor.AvailableNow || atVendor.AvailableFrom != "Banshee-44" {
		t.Errorf("vendor item = availableNow %v from %q; want true, Banshee-44", atVendor.AvailableNow, atVendor.AvailableFrom)
	}
	notSold := byHash[6666]
	if notSold.AvailableNow || notSold.AvailableFrom != "" {
		t.Errorf("unsold item flagged available: %+v", notSold)
	}
	wantSources := []sources.AcquisitionSource{
		{Text: "Monument to Lost Lights", Difficulty: sources.Easy},
		{Text: "Vault of Glass raid", Difficulty: sources.Challenging, RaidDungeon: true},
	}
	if len(notSold.AcquisitionSources) != len(wantSources) {
		t.Fatalf("acquisitionSources = %+v, want %+v", notSold.AcquisitionSources, wantSources)
	}
	for i := range wantSources {
		if notSold.AcquisitionSources[i] != wantSources[i] {
			t.Errorf("acquisitionSources[%d] = %+v, want %+v", i, notSold.AcquisitionSources[i], wantSources[i])
		}
	}
	var wire []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &wire); err != nil {
		t.Fatalf("decode wire shape: %v", err)
	}
	if _, exists := wire[1]["difficulty"]; exists {
		t.Errorf("wishlist item must not expose aggregate difficulty: %s", w.Body.Bytes())
	}
	if _, exists := wire[1]["sources"]; exists {
		t.Errorf("wishlist item must not expose legacy text-only sources: %s", w.Body.Bytes())
	}
}

// TestEnrichItems_TokenErrorBestEffort: a token-store error must not fail the
// request — availability just falls back to whatever the provider returns.
func TestEnrichItems_TokenErrorBestEffort(t *testing.T) {
	entries := &stubEntries{entries: []wishlist.StoredEntry{
		{ID: 1, ItemHash: 5555, Priority: wishlist.PriorityMedium, CreatedAt: time.Now()},
	}}
	live := &mockLiveVendors{hashes: map[uint32]string{5555: "Xûr"}}
	h := NewWishlistHandler(entries, nil, live, &mockTokens{err: fmt.Errorf("no token")})

	w := send(newTestRouter(h), http.MethodGet, "/api/wishlist", "")

	if w.Code != http.StatusOK {
		t.Fatalf("token error should not fail request; got %d", w.Code)
	}
}
