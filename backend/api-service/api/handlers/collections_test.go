package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/collections"

	"github.com/gin-gonic/gin"
)

func TestIsValidMembershipID(t *testing.T) {
	cases := []struct {
		name string
		id   string
		want bool
	}{
		{"valid", "4611686018467260757", true},
		{"min length 10", "1234567890", true},
		{"too short", "123456789", false},
		{"too long", strings.Repeat("1", 26), false},
		{"letters", "46116860184abc60757", false},
		{"empty", "", false},
		{"sql injection", "1; DROP TABLE users", false},
		{"negative", "-4611686018467260757", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isValidMembershipID(tc.id); got != tc.want {
				t.Errorf("isValidMembershipID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

func TestIsValidMembershipType(t *testing.T) {
	for _, valid := range []int{1, 2, 3, 4, 5, 6, 10, 254} {
		if !isValidMembershipType(valid) {
			t.Errorf("isValidMembershipType(%d) = false, want true", valid)
		}
	}
	for _, invalid := range []int{0, -1, 7, 255, 1000} {
		if isValidMembershipType(invalid) {
			t.Errorf("isValidMembershipType(%d) = true, want false", invalid)
		}
	}
}

func TestParseMembershipParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x/:membershipType/:membershipId", func(c *gin.Context) {
		mt, mid, ok := parseMembershipParams(c)
		if !ok {
			return // parseMembershipParams already wrote the error response
		}
		c.JSON(http.StatusOK, gin.H{"type": mt, "id": mid})
	})

	cases := []struct {
		name string
		path string
		want int
	}{
		{"valid", "/x/3/4611686018467260757", http.StatusOK},
		{"bad type", "/x/99/4611686018467260757", http.StatusBadRequest},
		{"non-numeric type", "/x/steam/4611686018467260757", http.StatusBadRequest},
		{"bad id", "/x/3/short", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Errorf("GET %s = %d, want %d", tc.path, w.Code, tc.want)
			}
		})
	}
}

func TestHandleBungieError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string // substring of the JSON "code" field
	}{
		{"privacy", &bungie.BungieError{ErrorCode: 5}, http.StatusForbidden, "PRIVACY_RESTRICTION"},
		{"not found", &bungie.BungieError{ErrorCode: 7}, http.StatusNotFound, "ACCOUNT_NOT_FOUND"},
		{"throttled", &bungie.BungieError{ErrorCode: 36, ThrottleSeconds: 12}, http.StatusTooManyRequests, "RATE_LIMITED"},
		{"other bungie error", &bungie.BungieError{ErrorCode: 99}, http.StatusBadGateway, "BUNGIE_ERROR"},
		{"manifest not ready", fmt.Errorf("wrap: %w", collections.ErrManifestNotReady), http.StatusServiceUnavailable, "MANIFEST_NOT_READY"},
		{"plain error", fmt.Errorf("boom"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			handleBungieError(c, tc.err)
			if w.Code != tc.wantCode {
				t.Errorf("status = %d, want %d", w.Code, tc.wantCode)
			}
			if !strings.Contains(w.Body.String(), tc.wantBody) {
				t.Errorf("body %s missing code %q", w.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestAvailableNowOverlay(t *testing.T) {
	items := map[string]collections.DestinyItem{"100": {ItemHash: "100"}, "200": {ItemHash: "200"}}
	live := map[uint32]string{
		100: "Banshee-44",
		999: "Xûr", // not in the collection → excluded
	}

	got := availableNowOverlay(items, live)

	if got["100"] != "Banshee-44" {
		t.Errorf("item 100 = %q, want Banshee-44", got["100"])
	}
	if _, ok := got["999"]; ok {
		t.Errorf("item 999 (not in collection) must be excluded")
	}
	if len(got) != 1 {
		t.Errorf("len = %d, want 1", len(got))
	}
}
