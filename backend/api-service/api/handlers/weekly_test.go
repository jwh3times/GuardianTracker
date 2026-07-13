package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"guardian-tracker/api-service/services/weekly"

	"github.com/gin-gonic/gin"
)

type fakeWeeklyService struct {
	characterID string
}

func (f *fakeWeeklyService) GetWeekly(_ context.Context, _ int, _, _, characterID string) (*weekly.Weekly, error) {
	f.characterID = characterID
	return &weekly.Weekly{}, nil
}

type fakeWeeklyTokenStore struct{}

func (fakeWeeklyTokenStore) GetValidToken(string) (string, error) { return "bungie-token", nil }

func TestWeeklyHandlerPassesOptionalCharacterID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeWeeklyService{}
	h := NewWeeklyHandler(svc, fakeWeeklyTokenStore{})
	router := gin.New()
	router.GET("/api/weekly/recommendations", func(c *gin.Context) {
		c.Set("membership_id", "member-1")
		c.Set("membership_type", 3)
		h.GetWeekly(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/weekly/recommendations?characterId=char-warlock", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if svc.characterID != "char-warlock" {
		t.Errorf("characterID = %q, want char-warlock", svc.characterID)
	}
}
