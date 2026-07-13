package e2efixture

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	manifestrepo "guardian-tracker/api-service/services/manifest"

	_ "github.com/mattn/go-sqlite3"
)

func TestGenerateManifestZipIsDeterministic(t *testing.T) {
	first, err := GenerateManifestZip()
	if err != nil {
		t.Fatalf("GenerateManifestZip first: %v", err)
	}
	second, err := GenerateManifestZip()
	if err != nil {
		t.Fatalf("GenerateManifestZip second: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("manifest archives differ between identical generations")
	}
}

func TestGenerateManifestZipContainsAllProductionTables(t *testing.T) {
	archive, err := GenerateManifestZip()
	if err != nil {
		t.Fatalf("GenerateManifestZip: %v", err)
	}
	dbPath := extractManifestDB(t, archive)

	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name")
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	var got []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatalf("scan table: %v", err)
		}
		got = append(got, name)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close table rows: %v", err)
	}
	want := ManifestTables()
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Fatalf("tables = %v, want %v", got, want)
	}

	for _, table := range want {
		var count int
		if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count == 0 {
			t.Errorf("%s has no deterministic fixture rows", table)
		}
	}

	var blob string
	if err := db.QueryRow("SELECT json FROM DestinyVendorDefinition WHERE id = ?", signedHash(XurVendorHash)).Scan(&blob); err != nil {
		t.Fatalf("query signed Xur hash: %v", err)
	}
	var vendor struct {
		Hash uint32 `json:"hash"`
	}
	if err := json.Unmarshal([]byte(blob), &vendor); err != nil {
		t.Fatalf("decode Xur: %v", err)
	}
	if vendor.Hash != XurVendorHash {
		t.Fatalf("Xur hash = %d, want %d", vendor.Hash, XurVendorHash)
	}
}

func TestGeneratedManifestExercisesProductionReaders(t *testing.T) {
	archive, err := GenerateManifestZip()
	if err != nil {
		t.Fatalf("GenerateManifestZip: %v", err)
	}
	dbPath := extractManifestDB(t, archive)
	repo, err := manifestrepo.NewRepository(dbPath)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	defer repo.Close()

	collectibles, err := repo.GetAllCollectiblesWithItems()
	if err != nil {
		t.Fatalf("GetAllCollectiblesWithItems: %v", err)
	}
	if len(collectibles) != 7 {
		t.Fatalf("collectibles = %d, want 7", len(collectibles))
	}

	perks, err := repo.GetWeaponPerks(fatebringerHash)
	if err != nil {
		t.Fatalf("GetWeaponPerks: %v", err)
	}
	if len(perks) != 2 || perks[0].Label != "Barrel" || perks[1].Label != "Trait 1" {
		t.Fatalf("Fatebringer perks = %+v", perks)
	}

	links, err := repo.GetCatalystLinks()
	if err != nil {
		t.Fatalf("GetCatalystLinks: %v", err)
	}
	if len(links) != 1 || links[0].WeaponName != "Sunshot" || len(links[0].ObjectiveHashes) != 1 || links[0].ObjectiveHashes[0] != 7101 {
		t.Fatalf("catalyst links = %+v", links)
	}
	if len(links[0].Catalysts) != 1 || links[0].Catalysts[0].Description != "Increases range and stability." {
		t.Fatalf("Sunshot catalyst = %+v", links[0].Catalysts)
	}

	destinationHash, destinationName, err := repo.ResolveVendorLocation(XurVendorHash, 0)
	if err != nil {
		t.Fatalf("ResolveVendorLocation: %v", err)
	}
	if destinationHash != towerDestinationHash || destinationName != "The Tower" {
		t.Fatalf("Xur location = %d/%q", destinationHash, destinationName)
	}
}

func extractManifestDB(t *testing.T, archive []byte) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	if len(zr.File) != 1 || zr.File[0].Name != ManifestContentName {
		t.Fatalf("zip entries = %+v, want only %q", zr.File, ManifestContentName)
	}
	entry, err := zr.File[0].Open()
	if err != nil {
		t.Fatalf("open content entry: %v", err)
	}
	defer entry.Close()
	dbPath := filepath.Join(t.TempDir(), ManifestContentName)
	out, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("create extracted DB: %v", err)
	}
	if _, err := io.Copy(out, entry); err != nil {
		out.Close()
		t.Fatalf("extract DB: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close extracted DB: %v", err)
	}
	return dbPath
}
