package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddleware_AllowlistAndResponseMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware([]string{"https://app.example"}))
	r.GET("/api/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	allowed := httptest.NewRecorder()
	allowedReq := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	allowedReq.Header.Set("Origin", "https://app.example")
	r.ServeHTTP(allowed, allowedReq)
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := allowed.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want Origin", got)
	}
	if got := allowed.Header().Get("Access-Control-Expose-Headers"); got != "X-Request-ID" {
		t.Fatalf("Access-Control-Expose-Headers = %q", got)
	}
	if got := allowed.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q", got)
	}

	denied := httptest.NewRecorder()
	deniedReq := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	deniedReq.Header.Set("Origin", "https://attacker.example")
	r.ServeHTTP(denied, deniedReq)
	if got := denied.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("denied Access-Control-Allow-Origin = %q, want empty", got)
	}
	if got := denied.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("denied Vary = %q, want Origin", got)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	r := gin.New()
	r.Use(corsMiddleware([]string{"https://app.example"}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/auth/refresh", nil)
	req.Header.Set("Origin", "https://app.example")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", w.Code)
	}
}
