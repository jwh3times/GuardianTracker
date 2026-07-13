package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"guardian-tracker/api-service/services/bungie"

	"github.com/gin-gonic/gin"
)

// readyManifestService builds a bungie.ManifestService whose IsReady() reports
// true by pointing it at a temp file that already exists on disk (IsReady
// stats the manifest DB path — mirrors the pattern in
// TestHealthHandlers/endpoints_test.go).
func readyManifestService(t *testing.T) *bungie.ManifestService {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")
	if err := os.WriteFile(dbPath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	return bungie.NewManifestService(bungie.NewClient("k", "http://x", 100, 100), dbPath, time.Hour)
}

type fakePinger struct{ err error }

func (f fakePinger) Ping(ctx context.Context) error { return f.err }

func TestReady_DBDown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(readyManifestService(t), fakePinger{err: errors.New("conn refused")})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/ready", nil)
	h.Ready(c)
	if w.Code != 503 {
		t.Fatalf("Ready with dead DB = %d, want 503", w.Code)
	}
}

func TestReady_NilPingerSkipsDBCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(readyManifestService(t), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/ready", nil)
	h.Ready(c)
	if w.Code != 200 {
		t.Fatalf("Ready degraded-mode = %d, want 200", w.Code)
	}
}

func TestManifestStatus_ExposesOnlyReadinessAndVersion(t *testing.T) {
	h := NewHealthHandler(readyManifestService(t), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/manifest/status", nil)
	h.ManifestStatus(c)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 2 {
		t.Fatalf("manifest status keys = %v, want exactly ready/version", body)
	}
	if _, ok := body["ready"]; !ok {
		t.Fatal("manifest status missing ready")
	}
	if _, ok := body["version"]; !ok {
		t.Fatal("manifest status missing version")
	}
}
