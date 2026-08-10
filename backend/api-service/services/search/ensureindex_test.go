package search

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"guardian-tracker/api-service/services/bungie"

	_ "github.com/mattn/go-sqlite3"
)

// writeVersionFile writes the sibling version file ManifestService reads, so a
// service constructed over dir reports a non-empty Version.
func writeVersionFile(t *testing.T, dir, version string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "manifest_version.txt"), []byte(version), 0644); err != nil {
		t.Fatalf("write version: %v", err)
	}
}

// versionOnlyService returns a service whose manifest version is known but whose
// database file does not exist. That is enough for the build-scheduling
// decisions, which never open SQLite.
func versionOnlyService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	writeVersionFile(t, dir, "v-test")
	dbPath := filepath.Join(dir, "manifest.sqlite")
	ms := bungie.NewManifestService(bungie.NewClient("k", "http://unused", 100, 100), dbPath, time.Hour)
	return NewService(ms, dbPath)
}

// The cold-start regression. If the very first BuildIndex failed, builtVersion
// stayed empty and SearchHandler returned 503 before Search — the only caller of
// the retry path — ever ran, so item search stayed dead until the next hourly
// manifest swap. EnsureIndex must recover the index once the cause clears.
func TestEnsureIndex_RecoversFromFailedColdStartBuild(t *testing.T) {
	requireSQLite(t)
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")
	writeVersionFile(t, dir, "v-test")

	// A manifest file with no item table: the build fails at the query and
	// leaves the index unbuilt, exactly as a truncated download would.
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE Unrelated (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	db.Close()

	svc := newTestService(t, dbPath)
	svc.BuildIndex()
	if svc.IsReady() {
		t.Fatal("build against a manifest with no item table should have failed")
	}

	// Whatever broke the first build is gone — here, the table now exists.
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE DestinyInventoryItemDefinition (id INTEGER PRIMARY KEY, json TEXT)`); err != nil {
		t.Fatalf("create item table: %v", err)
	}
	blob := `{"hash":1,"displayProperties":{"name":"Fatebringer"},"itemType":3,"itemSubType":9,"inventory":{"tierType":5}}`
	if _, err := db.Exec(`INSERT INTO DestinyInventoryItemDefinition (id, json) VALUES (0, ?)`, blob); err != nil {
		t.Fatalf("insert: %v", err)
	}
	db.Close()

	svc.EnsureIndex()

	waitFor(t, "the failed cold-start build to be retried", svc.IsReady)
	if res := svc.Search("fate", 10); len(res) != 1 {
		t.Errorf("search after recovery = %+v, want 1 result", res)
	}
}

// A failing build must not be retried once per request. shouldKickBuild carries
// the decision so it can be checked without spawning builds.
func TestShouldKickBuild_ThrottlesRetries(t *testing.T) {
	svc := versionOnlyService(t)
	now := time.Unix(1_700_000_000, 0).UTC()
	svc.now = func() time.Time { return now }

	if !svc.shouldKickBuild() {
		t.Fatal("first call must kick a build — the index has never been built")
	}
	if svc.shouldKickBuild() {
		t.Error("a second call inside the retry interval must not kick another build")
	}
	now = now.Add(buildRetryInterval - time.Second)
	if svc.shouldKickBuild() {
		t.Error("still inside the retry interval")
	}
	now = now.Add(2 * time.Second)
	if !svc.shouldKickBuild() {
		t.Error("past the retry interval, a failed build must be retried")
	}
}

func TestShouldKickBuild_SkipsWhenNoBuildIsWanted(t *testing.T) {
	t.Run("index already current for this manifest version", func(t *testing.T) {
		svc := versionOnlyService(t)
		svc.builtVersion = "v-test"
		if svc.shouldKickBuild() {
			t.Error("kicked a rebuild for a version that is already indexed")
		}
	})

	t.Run("build already running", func(t *testing.T) {
		svc := versionOnlyService(t)
		svc.building = true
		if svc.shouldKickBuild() {
			t.Error("kicked a second concurrent build")
		}
	})

	t.Run("manifest swap in progress", func(t *testing.T) {
		// BuildIndex refuses to run mid-swap, so kicking one here would burn the
		// retry budget on a guaranteed no-op and delay the real retry.
		svc := versionOnlyService(t)
		svc.CloseForSwap()
		if svc.shouldKickBuild() {
			t.Error("kicked a build during a manifest swap")
		}
	})

	t.Run("no manifest version yet", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "manifest.sqlite")
		ms := bungie.NewManifestService(bungie.NewClient("k", "http://unused", 100, 100), dbPath, time.Hour)
		svc := NewService(ms, dbPath)
		if svc.shouldKickBuild() {
			t.Error("kicked a build before any manifest was installed")
		}
	})

	t.Run("no manifest service", func(t *testing.T) {
		svc := NewService(nil, "")
		if svc.shouldKickBuild() {
			t.Error("kicked a build with no manifest service")
		}
	})
}
