package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"

	"github.com/gin-gonic/gin"
)

// fakeVersionStore satisfies UserVersionStore.
type fakeVersionStore struct {
	version int
	err     error
}

func (f *fakeVersionStore) GetTokenVersion(_ context.Context, _ string) (int, error) {
	return f.version, f.err
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
	refresh, _ := j.GenerateRefreshToken(testProfile(), 1)
	if w := doProtected(r, "Bearer "+refresh); w.Code != http.StatusUnauthorized {
		t.Fatalf("refresh-as-access: expected 401, got %d", w.Code)
	}
}

func TestMiddleware_ValidAccess_SetsContext(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	r := newMiddlewareRouter(j, nil)
	tok, _ := j.GenerateAccessToken(testProfile(), 7)
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
	revoker := NewRevocationChecker(&fakeVersionStore{version: 2}, cache.NewMemoryCache(time.Minute, time.Minute))
	r := newMiddlewareRouter(j, revoker)
	tok, _ := j.GenerateAccessToken(testProfile(), 1)
	if w := doProtected(r, "Bearer "+tok); w.Code != http.StatusUnauthorized {
		t.Fatalf("stale token_version: expected 401, got %d", w.Code)
	}
}

func TestMiddleware_TverMatch_200(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	revoker := NewRevocationChecker(&fakeVersionStore{version: 3}, cache.NewMemoryCache(time.Minute, time.Minute))
	r := newMiddlewareRouter(j, revoker)
	tok, _ := j.GenerateAccessToken(testProfile(), 3)
	if w := doProtected(r, "Bearer "+tok); w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRevocation_FailOpenOnDBError(t *testing.T) {
	revoker := NewRevocationChecker(&fakeVersionStore{err: fmt.Errorf("db down")}, cache.NewMemoryCache(time.Minute, time.Minute))
	if err := revoker.Check(context.Background(), "user-1", 1); err != nil {
		t.Fatalf("expected fail-open on DB error, got %v", err)
	}
}

func TestRevocation_CacheHitPath(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, time.Minute)
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
	revoker := NewRevocationChecker(nil, cache.NewMemoryCache(time.Minute, time.Minute))
	if err := revoker.Check(context.Background(), "anyone", 42); err != nil {
		t.Fatalf("degraded mode must skip the check, got %v", err)
	}
}
