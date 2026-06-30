package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"guardian-tracker/api-service/services/manifest"
)

type fakePerks struct {
	cols []manifest.PerkColumn
	err  error
}

func (f fakePerks) GetWeaponPerks(uint32) ([]manifest.PerkColumn, error) { return f.cols, f.err }
func (f fakePerks) GetItem(uint32) (*manifest.ItemView, error)           { return nil, f.err }

func itemsRouter(p itemProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewItemsHandler(p)
	r.GET("/api/items/:itemHash/perks", h.GetPerks)
	return r
}

func TestGetPerks_OK(t *testing.T) {
	r := itemsRouter(fakePerks{cols: []manifest.PerkColumn{
		{Role: "barrel", Label: "Barrel", Perks: []string{"Full Bore", "Arrowhead Brake"}},
	}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/items/1000/perks", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body struct {
		ItemHash    string                `json:"itemHash"`
		PerkColumns []manifest.PerkColumn `json:"perkColumns"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ItemHash != "1000" || len(body.PerkColumns) != 1 || body.PerkColumns[0].Label != "Barrel" {
		t.Errorf("body = %+v", body)
	}
}

func TestGetPerks_NonWeaponEmptyArray(t *testing.T) {
	r := itemsRouter(fakePerks{cols: nil})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/items/3000/perks", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// nil columns must serialize as [] not null.
	if got := w.Body.String(); !jsonHasEmptyPerkColumns(got) {
		t.Errorf("body = %s, want perkColumns: []", got)
	}
}

func jsonHasEmptyPerkColumns(s string) bool {
	var body struct {
		PerkColumns []manifest.PerkColumn `json:"perkColumns"`
	}
	if err := json.Unmarshal([]byte(s), &body); err != nil {
		return false
	}
	return body.PerkColumns != nil && len(body.PerkColumns) == 0
}

func TestGetPerks_BadHash(t *testing.T) {
	r := itemsRouter(fakePerks{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/items/notanumber/perks", nil))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetPerks_ManifestNotReady(t *testing.T) {
	r := itemsRouter(fakePerks{err: manifest.ErrNotReady})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/items/1000/perks", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

// fakeItemsProvider implements itemProvider for handler tests (both perks and item view).
type fakeItemsProvider struct {
	cols []manifest.PerkColumn
	view *manifest.ItemView
	err  error
}

func (f fakeItemsProvider) GetWeaponPerks(uint32) ([]manifest.PerkColumn, error) {
	return f.cols, f.err
}
func (f fakeItemsProvider) GetItem(uint32) (*manifest.ItemView, error) { return f.view, f.err }

// doGet builds an httptest.ResponseRecorder and invokes handler with :itemHash = hashParam.
func doGet(t *testing.T, handler gin.HandlerFunc, _ string, hashParam string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "itemHash", Value: hashParam}}
	handler(c)
	return w
}

func TestItemsHandler_GetItem(t *testing.T) {
	// 200 OK
	h := NewItemsHandler(&fakeItemsProvider{view: &manifest.ItemView{ItemHash: "100", Name: "Fatebringer"}})
	w := doGet(t, h.GetItem, "/api/items/100", "100")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Fatebringer") {
		t.Errorf("body = %s", w.Body.String())
	}

	// 400 bad hash
	if w := doGet(t, h.GetItem, "/api/items/abc", "abc"); w.Code != http.StatusBadRequest {
		t.Errorf("bad hash status = %d, want 400", w.Code)
	}

	// 404 unknown
	h404 := NewItemsHandler(&fakeItemsProvider{view: nil})
	if w := doGet(t, h404.GetItem, "/api/items/1", "1"); w.Code != http.StatusNotFound {
		t.Errorf("unknown status = %d, want 404", w.Code)
	}

	// 503 warming
	hWarm := NewItemsHandler(&fakeItemsProvider{err: manifest.ErrNotReady})
	if w := doGet(t, hWarm.GetItem, "/api/items/1", "1"); w.Code != http.StatusServiceUnavailable {
		t.Errorf("warming status = %d, want 503", w.Code)
	}
}
