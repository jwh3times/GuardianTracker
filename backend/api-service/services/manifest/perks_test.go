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

		// --- sniper 4000: intrinsic[0], scope[1] (pci "scopes", previously dropped) ---
		4000: `{"hash":4000,"itemType":3,"inventory":{"tierType":5},"sockets":{
			"socketCategories":[
				{"socketCategoryHash":3956125808,"socketIndexes":[0]},
				{"socketCategoryHash":4241085061,"socketIndexes":[1]}
			],
			"socketEntries":[
				{"singleInitialItemHash":6100,"reusablePlugSetHash":7000},
				{"singleInitialItemHash":6101,"reusablePlugSetHash":7001}
			]}}`,
		// --- sword 4001: intrinsic[0], blade[1] (pci "blades"), guard[2] (pci "guards") ---
		4001: `{"hash":4001,"itemType":3,"inventory":{"tierType":5},"sockets":{
			"socketCategories":[
				{"socketCategoryHash":3956125808,"socketIndexes":[0]},
				{"socketCategoryHash":4241085061,"socketIndexes":[1,2]}
			],
			"socketEntries":[
				{"singleInitialItemHash":6200,"reusablePlugSetHash":7002},
				{"singleInitialItemHash":6201,"reusablePlugSetHash":7003},
				{"singleInitialItemHash":6202,"reusablePlugSetHash":7004}
			]}}`,
		// --- fusion rifle 4002: intrinsic[0], battery[1] (pci "batteries") ---
		4002: `{"hash":4002,"itemType":3,"inventory":{"tierType":5},"sockets":{
			"socketCategories":[
				{"socketCategoryHash":3956125808,"socketIndexes":[0]},
				{"socketCategoryHash":4241085061,"socketIndexes":[1]}
			],
			"socketEntries":[
				{"singleInitialItemHash":6300,"reusablePlugSetHash":7005},
				{"singleInitialItemHash":6301,"reusablePlugSetHash":7006}
			]}}`,
		// --- Dead-Messenger-style 4003: intrinsic[0], trait[1] whose pool has the
		// empty-trait-socket placeholder FIRST, followed by 3 real frames perks ---
		4003: `{"hash":4003,"itemType":3,"inventory":{"tierType":5},"sockets":{
			"socketCategories":[
				{"socketCategoryHash":3956125808,"socketIndexes":[0]},
				{"socketCategoryHash":4241085061,"socketIndexes":[1]}
			],
			"socketEntries":[
				{"singleInitialItemHash":6350,"reusablePlugSetHash":7007},
				{"singleInitialItemHash":6400,"reusablePlugSetHash":7008}
			]}}`,
		// --- newer-style exotic 4004: intrinsic[0], trait[1], catalyst[2] — the
		// catalyst socket lives INSIDE the weapon-perks category itself (16/145
		// real exotics) and must never leak into perkColumns ---
		4004: `{"hash":4004,"itemType":3,"inventory":{"tierType":6},"sockets":{
			"socketCategories":[
				{"socketCategoryHash":3956125808,"socketIndexes":[0]},
				{"socketCategoryHash":4241085061,"socketIndexes":[1,2]}
			],
			"socketEntries":[
				{"singleInitialItemHash":6500,"reusablePlugSetHash":7009},
				{"singleInitialItemHash":6501,"reusablePlugSetHash":7010},
				{"singleInitialItemHash":0,"randomizedPlugSetHash":7011}
			]}}`,

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

		// --- sniper 4000 plugs ---
		6100: `{"hash":6100,"displayProperties":{"name":"Rapid-Fire Frame"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		6101: `{"hash":6101,"displayProperties":{"name":"Perfect Fifth"},"plug":{"plugCategoryIdentifier":"scopes"}}`,
		// --- sword 4001 plugs ---
		6200: `{"hash":6200,"displayProperties":{"name":"Caster Frame"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		6201: `{"hash":6201,"displayProperties":{"name":"Hungry Edge"},"plug":{"plugCategoryIdentifier":"blades"}}`,
		6202: `{"hash":6202,"displayProperties":{"name":"Balanced Guard"},"plug":{"plugCategoryIdentifier":"guards"}}`,
		// --- fusion 4002 plugs ---
		6300: `{"hash":6300,"displayProperties":{"name":"High-Impact Frame"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		6301: `{"hash":6301,"displayProperties":{"name":"Accelerated Coils"},"plug":{"plugCategoryIdentifier":"batteries"}}`,
		// --- Dead-Messenger-style 4003 plugs: empty-trait-socket FIRST, then 3 real perks ---
		6350: `{"hash":6350,"displayProperties":{"name":"Rapid-Fire Frame"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		6400: `{"hash":6400,"displayProperties":{"name":"Empty Traits Socket"},"plug":{"plugCategoryIdentifier":"crafting.recipes.empty_socket"}}`,
		6401: `{"hash":6401,"displayProperties":{"name":"Rewind Rounds"},"plug":{"plugCategoryIdentifier":"frames"}}`,
		6402: `{"hash":6402,"displayProperties":{"name":"Firefly"},"plug":{"plugCategoryIdentifier":"frames"}}`,
		6403: `{"hash":6403,"displayProperties":{"name":"Demolitionist"},"plug":{"plugCategoryIdentifier":"frames"}}`,
		// --- newer-style exotic 4004 plugs: trait col + catalyst col (empty + real) ---
		6500: `{"hash":6500,"displayProperties":{"name":"Adaptive Frame"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		6501: `{"hash":6501,"displayProperties":{"name":"Desperado"},"plug":{"plugCategoryIdentifier":"frames"}}`,
		6600: `{"hash":6600,"displayProperties":{"name":"Empty Catalyst Socket"},"plug":{"plugCategoryIdentifier":"v400.empty.exotic.masterwork"}}`,
		6601: `{"hash":6601,"displayProperties":{"name":"Fake Catalyst Trigger"},"plug":{"plugCategoryIdentifier":"catalysts"}}`,
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

		7000: `{"reusablePlugItems":[{"plugItemHash":6100}]}`,
		7001: `{"reusablePlugItems":[{"plugItemHash":6101}]}`,
		7002: `{"reusablePlugItems":[{"plugItemHash":6200}]}`,
		7003: `{"reusablePlugItems":[{"plugItemHash":6201}]}`,
		7004: `{"reusablePlugItems":[{"plugItemHash":6202}]}`,
		7005: `{"reusablePlugItems":[{"plugItemHash":6300}]}`,
		7006: `{"reusablePlugItems":[{"plugItemHash":6301}]}`,
		7007: `{"reusablePlugItems":[{"plugItemHash":6350}]}`,
		// Empty-trait-socket placeholder FIRST, then 3 real frames perks behind it.
		7008: `{"reusablePlugItems":[{"plugItemHash":6400},{"plugItemHash":6401},{"plugItemHash":6402},{"plugItemHash":6403}]}`,
		7009: `{"reusablePlugItems":[{"plugItemHash":6500}]}`,
		7010: `{"reusablePlugItems":[{"plugItemHash":6501}]}`,
		7011: `{"reusablePlugItems":[{"plugItemHash":6600},{"plugItemHash":6601}]}`,
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

// TestGetWeaponPerks_Scope covers a previously-dropped column: a sniper's scope
// slot (pci "scopes") was neither barrels/magazines/frames/origins, so the old
// first-plug allowlist silently discarded it.
func TestGetWeaponPerks_Scope(t *testing.T) {
	repo := perksRepo(t)
	cols, err := repo.GetWeaponPerks(4000)
	if err != nil {
		t.Fatalf("GetWeaponPerks: %v", err)
	}
	scope, ok := colByLabel(cols, "Scope")
	if !ok || scope.Role != "scope" || len(scope.Perks) != 1 || scope.Perks[0] != "Perfect Fifth" {
		t.Errorf("scope column = %+v (cols=%+v)", scope, cols)
	}
}

// TestGetWeaponPerks_SwordBladeGuard covers swords' blade/guard columns (pcis
// "blades"/"guards"), also previously dropped.
func TestGetWeaponPerks_SwordBladeGuard(t *testing.T) {
	repo := perksRepo(t)
	cols, err := repo.GetWeaponPerks(4001)
	if err != nil {
		t.Fatalf("GetWeaponPerks: %v", err)
	}
	blade, ok := colByLabel(cols, "Blade")
	if !ok || blade.Role != "blade" || len(blade.Perks) != 1 || blade.Perks[0] != "Hungry Edge" {
		t.Errorf("blade column = %+v (cols=%+v)", blade, cols)
	}
	guard, ok := colByLabel(cols, "Guard")
	if !ok || guard.Role != "guard" || len(guard.Perks) != 1 || guard.Perks[0] != "Balanced Guard" {
		t.Errorf("guard column = %+v (cols=%+v)", guard, cols)
	}
}

// TestGetWeaponPerks_Battery covers fusion rifles / linear fusion rifles' battery
// column (pci "batteries"), also previously dropped.
func TestGetWeaponPerks_Battery(t *testing.T) {
	repo := perksRepo(t)
	cols, err := repo.GetWeaponPerks(4002)
	if err != nil {
		t.Fatalf("GetWeaponPerks: %v", err)
	}
	battery, ok := colByLabel(cols, "Battery")
	if !ok || battery.Role != "battery" || len(battery.Perks) != 1 || battery.Perks[0] != "Accelerated Coils" {
		t.Errorf("battery column = %+v (cols=%+v)", battery, cols)
	}
}

// TestGetWeaponPerks_EmptySocketFirstKeepsColumn is the Dead Messenger
// regression: 'Empty Traits Socket' (crafting.recipes.empty_socket) is FIRST in
// the pool, but classification must skip past it (continue, never break) to find
// the 3 real frames perks behind it.
func TestGetWeaponPerks_EmptySocketFirstKeepsColumn(t *testing.T) {
	repo := perksRepo(t)
	cols, err := repo.GetWeaponPerks(4003)
	if err != nil {
		t.Fatalf("GetWeaponPerks: %v", err)
	}
	trait, ok := colByLabel(cols, "Trait 1")
	if !ok {
		t.Fatalf("trait column dropped entirely; got %+v", cols)
	}
	if len(trait.Perks) != 3 || trait.Perks[0] != "Rewind Rounds" || trait.Perks[1] != "Firefly" || trait.Perks[2] != "Demolitionist" {
		t.Errorf("trait perks = %+v, want 3 real perks (empty-socket excluded)", trait.Perks)
	}
}

// TestGetWeaponPerks_CatalystColumnExcluded: 16/145 catalyst-bearing exotics
// carry their catalyst socket inside the WEAPON PERKS category itself (not the
// separate WEAPON MODS category). GetWeaponPerks must not surface it as a normal
// perk column — catalysts are exposed separately via GetWeaponCatalysts.
func TestGetWeaponPerks_CatalystColumnExcluded(t *testing.T) {
	repo := perksRepo(t)
	cols, err := repo.GetWeaponPerks(4004)
	if err != nil {
		t.Fatalf("GetWeaponPerks: %v", err)
	}
	if len(cols) != 2 {
		t.Fatalf("columns = %d (%+v), want 2 (Intrinsic + Trait 1; catalyst column excluded)", len(cols), cols)
	}
	if _, ok := colByLabel(cols, "Perks"); ok {
		t.Errorf("catalyst column leaked into perkColumns under the generic fallback label")
	}
}
