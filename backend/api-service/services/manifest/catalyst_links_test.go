package manifest

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// writeCatalystLinksFixtureDB builds a small manifest with two exotics carrying
// catalyst sockets (one with objective hashes on its empty-placeholder plug, one
// without) and one legendary weapon (never included), to exercise
// GetCatalystLinks' full-table scan.
func writeCatalystLinksFixtureDB(t *testing.T, path string) {
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
		// Exotic with a catalyst socket: empty-placeholder plug carries the real
		// objective hashes (Sunshot shape — the resolved content plug itself often
		// carries none).
		9000: `{"hash":9000,"itemType":3,"inventory":{"tierType":6},
			"displayProperties":{"name":"Link Exotic One"},
			"sockets":{
				"socketCategories":[
					{"socketCategoryHash":3956125808,"socketIndexes":[0]},
					{"socketCategoryHash":2685412949,"socketIndexes":[1]}
				],
				"socketEntries":[
					{"singleInitialItemHash":9200},
					{"singleInitialItemHash":9010,"reusablePlugItems":[{"plugItemHash":9010},{"plugItemHash":9011}]}
				]}}`,
		// Second exotic, no objective hashes anywhere in its catalyst pool.
		9001: `{"hash":9001,"itemType":3,"inventory":{"tierType":6},
			"displayProperties":{"name":"Link Exotic Two"},
			"sockets":{
				"socketCategories":[
					{"socketCategoryHash":3956125808,"socketIndexes":[0]},
					{"socketCategoryHash":2685412949,"socketIndexes":[1]}
				],
				"socketEntries":[
					{"singleInitialItemHash":9201},
					{"singleInitialItemHash":9020,"reusablePlugItems":[{"plugItemHash":9020}]}
				]}}`,
		// Legendary weapon — never scanned (not exotic tier).
		9002: `{"hash":9002,"itemType":3,"inventory":{"tierType":5},
			"displayProperties":{"name":"Link Legendary"},
			"sockets":{
				"socketCategories":[
					{"socketCategoryHash":3956125808,"socketIndexes":[0]},
					{"socketCategoryHash":2685412949,"socketIndexes":[1]}
				],
				"socketEntries":[
					{"singleInitialItemHash":9202},
					{"singleInitialItemHash":9010,"reusablePlugItems":[{"plugItemHash":9010},{"plugItemHash":9011}]}
				]}}`,

		9200: `{"hash":9200,"displayProperties":{"name":"Intrinsic One"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		9201: `{"hash":9201,"displayProperties":{"name":"Intrinsic Two"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,
		9202: `{"hash":9202,"displayProperties":{"name":"Intrinsic Legendary"},"plug":{"plugCategoryIdentifier":"intrinsics"}}`,

		// Exotic One's catalyst socket: dummy content plug (itemType20, no
		// objectives) + "Upgrade Masterwork" (excluded from Catalysts, but its
		// objectiveHashes still count toward the link's ObjectiveHashes union).
		9010: `{"hash":9010,"itemType":20,
			"displayProperties":{"name":"Link Exotic One Catalyst","description":"Upgrades this weapon to a Masterwork."},
			"plug":{"plugCategoryIdentifier":"v300_link_one_masterwork"}}`,
		9011: `{"hash":9011,"itemType":19,
			"displayProperties":{"name":"Upgrade Masterwork","description":"Defeat targets to unlock."},
			"objectives":{"objectiveHashes":[5001,5002]},
			"plug":{"plugCategoryIdentifier":"v300_link_one_masterwork"}}`,

		// Exotic Two's catalyst socket: single named content plug, no objectives.
		9020: `{"hash":9020,"itemType":19,
			"displayProperties":{"name":"Link Exotic Two Perk","description":"Direct effect text."},
			"plug":{"plugCategoryIdentifier":"catalysts"}}`,
	}
	for hash, blob := range items {
		if _, err := db.Exec(`INSERT INTO DestinyInventoryItemDefinition (id, json) VALUES (?, ?)`, int32(hash), blob); err != nil {
			t.Fatalf("fixture item %d: %v", hash, err)
		}
	}
}

func catalystLinksRepo(t *testing.T) *Repository {
	t.Helper()
	requireSQLite(t)
	path := filepath.Join(t.TempDir(), "manifest.sqlite")
	writeCatalystLinksFixtureDB(t, path)
	repo, err := NewRepository(path)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestGetCatalystLinks(t *testing.T) {
	repo := catalystLinksRepo(t)
	links, err := repo.GetCatalystLinks()
	if err != nil {
		t.Fatalf("GetCatalystLinks: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("links = %+v, want 2 (legendary excluded)", links)
	}

	byName := map[string]CatalystLink{}
	for _, l := range links {
		byName[l.WeaponName] = l
	}

	one, ok := byName["Link Exotic One"]
	if !ok {
		t.Fatalf("missing Link Exotic One; got %+v", links)
	}
	if len(one.Catalysts) != 1 || one.Catalysts[0].Name != "Link Exotic One Catalyst" {
		t.Errorf("Exotic One catalysts = %+v", one.Catalysts)
	}
	if len(one.ObjectiveHashes) != 2 || one.ObjectiveHashes[0] != 5001 || one.ObjectiveHashes[1] != 5002 {
		t.Errorf("Exotic One objective hashes = %v, want [5001 5002] (from the excluded 'Upgrade Masterwork' plug)", one.ObjectiveHashes)
	}

	two, ok := byName["Link Exotic Two"]
	if !ok {
		t.Fatalf("missing Link Exotic Two; got %+v", links)
	}
	if len(two.Catalysts) != 1 || two.Catalysts[0].Name != "Link Exotic Two Perk" {
		t.Errorf("Exotic Two catalysts = %+v", two.Catalysts)
	}
	if len(two.ObjectiveHashes) != 0 {
		t.Errorf("Exotic Two objective hashes = %v, want none", two.ObjectiveHashes)
	}

	if _, ok := byName["Link Legendary"]; ok {
		t.Errorf("legendary weapon leaked into catalyst links")
	}
}
