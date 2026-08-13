package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/sources"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// --- mock wishlist store ---

type mockWishlistStore struct {
	userID       int64
	items        []db.WishlistItem
	addErr       error
	delFound     bool
	bulkAffected *int64
	lastBulkIDs  []int64
	getUserIDErr error
	bulkErr      error
}

func (m *mockWishlistStore) GetUserID(_ context.Context, _ string) (int64, error) {
	if m.getUserIDErr != nil {
		return 0, m.getUserIDErr
	}
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

func (m *mockWishlistStore) BulkDelete(_ context.Context, _ int64, ids []int64) (int64, error) {
	m.lastBulkIDs = ids
	if m.bulkErr != nil {
		return 0, m.bulkErr
	}
	if m.bulkAffected != nil {
		return *m.bulkAffected, nil
	}
	return int64(len(ids)), nil
}

func (m *mockWishlistStore) BulkSetPriority(_ context.Context, _ int64, ids []int64, _ int16) (int64, error) {
	m.lastBulkIDs = ids
	if m.bulkErr != nil {
		return 0, m.bulkErr
	}
	if m.bulkAffected != nil {
		return *m.bulkAffected, nil
	}
	return int64(len(ids)), nil
}

// --- mock prefs store ---

type mockPrefsStore struct {
	prefs                  *db.UserPreferences
	lastCompleteOnboarding bool
}

func (m *mockPrefsStore) Get(_ context.Context, userID int64) (*db.UserPreferences, error) {
	if m.prefs != nil {
		return m.prefs, nil
	}
	return &db.UserPreferences{UserID: userID, CardStyle: "framed", Personalize: true, UpdatedAt: time.Now()}, nil
}

func (m *mockPrefsStore) Upsert(_ context.Context, userID int64, cardStyle string, personalize, completeOnboarding bool) (*db.UserPreferences, error) {
	m.lastCompleteOnboarding = completeOnboarding
	onboardedAt := (*time.Time)(nil)
	if completeOnboarding {
		now := time.Now().UTC()
		onboardedAt = &now
	}
	return &db.UserPreferences{UserID: userID, CardStyle: cardStyle, Personalize: personalize, OnboardedAt: onboardedAt, UpdatedAt: time.Now()}, nil
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	degraded := db.NewStores(nil)
	h := NewWishlistHandler(degraded.Wishlist, nil, degraded.Prefs, nil, nil)
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
	h := NewWishlistHandler(spy, nil, nil, nil, nil)
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
func (s *membershipIDSpy) BulkDelete(ctx context.Context, userID int64, ids []int64) (int64, error) {
	return s.inner.BulkDelete(ctx, userID, ids)
}
func (s *membershipIDSpy) BulkSetPriority(ctx context.Context, userID int64, ids []int64, prio int16) (int64, error) {
	return s.inner.BulkSetPriority(ctx, userID, ids, prio)
}

func TestAddToWishlist_Duplicate_Returns409(t *testing.T) {
	store := &mockWishlistStore{
		userID: 42,
		addErr: &pgconn.PgError{Code: "23505"},
	}
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, manifest, nil, nil, nil)
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
			h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	h := NewWishlistHandler(store, nil, nil, nil, nil)
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
	degraded := db.NewStores(nil)
	h := NewWishlistHandler(degraded.Wishlist, nil, degraded.Prefs, nil, nil)
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
	if _, ok := resp["onboardedAt"]; !ok || resp["onboardedAt"] != nil {
		t.Errorf("expected onboardedAt=null, got %v", resp["onboardedAt"])
	}
}

func TestGetPreferences_ReturnsOnboardedAt(t *testing.T) {
	stamp := time.Date(2026, time.July, 12, 15, 30, 0, 0, time.UTC)
	store := &mockWishlistStore{userID: 42}
	prefs := &mockPrefsStore{prefs: &db.UserPreferences{
		UserID: 42, CardStyle: "framed", Personalize: true, OnboardedAt: &stamp,
	}}
	r := newTestRouter(NewWishlistHandler(store, nil, prefs, nil, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/preferences", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["onboardedAt"] != "2026-07-12T15:30:00Z" {
		t.Errorf("onboardedAt = %v, want RFC3339 server timestamp", resp["onboardedAt"])
	}
}

func TestUpdatePreferences_InvalidCardStyle_Returns400(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	prefs := &mockPrefsStore{}
	h := NewWishlistHandler(store, nil, prefs, nil, nil)
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
	h := NewWishlistHandler(store, nil, prefs, nil, nil)
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

func TestUpdatePreferences_CompletesOnboarding(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	prefs := &mockPrefsStore{}
	r := newTestRouter(NewWishlistHandler(store, nil, prefs, nil, nil))

	req := httptest.NewRequest(http.MethodPut, "/api/preferences", strings.NewReader(`{"onboardingComplete": true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !prefs.lastCompleteOnboarding {
		t.Fatal("completion flag was not passed to the preferences store")
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["onboardedAt"] == nil {
		t.Fatal("expected a server-generated onboardedAt timestamp")
	}
}

func TestUpdatePreferences_RejectsOnboardingReset(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	prefs := &mockPrefsStore{}
	r := newTestRouter(NewWishlistHandler(store, nil, prefs, nil, nil))

	req := httptest.NewRequest(http.MethodPut, "/api/preferences", strings.NewReader(`{"onboardingComplete": false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
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
	h := NewWishlistHandler(store, manifest, nil, nil, nil)
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

// TestEnrichItems_AvailabilityAndAcquisitionSources: live availability stays
// separate from the deterministic union of collectible provenance.
func TestEnrichItems_AvailabilityAndAcquisitionSources(t *testing.T) {
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
	h := NewWishlistHandler(store, manifest, nil, live, &mockTokens{token: "tok"})
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
	store := &mockWishlistStore{userID: 42, items: []db.WishlistItem{{ID: 1, UserID: 42, ItemHash: 5555, Priority: 1, CreatedAt: time.Now()}}}
	live := &mockLiveVendors{hashes: map[uint32]string{5555: "Xûr"}}
	h := NewWishlistHandler(store, nil, nil, live, &mockTokens{err: fmt.Errorf("no token")})
	r := newTestRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/api/wishlist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("token error should not fail request; got %d", w.Code)
	}
}

// --- bulk update tests ---

func bulkReq(body string) (*http.Request, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodPost, "/api/wishlist/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req, httptest.NewRecorder()
}

func TestBulkUpdate_Delete_PartialSuccess(t *testing.T) {
	two := int64(2)
	store := &mockWishlistStore{userID: 42, bulkAffected: &two} // only 2 of 3 owned
	h := NewWishlistHandler(store, nil, nil, nil, nil)
	r := newTestRouter(h)
	req, w := bulkReq(`{"action":"delete","ids":[1,2,999]}`)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct{ Updated, Skipped int }
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Updated != 2 || resp.Skipped != 1 {
		t.Errorf("got updated=%d skipped=%d, want 2/1", resp.Updated, resp.Skipped)
	}
}

func TestBulkUpdate_SetPriority(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	h := NewWishlistHandler(store, nil, nil, nil, nil)
	r := newTestRouter(h)
	req, w := bulkReq(`{"action":"set_priority","ids":[1,2],"priority":"HIGH"}`)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct{ Updated, Skipped int }
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Updated != 2 || resp.Skipped != 0 {
		t.Errorf("got updated=%d skipped=%d, want 2/0", resp.Updated, resp.Skipped)
	}
}

func TestBulkUpdate_Validation(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	h := NewWishlistHandler(store, nil, nil, nil, nil)
	r := newTestRouter(h)
	cases := []string{
		`{"action":"nope","ids":[1]}`,                               // bad action
		`{"action":"delete","ids":[]}`,                              // empty ids
		`{"action":"set_priority","ids":[1]}`,                       // missing priority
		`{"action":"set_priority","ids":[1],"priority":"CRITICAL"}`, // bad priority
	}
	for _, body := range cases {
		req, w := bulkReq(body)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %s: got %d, want 400", body, w.Code)
		}
	}
	// Over-cap ids (101) → 400
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = strconv.Itoa(i + 1)
	}
	req, w := bulkReq(`{"action":"delete","ids":[` + strings.Join(ids, ",") + `]}`)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("over-cap: got %d, want 400", w.Code)
	}
}

func TestBulkUpdate_DegradedMode_Returns503(t *testing.T) {
	degraded := db.NewStores(nil)
	h := NewWishlistHandler(degraded.Wishlist, nil, degraded.Prefs, nil, nil)
	r := newTestRouter(h)
	req, w := bulkReq(`{"action":"delete","ids":[1]}`)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestBulkUpdate_DedupesIDs(t *testing.T) {
	store := &mockWishlistStore{userID: 42}
	h := NewWishlistHandler(store, nil, nil, nil, nil)
	r := newTestRouter(h)
	req, w := bulkReq(`{"action":"delete","ids":[1,1,2]}`)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	want := []int64{1, 2}
	if len(store.lastBulkIDs) != len(want) || store.lastBulkIDs[0] != want[0] || store.lastBulkIDs[1] != want[1] {
		t.Fatalf("store.lastBulkIDs = %v, want %v (duplicates must be collapsed before reaching the store)", store.lastBulkIDs, want)
	}
	var resp struct{ Updated, Skipped int }
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Updated != 2 || resp.Skipped != 0 {
		t.Errorf("got updated=%d skipped=%d, want 2/0 (skipped must be computed from deduped count, not raw count)", resp.Updated, resp.Skipped)
	}
}

func TestBulkUpdate_GetUserIDError_Returns500(t *testing.T) {
	store := &mockWishlistStore{userID: 42, getUserIDErr: errors.New("db down")}
	h := NewWishlistHandler(store, nil, nil, nil, nil)
	r := newTestRouter(h)
	req, w := bulkReq(`{"action":"delete","ids":[1]}`)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBulkUpdate_StoreError_Returns500(t *testing.T) {
	store := &mockWishlistStore{userID: 42, bulkErr: errors.New("db down")}
	h := NewWishlistHandler(store, nil, nil, nil, nil)
	r := newTestRouter(h)
	req, w := bulkReq(`{"action":"delete","ids":[1]}`)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// TestBulkUpdate_ValidatesBeforeUserLookup proves action/priority validation runs
// before GetUserID: a bad action still returns 400 even when GetUserID would error
// (if validation ran after the lookup, this would be a 500).
func TestBulkUpdate_ValidatesBeforeUserLookup(t *testing.T) {
	store := &mockWishlistStore{userID: 42, getUserIDErr: errors.New("db down")}
	h := NewWishlistHandler(store, nil, nil, nil, nil)
	r := newTestRouter(h)
	req, w := bulkReq(`{"action":"nope","ids":[1]}`)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (validation before user lookup), got %d: %s", w.Code, w.Body.String())
	}
}
