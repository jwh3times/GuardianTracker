package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/config"

	"github.com/gin-gonic/gin"
)

const testJWTSecret = "test-jwt-secret-at-least-32-chars-long!!"

func newAuthHandler(t *testing.T) (*AuthHandler, *auth.JWT) {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       testJWTSecret,
		BungieClientID:  "client-123",
		AuthRedirectURI: "http://localhost:3000/auth/callback",
	}
	jwt := auth.NewJWT(cfg.JWTSecret, 24, 30)
	h := NewAuthHandler(jwt, newTokenStore(t), cfg, nil, nil, nil)
	return h, jwt
}

func TestGetBungieAuthURL(t *testing.T) {
	h, _ := newAuthHandler(t)
	r := gin.New()
	r.GET("/api/auth/bungie", h.GetBungieAuthURL)

	w := do(r, http.MethodGet, "/api/auth/bungie")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "authUrl") || !strings.Contains(body, "client-123") {
		t.Errorf("body missing auth url: %s", body)
	}
	if !strings.Contains(body, "state") {
		t.Errorf("body missing state: %s", body)
	}
}

func TestValidateAndProfileHandlers(t *testing.T) {
	h, _ := newAuthHandler(t)
	r := gin.New()
	withCtx := func(next gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("membership_id", testUserID)
			c.Set("display_name", "TestGuardian")
			c.Set("membership_type", 3)
			c.Set("platform", "Steam")
			next(c)
		}
	}
	r.GET("/validate", withCtx(h.ValidateToken))
	r.GET("/profile", withCtx(h.GetProfile))

	for _, path := range []string{"/validate", "/profile"} {
		w := do(r, http.MethodGet, path)
		if w.Code != http.StatusOK {
			t.Errorf("%s status = %d", path, w.Code)
		}
		if !strings.Contains(w.Body.String(), "TestGuardian") {
			t.Errorf("%s body missing user: %s", path, w.Body.String())
		}
	}
}

func TestLogout(t *testing.T) {
	h, _ := newAuthHandler(t)
	r := gin.New()
	r.POST("/logout", func(c *gin.Context) {
		c.Set("membership_id", testUserID)
		h.Logout(c)
	})
	if w := do(r, http.MethodPost, "/logout"); w.Code != http.StatusOK {
		t.Errorf("logout = %d, want 200", w.Code)
	}
}

func TestRefreshToken_Success(t *testing.T) {
	h, jwt := newAuthHandler(t)
	profile := &auth.BungieUserProfile{MembershipID: testUserID, DisplayName: "TestGuardian", MembershipType: 3}
	refresh, err := jwt.GenerateRefreshToken(profile, 1)
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)
	body := `{"refreshToken":"` + refresh + `"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("refresh = %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "token") {
		t.Errorf("missing new tokens: %s", w.Body.String())
	}
}

func TestRefreshToken_RejectsAccessTokenAndGarbage(t *testing.T) {
	h, jwt := newAuthHandler(t)
	profile := &auth.BungieUserProfile{MembershipID: testUserID, MembershipType: 3}
	// An access token must be rejected by the refresh endpoint (token-type claim).
	access, _ := jwt.GenerateAccessToken(profile, 1)

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	post := func(jsonBody string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := post(`{"refreshToken":"` + access + `"}`); code != http.StatusUnauthorized {
		t.Errorf("access-as-refresh = %d, want 401", code)
	}
	if code := post(`{"refreshToken":"not-a-jwt"}`); code != http.StatusUnauthorized {
		t.Errorf("garbage token = %d, want 401", code)
	}
	if code := post(`{}`); code != http.StatusBadRequest {
		t.Errorf("missing field = %d, want 400", code)
	}
}

func TestBungieCallback_ValidationBranches(t *testing.T) {
	h, _ := newAuthHandler(t)
	r := gin.New()
	r.POST("/callback", h.BungieCallback)

	post := func(form url.Values) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.ServeHTTP(w, req)
		return w.Code
	}

	// Empty code → 400.
	if code := post(url.Values{"state": {"x"}}); code != http.StatusBadRequest {
		t.Errorf("empty code = %d, want 400", code)
	}
	// Valid-length code but a forged/invalid state → 400.
	if code := post(url.Values{"code": {"abc123"}, "state": {"forged-state"}}); code != http.StatusBadRequest {
		t.Errorf("bad state = %d, want 400", code)
	}
}
