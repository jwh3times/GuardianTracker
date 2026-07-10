package manifest

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// writeCatalystsFixtureDB builds a minimal manifest covering the real-world
// catalyst-socket shapes verified against the live manifest: an old-style
// (v300) single catalyst whose real effect text lives on a same-named itemType
// 19 row elsewhere in the table, a newer-style catalyst living inside the
// weapon-perks category with perk-hash-driven effect text, a multi-catalyst
// exotic, an exotic with no catalyst socket at all (e.g. Legend of Acrius), and
// a non-exotic weapon.
func writeCatalystsFixtureDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	defer db.Close()

	for _, ddl := range []string{
		`CREATE TABLE DestinyInventoryItemDefinition (id INTEGER PRIMARY KEY, json TEXT)`,
		`CREATE TABLE DestinyPlugSetDefinition (id INTEGER PRIMARY KEY, json TEXT)`,
		`CREATE TABLE DestinySandboxPerkDefinition (id INTEGER PRIMARY KEY, json TEXT)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("fixture ddl: %v", err)
		}
	}

	items := map[uint32]string{
		// --- 8000: old-style single catalyst (Sunshot shape). Catalyst socket in
		// WEAPON MODS (2685412949) at socket index 1. Pool: dummy content plug
		// (itemType 20, generic description) + "Upgrade Masterwork" (excluded by
		// name). The real effect text lives on a SAME-NAMED itemType 19 row (8100)
		// elsewhere in the table, resolved via its displayable sandbox perk.
		8000: `{"hash":8000,"itemType":3,"inventory":{"tierType":6},
			"displayProperties":{"name":"Test Exotic Old"},
			"sockets":{
				"socketCategories":[
					{"socketCategoryHash":3956125808,"socketIndexes":[0]},
					{"socketCategoryHash":2685412949,"socketIndexes":[1]}
				],
				"socketEntries":[
					{"singleInitialItemHash":8300},
					{"singleInitialItemHash":8010,"reusablePlugItems":[{"plugItemHash":8010},{"plugItemHash":8011}]}
				]}}`,
		// --- 8001: newer-style catalyst living INSIDE the weapon-perks category
		// (16/145 real exotics). Trait[1] + catalyst[2] both in 4241085061. ---
		8001: `{"hash":8001,"itemType":3,"inventory":{"tierType":6},
			"displayProperties":{"name":"Test Exotic New"},
			"sockets":{
				"socketCategories":[
					{"socketCategoryHash":3956125808,"socketIndexes":[0]},
					{"socketCategoryHash":4241085061,"socketIndexes":[1,2]}
				],
				"socketEntries":[
					{"singleInitialItemHash":8301},
					{"singleInitialItemHash":8302,"reusablePlugItems":[{"plugItemHash":8302}]},
					{"randomizedPlugSetHash":8500}
				]}}`,
		// --- 8002: multi-catalyst exotic (Wish-Keeper/Revision Zero shape) — 3
		// distinct named catalyst-perk plugs in one WEAPON MODS socket. ---
		8002: `{"hash":8002,"itemType":3,"inventory":{"tierType":6},
			"displayProperties":{"name":"Test Exotic Multi"},
			"sockets":{
				"socketCategories":[
					{"socketCategoryHash":3956125808,"socketIndexes":[0]},
					{"socketCategoryHash":2685412949,"socketIndexes":[1]}
				],
				"socketEntries":[
					{"singleInitialItemHash":8303},
					{"singleInitialItemHash":8600,"reusablePlugItems":[{"plugItemHash":8600},{"plugItemHash":8601},{"plugItemHash":8602}]}
				]}}`,
		// --- 8003: exotic with NO catalyst socket at all (Legend of Acrius shape). ---
		8003: `{"hash":8003,"itemType":3,"inventory":{"tierType":6},
			"displayProperties":{"name":"Test Exotic No Catalyst"},
			"sockets":{
				"socketCategories":[
					{"socketCategoryHash":3956125808,"socketIndexes":[0]},
					{"socketCategoryHash":4241085061,"socketIndexes":[1]}
				],
				"socketEntries":[
					{"singleInitialItemHash":8304},
					{"singleInitialItemHash":8305}
				]}}`,
		// --- 8004: non-exotic (legendary) weapon — never has a catalyst. ---
		8004: `{"hash":8004,"itemType":3,"inventory":{"tierType":5},
			"displayProperties":{"name":"Test Legendary"},
			"sockets":{
				"socketCategories":[
					{"socketCategoryHash":3956125808,"socketIndexes":[0]},
					{"socketCategoryHash":2685412949,"socketIndexes":[1]}
				],
				"socketEntries":[
					{"singleInitialItemHash":8306},
					{"singleInitialItemHash":8010,"reusablePlugItems":[{"plugItemHash":8010},{"plugItemHash":8011}]}
				]}}`,

		// --- shared intrinsic plugs (not under test) ---
		8300: `{"hash":8300,"displayProperties":{"name":"Intrinsic Old"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		8301: `{"hash":8301,"displayProperties":{"name":"Intrinsic New"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		8302: `{"hash":8302,"displayProperties":{"name":"Desperado"},"plug":{"plugCategoryIdentifier":"frames"}}`,
		8303: `{"hash":8303,"displayProperties":{"name":"Intrinsic Multi"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		8304: `{"hash":8304,"displayProperties":{"name":"Intrinsic None"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		8305: `{"hash":8305,"displayProperties":{"name":"Ordinary Trait"},"plug":{"plugCategoryIdentifier":"frames"}}`,
		8306: `{"hash":8306,"displayProperties":{"name":"Intrinsic Legendary"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,

		// --- 8000's catalyst-socket pool: dummy content plug (itemType 20, generic
		// text) + excluded "Upgrade Masterwork" plug ---
		8010: `{"hash":8010,"itemType":20,
			"displayProperties":{"name":"Test Exotic Old Catalyst","description":"Upgrades this weapon to a Masterwork. Once upgraded, the weapon will obtain enhanced capabilities."},
			"plug":{"plugCategoryIdentifier":"v300_old_test_masterwork"}}`,
		8011: `{"hash":8011,"itemType":19,
			"displayProperties":{"name":"Upgrade Masterwork","description":"Defeat targets to unlock this upgrade."},
			"plug":{"plugCategoryIdentifier":"v300_old_test_masterwork"}}`,

		// --- same-named itemType19 row elsewhere in the table carrying the real
		// effect text via a displayable sandbox perk (old-style v300 pattern) ---
		8100: `{"hash":8100,"itemType":19,
			"displayProperties":{"name":"Test Exotic Old Catalyst","description":"When upgraded to a Masterwork, this weapon gains additional perks."},
			"plug":{"plugCategoryIdentifier":"catalysts"},
			"perks":[{"perkHash":9001,"perkVisibility":0},{"perkHash":9002,"perkVisibility":1}]}`,

		// --- 8001's catalyst socket: empty placeholder (in plugset 8500) is
		// excluded automatically once we resolve the plugset ---

		// --- 8302 real trait plug already defined above ---

		// --- newer-style catalyst content plug, perk-hash-driven effect text ---
		8600: `{"hash":8600,"itemType":19,
			"displayProperties":{"name":"Catalyst Perk One","description":""},
			"plug":{"plugCategoryIdentifier":"catalysts"},
			"perks":[{"perkHash":9010,"perkVisibility":0}]}`,
		8601: `{"hash":8601,"itemType":19,
			"displayProperties":{"name":"Catalyst Perk Two","description":""},
			"plug":{"plugCategoryIdentifier":"catalysts"},
			"perks":[{"perkHash":9011,"perkVisibility":0}]}`,
		8602: `{"hash":8602,"itemType":19,
			"displayProperties":{"name":"Catalyst Perk Three","description":""},
			"plug":{"plugCategoryIdentifier":"catalysts"},
			"perks":[{"perkHash":9012,"perkVisibility":0}]}`,
	}
	for hash, blob := range items {
		if _, err := db.Exec(`INSERT INTO DestinyInventoryItemDefinition (id, json) VALUES (?, ?)`, int32(hash), blob); err != nil {
			t.Fatalf("fixture item %d: %v", hash, err)
		}
	}

	// 8001's catalyst socket resolves via randomizedPlugSetHash 8500: empty
	// placeholder FIRST, then the real content plug — proving the empty plug
	// never blocks resolution of the real one behind it.
	plugItems8001 := map[uint32]string{
		1498917124: `{"hash":1498917124,"itemType":19,
			"displayProperties":{"name":"Empty Catalyst Socket","description":"An Exotic catalyst can be inserted into this socket."},
			"plug":{"plugCategoryIdentifier":"v400.empty.exotic.masterwork"}}`,
		8700: `{"hash":8700,"itemType":19,
			"displayProperties":{"name":"Real Catalyst Perk","description":"Generic fallback text."},
			"plug":{"plugCategoryIdentifier":"catalysts"},
			"perks":[{"perkHash":9020,"perkVisibility":0}]}`,
	}
	for hash, blob := range plugItems8001 {
		if _, err := db.Exec(`INSERT INTO DestinyInventoryItemDefinition (id, json) VALUES (?, ?)`, int32(hash), blob); err != nil {
			t.Fatalf("fixture item %d: %v", hash, err)
		}
	}

	plugSets := map[uint32]string{
		8500: `{"reusablePlugItems":[{"plugItemHash":1498917124},{"plugItemHash":8700}]}`,
	}
	for hash, blob := range plugSets {
		if _, err := db.Exec(`INSERT INTO DestinyPlugSetDefinition (id, json) VALUES (?, ?)`, int32(hash), blob); err != nil {
			t.Fatalf("fixture plugset %d: %v", hash, err)
		}
	}

	sandboxPerks := map[uint32]string{
		// 8100's perks: first is NOT displayable, second IS — proves the resolver
		// picks the first *displayable* one, not just the first in the array.
		9001: `{"hash":9001,"isDisplayable":false,"displayProperties":{"name":"","description":""}}`,
		9002: `{"hash":9002,"isDisplayable":true,"displayProperties":{"name":"Loose Change","description":"Old-style resolved effect text."}}`,
		9010: `{"hash":9010,"isDisplayable":true,"displayProperties":{"name":"Perk One","description":"Effect text one."}}`,
		9011: `{"hash":9011,"isDisplayable":true,"displayProperties":{"name":"Perk Two","description":"Effect text two."}}`,
		9012: `{"hash":9012,"isDisplayable":true,"displayProperties":{"name":"Perk Three","description":"Effect text three."}}`,
		9020: `{"hash":9020,"isDisplayable":true,"displayProperties":{"name":"Real Perk","description":"New-style resolved effect text."}}`,
	}
	for hash, blob := range sandboxPerks {
		if _, err := db.Exec(`INSERT INTO DestinySandboxPerkDefinition (id, json) VALUES (?, ?)`, int32(hash), blob); err != nil {
			t.Fatalf("fixture sandbox perk %d: %v", hash, err)
		}
	}
}

func catalystsRepo(t *testing.T) *Repository {
	t.Helper()
	requireSQLite(t)
	path := filepath.Join(t.TempDir(), "manifest.sqlite")
	writeCatalystsFixtureDB(t, path)
	repo, err := NewRepository(path)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestGetWeaponCatalysts_OldStyleSameNamedFallback(t *testing.T) {
	repo := catalystsRepo(t)
	cats, err := repo.GetWeaponCatalysts(8000)
	if err != nil {
		t.Fatalf("GetWeaponCatalysts: %v", err)
	}
	if len(cats) != 1 {
		t.Fatalf("catalysts = %+v, want 1 entry", cats)
	}
	if cats[0].Name != "Test Exotic Old Catalyst" {
		t.Errorf("name = %q", cats[0].Name)
	}
	if cats[0].Description != "Old-style resolved effect text." {
		t.Errorf("description = %q, want the same-named itemType19 row's displayable sandbox-perk text", cats[0].Description)
	}
}

func TestGetWeaponCatalysts_NewerStyleInsideWeaponPerksCategory(t *testing.T) {
	repo := catalystsRepo(t)
	cats, err := repo.GetWeaponCatalysts(8001)
	if err != nil {
		t.Fatalf("GetWeaponCatalysts: %v", err)
	}
	if len(cats) != 1 {
		t.Fatalf("catalysts = %+v, want 1 entry", cats)
	}
	if cats[0].Name != "Real Catalyst Perk" {
		t.Errorf("name = %q, want the plug's own name (new-style plugs are named after the perk)", cats[0].Name)
	}
	if cats[0].Description != "New-style resolved effect text." {
		t.Errorf("description = %q, want the perk-hash-driven text", cats[0].Description)
	}
}

func TestGetWeaponCatalysts_MultiCatalyst(t *testing.T) {
	repo := catalystsRepo(t)
	cats, err := repo.GetWeaponCatalysts(8002)
	if err != nil {
		t.Fatalf("GetWeaponCatalysts: %v", err)
	}
	if len(cats) != 3 {
		t.Fatalf("catalysts = %+v, want 3 entries (multi-catalyst exotic)", cats)
	}
	names := []string{cats[0].Name, cats[1].Name, cats[2].Name}
	want := []string{"Catalyst Perk One", "Catalyst Perk Two", "Catalyst Perk Three"}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("catalyst[%d] name = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestGetWeaponCatalysts_NoCatalystSocketReturnsNil(t *testing.T) {
	repo := catalystsRepo(t)
	cats, err := repo.GetWeaponCatalysts(8003)
	if err != nil {
		t.Fatalf("GetWeaponCatalysts: %v", err)
	}
	if cats != nil {
		t.Errorf("catalysts = %+v, want nil (no catalyst socket, e.g. Legend of Acrius)", cats)
	}
}

func TestGetWeaponCatalysts_NonExoticReturnsNil(t *testing.T) {
	repo := catalystsRepo(t)
	cats, err := repo.GetWeaponCatalysts(8004)
	if err != nil {
		t.Fatalf("GetWeaponCatalysts: %v", err)
	}
	if cats != nil {
		t.Errorf("catalysts = %+v, want nil (legendary weapons never have catalysts)", cats)
	}
}

func TestGetWeaponCatalysts_UnknownHashReturnsNil(t *testing.T) {
	repo := catalystsRepo(t)
	cats, err := repo.GetWeaponCatalysts(424242)
	if err != nil {
		t.Fatalf("GetWeaponCatalysts: %v", err)
	}
	if cats != nil {
		t.Errorf("catalysts = %+v, want nil", cats)
	}
}
