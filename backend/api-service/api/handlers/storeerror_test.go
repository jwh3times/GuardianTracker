package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
)

func TestHandleStoreError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name     string
		err      error
		want     int
		wantCode string
		handled  bool
	}{
		{"unavailable", db.ErrUnavailable, http.StatusServiceUnavailable, "DB_UNAVAILABLE", true},
		{"wrapped unavailable", fmt.Errorf("listing: %w", db.ErrUnavailable), http.StatusServiceUnavailable, "DB_UNAVAILABLE", true},
		{"other failure", errors.New("boom"), http.StatusInternalServerError, "INTERNAL_ERROR", true},
		{"no error", nil, http.StatusOK, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

			if got := HandleStoreError(c, tc.err, "test"); got != tc.handled {
				t.Fatalf("handled = %v, want %v", got, tc.handled)
			}
			if !tc.handled {
				return
			}
			if w.Code != tc.want {
				t.Errorf("status = %d, want %d", w.Code, tc.want)
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("body: %v", err)
			}
			if body["code"] != tc.wantCode {
				t.Errorf("code = %v, want %q", body["code"], tc.wantCode)
			}
		})
	}
}

// Regression: these three admin handlers dereference their store with no guard.
// They were safe only because RequireAdmin happened to reject the request first
// — and only because main.go derived both the middleware's enabled flag and the
// handler's store from the same expression. Wired directly, as here, they used
// to panic on a nil interface. Degraded mode must be a 503 on its own merits.
func TestAdminHandlers_DegradedModeIs503NotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	degraded := db.NewStores(nil)
	h := NewAdminHandler(degraded.Users, degraded.Flags, cache.NewMemoryCache(0, 0))

	r := gin.New()
	r.GET("/users", h.ListUsers)
	r.GET("/flags", h.ListFlags)
	r.PUT("/flags/:key", h.UpdateFlag)

	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/users", ""},
		{http.MethodGet, "/flags", ""},
		{http.MethodPut, "/flags/cosmetics", `{"enabled":true}`},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			var req *http.Request
			if tc.body == "" {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			} else {
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			}
			// No panic, and a diagnosable status.
			r.ServeHTTP(w, req)
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("status = %d, want 503; body = %s", w.Code, w.Body.String())
			}
		})
	}
}
