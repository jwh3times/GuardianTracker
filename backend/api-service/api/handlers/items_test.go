package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"guardian-tracker/api-service/services/manifest"
)

type fakePerks struct {
	cols []manifest.PerkColumn
	err  error
}

func (f fakePerks) GetWeaponPerks(uint32) ([]manifest.PerkColumn, error) { return f.cols, f.err }

func itemsRouter(p weaponPerksProvider) *gin.Engine {
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
