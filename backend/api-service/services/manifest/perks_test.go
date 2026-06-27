package manifest

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// writePerksFixtureDB builds a minimal manifest with weapon/exotic/armor items
// plus the plug sets and plug-item defs their sockets reference.
func writePerksFixtureDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	defer db.Close()

	for _, ddl := range []string{
		`CREATE TABLE DestinyInventoryItemDefinition (id INTEGER PRIMARY KEY, json TEXT)`,
		`CREATE TABLE DestinyPlugSetDefinition (id INTEGER PRIMARY KEY, json TEXT)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("fixture ddl: %v", err)
		}
	}

	items := map[uint32]string{
		// --- weapon 1000: intrinsic[0], barrel[1] random, trait[2] random, tracker[3] ---
		1000: `{"hash":1000,"itemType":3,"inventory":{"tierType":5},"sockets":{
			"socketCategories":[
				{"socketCategoryHash":3956125808,"socketIndexes":[0]},
				{"socketCategoryHash":4241085061,"socketIndexes":[1,2,3]}
			],
			"socketEntries":[
				{"singleInitialItemHash":5100,"reusablePlugSetHash":5000},
				{"singleInitialItemHash":5101,"randomizedPlugSetHash":5001},
				{"singleInitialItemHash":5102,"randomizedPlugSetHash":5002},
				{"singleInitialItemHash":5103,"reusablePlugSetHash":5003}
			]}}`,
		// --- exotic 2000: intrinsic[0] fixed, perk[1] fixed, tracker[2] ---
		2000: `{"hash":2000,"itemType":3,"inventory":{"tierType":6},"sockets":{
			"socketCategories":[
				{"socketCategoryHash":3956125808,"socketIndexes":[0]},
				{"socketCategoryHash":4241085061,"socketIndexes":[1,2]}
			],
			"socketEntries":[
				{"singleInitialItemHash":5200,"reusablePlugSetHash":5004},
				{"singleInitialItemHash":5201,"reusablePlugSetHash":5005},
				{"singleInitialItemHash":5103,"reusablePlugSetHash":5003}
			]}}`,
		// --- armor 3000: not a weapon (itemType 2), armor categories only ---
		3000: `{"hash":3000,"itemType":2,"inventory":{"tierType":5},"sockets":{
			"socketCategories":[{"socketCategoryHash":3154740035,"socketIndexes":[0]}],
			"socketEntries":[{"singleInitialItemHash":9999,"randomizedPlugSetHash":5006}]}}`,

		// --- plug item defs (name + plugCategoryIdentifier) ---
		5100: `{"hash":5100,"displayProperties":{"name":"Adaptive Frame"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		5101: `{"hash":5101,"displayProperties":{"name":"Arrowhead Brake"},"plug":{"plugCategoryIdentifier":"barrels"}}`,
		5110: `{"hash":5110,"displayProperties":{"name":"Full Bore"},"plug":{"plugCategoryIdentifier":"barrels"}}`,
		5102: `{"hash":5102,"displayProperties":{"name":"Rampage"},"plug":{"plugCategoryIdentifier":"frames"}}`,
		5111: `{"hash":5111,"displayProperties":{"name":"Rampage"},"plug":{"plugCategoryIdentifier":"frames"}}`,
		5112: `{"hash":5112,"displayProperties":{"name":"Frenzy"},"plug":{"plugCategoryIdentifier":"frames"}}`,
		5113: `{"hash":5113,"displayProperties":{"name":"Retired Perk"},"plug":{"plugCategoryIdentifier":"frames"}}`,
		5103: `{"hash":5103,"displayProperties":{"name":"Kill Tracker"},"plug":{"plugCategoryIdentifier":"v400.plugs.weapons.masterworks.trackers"}}`,
		5200: `{"hash":5200,"displayProperties":{"name":"Impact Casing"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		5201: `{"hash":5201,"displayProperties":{"name":"Harbinger's Pulse"},"plug":{"plugCategoryIdentifier":"frames"}}`,
	}
	for hash, blob := range items {
		if _, err := db.Exec(`INSERT INTO DestinyInventoryItemDefinition (id, json) VALUES (?, ?)`, int32(hash), blob); err != nil {
			t.Fatalf("fixture item %d: %v", hash, err)
		}
	}

	plugSets := map[uint32]string{
		5000: `{"reusablePlugItems":[{"plugItemHash":5100}]}`,
		5001: `{"reusablePlugItems":[{"plugItemHash":5101,"currentlyCanRoll":true},{"plugItemHash":5110}]}`,
		// trait pool: Rampage twice (dedupe), Frenzy, and a retired perk (currentlyCanRoll:false → excluded)
		5002: `{"reusablePlugItems":[{"plugItemHash":5102,"currentlyCanRoll":true},{"plugItemHash":5111,"currentlyCanRoll":true},{"plugItemHash":5112},{"plugItemHash":5113,"currentlyCanRoll":false}]}`,
		5003: `{"reusablePlugItems":[{"plugItemHash":5103}]}`,
		5004: `{"reusablePlugItems":[{"plugItemHash":5200}]}`,
		5005: `{"reusablePlugItems":[{"plugItemHash":5201}]}`,
	}
	for hash, blob := range plugSets {
		if _, err := db.Exec(`INSERT INTO DestinyPlugSetDefinition (id, json) VALUES (?, ?)`, int32(hash), blob); err != nil {
			t.Fatalf("fixture plugset %d: %v", hash, err)
		}
	}
}

func perksRepo(t *testing.T) *Repository {
	t.Helper()
	requireSQLite(t) // defined in repository_test.go
	path := filepath.Join(t.TempDir(), "manifest.sqlite")
	writePerksFixtureDB(t, path)
	repo, err := NewRepository(path)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func colByLabel(cols []PerkColumn, label string) (PerkColumn, bool) {
	for _, c := range cols {
		if c.Label == label {
			return c, true
		}
	}
	return PerkColumn{}, false
}

func TestGetWeaponPerks_RandomRollLegendary(t *testing.T) {
	repo := perksRepo(t)
	cols, err := repo.GetWeaponPerks(1000)
	if err != nil {
		t.Fatalf("GetWeaponPerks: %v", err)
	}
	// Expect: Intrinsic, Barrel, Trait 1 — tracker column dropped entirely.
	if len(cols) != 3 {
		t.Fatalf("columns = %d (%+v), want 3", len(cols), cols)
	}
	if cols[0].Label != "Intrinsic" || len(cols[0].Perks) != 1 || cols[0].Perks[0] != "Adaptive Frame" {
		t.Errorf("intrinsic column = %+v", cols[0])
	}
	barrel, ok := colByLabel(cols, "Barrel")
	if !ok || len(barrel.Perks) != 2 || barrel.Perks[0] != "Arrowhead Brake" || barrel.Perks[1] != "Full Bore" {
		t.Errorf("barrel column = %+v", barrel)
	}
	trait, ok := colByLabel(cols, "Trait 1")
	if !ok {
		t.Fatalf("missing Trait 1 column; got %+v", cols)
	}
	// Rampage deduped to one entry; Frenzy present; retired (currentlyCanRoll:false) excluded.
	if len(trait.Perks) != 2 || trait.Perks[0] != "Rampage" || trait.Perks[1] != "Frenzy" {
		t.Errorf("trait column = %+v, want [Rampage Frenzy]", trait.Perks)
	}
	if _, bad := colByLabel(cols, "Trait 2"); bad {
		t.Errorf("unexpected Trait 2 column")
	}
}

func TestGetWeaponPerks_Exotic(t *testing.T) {
	repo := perksRepo(t)
	cols, err := repo.GetWeaponPerks(2000)
	if err != nil {
		t.Fatalf("GetWeaponPerks: %v", err)
	}
	// Intrinsic + one trait column; tracker dropped.
	if len(cols) != 2 {
		t.Fatalf("columns = %d (%+v), want 2", len(cols), cols)
	}
	if cols[0].Label != "Intrinsic" || cols[0].Perks[0] != "Impact Casing" {
		t.Errorf("intrinsic = %+v", cols[0])
	}
	trait, ok := colByLabel(cols, "Trait 1")
	if !ok || trait.Perks[0] != "Harbinger's Pulse" {
		t.Errorf("trait = %+v", trait)
	}
}

func TestGetWeaponPerks_NonWeaponReturnsNil(t *testing.T) {
	repo := perksRepo(t)
	cols, err := repo.GetWeaponPerks(3000) // armor
	if err != nil {
		t.Fatalf("GetWeaponPerks: %v", err)
	}
	if cols != nil {
		t.Errorf("armor columns = %+v, want nil", cols)
	}
}

func TestGetWeaponPerks_UnknownHashReturnsNil(t *testing.T) {
	repo := perksRepo(t)
	cols, err := repo.GetWeaponPerks(424242)
	if err != nil {
		t.Fatalf("GetWeaponPerks: %v", err)
	}
	if cols != nil {
		t.Errorf("unknown hash columns = %+v, want nil", cols)
	}
}
