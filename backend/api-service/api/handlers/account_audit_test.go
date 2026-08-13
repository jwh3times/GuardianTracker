package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAccountSetRole_AuditsOptIn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spy := &spyAudit{}
	h := NewUserHandler(&fakeRoleStore{}, nil, nil, spy)

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
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if got := spy.types(); len(got) != 1 || got[0] != "role.optin" {
		t.Errorf("audit events = %v, want [role.optin]", got)
	}
}
