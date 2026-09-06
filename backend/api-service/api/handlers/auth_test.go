package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/config"
	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
)

// These tests cover the HTTP surface of the session lifecycle: status codes,
// the refresh cookie, audit events and response bodies. What a session *is* —
// issuance, rotation, reuse detection, the degraded path — is tested against
// auth.SessionIssuer directly in auth/session_test.go, without gin.

const testJWTSecret = "test-jwt-secret-at-least-32-chars-long!!"

// fakeAudit captures emitted audit events so tests can assert on event type and
// details. Satisfies the AuditLogger interface.
type fakeAudit struct{ events []db.AuditEvent }

func (f *fakeAudit) Log(_ context.Context, ev db.AuditEvent) error {
	f.events = append(f.events, ev)
	return nil
}

// fakeUserStore is an in-memory store satisfying both auth.SessionStore and
// auth.UserAuthStore. It mirrors RotateSession's lock-CAS-with-reuse-delete: a
// presented jti must match the session's current jti, otherwise the session is
// revoked (deleted).
type fakeUserStore struct {
	sessions map[string]string          // sid -> current refresh jti
	rotateFn func() (bool, bool, error) // optional override for RotateSession
	version  int                        // token_version reported to the revoker
	role     int16
	notFound bool
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{sessions: map[string]string{}, version: 1}
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
func (f *fakeUserStore) GetAuthInfo(_ context.Context, _ string) (int, int16, bool, error) {
	return f.version, f.role, !f.notFound, nil
}
func (f *fakeUserStore) SessionExists(_ context.Context, id string) (bool, error) {
	_, ok := f.sessions[id]
	return ok, nil
}

// newAuthHandlerWith builds the handler over a real SessionIssuer, so the
// mapping under test is the one production uses.
func newAuthHandlerWith(t *testing.T, store *fakeUserStore, audit AuditLogger) (*AuthHandler, *auth.JWT) {
	t.Helper()
	cfg := &config.Config{JWTSecret: testJWTSecret}
	jwt := auth.NewJWT(cfg.JWTSecret, 24, 30)
	issuer := auth.NewSessionIssuer(auth.SessionDeps{
		JWT:     jwt,
		Tokens:  newTokenStore(t),
		Users:   store,
		Revoker: auth.NewRevocationChecker(store, cache.NewNoOpCache()),
		Cache:   cache.NewNoOpCache(),
		State:   auth.NewStateSigner(cfg.JWTSecret),
		OAuth: auth.OAuthConfig{
			ClientID:     "client-123",
			AuthorizeURL: "https://www.bungie.net/en/OAuth/Authorize",
			RedirectURI:  "http://localhost:3000/auth/callback",
		},
	})
	return NewAuthHandler(issuer, cfg, audit), jwt
}

func newAuthHandler(t *testing.T) (*AuthHandler, *auth.JWT) {
	t.Helper()
	h, jwt := newAuthHandlerWith(t, newFakeUserStore(), nil)
	return h, jwt
}

func newRefreshRequest(path, token string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: token})
	}
	return req
}

func findRefreshCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == refreshCookieName {
			return cookie
		}
	}
	t.Fatalf("response did not set %s", refreshCookieName)
	return nil
}

func TestRefreshCookie_ProductionAttributes(t *testing.T) {
	h, _ := newAuthHandler(t)
	h.cfg.GoEnv = "production"
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h.setRefreshCookie(c, &auth.Session{
		RefreshToken:     "refresh-jwt",
		RefreshExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	})

	cookie := findRefreshCookie(t, w)
	if cookie.Value != "refresh-jwt" || cookie.Domain != "" || cookie.Path != refreshCookiePath {
		t.Fatalf("refresh cookie scope = %+v", cookie)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("refresh cookie security attributes = %+v", cookie)
	}
	if cookie.MaxAge <= 0 || !cookie.Expires.After(time.Now()) {
		t.Fatalf("refresh cookie expiry = %+v", cookie)
	}
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
			c.Set("role", auth.RoleBeta)
			next(c)
		}
	}
	r.GET("/validate", withCtx(h.ValidateToken))
	r.GET("/profile", withCtx(h.GetSessionUser))

	// Both answer from the same builder, so both carry the same fields —
	// including the role that /validate used to drop.
	for _, path := range []string{"/validate", "/profile"} {
		w := do(r, http.MethodGet, path)
		if w.Code != http.StatusOK {
			t.Errorf("%s status = %d", path, w.Code)
		}
		body := w.Body.String()
		for _, want := range []string{"TestGuardian", `"platform":"steam"`, `"role":"beta"`} {
			if !strings.Contains(body, want) {
				t.Errorf("%s body missing %s: %s", path, want, body)
			}
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
	w := do(r, http.MethodPost, "/logout")
	if w.Code != http.StatusOK {
		t.Errorf("logout = %d, want 200", w.Code)
	}
	if cookie := findRefreshCookie(t, w); cookie.MaxAge >= 0 {
		t.Fatalf("logout cookie MaxAge = %d, want negative", cookie.MaxAge)
	}
}

func TestRefreshToken_Success(t *testing.T) {
	store := newFakeUserStore()
	h, jwt := newAuthHandlerWith(t, store, nil)
	profile := &auth.DestinyMembership{MembershipID: testUserID, DisplayName: "TestGuardian", MembershipType: 3}
	const sid = "sess-1"
	refresh, jti, err := jwt.GenerateRefreshToken(profile, 1, sid)
	if err != nil {
		t.Fatal(err)
	}
	store.sessions[sid] = jti

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRefreshRequest("/refresh", refresh))

	if w.Code != http.StatusOK {
		t.Fatalf("refresh = %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "token") || strings.Contains(w.Body.String(), "refreshToken") {
		t.Errorf("unexpected token response: %s", w.Body.String())
	}
	// The refresh body carries the same user object the callback returns; it
	// used to silently omit role, which the shared builder makes impossible.
	if !strings.Contains(w.Body.String(), `"role":`) {
		t.Errorf("refresh response omitted role: %s", w.Body.String())
	}
	cookie := findRefreshCookie(t, w)
	if cookie.Value == "" || cookie.Value == refresh {
		t.Fatal("refresh cookie was not rotated")
	}
	if !cookie.HttpOnly || cookie.Path != refreshCookiePath || cookie.SameSite != http.SameSiteLaxMode || cookie.Secure {
		t.Fatalf("refresh cookie attributes = %+v", cookie)
	}
}

func TestRefreshToken_RejectsAccessTokenAndGarbage(t *testing.T) {
	h, jwt := newAuthHandler(t)
	profile := &auth.DestinyMembership{MembershipID: testUserID, MembershipType: 3}
	// An access token must be rejected by the refresh endpoint (token-type claim).
	access, _ := jwt.GenerateAccessToken(profile, 1, "")

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)

	post := func(token string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newRefreshRequest("/refresh", token))
		return w
	}

	if w := post(access); w.Code != http.StatusUnauthorized {
		t.Errorf("access-as-refresh = %d, want 401", w.Code)
	} else if findRefreshCookie(t, w).MaxAge >= 0 {
		t.Error("invalid refresh did not expire the cookie")
	}
	if w := post("not-a-jwt"); w.Code != http.StatusUnauthorized {
		t.Errorf("garbage token = %d, want 401", w.Code)
	}
	if w := post(""); w.Code != http.StatusUnauthorized {
		t.Errorf("missing cookie = %d, want 401", w.Code)
	}

	// A legacy body token is not a compatibility bridge: without the cookie it
	// is rejected even when the JSON contains an otherwise valid refresh token.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{"refreshToken":"`+access+`"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("body-only token = %d, want 401", w.Code)
	}
}

// TestRefreshToken_ReuseRejected verifies per-session rotation over HTTP: the
// first use rotates, and replaying that now-superseded token is rejected with
// 401 and the session revoked.
func TestRefreshToken_ReuseRejected(t *testing.T) {
	store := newFakeUserStore()
	h, jwt := newAuthHandlerWith(t, store, nil)
	profile := &auth.DestinyMembership{MembershipID: testUserID, DisplayName: "TestGuardian", MembershipType: 3}
	const sid = "sess-1"
	refresh, jti, err := jwt.GenerateRefreshToken(profile, 1, sid)
	if err != nil {
		t.Fatal(err)
	}
	store.sessions[sid] = jti // simulate login having created this session

	r := gin.New()
	r.POST("/refresh", h.RefreshToken)
	post := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newRefreshRequest("/refresh", refresh))
		return w
	}

	if w := post(); w.Code != http.StatusOK {
		t.Fatalf("first refresh = %d, want 200: %s", w.Code, w.Body.String())
	}
	if w := post(); w.Code != http.StatusUnauthorized {
		t.Fatalf("reused refresh = %d, want 401", w.Code)
	} else if findRefreshCookie(t, w).MaxAge >= 0 {
		t.Fatal("reuse rejection did not expire refresh cookie")
	}
	if _, alive := store.sessions[sid]; alive {
		t.Error("reuse should have revoked (deleted) the session")
	}
}

// TestLogout_DeletesCurrentSessionOnly pins the multi-device contract: this-device
// logout ends the caller's session and leaves other devices' sessions intact.
func TestLogout_DeletesCurrentSessionOnly(t *testing.T) {
	store := newFakeUserStore()
	store.sessions["this-device"] = "jti-a"
	store.sessions["other-device"] = "jti-b"
	h, _ := newAuthHandlerWith(t, store, nil)

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
	h, _ := newAuthHandlerWith(t, store, nil)

	r := gin.New()
	r.POST("/logout/all", func(c *gin.Context) {
		c.Set("membership_id", testUserID)
		h.LogoutAll(c)
	})
	w := do(r, http.MethodPost, "/logout/all")
	if w.Code != http.StatusOK {
		t.Fatalf("logout-all = %d, want 200", w.Code)
	}
	if cookie := findRefreshCookie(t, w); cookie.MaxAge >= 0 {
		t.Fatalf("logout-all cookie MaxAge = %d, want negative", cookie.MaxAge)
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
	h, _ := newAuthHandlerWith(t, store, audit)

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

// TestSessionFailureMapping drives every auth.Reason through failSession and
// pins the whole table: status, message, audit event, audit reason, and whether
// the refresh cookie is cleared. This is the mapping the two endpoints share,
// and the reason the response copy stopped being hand-written per branch.
func TestSessionFailureMapping(t *testing.T) {
	cases := []struct {
		name       string
		table      map[auth.Reason]authFailure
		err        error
		wantStatus int
		wantBody   string
		wantEvent  string // "" = no audit event
		wantReason string // "" = no reason detail
		wantExpire bool
	}{
		{"login invalid code", loginFailures, &auth.SessionError{Reason: auth.ReasonInvalidCode},
			http.StatusBadRequest, "Invalid authorization code", "login.failure", "invalid_code", false},
		{"login invalid state", loginFailures, &auth.SessionError{Reason: auth.ReasonInvalidState},
			http.StatusBadRequest, "Invalid or expired state", "login.failure", "invalid_state", false},
		{"login code exchange", loginFailures, &auth.SessionError{Reason: auth.ReasonCodeExchange},
			http.StatusInternalServerError, "Failed to complete authentication", "login.failure", "code_exchange", false},
		{"login membership fetch", loginFailures, &auth.SessionError{Reason: auth.ReasonMembershipFetch},
			http.StatusInternalServerError, "Failed to retrieve Destiny membership", "login.failure", "profile_fetch", false},
		{"login session write", loginFailures, &auth.SessionError{Reason: auth.ReasonSessionWrite},
			http.StatusInternalServerError, "Failed to create session", "", "", false},
		{"login token mint", loginFailures, &auth.SessionError{Reason: auth.ReasonTokenMint},
			http.StatusInternalServerError, "Failed to create session", "", "", false},
		{"refresh missing cookie", refreshFailures, &auth.SessionError{Reason: auth.ReasonMissingCookie},
			http.StatusUnauthorized, "Invalid or expired refresh token", "refresh.failure", "missing_cookie", true},
		{"refresh invalid token", refreshFailures, &auth.SessionError{Reason: auth.ReasonInvalidToken},
			http.StatusUnauthorized, "Invalid or expired refresh token", "refresh.failure", "invalid_token", true},
		{"refresh revoked", refreshFailures, &auth.SessionError{Reason: auth.ReasonRevoked},
			http.StatusUnauthorized, "revoked", "refresh.failure", "revoked", true},
		{"refresh reuse", refreshFailures, &auth.SessionError{Reason: auth.ReasonReuse},
			http.StatusUnauthorized, "security reasons", "refresh.reuse", "", true},
		{"refresh expired", refreshFailures, &auth.SessionError{Reason: auth.ReasonExpired},
			http.StatusUnauthorized, "Session has expired", "refresh.failure", "expired", true},
		{"refresh token mint", refreshFailures, &auth.SessionError{Reason: auth.ReasonTokenMint},
			http.StatusInternalServerError, "Failed to refresh session", "", "", false},
		{"reconnect authorization write", reconnectFailures, &auth.SessionError{Reason: auth.ReasonAuthorizationWrite, MembershipID: testUserID},
			http.StatusServiceUnavailable, "Failed to save Bungie authorization", "reconnect.failure", "authorization_write", false},
		// A reason with no table entry, and an error that is not a session
		// failure at all, must both surface as 500 rather than as success.
		{"unmapped reason", loginFailures, &auth.SessionError{Reason: auth.Reason("something_new")},
			http.StatusInternalServerError, "Failed to create session", "", "", false},
		{"foreign error", loginFailures, errors.New("boom"),
			http.StatusInternalServerError, "Failed to create session", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			audit := &fakeAudit{}
			h, _ := newAuthHandlerWith(t, newFakeUserStore(), audit)
			r := gin.New()
			r.POST("/x", func(c *gin.Context) { h.failSession(c, tc.err, tc.table) })

			w := do(r, http.MethodPost, "/x")
			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
			if !strings.Contains(w.Body.String(), tc.wantBody) {
				t.Errorf("body = %s, want it to contain %q", w.Body.String(), tc.wantBody)
			}
			if tc.wantEvent == "" {
				if len(audit.events) != 0 {
					t.Errorf("audit events = %+v, want none", audit.events)
				}
			} else {
				if len(audit.events) != 1 || audit.events[0].EventType != tc.wantEvent {
					t.Fatalf("audit events = %+v, want one %s", audit.events, tc.wantEvent)
				}
				if audit.events[0].Outcome != "failure" {
					t.Errorf("outcome = %q, want failure", audit.events[0].Outcome)
				}
				got, _ := audit.events[0].Details["reason"].(string)
				if got != tc.wantReason {
					t.Errorf("audit reason = %q, want %q", got, tc.wantReason)
				}
			}
			// A failure either clears the cookie or leaves it entirely alone;
			// it must never write a live one.
			expired := false
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name != refreshCookieName {
					continue
				}
				if cookie.MaxAge >= 0 || cookie.Value != "" {
					t.Fatalf("failure wrote a live refresh cookie: %+v", cookie)
				}
				expired = true
			}
			if expired != tc.wantExpire {
				t.Errorf("cookie expired = %v, want %v", expired, tc.wantExpire)
			}
		})
	}
}

// TestSessionFailureAudit_CarriesIdentity pins why SessionError carries the
// membership and session ids: once the issuer owns refresh-token validation,
// the handler has no claims to read them from, and the audit trail would
// quietly lose the actor on exactly the security-relevant events.
func TestSessionFailureAudit_CarriesIdentity(t *testing.T) {
	audit := &fakeAudit{}
	h, _ := newAuthHandlerWith(t, newFakeUserStore(), audit)
	r := gin.New()
	r.POST("/x", func(c *gin.Context) {
		h.failSession(c, &auth.SessionError{
			Reason: auth.ReasonReuse, MembershipID: testUserID, SessionID: "sess-9",
		}, refreshFailures)
	})

	do(r, http.MethodPost, "/x")
	if len(audit.events) != 1 {
		t.Fatalf("audit events = %+v, want one", audit.events)
	}
	if ev := audit.events[0]; ev.ActorMembershipID != testUserID || ev.SessionID != "sess-9" {
		t.Errorf("audit identity = %q/%q, want %q/sess-9", ev.ActorMembershipID, ev.SessionID, testUserID)
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

func TestReconnectBungie_PreservesGuardianSession(t *testing.T) {
	store := newFakeUserStore()
	store.sessions["existing-session"] = "existing-jti"
	bungie := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "GetMembershipsForCurrentUser") {
			fmt.Fprintf(w, `{"Response":{"destinyMemberships":[{"membershipType":3,"membershipId":"%s","displayName":"TestGuardian"}],"primaryMembershipId":"%s"},"ErrorCode":1}`, testUserID, testUserID)
			return
		}
		fmt.Fprint(w, `{"access_token":"new-bungie-access","expires_in":3600}`)
	}))
	t.Cleanup(bungie.Close)

	cfg := &config.Config{JWTSecret: testJWTSecret}
	issuer := auth.NewSessionIssuer(auth.SessionDeps{
		JWT:     auth.NewJWT(cfg.JWTSecret, 24, 30),
		Tokens:  newTokenStore(t),
		Users:   store,
		Revoker: auth.NewRevocationChecker(store, cache.NewNoOpCache()),
		Cache:   cache.NewNoOpCache(),
		State:   auth.NewStateSigner(cfg.JWTSecret),
		OAuth: auth.OAuthConfig{
			ClientID:   "client-123",
			TokenURL:   bungie.URL,
			APIBaseURL: bungie.URL,
		},
	})
	audit := &fakeAudit{}
	h := NewAuthHandler(issuer, cfg, audit)
	_, state, nonce, err := issuer.AuthorizeURL()
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.POST("/reconnect", func(c *gin.Context) {
		c.Set("membership_id", testUserID)
		h.ReconnectBungie(c)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/reconnect", strings.NewReader(url.Values{
		"code": {"reconnect-code"}, "state": {state},
	}.Encode()))
	req.AddCookie(&http.Cookie{Name: h.oauthCookieName(), Value: nonce})
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("reconnect = %d, body=%s", w.Code, w.Body.String())
	}
	if got := store.sessions["existing-session"]; got != "existing-jti" || len(store.sessions) != 1 {
		t.Fatalf("Guardian sessions changed: %+v", store.sessions)
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == refreshCookieName {
			t.Fatal("reconnect wrote refresh cookie")
		}
	}
	if len(audit.events) != 1 {
		t.Fatalf("audit events = %+v, want one reconnect.success", audit.events)
	}
	if event := audit.events[0]; event.EventType != "reconnect.success" || event.ActorMembershipID != testUserID || len(event.Details) != 0 {
		t.Fatalf("success audit = %+v, want value-free reconnect.success for authenticated membership", event)
	}
}

func TestReconnectBungie_RejectsForgedState(t *testing.T) {
	audit := &fakeAudit{}
	h, _ := newAuthHandlerWith(t, newFakeUserStore(), audit)
	r := gin.New()
	r.POST("/reconnect", func(c *gin.Context) {
		c.Set("membership_id", testUserID)
		h.ReconnectBungie(c)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/reconnect", strings.NewReader(url.Values{
		"code": {"code"}, "state": {"forged"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("forged reconnect state = %d, want 400", w.Code)
	}
	if len(audit.events) != 1 {
		t.Fatalf("audit events = %+v, want one reconnect.failure", audit.events)
	}
	event := audit.events[0]
	if event.EventType != "reconnect.failure" || event.ActorMembershipID != testUserID {
		t.Fatalf("failure audit identity = %+v", event)
	}
	if got := event.Details["reason"]; got != "invalid_state" {
		t.Fatalf("failure reason = %v, want invalid_state", got)
	}
	if _, present := event.Details["code"]; present {
		t.Fatal("failure audit recorded OAuth code")
	}
	if _, present := event.Details["state"]; present {
		t.Fatal("failure audit recorded OAuth state")
	}
}
