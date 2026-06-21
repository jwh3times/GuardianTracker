package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/config"
	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
)

// fakeAudit captures emitted audit events so tests can assert on event type and
// details. Satisfies the AuditLogger interface.
type fakeAudit struct{ events []db.AuditEvent }

func (f *fakeAudit) Log(_ context.Context, ev db.AuditEvent) error {
	f.events = append(f.events, ev)
	return nil
}

// fakeUserStore is an in-memory UserStore for session-rotation handler tests. It
// mirrors RotateSession's lock-CAS-with-reuse-delete: a presented jti must match the
// session's current jti, otherwise the session is revoked (deleted).
type fakeUserStore struct {
	sessions map[string]string          // sid -> current refresh jti
	rotateFn func() (bool, bool, error) // optional override for RotateSession
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{sessions: map[string]string{}}
}

func (f *fakeUserStore) Upsert(_ context.Context, _ string, _ int16, _ string, _ bool) (int64, int, int16, error) {
	return 1, 1, 0, nil
}
func (f *fakeUserStore) BumpTokenVersion(_ context.Context, _ string) error { return nil }
func (f *fakeUserStore) CreateSession(_ context.Context, id, _, jti, _ string, _ time.Time) error {
	f.sessions[id] = jti
	return nil
}
func (f *fakeUserStore) RotateSession(_ context.Context, id, _, oldJTI, newJTI string, _ time.Time) (bool, bool, error) {
	if f.rotateFn != nil {
		return f.rotateFn()
	}
	cur, ok := f.sessions[id]
	if !ok {
		return false, false, nil // unknown / expired session
	}
	if cur != oldJTI {
		delete(f.sessions, id) // replay of a rotated token — revoke the family
		return false, true, nil
	}
	f.sessions[id] = newJTI
	return true, false, nil
}
func (f *fakeUserStore) DeleteSession(_ context.Context, id string) error {
	delete(f.sessions, id)
	return nil
}
func (f *fakeUserStore) DeleteUserSessions(_ context.Context, _ string) error {
	f.sessions = map[string]string{}
	return nil
}

func newAuthHandlerWithStore(t *testing.T, store UserStore) (*AuthHandler, *auth.JWT) {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       testJWTSecret,
		BungieClientID:  "client-123",
		AuthRedirectURI: "http://localhost:3000/auth/callback",
	}
	jwt := auth.NewJWT(cfg.JWTSecret, 24, 30)
	h := NewAuthHandler(jwt, newTokenStore(t), cfg, store, nil, nil, nil)
	return h, jwt
}

func newAuthHandlerWithStoreAndAudit(t *testing.T, store UserStore, audit AuditLogger) (*AuthHandler, *auth.JWT) {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       testJWTSecret,
		BungieClientID:  "client-123",
		AuthRedirectURI: "http://localhost:3000/auth/callback",
	}
	jwt := auth.NewJWT(cfg.JWTSecret, 24, 30)
	h := NewAuthHandler(jwt, newTokenStore(t), cfg, store, nil, nil, audit)
	return h, jwt
}

const testJWTSecret = "test-jwt-secret-at-least-32-chars-long!!"

func newAuthHandler(t *testing.T) (*AuthHandler, *auth.JWT) {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       testJWTSecret,
		BungieClientID:  "client-123",
		AuthRedirectURI: "http://localhost:3000/auth/callback",
	}
	jwt := auth.NewJWT(cfg.JWTSecret, 24, 30)
	h := NewAuthHandler(jwt, newTokenStore(t), cfg, nil, nil, nil, nil)
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
	refresh, _, err := jwt.GenerateRefreshToken(profile, 1, "")
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
	access, _ := jwt.GenerateAccessToken(profile, 1, "")

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

// TestRefreshToken_ReuseRejected verifies per-session rotation: the first use of a
// refresh token rotates successfully, and replaying that now-superseded token is
// rejected with 401 — the session is revoked (limitation #3 fix).
func TestRefreshToken_ReuseRejected(t *testing.T) {
	store := newFakeUserStore()
	h, jwt := newAuthHandlerWithStore(t, store)
	profile := &auth.BungieUserProfile{MembershipID: testUserID, DisplayName: "TestGuardian", MembershipType: 3}
	const sid = "sess-1"
	refresh, jti, err := jwt.GenerateRefreshToken(profile, 1, sid)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate login having created this session.
	if err := store.CreateSession(context.Background(), sid, testUserID, jti, "", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)
	post := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{"refreshToken":"`+refresh+`"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	// First use rotates the session successfully.
	if w := post(); w.Code != http.StatusOK {
		t.Fatalf("first refresh = %d, want 200: %s", w.Code, w.Body.String())
	}
	// The original token is now superseded — replaying it revokes the session (401).
	if w := post(); w.Code != http.StatusUnauthorized {
		t.Fatalf("reused refresh = %d, want 401", w.Code)
	}
	if _, alive := store.sessions[sid]; alive {
		t.Error("reuse should have revoked (deleted) the session")
	}
}

// TestRefreshToken_ReuseRejectedOnCommitError verifies reuse wins over a DB error:
// when RotateSession reports reused=true alongside an error (e.g. the revoking commit
// failed), the refresh is still rejected rather than fail-open.
func TestRefreshToken_ReuseRejectedOnCommitError(t *testing.T) {
	store := newFakeUserStore()
	store.rotateFn = func() (bool, bool, error) { return false, true, errors.New("commit failed") }
	h, jwt := newAuthHandlerWithStore(t, store)
	profile := &auth.BungieUserProfile{MembershipID: testUserID, MembershipType: 3}
	refresh, _, err := jwt.GenerateRefreshToken(profile, 1, "sess-x")
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{"refreshToken":"`+refresh+`"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("reuse+commit-error refresh = %d, want 401", w.Code)
	}
}

// TestRefreshToken_LegacyTokenAdopted verifies a pre-session refresh token (no sid)
// is accepted once and adopted into a fresh session, so the rollout logs nobody out.
func TestRefreshToken_LegacyTokenAdopted(t *testing.T) {
	store := newFakeUserStore()
	h, jwt := newAuthHandlerWithStore(t, store)
	profile := &auth.BungieUserProfile{MembershipID: testUserID, MembershipType: 3}
	refresh, _, err := jwt.GenerateRefreshToken(profile, 1, "") // no sid
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{"refreshToken":"`+refresh+`"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("legacy refresh = %d, want 200: %s", w.Code, w.Body.String())
	}
	if len(store.sessions) != 1 {
		t.Errorf("expected one adopted session, got %d", len(store.sessions))
	}
}

// TestLogout_DeletesCurrentSessionOnly pins the multi-device contract: this-device
// logout ends the caller's session and leaves other devices' sessions intact.
func TestLogout_DeletesCurrentSessionOnly(t *testing.T) {
	store := newFakeUserStore()
	store.sessions["this-device"] = "jti-a"
	store.sessions["other-device"] = "jti-b"
	h, _ := newAuthHandlerWithStore(t, store)

	r := gin.New()
	r.POST("/logout", func(c *gin.Context) {
		c.Set("membership_id", testUserID)
		c.Set("session_id", "this-device")
		h.Logout(c)
	})
	if w := do(r, http.MethodPost, "/logout"); w.Code != http.StatusOK {
		t.Fatalf("logout = %d, want 200", w.Code)
	}
	if _, present := store.sessions["this-device"]; present {
		t.Error("current session should be deleted")
	}
	if _, kept := store.sessions["other-device"]; !kept {
		t.Error("another device's session must remain logged in")
	}
}

// TestLogoutAll clears every session and returns 200.
func TestLogoutAll(t *testing.T) {
	store := newFakeUserStore()
	store.sessions["s1"] = "j1"
	store.sessions["s2"] = "j2"
	h, _ := newAuthHandlerWithStore(t, store)

	r := gin.New()
	r.POST("/logout/all", func(c *gin.Context) {
		c.Set("membership_id", testUserID)
		h.LogoutAll(c)
	})
	if w := do(r, http.MethodPost, "/logout/all"); w.Code != http.StatusOK {
		t.Fatalf("logout-all = %d, want 200", w.Code)
	}
	if len(store.sessions) != 0 {
		t.Errorf("expected all sessions cleared, got %d", len(store.sessions))
	}
}

// TestLogout_EmitsLogoutSessionEvent pins the audit taxonomy: a single-device
// logout is recorded as "logout.session" so it shares the "logout." family with
// "logout.all" and the admin Logouts filter (prefix match) catches both.
func TestLogout_EmitsLogoutSessionEvent(t *testing.T) {
	store := newFakeUserStore()
	store.sessions["this-device"] = "jti-a"
	audit := &fakeAudit{}
	h, _ := newAuthHandlerWithStoreAndAudit(t, store, audit)

	r := gin.New()
	r.POST("/logout", func(c *gin.Context) {
		c.Set("membership_id", testUserID)
		c.Set("session_id", "this-device")
		h.Logout(c)
	})
	if w := do(r, http.MethodPost, "/logout"); w.Code != http.StatusOK {
		t.Fatalf("logout = %d, want 200", w.Code)
	}
	if len(audit.events) != 1 || audit.events[0].EventType != "logout.session" {
		t.Fatalf("audit events = %+v, want one logout.session", audit.events)
	}
}

// TestRefreshToken_ExpiredSessionAuditReason pins the refresh.failure reason for an
// unknown/expired session as "expired" (per spec), not "expired_session".
func TestRefreshToken_ExpiredSessionAuditReason(t *testing.T) {
	store := newFakeUserStore() // no sessions → RotateSession reports rotated=false
	audit := &fakeAudit{}
	h, jwt := newAuthHandlerWithStoreAndAudit(t, store, audit)
	profile := &auth.BungieUserProfile{MembershipID: testUserID, MembershipType: 3}
	refresh, _, err := jwt.GenerateRefreshToken(profile, 1, "sess-unknown")
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{"refreshToken":"`+refresh+`"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expired-session refresh = %d, want 401: %s", w.Code, w.Body.String())
	}
	if len(audit.events) != 1 {
		t.Fatalf("audit events = %+v, want one refresh.failure", audit.events)
	}
	ev := audit.events[0]
	if ev.EventType != "refresh.failure" {
		t.Errorf("event type = %q, want refresh.failure", ev.EventType)
	}
	if got := ev.Details["reason"]; got != "expired" {
		t.Errorf("reason = %v, want \"expired\"", got)
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
