package search

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"guardian-tracker/api-service/services/bungie"

	_ "github.com/mattn/go-sqlite3"
)

// writeLargeManifestDB creates a manifest whose item table is big enough that a
// full BuildIndex scan is still running when the test calls CloseForSwap. The
// rows are inserted in one transaction to keep fixture setup fast.
func writeLargeManifestDB(t *testing.T, rowCount int) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE DestinyInventoryItemDefinition (id INTEGER PRIMARY KEY, json TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO DestinyInventoryItemDefinition (id, json) VALUES (?, ?)`)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	for i := range rowCount {
		blob := fmt.Sprintf(
			`{"hash":%d,"displayProperties":{"name":"Item %d"},"itemType":3,"itemSubType":9,"inventory":{"tierType":5}}`,
			i+1, i+1,
		)
		if _, err := stmt.Exec(i, blob); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	db.Close()

	if err := os.WriteFile(filepath.Join(dir, "manifest_version.txt"), []byte("v-old"), 0644); err != nil {
		t.Fatalf("write version: %v", err)
	}
	return dbPath
}

func newTestService(t *testing.T, dbPath string) *Service {
	t.Helper()
	ms := bungie.NewManifestService(bungie.NewClient("k", "http://unused", 100, 100), dbPath, time.Hour)
	return NewService(ms, dbPath)
}

// waitFor polls cond until it holds or the deadline passes.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func (s *Service) isBuilding() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.building
}

// The regression test for the swap race: the search service opens its own SQLite
// handle on the manifest, so an in-flight BuildIndex used to hold that handle
// across os.Rename — failing the rename on Windows (the app then silently keeps
// serving the old manifest) and leaving Linux reading a deleted inode.
// CloseForSwap must not return until the build has released the handle.
func TestCloseForSwap_WaitsForInFlightBuild(t *testing.T) {
	requireSQLite(t)
	dbPath := writeLargeManifestDB(t, 30000)
	svc := newTestService(t, dbPath)

	go svc.BuildIndex()
	waitFor(t, "build to start", svc.isBuilding)

	svc.CloseForSwap()

	if svc.isBuilding() {
		t.Fatal("CloseForSwap returned while a build was still running — the manifest handle is still open")
	}

	// With no handle outstanding, the swap's rename succeeds. On Linux this
	// passes either way; on Windows it is the actual failure the bug produced.
	newPath := dbPath + ".new"
	if err := os.WriteFile(newPath, []byte("replacement"), 0644); err != nil {
		t.Fatalf("write replacement: %v", err)
	}
	if err := os.Rename(newPath, dbPath); err != nil {
		t.Fatalf("rename over manifest failed after CloseForSwap: %v", err)
	}
}

func TestBuildIndex_RefusesWhileSwapping(t *testing.T) {
	requireSQLite(t)
	dbPath := writeManifestDB(t)
	svc := newTestService(t, dbPath)

	svc.CloseForSwap()
	svc.BuildIndex()

	if svc.IsReady() {
		t.Error("BuildIndex built an index during a manifest swap; it must wait for Reopen")
	}

	// Reopen re-admits builds. It deliberately does not build on its own — the
	// OnVersionChanged kicks BuildIndex once the new version is installed.
	svc.Reopen()
	svc.BuildIndex()

	if !svc.IsReady() {
		t.Fatal("BuildIndex did not build after Reopen")
	}
	if res := svc.Search("fate", 10); len(res) != 1 {
		t.Errorf("search after Reopen = %+v, want 1 result", res)
	}
}

// An aborted build must not stamp builtVersion, or the index would be
// permanently wrong for that version: ensureIndex only rebuilds when
// builtVersion differs from the manifest's current version.
func TestBuildIndex_AbortedBuildDoesNotStampVersion(t *testing.T) {
	requireSQLite(t)
	dbPath := writeLargeManifestDB(t, 30000)
	svc := newTestService(t, dbPath)

	go svc.BuildIndex()
	waitFor(t, "build to start", svc.isBuilding)
	svc.CloseForSwap()

	if svc.IsReady() {
		// The build finished before the swap was requested — a legitimate race,
		// and not the case this test is about.
		t.Skip("build completed before CloseForSwap; timing did not exercise the abort path")
	}
	if v := svc.builtVersion; v != "" {
		t.Errorf("aborted build stamped builtVersion = %q, want empty", v)
	}
}
