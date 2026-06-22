package manifest

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"guardian-tracker/api-service/services/bungie"

	_ "github.com/mattn/go-sqlite3"
)

// requireSQLite skips the test when the sqlite3 driver is a CGO-disabled stub
// (e.g. local Windows without a C toolchain). CI runs these on Linux with cgo.
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

// writeFixtureDB creates a minimal manifest SQLite database at path.
func writeFixtureDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	defer db.Close()

	for _, ddl := range []string{
		`CREATE TABLE DestinyInventoryItemDefinition (id INTEGER PRIMARY KEY, json TEXT)`,
		`CREATE TABLE DestinyCollectibleDefinition (id INTEGER PRIMARY KEY, json TEXT)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("fixture ddl: %v", err)
		}
	}

	items := map[uint32]string{
		// Legendary hand cannon
		100: `{"hash":100,"displayProperties":{"name":"Fatebringer","icon":"/icons/fb.png"},
			"itemType":3,"itemSubType":9,"inventory":{"tierType":5}}`,
		// Exotic rocket launcher
		200: `{"hash":200,"displayProperties":{"name":"Gjallarhorn","icon":"/icons/gj.png"},
			"itemType":3,"itemSubType":10,"inventory":{"tierType":6}}`,
		// Legendary armor
		300: `{"hash":300,"displayProperties":{"name":"Helm of Tests"},
			"itemType":2,"inventory":{"tierType":5},"equippingBlock":{"equipmentSlotTypeHash":3448274439}}`,
		// Cosmetic (ship, itemType 21)
		400: `{"hash":400,"displayProperties":{"name":"Test Ship"},"itemType":21,"inventory":{"tierType":5}}`,
	}
	for hash, blob := range items {
		if _, err := db.Exec(`INSERT INTO DestinyInventoryItemDefinition (id, json) VALUES (?, ?)`, int32(hash), blob); err != nil {
			t.Fatalf("fixture item %d: %v", hash, err)
		}
	}

	collectibles := map[uint32]string{
		1000: `{"hash":1000,"itemHash":100,"sourceString":"Vault of Glass raid","displayProperties":{"name":"Fatebringer"}}`,
		2000: `{"hash":2000,"itemHash":200,"sourceString":"Exotic quest","displayProperties":{"name":"Gjallarhorn"}}`,
		3000: `{"hash":3000,"itemHash":300,"sourceString":"World drops","displayProperties":{"name":"Helm of Tests"}}`,
		4000: `{"hash":4000,"itemHash":400,"sourceString":"Eververse","displayProperties":{"name":"Test Ship"}}`,
	}
	for hash, blob := range collectibles {
		if _, err := db.Exec(`INSERT INTO DestinyCollectibleDefinition (id, json) VALUES (?, ?)`, int32(hash), blob); err != nil {
			t.Fatalf("fixture collectible %d: %v", hash, err)
		}
	}

	if _, err := db.Exec(`CREATE TABLE DestinyPresentationNodeDefinition (id INTEGER PRIMARY KEY, json TEXT)`); err != nil {
		t.Fatalf("fixture pnode ddl: %v", err)
	}
	nodes := map[uint32]string{
		10: `{"hash":10,"displayProperties":{"name":"Weapons"},"children":{"presentationNodes":[{"presentationNodeHash":11}],"collectibles":[]}}`,
		11: `{"hash":11,"displayProperties":{"name":"Hand Cannons"},"children":{"presentationNodes":[],"collectibles":[{"collectibleHash":1000}]}}`,
	}
	for hash, blob := range nodes {
		if _, err := db.Exec(`INSERT INTO DestinyPresentationNodeDefinition (id, json) VALUES (?, ?)`, int32(hash), blob); err != nil {
			t.Fatalf("fixture pnode %d: %v", hash, err)
		}
	}
}

func fixtureRepo(t *testing.T) (*Repository, string) {
	t.Helper()
	requireSQLite(t)
	path := filepath.Join(t.TempDir(), "manifest.sqlite")
	writeFixtureDB(t, path)
	repo, err := NewRepository(path)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo, path
}

func TestRepository_GetItemsByHashes(t *testing.T) {
	repo, _ := fixtureRepo(t)
	defs, err := repo.GetItemsByHashes([]uint32{100, 200, 999})
	if err != nil {
		t.Fatalf("GetItemsByHashes: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("len = %d, want 2 (unknown hash omitted)", len(defs))
	}
	if defs[100].DisplayProperties.Name != "Fatebringer" {
		t.Errorf("item 100 = %+v", defs[100])
	}
	if defs[200].Inventory.TierType != bungie.TierTypeExotic {
		t.Errorf("item 200 tier = %d", defs[200].Inventory.TierType)
	}
}

func TestRepository_GetCollectiblesByItemHashes(t *testing.T) {
	repo, _ := fixtureRepo(t)
	cols, err := repo.GetCollectiblesByItemHashes([]uint32{100, 400, 999})
	if err != nil {
		t.Fatalf("GetCollectiblesByItemHashes: %v", err)
	}
	if len(cols) != 2 {
		t.Fatalf("len = %d, want 2", len(cols))
	}
	if cols[100].SourceString != "Vault of Glass raid" {
		t.Errorf("source for 100 = %q", cols[100].SourceString)
	}
}

func TestRepository_GetWeaponTypesByName(t *testing.T) {
	repo, _ := fixtureRepo(t)
	types, err := repo.GetWeaponTypesByName()
	if err != nil {
		t.Fatalf("GetWeaponTypesByName: %v", err)
	}
	if got := types["fatebringer"]; got != bungie.GetWeaponTypeName(9) {
		t.Errorf("fatebringer type = %q", got)
	}
	if _, ok := types["helm of tests"]; ok {
		t.Error("armor leaked into weapon types map")
	}
}

func TestRepository_Reconnect(t *testing.T) {
	repo, _ := fixtureRepo(t)
	if err := repo.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := repo.Reconnect(); err != nil {
		t.Fatalf("Reconnect: %v", err)
	}
	if _, err := repo.GetItemsByHashes([]uint32{100}); err != nil {
		t.Fatalf("query after Reconnect: %v", err)
	}
}

func TestPresentationNodeDef_ParsesCollectibles(t *testing.T) {
	blob := `{"hash":7,"displayProperties":{"name":"Hand Cannons","icon":"/i/hc.png"},
		"children":{"presentationNodes":[{"presentationNodeHash":8}],
		"collectibles":[{"collectibleHash":1000},{"collectibleHash":2000}]}}`
	var def PresentationNodeDef
	if err := json.Unmarshal([]byte(blob), &def); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(def.Children.Collectibles) != 2 || def.Children.Collectibles[0].CollectibleHash != 1000 {
		t.Fatalf("collectibles = %+v", def.Children.Collectibles)
	}
	if def.Children.PresentationNodes[0].PresentationNodeHash != 8 {
		t.Fatalf("child node = %+v", def.Children.PresentationNodes)
	}
}

func TestRepository_GetAllPresentationNodes(t *testing.T) {
	repo, _ := fixtureRepo(t)
	nodes, err := repo.GetAllPresentationNodes()
	if err != nil {
		t.Fatalf("GetAllPresentationNodes: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("len = %d, want 2", len(nodes))
	}
	if nodes[11].Children.Collectibles[0].CollectibleHash != 1000 {
		t.Errorf("node 11 collectibles = %+v", nodes[11].Children.Collectibles)
	}
	if nodes[10].Children.PresentationNodes[0].PresentationNodeHash != 11 {
		t.Errorf("node 10 children = %+v", nodes[10].Children.PresentationNodes)
	}
}

func TestCollectibleCategory(t *testing.T) {
	mk := func(itemType, tier int) *bungie.InventoryItemDefinition {
		d := &bungie.InventoryItemDefinition{ItemType: itemType}
		d.Inventory.TierType = tier
		return d
	}
	cases := []struct {
		name string
		item *bungie.InventoryItemDefinition
		want string
	}{
		{"legendary weapon", mk(bungie.ItemTypeWeapon, bungie.TierTypeLegendary), "weapons"},
		{"exotic weapon", mk(bungie.ItemTypeWeapon, bungie.TierTypeExotic), "exotics"},
		{"legendary armor", mk(bungie.ItemTypeArmor, bungie.TierTypeLegendary), "armor"},
		{"exotic armor", mk(bungie.ItemTypeArmor, bungie.TierTypeExotic), "exotics"},
		{"ship cosmetic", mk(21, bungie.TierTypeLegendary), "cosmetics"},
		{"ghost cosmetic", mk(24, bungie.TierTypeRare), "cosmetics"},
		{"mod (uncategorized)", mk(19, bungie.TierTypeCommon), ""},
		{"nil", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CollectibleCategory(tc.item); got != tc.want {
				t.Errorf("CollectibleCategory = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProvider_LazyOpenAndSwap(t *testing.T) {
	requireSQLite(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.sqlite")
	p := NewProvider(path)
	t.Cleanup(func() { p.Close() })

	// File absent → "manifest not ready" error, no panic.
	if _, err := p.GetItemsByHashes([]uint32{100}); err == nil {
		t.Fatal("expected error while manifest file is absent")
	}

	// File appears → lazy open succeeds without recreating the Provider (B4).
	writeFixtureDB(t, path)
	defs, err := p.GetItemsByHashes([]uint32{100})
	if err != nil {
		t.Fatalf("GetItemsByHashes after download: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("len = %d, want 1", len(defs))
	}

	// Simulate the hourly swap (B5): close handles, replace the file, reopen.
	p.CloseForSwap()
	newPath := filepath.Join(dir, "manifest.sqlite.tmp")
	writeFixtureDB(t, newPath)
	// Mirrors the os.Rename the manifest updater performs — on Windows this
	// only succeeds because CloseForSwap released the SQLite handle (B5).
	if err := os.Rename(newPath, path); err != nil {
		t.Fatalf("swap rename failed (handles still open?): %v", err)
	}
	if err := p.Reopen(); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if _, err := p.GetItemsByHashes([]uint32{200}); err != nil {
		t.Fatalf("query after swap: %v", err)
	}
}
