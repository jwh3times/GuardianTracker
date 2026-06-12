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

// requireSQLite skips when the sqlite3 driver is a CGO-disabled stub (local
// Windows without a C toolchain). CI runs these on Linux with cgo enabled.
func requireSQLite(t *testing.T) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Skipf("sqlite3 driver unavailable: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("sqlite3 driver unavailable (CGO disabled?): %v", err)
	}
}

// writeManifestDB creates a minimal manifest SQLite file with the rows needed to
// exercise BuildIndex, plus the sibling version file the ManifestService reads.
func writeManifestDB(t *testing.T) string {
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
	rows := []string{
		// weapon → "Hand Cannon"
		`{"hash":1,"displayProperties":{"name":"Fatebringer","icon":"/fb.png"},"itemType":3,"itemSubType":9,"inventory":{"tierType":5}}`,
		// armor → "Helmet"
		`{"hash":2,"displayProperties":{"name":"Helm of Saint"},"itemType":2,"inventory":{"tierType":6},"equippingBlock":{"equipmentSlotTypeHash":3448274439}}`,
		// emblem → "Emblem"
		`{"hash":3,"displayProperties":{"name":"Cool Emblem"},"itemType":14}`,
		// ship → "Ship"
		`{"hash":4,"displayProperties":{"name":"Star Ship"},"itemType":21}`,
		// excluded type (Mod=19) — must not be indexed
		`{"hash":5,"displayProperties":{"name":"Targeting Mod"},"itemType":19}`,
		// included type but empty name — skipped
		`{"hash":6,"displayProperties":{"name":""},"itemType":3,"itemSubType":6}`,
		// malformed JSON — skipped without aborting the build
		`{not json`,
	}
	for i, blob := range rows {
		if _, err := db.Exec(`INSERT INTO DestinyInventoryItemDefinition (id, json) VALUES (?, ?)`, i, blob); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	db.Close()

	versionPath := filepath.Join(dir, "manifest_version.txt")
	if err := os.WriteFile(versionPath, []byte("v-test"), 0644); err != nil {
		t.Fatalf("write version: %v", err)
	}
	return dbPath
}

func TestBuildIndex_IndexesIncludedTypes(t *testing.T) {
	requireSQLite(t)
	dbPath := writeManifestDB(t)
	ms := bungie.NewManifestService(bungie.NewClient("k", "http://unused", 100, 100), dbPath, time.Hour)
	if ms.Version() != "v-test" {
		t.Fatalf("manifest version = %q, want v-test", ms.Version())
	}

	svc := NewService(ms, dbPath)
	if svc.IsReady() {
		t.Error("IsReady should be false before BuildIndex")
	}

	svc.BuildIndex()

	if !svc.IsReady() {
		t.Fatal("IsReady should be true after BuildIndex")
	}

	// Weapon resolves its sub-type name.
	res := svc.Search("fate", 10)
	if len(res) != 1 || res[0].Type != "Hand Cannon" || res[0].Rarity != "Legendary" {
		t.Errorf("search 'fate' = %+v", res)
	}
	// Armor resolves its slot name.
	if res := svc.Search("helm", 10); len(res) != 1 || res[0].Type != "Helmet" {
		t.Errorf("search 'helm' = %+v", res)
	}
	// Excluded mod type is absent.
	if res := svc.Search("targeting", 10); len(res) != 0 {
		t.Errorf("mod should not be indexed, got %+v", res)
	}
	// Empty-name and malformed rows were skipped → exactly 4 indexed entries.
	if got := svc.Search("", 50); len(got) != 4 {
		t.Errorf("indexed entries = %d, want 4", len(got))
	}
}

func TestBuildIndex_NoManifestServiceIsNoOp(t *testing.T) {
	svc := NewService(nil, "")
	svc.BuildIndex() // must not panic
	if svc.IsReady() {
		t.Error("IsReady should stay false with no manifest service")
	}
}

func TestItemTypeName(t *testing.T) {
	mk := func(itemType, subType int, slot uint32) bungie.InventoryItemDefinition {
		d := bungie.InventoryItemDefinition{ItemType: itemType, ItemSubType: subType}
		d.EquippingBlock.EquipmentSlotTypeHash = slot
		return d
	}
	cases := []struct {
		name string
		def  bungie.InventoryItemDefinition
		want string
	}{
		{"weapon", mk(3, bungie.WeaponSubTypeBow, 0), "Bow"},
		{"armor", mk(2, 0, bungie.SlotLegs), "Leg Armor"},
		{"emblem", mk(14, 0, 0), "Emblem"},
		{"ship", mk(21, 0, 0), "Ship"},
		{"sparrow", mk(22, 0, 0), "Sparrow"},
		{"emote", mk(23, 0, 0), "Emote"},
		{"ghost", mk(24, 0, 0), "Ghost"},
		{"default", mk(99, 0, 0), "Item"},
	}
	for _, tc := range cases {
		if got := itemTypeName(tc.def); got != tc.want {
			t.Errorf("itemTypeName(%s) = %q, want %q", tc.name, got, tc.want)
		}
	}
}
