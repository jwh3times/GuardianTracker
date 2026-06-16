package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/bungie"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// --- mock wishlist store ---

type mockWishlistStore struct {
	userID   int64
	items    []db.WishlistItem
	addErr   error
	delFound bool
}

func (m *mockWishlistStore) GetUserID(_ context.Context, _ string) (int64, error) {
	return m.userID, nil
}

func (m *mockWishlistStore) List(_ context.Context, _ int64) ([]db.WishlistItem, error) {
	return m.items, nil
}

func (m *mockWishlistStore) Add(_ context.Context, _ int64, hash uint32, prio int16, notes string) (*db.WishlistItem, error) {
	if m.addErr != nil {
		return nil, m.addErr
	}
	return &db.WishlistItem{ID: 1, ItemHash: hash, Priority: prio, Notes: notes, CreatedAt: time.Now()}, nil
}

func (m *mockWishlistStore) Update(_ context.Context, _, id int64, prio *int16, notes *string) (*db.WishlistItem, error) {
	if len(m.items) == 0 {
		return nil, pgx.ErrNoRows
	}
	it := m.items[0]
	if prio != nil {
		it.Priority = *prio
	}
	if notes != nil {
		it.Notes = *notes
	}
	return &it, nil
}

func (m *mockWishlistStore) Delete(_ context.Context, _, _ int64) (bool, error) {
	return m.delFound, nil
}

// --- mock prefs store ---

type mockPrefsStore struct {
	prefs *db.UserPreferences
}

func (m *mockPrefsStore) Get(_ context.Context, userID int64) (*db.UserPreferences, error) {
	if m.prefs != nil {
		return m.prefs, nil
	}
	return &db.UserPreferences{UserID: userID, CardStyle: "framed", Personalize: true, UpdatedAt: time.Now()}, nil
}

func (m *mockPrefsStore) Upsert(_ context.Context, userID int64, cardStyle string, personalize bool) (*db.UserPreferences, error) {
	return &db.UserPreferences{UserID: userID, CardStyle: cardStyle, Personalize: personalize, UpdatedAt: time.Now()}, nil
}

// --- mock manifest ---

type mockManifest struct {
	defs map[uint32]*bungie.InventoryItemDefinition
	cols map[uint32]*bungie.CollectibleDefinition
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

func (m *mockManifest) GetCollectiblesByItemHashes(hashes []uint32) (map[uint32]*bungie.CollectibleDefinition, error) {
	if m.cols == nil {
		return map[uint32]*bungie.CollectibleDefinition{}, nil
	}
	out := make(map[uint32]*bungie.CollectibleDefinition)
	for _, h := range hashes {
		if col, ok := m.cols[h]; ok {
			out[h] = col
		}
	}
	return out, nil
}

// --- mock Xûr inventory ---

type mockXur struct {
	hashes map[uint32]struct{}
}

func (m *mockXur) XurItemHashes(_ context.Context) map[uint32]struct{} {
	if m.hashes == nil {
		return map[uint32]struct{}{}
	}
	return m.hashes
}

// --- router setup helper ---

func newTestRouter(h *WishlistHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("membership_id", "test-member-123")
		c.Next()
	})
	r.GET("/api/wishlist", h.GetWishlist)
	r.POST("/api/wishlist", h.AddToWishlist)
	r.PUT("/api/wishlist/:id", h.UpdateWishlistItem)
	r.DELETE("/api/wishlist/:id", h.RemoveFromWishlist)
	r.GET("/api/preferences", h.GetPreferences)
	r.PUT("/api/preferences", h.UpdatePreferences)
	return r
}

// --- tests ---

func TestGetWishlist_ReturnsItems(t *testing.T) {
	store := &mockWishlistStore{
		userID: 42,
		items: []db.WishlistItem{
			{ID: 1, UserID: 42, ItemHash: 1234, Priority: 2, Notes: "nice roll", CreatedAt: time.Now()},
		},
	}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/wishlist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []wishlistResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp))
	}
	if resp[0].ID != "1" {
		t.Errorf("expected id=1, got %s", resp[0].ID)
	}
	if resp[0].Priority != "HIGH" {
		t.Errorf("expected priority=HIGH, got %s", resp[0].Priority)
	}
}

func TestGetWishlist_DegradedMode_Returns503(t *testing.T) {
	h := NewWishlistHandler(nil, nil, nil, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/wishlist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestGetWishlist_UsesJWTMembershipID(t *testing.T) {
	capturedMembershipID := ""
	store := &mockWishlistStore{userID: 7, items: []db.WishlistItem{}}
	// Override GetUserID to capture what was passed
	spy := &membershipIDSpy{inner: store, captured: &capturedMembershipID}
	h := NewWishlistHandler(spy, nil, nil, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/wishlist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedMembershipID != "test-member-123" {
		t.Errorf("expected membership_id=test-member-123, got %q", capturedMembershipID)
	}
}

// membershipIDSpy wraps mockWishlistStore and records the membershipID passed to GetUserID.
type membershipIDSpy struct {
	inner    *mockWishlistStore
	captured *string
}

func (s *membershipIDSpy) GetUserID(ctx context.Context, membershipID string) (int64, error) {
	*s.captured = membershipID
	return s.inner.GetUserID(ctx, membershipID)
}
func (s *membershipIDSpy) List(ctx context.Context, userID int64) ([]db.WishlistItem, error) {
	return s.inner.List(ctx, userID)
}
func (s *membershipIDSpy) Add(ctx context.Context, userID int64, hash uint32, prio int16, notes string) (*db.WishlistItem, error) {
	return s.inner.Add(ctx, userID, hash, prio, notes)
}
func (s *membershipIDSpy) Update(ctx context.Context, userID, id int64, prio *int16, notes *string) (*db.WishlistItem, error) {
	return s.inner.Update(ctx, userID, id, prio, notes)
}
func (s *membershipIDSpy) Delete(ctx context.Context, userID, id int64) (bool, error) {
	return s.inner.Delete(ctx, userID, id)
}

func TestAddToWishlist_Duplicate_Returns409(t *testing.T) {
	store := &mockWishlistStore{
		userID: 42,
		addErr: &pgconn.PgError{Code: "23505"},
	}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	body := `{"itemHash": 1234567}`
	req := httptest.NewRequest(http.MethodPost, "/api/wishlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddToWishlist_NotesTooLong_Returns400(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	longNotes := strings.Repeat("x", 501)
	body, _ := json.Marshal(map[string]any{
		"itemHash": 1234567,
		"notes":    longNotes,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/wishlist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddToWishlist_InvalidPriority_Returns400(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	body := `{"itemHash": 1234567, "priority": "CRITICAL"}`
	req := httptest.NewRequest(http.MethodPost, "/api/wishlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddToWishlist_MissingItemHash_Returns400(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	body := `{"priority": "HIGH"}`
	req := httptest.NewRequest(http.MethodPost, "/api/wishlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddToWishlist_ManifestValidation_UnknownHash_Returns400(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	manifest := &mockManifest{defs: map[uint32]*bungie.InventoryItemDefinition{}} // empty — hash not found
	h := NewWishlistHandler(store, manifest, nil, nil)
	r := newTestRouter(h)

	body := `{"itemHash": 9999999}`
	req := httptest.NewRequest(http.MethodPost, "/api/wishlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddToWishlist_PriorityMapping(t *testing.T) {
	cases := []struct {
		priority string
		expected int16
	}{
		{"LOW", 0},
		{"MEDIUM", 1},
		{"HIGH", 2},
		{"URGENT", 3},
	}

	for _, tc := range cases {
		t.Run(tc.priority, func(t *testing.T) {
			store := &mockWishlistStore{userID: 42}
			h := NewWishlistHandler(store, nil, nil, nil)
			r := newTestRouter(h)

			body := fmt.Sprintf(`{"itemHash": 1234567, "priority": "%s"}`, tc.priority)
			req := httptest.NewRequest(http.MethodPost, "/api/wishlist", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
			}
			var resp wishlistResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if resp.Priority != tc.priority {
				t.Errorf("expected priority=%s, got %s", tc.priority, resp.Priority)
			}
		})
	}
}

func TestAddToWishlist_DefaultPriorityIsMedium(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	body := `{"itemHash": 1234567}`
	req := httptest.NewRequest(http.MethodPost, "/api/wishlist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp wishlistResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Priority != "MEDIUM" {
		t.Errorf("expected default priority=MEDIUM, got %s", resp.Priority)
	}
}

func TestRemoveFromWishlist_NotFound_Returns404(t *testing.T) {
	store := &mockWishlistStore{userID: 42, delFound: false}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/wishlist/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRemoveFromWishlist_Success_Returns204(t *testing.T) {
	store := &mockWishlistStore{userID: 42, delFound: true}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/wishlist/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRemoveFromWishlist_InvalidID_Returns400(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/wishlist/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateWishlistItem_InvalidPriority_Returns400(t *testing.T) {
	store := &mockWishlistStore{
		userID: 42,
		items:  []db.WishlistItem{{ID: 1, UserID: 42, ItemHash: 1234, Priority: 1}},
	}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	body := `{"priority": "LEGENDARY"}`
	req := httptest.NewRequest(http.MethodPut, "/api/wishlist/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateWishlistItem_NotFound_Returns404(t *testing.T) {
	store := &mockWishlistStore{userID: 42, items: []db.WishlistItem{}} // empty = ErrNoRows
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	body := `{"priority": "HIGH"}`
	req := httptest.NewRequest(http.MethodPut, "/api/wishlist/99", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateWishlistItem_NotesTooLong_Returns400(t *testing.T) {
	store := &mockWishlistStore{
		userID: 42,
		items:  []db.WishlistItem{{ID: 1, UserID: 42, ItemHash: 1234, Priority: 1}},
	}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	longNotes := strings.Repeat("y", 501)
	body, _ := json.Marshal(map[string]any{"notes": longNotes})
	req := httptest.NewRequest(http.MethodPut, "/api/wishlist/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateWishlistItem_Success(t *testing.T) {
	store := &mockWishlistStore{
		userID: 42,
		items:  []db.WishlistItem{{ID: 1, UserID: 42, ItemHash: 1234, Priority: 1, Notes: "original", CreatedAt: time.Now()}},
	}
	h := NewWishlistHandler(store, nil, nil, nil)
	r := newTestRouter(h)

	body := `{"priority": "URGENT", "notes": "updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/wishlist/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp wishlistResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Priority != "URGENT" {
		t.Errorf("expected priority=URGENT, got %s", resp.Priority)
	}
	if resp.Notes != "updated" {
		t.Errorf("expected notes=updated, got %s", resp.Notes)
	}
}

func TestGetPreferences_DegradedMode_ReturnsDefaults(t *testing.T) {
	h := NewWishlistHandler(nil, nil, nil, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/preferences", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["cardStyle"] != "framed" {
		t.Errorf("expected cardStyle=framed, got %v", resp["cardStyle"])
	}
	if resp["personalize"] != true {
		t.Errorf("expected personalize=true, got %v", resp["personalize"])
	}
}

func TestUpdatePreferences_InvalidCardStyle_Returns400(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	prefs := &mockPrefsStore{}
	h := NewWishlistHandler(store, nil, prefs, nil)
	r := newTestRouter(h)

	body := `{"cardStyle": "giant"}`
	req := httptest.NewRequest(http.MethodPut, "/api/preferences", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdatePreferences_Success(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	prefs := &mockPrefsStore{}
	h := NewWishlistHandler(store, nil, prefs, nil)
	r := newTestRouter(h)

	body := `{"cardStyle": "compact", "personalize": false}`
	req := httptest.NewRequest(http.MethodPut, "/api/preferences", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["cardStyle"] != "compact" {
		t.Errorf("expected cardStyle=compact, got %v", resp["cardStyle"])
	}
	if resp["personalize"] != false {
		t.Errorf("expected personalize=false, got %v", resp["personalize"])
	}
}

func TestEnrichItems_WithManifest(t *testing.T) {
	store := &mockWishlistStore{
		userID: 42,
		items: []db.WishlistItem{
			{ID: 1, UserID: 42, ItemHash: 5555, Priority: 3, CreatedAt: time.Now()},
		},
	}
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
	h := NewWishlistHandler(store, manifest, nil, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/wishlist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []wishlistResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp))
	}
	if resp[0].Name != "Gjallarhorn" {
		t.Errorf("expected Name=Gjallarhorn, got %s", resp[0].Name)
	}
	if resp[0].Rarity != "Exotic" {
		t.Errorf("expected Rarity=Exotic, got %s", resp[0].Rarity)
	}
	if resp[0].ItemType != "Rocket Launcher" {
		t.Errorf("expected ItemType=Rocket Launcher, got %s", resp[0].ItemType)
	}
	if resp[0].Priority != "URGENT" {
		t.Errorf("expected Priority=URGENT, got %s", resp[0].Priority)
	}
}

// TestEnrichItems_AvailabilityAndSources is the B6 test: items Xûr sells are
// flagged availableNow, and sources come from the collectible's sourceString.
func TestEnrichItems_AvailabilityAndSources(t *testing.T) {
	store := &mockWishlistStore{
		userID: 42,
		items: []db.WishlistItem{
			{ID: 1, UserID: 42, ItemHash: 5555, Priority: 1, CreatedAt: time.Now()},
			{ID: 2, UserID: 42, ItemHash: 6666, Priority: 1, CreatedAt: time.Now()},
		},
	}
	gjally := &bungie.InventoryItemDefinition{Hash: 5555}
	gjally.DisplayProperties.Name = "Gjallarhorn"
	gjally.DisplayProperties.Icon = "/icons/gjally.png"
	other := &bungie.InventoryItemDefinition{Hash: 6666}
	other.DisplayProperties.Name = "Fatebringer"
	manifest := &mockManifest{
		defs: map[uint32]*bungie.InventoryItemDefinition{5555: gjally, 6666: other},
		cols: map[uint32]*bungie.CollectibleDefinition{
			6666: {ItemHash: 6666, SourceString: "Vault of Glass raid"},
		},
	}
	xur := &mockXur{hashes: map[uint32]struct{}{5555: {}}}
	h := NewWishlistHandler(store, manifest, nil, xur)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/wishlist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []wishlistResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp))
	}

	byHash := map[uint32]wishlistResponse{}
	for _, it := range resp {
		byHash[it.ItemHash] = it
	}

	atXur := byHash[5555]
	if !atXur.AvailableNow || atXur.AvailableFrom != "Xûr" {
		t.Errorf("Xûr-stocked item = availableNow %v from %q; want true, Xûr", atXur.AvailableNow, atXur.AvailableFrom)
	}
	if atXur.Icon != "/icons/gjally.png" {
		t.Errorf("icon = %q", atXur.Icon)
	}

	notAtXur := byHash[6666]
	if notAtXur.AvailableNow || notAtXur.AvailableFrom != "" {
		t.Errorf("non-Xûr item flagged available: %+v", notAtXur)
	}
	if len(notAtXur.Sources) != 1 || notAtXur.Sources[0] != "Vault of Glass raid" {
		t.Errorf("sources = %v, want [Vault of Glass raid]", notAtXur.Sources)
	}
}
