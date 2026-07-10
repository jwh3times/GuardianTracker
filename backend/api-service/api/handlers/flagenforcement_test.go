package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
)

// enforceRouter mounts RequireFlag (backed by a real FlagResolver over a fake
// store) after injecting a fixed role, then a 200 terminal handler.
func enforceRouter(role int, flags []db.FeatureFlag, key string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	resolver := NewFlagResolver(&fakeFlagLister{flags: flags}, cache.NewMemoryCache(time.Minute, time.Minute))
	authz := auth.NewAuthz(true)
	r := gin.New()
	r.GET("/x",
		func(c *gin.Context) { c.Set("role", role); c.Next() },
		authz.RequireFlag(resolver, key),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) },
	)
	return r
}

func TestRequireFlag_EndToEndWithRealResolver(t *testing.T) {
	flags := []db.FeatureFlag{
		{Key: FlagCatalystsCrafting, Enabled: true, MinTier: 0},
		{Key: FlagGlobalSearch, Enabled: false, MinTier: 0},
	}
	cases := []struct {
		name string
		role int
		key  string
		want int
	}{
		{"accessible", auth.RoleStandard, FlagCatalystsCrafting, http.StatusOK},
		{"disabled-404", auth.RoleAdmin, FlagGlobalSearch, http.StatusNotFound},
		{"unknown-key-open", auth.RoleStandard, "not-wired", http.StatusOK},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		enforceRouter(c.role, flags, c.key).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		if w.Code != c.want {
			t.Errorf("%s: got %d, want %d", c.name, w.Code, c.want)
		}
	}
}

func TestRequireFlag_UnderTier403(t *testing.T) {
	flags := []db.FeatureFlag{{Key: FlagTriumphsSeals, Enabled: true, MinTier: 2}}
	w := httptest.NewRecorder()
	enforceRouter(auth.RoleStandard, flags, FlagTriumphsSeals).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusForbidden {
		t.Errorf("under-tier: got %d, want 403", w.Code)
	}
}
