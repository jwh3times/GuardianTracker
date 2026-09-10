package manifest

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"guardian-tracker/api-service/services/bungie"

	_ "github.com/mattn/go-sqlite3"
)

// chunkFixtureHashes returns n item hashes that straddle the signed/unsigned
// boundary: the first half fit in an int32, the second half do not and are
// stored as negative row ids. Mixing them means a chunk split lands in the
// middle of the wrapped range rather than only ever at a sign change.
func chunkFixtureHashes(n int) []uint32 {
	hashes := make([]uint32, 0, n)
	for i := range n / 2 {
		hashes = append(hashes, uint32(100000+i))
	}
	for i := range n - len(hashes) {
		hashes = append(hashes, uint32(3_000_000_000+i))
	}
	return hashes
}

// bulkItemRepo builds a manifest fixture holding exactly the given item hashes.
func bulkItemRepo(t *testing.T, hashes []uint32) *Repository {
	t.Helper()
	requireSQLite(t)
	path := t.TempDir() + "/manifest.sqlite"

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE DestinyInventoryItemDefinition (id INTEGER PRIMARY KEY, json TEXT)`,
		`CREATE TABLE DestinyCollectibleDefinition (id INTEGER PRIMARY KEY, json TEXT)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			db.Close()
			t.Fatalf("fixture ddl: %v", err)
		}
	}
	tx, err := db.Begin()
	if err != nil {
		db.Close()
		t.Fatalf("fixture tx: %v", err)
	}
	for i, hash := range hashes {
		blob := fmt.Sprintf(`{"hash":%d,"displayProperties":{"name":"Item %d"},"itemType":3,"inventory":{"tierType":5}}`, hash, hash)
		if _, err := tx.Exec(`INSERT INTO DestinyInventoryItemDefinition (id, json) VALUES (?, ?)`, hashToDBKey(hash), blob); err != nil {
			db.Close()
			t.Fatalf("fixture item %d: %v", hash, err)
		}
		// One collectible per item, on its own hash space, so the itemHash join
		// chunks over the same input.
		col := fmt.Sprintf(`{"hash":%d,"itemHash":%d,"sourceString":"Source %d","displayProperties":{"name":"Item %d"}}`, 1+i, hash, hash, hash)
		if _, err := tx.Exec(`INSERT INTO DestinyCollectibleDefinition (id, json) VALUES (?, ?)`, int64(1+i), col); err != nil {
			db.Close()
			t.Fatalf("fixture collectible for %d: %v", hash, err)
		}
	}
	if err := tx.Commit(); err != nil {
		db.Close()
		t.Fatalf("fixture commit: %v", err)
	}
	db.Close()

	repo, err := NewRepository(path)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

// countingQuery wraps a row-id query, counting how many statements it builds.
// statement is called exactly once per chunk, so the count is the chunk count.
func countingQuery(table string, chunks *int) chunkedQuery {
	q := byRowID(table, "counting")
	inner := q.statement
	q.statement = func(placeholders string) string {
		*chunks++
		return inner(placeholders)
	}
	return q
}

// A batch larger than chunkSize must be split, and every requested definition
// must still come back exactly once. The count is asserted directly rather than
// inferred from the result: SQLite's modern parameter ceiling is far above
// chunkSize, so an unchunked query over this many hashes would still succeed
// and a result-only assertion would pass with the chunking removed.
func TestQueryDefsChunked_SplitsBatchAndReturnsEverything(t *testing.T) {
	const total = 1201 // deliberately not a multiple of chunkSize
	hashes := chunkFixtureHashes(total)
	repo := bulkItemRepo(t, hashes)

	chunks := 0
	seen := make(map[uint32]int, total)
	err := queryDefsChunked(repo.db, hashes, countingQuery("DestinyInventoryItemDefinition", &chunks),
		func(id uint32, _ *bungie.InventoryItemDefinition) { seen[id]++ })
	if err != nil {
		t.Fatalf("queryDefsChunked: %v", err)
	}

	if want := 3; chunks != want { // 500 + 500 + 201
		t.Errorf("chunks = %d, want %d", chunks, want)
	}
	if len(seen) != total {
		t.Fatalf("distinct definitions = %d, want %d", len(seen), total)
	}
	for _, hash := range hashes {
		if seen[hash] != 1 {
			t.Fatalf("hash %d seen %d times, want exactly 1", hash, seen[hash])
		}
	}
}

// An input that divides evenly must not issue a trailing empty chunk.
func TestQueryDefsChunked_ExactMultipleIssuesNoEmptyChunk(t *testing.T) {
	hashes := chunkFixtureHashes(2 * chunkSize)
	repo := bulkItemRepo(t, hashes)

	chunks := 0
	count := 0
	err := queryDefsChunked(repo.db, hashes, countingQuery("DestinyInventoryItemDefinition", &chunks),
		func(uint32, *bungie.InventoryItemDefinition) { count++ })
	if err != nil {
		t.Fatalf("queryDefsChunked: %v", err)
	}
	if chunks != 2 {
		t.Errorf("chunks = %d, want 2", chunks)
	}
	if count != 2*chunkSize {
		t.Errorf("definitions = %d, want %d", count, 2*chunkSize)
	}
}

// An empty request must not reach the database at all.
func TestQueryDefsChunked_EmptyRequestIssuesNoStatement(t *testing.T) {
	repo := bulkItemRepo(t, chunkFixtureHashes(4))

	chunks := 0
	err := queryDefsChunked(repo.db, nil, countingQuery("DestinyInventoryItemDefinition", &chunks),
		func(uint32, *bungie.InventoryItemDefinition) { t.Error("collect called for an empty request") })
	if err != nil {
		t.Fatalf("queryDefsChunked: %v", err)
	}
	if chunks != 0 {
		t.Errorf("chunks = %d, want 0", chunks)
	}
}

// GetItemsByHashes is reached with more hashes than one statement should carry
// (a cold collections fetch covers thousands of items), so it must chunk like
// every other batched read rather than building one oversized IN-clause.
func TestRepository_GetItemsByHashesAcrossChunks(t *testing.T) {
	hashes := chunkFixtureHashes(1201)
	repo := bulkItemRepo(t, hashes)

	defs, err := repo.GetItemsByHashes(hashes)
	if err != nil {
		t.Fatalf("GetItemsByHashes: %v", err)
	}
	if len(defs) != len(hashes) {
		t.Fatalf("len = %d, want %d", len(defs), len(hashes))
	}
	for _, hash := range hashes {
		def, ok := defs[hash]
		if !ok {
			t.Fatalf("hash %d missing from a multi-chunk batch", hash)
		}
		if def.DisplayProperties.Name != fmt.Sprintf("Item %d", hash) {
			t.Fatalf("hash %d resolved to %q, want its own definition", hash, def.DisplayProperties.Name)
		}
	}
}

// The itemHash join binds the plain unsigned hash rather than the signed row id,
// so it chunks on a different encoding and needs its own boundary coverage.
func TestGetAcquisitionRows_AcrossChunks(t *testing.T) {
	hashes := chunkFixtureHashes(1201)
	repo := bulkItemRepo(t, hashes)

	rows, err := repo.GetAcquisitionRows(hashes)
	if err != nil {
		t.Fatalf("GetAcquisitionRows: %v", err)
	}
	if len(rows.Items) != len(hashes) {
		t.Fatalf("Items = %d, want %d", len(rows.Items), len(hashes))
	}
	if len(rows.Collectibles) != len(hashes) {
		t.Fatalf("Collectibles = %d, want %d", len(rows.Collectibles), len(hashes))
	}
	for _, hash := range hashes {
		cols := rows.Collectibles[hash]
		if len(cols) != 1 {
			t.Fatalf("collectibles for %d = %d, want exactly 1", hash, len(cols))
		}
		if cols[0].SourceString != fmt.Sprintf("Source %d", hash) {
			t.Fatalf("hash %d joined to %q, want its own collectible", hash, cols[0].SourceString)
		}
	}
}

// rawRepo builds a manifest fixture from rows given verbatim, so a test can make
// a row id and the hash inside its blob disagree — which the shared fixture,
// where the two always match, cannot express.
func rawRepo(t *testing.T, ddl string, rows map[int64]string) *Repository {
	t.Helper()
	requireSQLite(t)
	path := t.TempDir() + "/manifest.sqlite"

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	if _, err := db.Exec(ddl); err != nil {
		db.Close()
		t.Fatalf("fixture ddl: %v", err)
	}
	for id, blob := range rows {
		if _, err := db.Exec("INSERT INTO "+tableOf(ddl)+" (id, json) VALUES (?, ?)", id, blob); err != nil {
			db.Close()
			t.Fatalf("fixture row %d: %v", id, err)
		}
	}
	db.Close()

	repo, err := NewRepository(path)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

// tableOf pulls the table name out of a CREATE TABLE statement.
func tableOf(ddl string) string {
	return strings.Fields(ddl)[2]
}

// The manifest is a third-party artifact. One definition that fails to decode
// must be skipped rather than failing the whole batch, or a single bad blob
// upstream would deny the caller every other definition it asked for.
func TestQueryDefsChunked_MalformedBlobSkippedNotFatal(t *testing.T) {
	repo := rawRepo(t, "CREATE TABLE DestinyInventoryItemDefinition (id INTEGER PRIMARY KEY, json TEXT)", map[int64]string{
		100: `{"hash":100,"displayProperties":{"name":"Fatebringer"}}`,
		200: `{"hash":200,"displayProperties":{"name":`, // truncated mid-object
		300: `{"hash":300,"displayProperties":{"name":"Gjallarhorn"}}`,
	})

	defs, err := repo.GetItemsByHashes([]uint32{100, 200, 300})
	if err != nil {
		t.Fatalf("GetItemsByHashes returned an error for one malformed blob: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("len = %d, want 2 (the malformed row skipped, the others kept)", len(defs))
	}
	if defs[100] == nil || defs[300] == nil {
		t.Errorf("well-formed definitions lost alongside the malformed one: %v", defs)
	}
	if _, ok := defs[200]; ok {
		t.Error("malformed row should be absent, not present and empty")
	}
}

// Item definitions are keyed by the row id the caller asked for, so a batch stays
// addressable by its own input even if a blob's self-reported hash disagrees.
// This is the opposite of the keying used for records and presentation nodes
// below; the asymmetry is pre-existing and deliberate, and these two tests pin
// each side of it so a future unification is a visible decision, not a slip.
func TestRepository_ItemsKeyedByRowID(t *testing.T) {
	repo := rawRepo(t, "CREATE TABLE DestinyInventoryItemDefinition (id INTEGER PRIMARY KEY, json TEXT)", map[int64]string{
		100: `{"hash":999,"displayProperties":{"name":"Disagreeing Item"}}`,
	})

	defs, err := repo.GetItemsByHashes([]uint32{100})
	if err != nil {
		t.Fatalf("GetItemsByHashes: %v", err)
	}
	if defs[100] == nil {
		t.Fatalf("item absent under the requested hash; keys = %v", defs)
	}
	if _, ok := defs[999]; ok {
		t.Error("item keyed by the blob's hash instead of the requested row id")
	}
}

// Presentation nodes are keyed by the hash inside the blob, not by the row id.
func TestRepository_PresentationNodesKeyedByBlobHash(t *testing.T) {
	repo := rawRepo(t, "CREATE TABLE DestinyPresentationNodeDefinition (id INTEGER PRIMARY KEY, json TEXT)", map[int64]string{
		50: `{"hash":51,"displayProperties":{"name":"Disagreeing Node"}}`,
	})

	defs, err := repo.GetPresentationNodeDefinitions([]uint32{50})
	if err != nil {
		t.Fatalf("GetPresentationNodeDefinitions: %v", err)
	}
	if defs[51] == nil {
		t.Fatalf("node absent under the blob's own hash; keys = %v", defs)
	}
	if _, ok := defs[50]; ok {
		t.Error("node keyed by the row id instead of the blob's hash")
	}
}
