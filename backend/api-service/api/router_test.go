package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/api/handlers"
	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/config"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

const (
	testSecret       = "test-secret-at-least-32-characters-long"
	testMembershipID = "4611686018467260757"
)

// stubAuthStore drives auth.RevocationChecker without a database, so the role
// the JWT middleware resolves is whatever a test asks for.
type stubAuthStore struct{ role int16 }

func (s stubAuthStore) GetAuthInfo(context.Context, string) (int, int16, bool, error) {
	return 1, s.role, true, nil
}
func (s stubAuthStore) SessionExists(context.Context, string) (bool, error) { return true, nil }

// stubFlags reports every flag enabled except the ones named in disabled.
type stubFlags struct{ disabled map[string]bool }

func (s stubFlags) Resolve(_ context.Context, key string) (enabled bool, minTier int, found bool, err error) {
	return !s.disabled[key], 0, true, nil
}

// newTestRouter builds the real route table. The handlers are constructed with
// nil dependencies on purpose: every assertion here is about what happens
// *before* a handler runs, and a nil handler dependency makes "a request
// unexpectedly reached the handler" loud instead of silent.
func newTestRouter(t *testing.T, role int16, authzEnabled bool, disabledFlags ...string) *gin.Engine {
	t.Helper()

	disabled := map[string]bool{}
	for _, k := range disabledFlags {
		disabled[k] = true
	}

	cfg := &config.Config{
		CORSAllowedOrigins: []string{"https://app.example"},
		MaxBodyBytes:       1 << 20,
		AuthRateLimitRPS:   1000,
		AuthRateLimitBurst: 1000,
		JWTSecret:          testSecret,
	}

	return NewRouter(Deps{
		Config:  cfg,
		Logger:  slog.New(slog.DiscardHandler),
		JWT:     auth.NewJWTWithTTL(testSecret, time.Hour, 90),
		Revoker: auth.NewRevocationChecker(stubAuthStore{role: role}, cache.NewNoOpCache()),
		Authz:   auth.NewAuthz(authzEnabled),
		Flags:   stubFlags{disabled: disabled},
		Handlers: Handlers{
			Health:      handlers.NewHealthHandler(nil, nil),
			Auth:        handlers.NewAuthHandler(nil, nil, cfg, nil, nil, nil, nil),
			Wishlist:    handlers.NewWishlistHandler(nil, nil, nil, nil, nil),
			Account:     handlers.NewAccountHandler(nil, nil, nil, nil),
			Admin:       handlers.NewAdminHandler(nil, nil, nil),
			Audit:       handlers.NewAuditHandler(nil),
			Characters:  handlers.NewCharactersHandler(nil, nil),
			Collections: handlers.NewCollectionsHandler(nil, nil, nil, nil, nil),
			Items:       handlers.NewItemsHandler(nil),
			Weekly:      handlers.NewWeeklyHandler(nil, nil),
			Records:     handlers.NewRecordsHandler(nil, nil),
			Search:      handlers.NewSearchHandler(nil),
		},
	})
}

// accessToken mints a token the real JWT middleware will accept.
func accessToken(t *testing.T) string {
	t.Helper()
	j := auth.NewJWTWithTTL(testSecret, time.Hour, 90)
	tok, err := j.GenerateAccessToken(&auth.BungieUserProfile{
		MembershipID:   testMembershipID,
		MembershipType: 3,
		DisplayName:    "TestGuardian",
	}, 1, "session-1")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	return tok
}

// dispatch serves one request and returns its status.
//
// The handlers behind these routes are wired with nil dependencies, so a
// request that gets past the gate under test lands on one and fails — as a 500
// from the observability middleware's panic recovery, or as whatever a
// nil-dependency handler happens to return. Any of those is "not 401" and the
// assertions read it as the failure it is.
func dispatch(r *gin.Engine, method, target, bearer string) int {
	req := httptest.NewRequest(method, target, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// fillParams makes a gin route pattern requestable.
func fillParams(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") {
			parts[i] = "1"
		}
	}
	return strings.Join(parts, "/")
}

// publicAPIRoutes are the only /api routes that intentionally serve an
// unauthenticated caller.
//
// This list lives in the test, not in the router, on purpose: adding a route
// outside the authenticated group fails TestEveryAPIRouteRequiresAuthentication
// until someone comes here and says so deliberately. If the router owned the
// list, the router could grant itself the exemption.
var publicAPIRoutes = map[string]bool{
	"GET /api/manifest/status":       true,
	"GET /api/auth/bungie":           true,
	"POST /api/auth/bungie/callback": true,
	"POST /api/auth/refresh":         true,
}

// TestEveryAPIRouteRequiresAuthentication is the regression guard for the
// failure this card exists to remove: an /api route registered without the JWT
// middleware is published to the world with no error and no log.
//
// It walks the route table gin actually built, so a route added later is
// covered without anyone remembering to extend this test.
func TestEveryAPIRouteRequiresAuthentication(t *testing.T) {
	r := newTestRouter(t, auth.RoleStandard, true)

	unvisited := make(map[string]bool, len(publicAPIRoutes))
	for k := range publicAPIRoutes {
		unvisited[k] = true
	}

	var checked int
	for _, route := range r.Routes() {
		if !strings.HasPrefix(route.Path, "/api/") {
			continue
		}
		key := route.Method + " " + route.Path
		if publicAPIRoutes[key] {
			delete(unvisited, key)
			continue
		}
		checked++
		if got := dispatch(r, route.Method, fillParams(route.Path), ""); got != http.StatusUnauthorized {
			t.Errorf("%s without a token = %d, want 401 — the route is not behind the authentication gate", key, got)
		}
	}

	if checked == 0 {
		t.Fatal("no authenticated /api routes found; the route table did not build")
	}
	for key := range unvisited {
		t.Errorf("publicAPIRoutes lists %q, which is not a registered route", key)
	}
}

// TestPublicRoutesStayReachable is the other direction: the authentication gate
// must not creep onto login. Applying it one group too high would 401 the whole
// OAuth flow, which the test above cannot see.
func TestPublicRoutesStayReachable(t *testing.T) {
	r := newTestRouter(t, auth.RoleStandard, true)

	for key := range publicAPIRoutes {
		method, path, _ := strings.Cut(key, " ")
		// Callback and refresh additionally require an allowlisted Origin, so
		// they legitimately answer 403 here. Only 401 is the failure.
		if got := dispatch(r, method, path, ""); got == http.StatusUnauthorized {
			t.Errorf("%s = 401; public route is behind the authentication gate", key)
		}
	}

	for _, path := range []string{"/health", "/ready"} {
		if got := dispatch(r, http.MethodGet, path, ""); got == http.StatusUnauthorized {
			t.Errorf("%s = 401; health probes must not require a token", path)
		}
	}
}

// TestFlagGatedRoutesEnforceTheirOwnFlag pins each flag-gated route to the flag
// key it claims. RequireFlag fails open on a key it cannot resolve, so gating a
// route on a mistyped or wrong key gates nothing at all — silently.
//
// Disabling exactly one key at a time makes a wrong pairing visible: the route
// either fails to 404 when its own flag is off, or 404s when someone else's is.
func TestFlagGatedRoutesEnforceTheirOwnFlag(t *testing.T) {
	type gated struct {
		method, path, flag string
	}
	routes := []gated{
		{http.MethodGet, "/api/weekly/recommendations", handlers.FlagWeeklyPlanner},
		{http.MethodGet, "/api/items/search", handlers.FlagGlobalSearch},
		{http.MethodGet, "/api/catalysts/3/1", handlers.FlagCatalystsCrafting},
		{http.MethodGet, "/api/crafting/3/1", handlers.FlagCatalystsCrafting},
		{http.MethodGet, "/api/seals/3/1", handlers.FlagTriumphsSeals},
	}
	allFlags := []string{
		handlers.FlagWeeklyPlanner,
		handlers.FlagGlobalSearch,
		handlers.FlagCatalystsCrafting,
		handlers.FlagTriumphsSeals,
	}

	token := accessToken(t)
	for _, off := range allFlags {
		r := newTestRouter(t, auth.RoleStandard, true, off)
		for _, rt := range routes {
			got := dispatch(r, rt.method, rt.path, token)
			blocked := got == http.StatusNotFound
			want := rt.flag == off
			if blocked != want {
				t.Errorf("with %q disabled, %s %s blocked=%v (status %d), want blocked=%v",
					off, rt.method, rt.path, blocked, got, want)
			}
		}
	}
}

// TestAdminRoutesRejectNonAdmins covers the coupling B4 left behind: the admin
// handlers are only safe because RequireAdmin runs first. Asserting it on the
// real route table is what makes that a guarantee rather than a coincidence.
func TestAdminRoutesRejectNonAdmins(t *testing.T) {
	adminRoutes := func(r *gin.Engine) []gin.RouteInfo {
		var out []gin.RouteInfo
		for _, route := range r.Routes() {
			if strings.HasPrefix(route.Path, "/api/admin/") {
				out = append(out, route)
			}
		}
		return out
	}

	token := accessToken(t)

	t.Run("non-admin gets 403", func(t *testing.T) {
		r := newTestRouter(t, auth.RoleStandard, true)
		routes := adminRoutes(r)
		if len(routes) != 5 {
			t.Fatalf("found %d admin routes, want 5", len(routes))
		}
		for _, route := range routes {
			got := dispatch(r, route.Method, fillParams(route.Path), token)
			if got != http.StatusForbidden {
				t.Errorf("%s %s as standard user = %d, want 403", route.Method, route.Path, got)
			}
		}
	})

	t.Run("degraded build gets 503, not a panic", func(t *testing.T) {
		// Roles are not authoritative without a database, so every admin route
		// must refuse before reaching a handler wired to a degraded store.
		r := newTestRouter(t, auth.RoleAdmin, false)
		for _, route := range adminRoutes(r) {
			got := dispatch(r, route.Method, fillParams(route.Path), token)
			if got != http.StatusServiceUnavailable {
				t.Errorf("%s %s in degraded mode = %d, want 503", route.Method, route.Path, got)
			}
		}
	})
}

// TestInvalidTokenIsRejected guards the middleware's other two refusals, which
// the table test cannot distinguish from a missing header.
func TestInvalidTokenIsRejected(t *testing.T) {
	r := newTestRouter(t, auth.RoleStandard, true)

	if got := dispatch(r, http.MethodGet, "/api/wishlist", "not-a-jwt"); got != http.StatusUnauthorized {
		t.Errorf("garbage token = %d, want 401", got)
	}

	// A refresh token must not authenticate an API call.
	j := auth.NewJWTWithTTL(testSecret, time.Hour, 90)
	refresh, _, err := j.GenerateRefreshToken(&auth.BungieUserProfile{
		MembershipID: testMembershipID, MembershipType: 3,
	}, 1, "session-1")
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if got := dispatch(r, http.MethodGet, "/api/wishlist", refresh); got != http.StatusUnauthorized {
		t.Errorf("refresh token on an API route = %d, want 401", got)
	}
}
