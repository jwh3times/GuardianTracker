package bungie

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManifestService_ReadsVersionFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")
	if err := os.WriteFile(filepath.Join(dir, "manifest_version.txt"), []byte(" v9 \n"), 0644); err != nil {
		t.Fatal(err)
	}
	ms := NewManifestService(NewClient("k", "http://x", 100, 100), dbPath, time.Hour)
	if ms.Version() != "v9" {
		t.Errorf("Version() = %q, want v9 (trimmed)", ms.Version())
	}
	if ms.GetCurrentVersion() != "v9" {
		t.Errorf("GetCurrentVersion() = %q", ms.GetCurrentVersion())
	}
	if ms.GetDBPath() != dbPath {
		t.Errorf("GetDBPath() = %q, want %q", ms.GetDBPath(), dbPath)
	}
}

func TestManifestService_IsReady(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")
	ms := NewManifestService(NewClient("k", "http://x", 100, 100), dbPath, time.Hour)
	if ms.IsReady() {
		t.Error("IsReady should be false before the db file exists")
	}
	if err := os.WriteFile(dbPath, []byte("db"), 0644); err != nil {
		t.Fatal(err)
	}
	if !ms.IsReady() {
		t.Error("IsReady should be true once the db file exists")
	}
}

func TestRegisterSwapHooks_RunsRegisteredHooks(t *testing.T) {
	ms := NewManifestService(NewClient("k", "http://x", 100, 100), "db.sqlite", time.Hour)
	var before int
	var afterVersion string
	ms.RegisterSwapHooks(func() { before++ }, func(v string) { afterVersion = v })
	// nil hooks must be ignored without panicking.
	ms.RegisterSwapHooks(nil, nil)

	ms.runBeforeSwapHooks()
	ms.runAfterSwapHooks("v2")
	if before != 1 {
		t.Errorf("before hook ran %d times, want 1", before)
	}
	if afterVersion != "v2" {
		t.Errorf("after hook version = %q, want v2", afterVersion)
	}
}

func TestCheckForUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"version":"new-version",
			"mobileWorldContentPaths":{"en":"/world.content"}}}`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	ms := NewManifestService(NewClient("k", srv.URL, 100, 100), filepath.Join(dir, "m.sqlite"), time.Hour)

	// No current version → update needed.
	needs, version, err := ms.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate: %v", err)
	}
	if !needs || version != "new-version" {
		t.Errorf("needs=%v version=%q, want true/new-version", needs, version)
	}

	// Same version already installed → no update.
	ms.currentVersion = "new-version"
	if needs, _, _ := ms.CheckForUpdate(context.Background()); needs {
		t.Error("no update should be needed when versions match")
	}
}

func TestCheckForUpdate_PropagatesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":5,"ErrorStatus":"SystemDisabled"}`)
	}))
	defer srv.Close()

	ms := NewManifestService(NewClient("k", srv.URL, 100, 100), "m.sqlite", time.Hour)
	if _, _, err := ms.CheckForUpdate(context.Background()); err == nil {
		t.Fatal("expected error when Bungie returns ErrorCode 5")
	}
}

func zipWith(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractManifest_WritesContentFileAndRunsHooks(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "manifest.sqlite") // nested dir is created
	ms := NewManifestService(NewClient("k", "http://x", 100, 100), dbPath, time.Hour)

	var beforeRan bool
	ms.RegisterSwapHooks(func() { beforeRan = true }, nil)

	zipData := zipWith(t, "world_en.content", []byte("SQLITE-BYTES"))
	if err := ms.extractManifest(zipData); err != nil {
		t.Fatalf("extractManifest: %v", err)
	}
	if !beforeRan {
		t.Error("before-swap hook should run before the file rename")
	}
	got, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read installed db: %v", err)
	}
	if string(got) != "SQLITE-BYTES" {
		t.Errorf("installed db = %q", got)
	}
}

func TestExtractManifest_Errors(t *testing.T) {
	ms := NewManifestService(NewClient("k", "http://x", 100, 100), filepath.Join(t.TempDir(), "m.sqlite"), time.Hour)

	if err := ms.extractManifest([]byte("not a zip")); err == nil {
		t.Error("expected error for invalid zip data")
	}
	// Valid zip but no .content member.
	zipData := zipWith(t, "readme.txt", []byte("hi"))
	if err := ms.extractManifest(zipData); err == nil {
		t.Error("expected error when no .content file is present")
	}
}

func TestEnsureReady_ReadyAndWithinInterval_NoNetwork(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")
	if err := os.WriteFile(dbPath, []byte("db"), 0644); err != nil {
		t.Fatal(err)
	}
	// Point the client at a server that fails the test if it's ever called.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("EnsureReady should not hit the network when fresh")
	}))
	defer srv.Close()

	ms := NewManifestService(NewClient("k", srv.URL, 100, 100), dbPath, time.Hour)
	ms.lastCheck = time.Now() // fresh → skip the update check
	if err := ms.EnsureReady(context.Background()); err != nil {
		t.Fatalf("EnsureReady: %v", err)
	}
}
