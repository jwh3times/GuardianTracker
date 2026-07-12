package search

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"guardian-tracker/api-service/services/bungie"
)

func snapshotEntries() []Entry {
	return []Entry{
		{Hash: 1, Name: "Hawkmoon", Icon: "/hawkmoon.png", Type: "Hand Cannon", Rarity: "Exotic", nameLower: "hawkmoon"},
		{Hash: 2, Name: "Vex Mythoclast", Icon: "/vex.png", Type: "Fusion Rifle", Rarity: "Exotic", nameLower: "vex mythoclast"},
	}
}

func writeSnapshotFile(t *testing.T, path string, snapshot indexSnapshot) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	writer := gzip.NewWriter(file)
	if err := json.NewEncoder(writer).Encode(snapshot); err != nil {
		file.Close()
		t.Fatalf("encode snapshot: %v", err)
	}
	if err := writer.Close(); err != nil {
		file.Close()
		t.Fatalf("close gzip: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close snapshot: %v", err)
	}
}

func TestSnapshotRoundTripAndStartupRestore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")
	entries := snapshotEntries()
	writer := &Service{snapshotDir: dir}
	if err := writer.saveSnapshot("v/1", entries); err != nil {
		t.Fatalf("saveSnapshot: %v", err)
	}

	versionPath := filepath.Join(dir, "manifest_version.txt")
	if err := os.WriteFile(versionPath, []byte("v/1"), 0644); err != nil {
		t.Fatalf("write version: %v", err)
	}
	ms := bungie.NewManifestService(bungie.NewClient("k", "http://unused", 100, 100), dbPath, time.Hour)
	svc := NewService(ms, dbPath)
	if !svc.IsReady() {
		t.Fatal("matching snapshot should make the service ready")
	}
	results := svc.Search("HAWK", 10)
	if len(results) != 1 || results[0].Name != "Hawkmoon" {
		t.Fatalf("restored search results = %+v", results)
	}
	if results[0].Icon != "/hawkmoon.png" || results[0].nameLower != "hawkmoon" {
		t.Fatalf("restored entry metadata = %+v", results[0])
	}
}

func TestSnapshotRejectsStaleAndCorruptFiles(t *testing.T) {
	dir := t.TempDir()
	svc := &Service{snapshotDir: dir}
	path := svc.snapshotPath("v2")

	writeSnapshotFile(t, path, indexSnapshot{
		Format:  searchSnapshotFormat,
		Version: "v1",
		Entries: snapshotEntries(),
	})
	if svc.loadSnapshot("v2") {
		t.Fatal("stale snapshot should not load")
	}

	if err := os.WriteFile(path, []byte("not gzip"), 0644); err != nil {
		t.Fatalf("write corrupt snapshot: %v", err)
	}
	if svc.loadSnapshot("v2") {
		t.Fatal("corrupt snapshot should not load")
	}
	if svc.IsReady() {
		t.Fatal("rejected snapshot should not mark the service ready")
	}
}

func TestSaveSnapshotPrunesOlderVersions(t *testing.T) {
	dir := t.TempDir()
	svc := &Service{snapshotDir: dir}
	if err := svc.saveSnapshot("v1", snapshotEntries()); err != nil {
		t.Fatalf("save v1: %v", err)
	}
	if err := svc.saveSnapshot("v2", snapshotEntries()); err != nil {
		t.Fatalf("save v2: %v", err)
	}

	if _, err := os.Stat(svc.snapshotPath("v1")); !os.IsNotExist(err) {
		t.Fatalf("v1 snapshot still exists, stat error = %v", err)
	}
	if _, err := os.Stat(svc.snapshotPath("v2")); err != nil {
		t.Fatalf("v2 snapshot missing: %v", err)
	}
}

func TestSanitizeVersionKeepsSnapshotInsideDirectory(t *testing.T) {
	svc := &Service{snapshotDir: `C:\data`}
	path := svc.snapshotPath(`../release/v1`)
	if filepath.Dir(path) != `C:\data` {
		t.Fatalf("snapshot path escaped directory: %q", path)
	}
	if filepath.Base(path) != "search_index_release_v1.json.gz" {
		t.Fatalf("snapshot filename = %q", filepath.Base(path))
	}
}
