package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAccountSetRole_PassesAuditMetadataToAtomicMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeRoleStore{}
	h := NewUserHandler(store, nil, nil)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("membership_id", "mid-1")
		c.Set("role", 0) // standard (not admin) so opt-in is allowed
		c.Next()
	})
	r.PUT("/api/account/role", h.SetRole)

	req := httptest.NewRequest(http.MethodPut, "/api/account/role",
		strings.NewReader(`{"role":"beta"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "fixture-agent")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if store.lastMembership != "mid-1" || store.lastIP != "127.0.0.1" || store.lastUserAgent != "fixture-agent" {
		t.Errorf("atomic role mutation metadata = %+v", store)
	}
}
