package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// roleRouter builds a router that injects a fixed role, then applies the guard.
func roleRouter(role int, guard gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) { c.Set("role", role); c.Next() }, guard, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func statusFor(role int, guard gin.HandlerFunc) int {
	w := httptest.NewRecorder()
	roleRouter(role, guard).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	return w.Code
}

func TestRequireAdmin_DegradedAlways503(t *testing.T) {
	authz := NewAuthz(false) // no DB
	for _, role := range []int{RoleStandard, RoleAlpha, RoleAdmin} {
		if got := statusFor(role, authz.RequireAdmin()); got != http.StatusServiceUnavailable {
			t.Errorf("degraded RequireAdmin role=%d: got %d, want 503", role, got)
		}
	}
}

func TestRequireAdmin_Enabled(t *testing.T) {
	authz := NewAuthz(true)
	cases := []struct {
		role int
		want int
	}{
		{RoleStandard, http.StatusForbidden},
		{RoleBeta, http.StatusForbidden},
		{RoleAlpha, http.StatusForbidden},
		{RoleAdmin, http.StatusOK},
	}
	for _, c := range cases {
		if got := statusFor(c.role, authz.RequireAdmin()); got != c.want {
			t.Errorf("RequireAdmin role=%d: got %d, want %d", c.role, got, c.want)
		}
	}
}

func TestRequireTier_Matrix(t *testing.T) {
	authz := NewAuthz(true)
	// For each minTier, a role below is 403 and a role at/above is 200.
	for _, min := range []int{RoleStandard, RoleBeta, RoleAlpha} {
		for _, role := range []int{RoleStandard, RoleBeta, RoleAlpha, RoleAdmin} {
			want := http.StatusOK
			if role < min {
				want = http.StatusForbidden
			}
			if got := statusFor(role, authz.RequireTier(min)); got != want {
				t.Errorf("RequireTier(min=%d) role=%d: got %d, want %d", min, role, got, want)
			}
		}
	}
}

func TestRoleNameParseRoundTrip(t *testing.T) {
	for _, name := range []string{"standard", "beta", "alpha", "admin"} {
		r, ok := ParseRole(name)
		if !ok {
			t.Fatalf("ParseRole(%q) not ok", name)
		}
		if RoleName(r) != name {
			t.Errorf("RoleName(%d) = %q, want %q", r, RoleName(r), name)
		}
	}
	if _, ok := ParseRole("superuser"); ok {
		t.Error("ParseRole(superuser) should be !ok")
	}
	if RoleName(99) != "standard" {
		t.Errorf("RoleName(99) = %q, want standard fallback", RoleName(99))
	}
}
