package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"

	"github.com/gin-gonic/gin"
)

// fakeVersionStore satisfies UserAuthStore.
type fakeVersionStore struct {
	version      int
	role         int16
	err          error
	missing      bool
	sessionOK    bool // returned by SessionExists
	sessionErr   error
	authCalls    int
	sessionCalls int
}

func (f *fakeVersionStore) GetAuthInfo(_ context.Context, _ string) (int, int16, bool, error) {
	f.authCalls++
	return f.version, f.role, !f.missing, f.err
}

func (f *fakeVersionStore) SessionExists(_ context.Context, _ string) (bool, error) {
	f.sessionCalls++
	return f.sessionOK, f.sessionErr
}

func newMiddlewareRouter(j *JWT, revoker *RevocationChecker) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", j.Middleware(revoker), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"membership_id": c.GetString("membership_id"),
			"platform":      c.GetString("platform"),
			"tver":          c.GetInt("token_version"),
		})
	})
	return r
}

func doProtected(r *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMiddleware_MissingHeader_401(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	r := newMiddlewareRouter(j, nil)
	if w := doProtected(r, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_GarbageToken_401(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	r := newMiddlewareRouter(j, nil)
	if w := doProtected(r, "Bearer not.a.jwt"); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// TestMiddleware_RefreshTokenAsAccess_401 pins the token-type confusion guard:
// a valid *refresh* token must never authorize an access-protected route.
func TestMiddleware_RefreshTokenAsAccess_401(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	r := newMiddlewareRouter(j, nil)
	refresh, _, _ := j.GenerateRefreshToken(testProfile(), 1, "")
	if w := doProtected(r, "Bearer "+refresh); w.Code != http.StatusUnauthorized {
		t.Fatalf("refresh-as-access: expected 401, got %d", w.Code)
	}
}

func TestMiddleware_ValidAccess_SetsContext(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	r := newMiddlewareRouter(j, nil)
	tok, _ := j.GenerateAccessToken(testProfile(), 7, "")
	w := doProtected(r, "Bearer "+tok)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"4611686018467260757", "steam", "7"} {
		if !strings.Contains(body, want) {
			t.Errorf("response %s missing %q", body, want)
		}
	}
}

func TestMiddleware_TverMismatch_401(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	// DB says version 2; the JWT carries version 1 (issued before a logout).
	revoker := NewRevocationChecker(&fakeVersionStore{version: 2}, cache.NewMemoryCache(time.Minute, 0))
	r := newMiddlewareRouter(j, revoker)
	tok, _ := j.GenerateAccessToken(testProfile(), 1, "")
	if w := doProtected(r, "Bearer "+tok); w.Code != http.StatusUnauthorized {
		t.Fatalf("stale token_version: expected 401, got %d", w.Code)
	}
}

func TestMiddleware_TverMatch_200(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	revoker := NewRevocationChecker(&fakeVersionStore{version: 3}, cache.NewMemoryCache(time.Minute, 0))
	r := newMiddlewareRouter(j, revoker)
	tok, _ := j.GenerateAccessToken(testProfile(), 3, "")
	if w := doProtected(r, "Bearer "+tok); w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// TestMiddleware_SessionRevoked_401 pins this-device logout: a token whose
// token_version still matches must be rejected once its session no longer exists.
func TestMiddleware_SessionRevoked_401(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	// token_version matches (3), but the session is gone (sessionOK=false).
	revoker := NewRevocationChecker(&fakeVersionStore{version: 3, sessionOK: false}, cache.NewMemoryCache(time.Minute, 0))
	r := newMiddlewareRouter(j, revoker)
	tok, _ := j.GenerateAccessToken(testProfile(), 3, "sess-gone")
	if w := doProtected(r, "Bearer "+tok); w.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session: expected 401, got %d", w.Code)
	}
}

// TestMiddleware_SessionValid_200 confirms a live session passes the session check.
func TestMiddleware_SessionValid_200(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	revoker := NewRevocationChecker(&fakeVersionStore{version: 3, sessionOK: true}, cache.NewMemoryCache(time.Minute, 0))
	r := newMiddlewareRouter(j, revoker)
	tok, _ := j.GenerateAccessToken(testProfile(), 3, "sess-live")
	if w := doProtected(r, "Bearer "+tok); w.Code != http.StatusOK {
		t.Fatalf("live session: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRevocation_FailOpenOnDBError(t *testing.T) {
	revoker := NewRevocationChecker(&fakeVersionStore{err: fmt.Errorf("db down")}, cache.NewMemoryCache(time.Minute, 0))
	if err := revoker.Check(context.Background(), "user-1", 1); err != nil {
		t.Fatalf("expected fail-open on DB error, got %v", err)
	}
}

func TestRevocation_MissingUserFailsClosed(t *testing.T) {
	store := &fakeVersionStore{missing: true}
	revoker := NewRevocationChecker(store, cache.NewMemoryCache(time.Minute, 0))
	if err := revoker.Check(context.Background(), "deleted-user", 1); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Check() = %v, want ErrUserNotFound", err)
	}
}

func TestRevocation_WrongTypedUserCacheValueIsRepaired(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, 0)
	c.Set("tver:user-1", "corrupt", time.Minute)
	store := &fakeVersionStore{version: 4, role: RoleAlpha}
	revoker := NewRevocationChecker(store, c)

	role, err := revoker.Resolve(context.Background(), "user-1", 4, "")
	if err != nil {
		t.Fatalf("Resolve() = %v", err)
	}
	if role != RoleAlpha {
		t.Fatalf("role = %d, want %d", role, RoleAlpha)
	}
	if store.authCalls != 1 {
		t.Fatalf("GetAuthInfo calls = %d, want 1", store.authCalls)
	}
	v, ok := c.Get("tver:user-1")
	if _, typed := v.(authInfo); !ok || !typed {
		t.Fatalf("cache value was not repaired: %#v", v)
	}
}

func TestRevocation_WrongTypedSessionCacheValueIsRepaired(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, 0)
	c.Set("sess:session-1", "corrupt", time.Minute)
	store := &fakeVersionStore{version: 1, sessionOK: true}
	revoker := NewRevocationChecker(store, c)

	if _, err := revoker.Resolve(context.Background(), "user-1", 1, "session-1"); err != nil {
		t.Fatalf("Resolve() = %v", err)
	}
	if store.sessionCalls != 1 {
		t.Fatalf("SessionExists calls = %d, want 1", store.sessionCalls)
	}
	if v, ok := c.Get("sess:session-1"); !ok || v != true {
		t.Fatalf("session cache value was not repaired: %#v", v)
	}
}

func TestRevocation_CacheHitPath(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, 0)
	store := &fakeVersionStore{version: 2}
	revoker := NewRevocationChecker(store, c)

	// First check populates the cache from the DB.
	if err := revoker.Check(context.Background(), "user-1", 2); err != nil {
		t.Fatalf("matching version rejected: %v", err)
	}
	// DB now changes, but the cached value still answers within the TTL window.
	store.version = 99
	if err := revoker.Check(context.Background(), "user-1", 2); err != nil {
		t.Fatalf("cached version should still pass within TTL: %v", err)
	}
	// A mismatched claim against the cached value is rejected.
	if err := revoker.Check(context.Background(), "user-1", 1); err == nil {
		t.Fatal("stale claimed version passed against cached value")
	}
}

func TestRevocation_NilStoreDegradedMode(t *testing.T) {
	revoker := NewRevocationChecker(nil, cache.NewMemoryCache(time.Minute, 0))
	if err := revoker.Check(context.Background(), "anyone", 42); err != nil {
		t.Fatalf("degraded mode must skip the check, got %v", err)
	}
}
