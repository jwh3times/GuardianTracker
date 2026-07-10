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
	cats []manifest.WeaponCatalyst
	err  error
}

func (f fakePerks) GetWeaponPerks(uint32) ([]manifest.PerkColumn, error) { return f.cols, f.err }
func (f fakePerks) GetItem(uint32) (*manifest.ItemView, error)           { return nil, f.err }
func (f fakePerks) GetCatalysts(uint32) ([]manifest.WeaponCatalyst, error) {
	return f.cats, f.err
}

func itemsRouter(p itemProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewItemsHandler(p)
	r.GET("/api/items/:itemHash/perks", h.GetPerks)
	return r
}

func TestGetPerks_OK(t *testing.T) {
	r := itemsRouter(fakePerks{
		cols: []manifest.PerkColumn{
			{Role: "barrel", Label: "Barrel", Perks: []string{"Full Bore", "Arrowhead Brake"}},
		},
		cats: []manifest.WeaponCatalyst{{Name: "Loose Change", Description: "text"}},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/items/1000/perks", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body struct {
		ItemHash    string                    `json:"itemHash"`
		PerkColumns []manifest.PerkColumn     `json:"perkColumns"`
		Catalysts   []manifest.WeaponCatalyst `json:"catalysts"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ItemHash != "1000" || len(body.PerkColumns) != 1 || body.PerkColumns[0].Label != "Barrel" {
		t.Errorf("body = %+v", body)
	}
	if len(body.Catalysts) != 1 || body.Catalysts[0].Name != "Loose Change" {
		t.Errorf("catalysts = %+v", body.Catalysts)
	}
}

func TestGetPerks_NonWeaponEmptyArray(t *testing.T) {
	r := itemsRouter(fakePerks{cols: nil, cats: nil})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/items/3000/perks", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// nil columns/catalysts must serialize as [] not null.
	if got := w.Body.String(); !jsonHasEmptyPerkColumns(got) {
		t.Errorf("body = %s, want perkColumns: []", got)
	}
	if got := w.Body.String(); !jsonHasEmptyCatalysts(got) {
		t.Errorf("body = %s, want catalysts: []", got)
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

func jsonHasEmptyCatalysts(s string) bool {
	var body struct {
		Catalysts []manifest.WeaponCatalyst `json:"catalysts"`
	}
	if err := json.Unmarshal([]byte(s), &body); err != nil {
		return false
	}
	return body.Catalysts != nil && len(body.Catalysts) == 0
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

// catalystsOnlyErrProvider succeeds on perks but fails (manifest not ready) on
// catalysts — proves GetPerks propagates a catalyst-source error too, not just
// a perks-source one.
type catalystsOnlyErrProvider struct{}

func (catalystsOnlyErrProvider) GetWeaponPerks(uint32) ([]manifest.PerkColumn, error) {
	return nil, nil
}
func (catalystsOnlyErrProvider) GetItem(uint32) (*manifest.ItemView, error) { return nil, nil }
func (catalystsOnlyErrProvider) GetCatalysts(uint32) ([]manifest.WeaponCatalyst, error) {
	return nil, manifest.ErrNotReady
}

func TestGetPerks_CatalystsManifestNotReady(t *testing.T) {
	r := itemsRouter(catalystsOnlyErrProvider{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/items/1000/perks", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

// fakeItemsProvider implements itemProvider for handler tests (perks, item view, catalysts).
type fakeItemsProvider struct {
	cols []manifest.PerkColumn
	cats []manifest.WeaponCatalyst
	view *manifest.ItemView
	err  error
}

func (f fakeItemsProvider) GetWeaponPerks(uint32) ([]manifest.PerkColumn, error) {
	return f.cols, f.err
}
func (f fakeItemsProvider) GetItem(uint32) (*manifest.ItemView, error) { return f.view, f.err }
func (f fakeItemsProvider) GetCatalysts(uint32) ([]manifest.WeaponCatalyst, error) {
	return f.cats, f.err
}

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
