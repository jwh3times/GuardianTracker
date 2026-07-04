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

// writeZipFile writes zip bytes to a temp file and returns its path —
// extractManifest now reads the zip from disk rather than from a byte slice.
func writeZipFile(t *testing.T, zipData []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.zip")
	if err := os.WriteFile(path, zipData, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractManifest_WritesContentFileAndRunsHooks(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "manifest.sqlite") // nested dir is created
	ms := NewManifestService(NewClient("k", "http://x", 100, 100), dbPath, time.Hour)

	var beforeRan bool
	ms.RegisterSwapHooks(func() { beforeRan = true }, nil)

	zipData := zipWith(t, "world_en.content", []byte("SQLITE-BYTES"))
	zipPath := writeZipFile(t, zipData)
	if err := ms.extractManifest(zipPath); err != nil {
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

	badZipPath := filepath.Join(t.TempDir(), "not-a-zip")
	if err := os.WriteFile(badZipPath, []byte("not a zip"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ms.extractManifest(badZipPath); err == nil {
		t.Error("expected error for invalid zip data")
	}
	// Valid zip but no .content member.
	zipData := zipWith(t, "readme.txt", []byte("hi"))
	zipPath := writeZipFile(t, zipData)
	if err := ms.extractManifest(zipPath); err == nil {
		t.Error("expected error when no .content file is present")
	}
}

func TestExtractManifest_MissingZipFile(t *testing.T) {
	ms := NewManifestService(NewClient("k", "http://x", 100, 100), filepath.Join(t.TempDir(), "m.sqlite"), time.Hour)
	if err := ms.extractManifest(filepath.Join(t.TempDir(), "does-not-exist.zip")); err == nil {
		t.Error("expected error when the zip file does not exist")
	}
}

func TestDownload_HappyPath_UsesCDNBaseURLAndRunsHooks(t *testing.T) {
	zipData := zipWith(t, "world_en.content", []byte("SQLITE-BYTES"))
	var cdnPathRequested string
	cdnSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cdnPathRequested = r.URL.Path
		if got := r.Header.Get("X-API-Key"); got != "k" {
			t.Errorf("X-API-Key on CDN download = %q, want k", got)
		}
		w.Write(zipData)
	}))
	defer cdnSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"version":"v-cdn-test",
			"mobileWorldContentPaths":{"en":"/common/destiny2_content/sqlite/en/world.content"}}}`)
	}))
	defer apiSrv.Close()

	client := NewClient("k", apiSrv.URL, 100, 100)
	client.SetCDNBaseURL(cdnSrv.URL)

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")
	ms := NewManifestService(client, dbPath, time.Hour)

	var afterVersion string
	ms.RegisterSwapHooks(nil, func(v string) { afterVersion = v })

	if err := ms.Download(context.Background()); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if cdnPathRequested != "/common/destiny2_content/sqlite/en/world.content" {
		t.Errorf("cdn path requested = %q", cdnPathRequested)
	}
	got, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read installed db: %v", err)
	}
	if string(got) != "SQLITE-BYTES" {
		t.Errorf("installed db = %q", got)
	}
	if afterVersion != "v-cdn-test" {
		t.Errorf("after-swap hook version = %q, want v-cdn-test", afterVersion)
	}
	if ms.Version() != "v-cdn-test" {
		t.Errorf("Version() = %q", ms.Version())
	}
	// The temp zip file must not be left behind.
	if _, err := os.Stat(dbPath + ".zip.tmp"); !os.IsNotExist(err) {
		t.Errorf("zip.tmp should be removed after extraction, stat err = %v", err)
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
