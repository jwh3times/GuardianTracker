package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPerIPRateLimit_Blocks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", PerIPRateLimit(1, 2), func(c *gin.Context) { c.Status(200) })

	codes := []int{}
	for i := 0; i < 4; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/x", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		r.ServeHTTP(w, req)
		codes = append(codes, w.Code)
	}
	// burst 2 → first two pass, then 429s.
	if codes[0] != 200 || codes[1] != 200 || codes[2] != 429 || codes[3] != 429 {
		t.Fatalf("codes = %v, want [200 200 429 429]", codes)
	}
	// A different IP has its own bucket.
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("second IP got %d, want 200", w.Code)
	}
}

func TestMaxBodyBytes_Rejects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodyBytes(64))
	r.POST("/x", func(c *gin.Context) {
		var v map[string]any
		if err := c.ShouldBindJSON(&v); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "body too large or invalid"})
			return
		}
		c.Status(200)
	})

	small := httptest.NewRecorder()
	r.ServeHTTP(small, httptest.NewRequest("POST", "/x", strings.NewReader(`{"a":1}`)))
	if small.Code != 200 {
		t.Fatalf("small body got %d, want 200", small.Code)
	}

	// VALID JSON that exceeds the 64-byte cap — an invalid payload would 400
	// from JSON syntax alone and prove nothing about MaxBytesReader.
	oversized := `{"a":"` + strings.Repeat("a", 128) + `"}`
	big := httptest.NewRecorder()
	r.ServeHTTP(big, httptest.NewRequest("POST", "/x", strings.NewReader(oversized)))
	if big.Code != 400 {
		t.Fatalf("big body got %d, want 400", big.Code)
	}
}

func TestAPISecurityHeaders(t *testing.T) {
	r := gin.New()
	r.Use(APISecurityHeaders())
	r.GET("/api/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/health", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	api := httptest.NewRecorder()
	r.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/x", nil))
	if got := api.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if got := api.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q", got)
	}

	health := httptest.NewRecorder()
	r.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if got := health.Header().Get("X-Content-Type-Options"); got != "" {
		t.Fatalf("health X-Content-Type-Options = %q, want empty", got)
	}
}

func TestNoStore(t *testing.T) {
	r := gin.New()
	r.GET("/auth", NoStore(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth", nil))
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

func TestRequireAllowedOrigin_ExactMatchOnly(t *testing.T) {
	r := gin.New()
	r.POST("/auth", RequireAllowedOrigin([]string{" https://app.example ", "http://localhost:5273"}), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for _, tc := range []struct {
		name   string
		origin string
		want   int
	}{
		{name: "allowed", origin: "https://app.example", want: http.StatusNoContent},
		{name: "missing", want: http.StatusForbidden},
		{name: "prefix attack", origin: "https://app.example.attacker.test", want: http.StatusForbidden},
		{name: "different port", origin: "http://localhost:5274", want: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/auth", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d", w.Code, tc.want)
			}
		})
	}
}
