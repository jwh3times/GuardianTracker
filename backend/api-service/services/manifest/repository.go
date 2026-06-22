package manifest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"sync"

	"guardian-tracker/api-service/services/bungie"

	_ "github.com/mattn/go-sqlite3"
)

// Repository handles read-only queries against the Bungie manifest SQLite database.
type Repository struct {
	dbPath string
	db     *sql.DB
	mu     sync.RWMutex
}

func NewRepository(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro&cache=shared")
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest database: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to manifest database: %w", err)
	}
	return &Repository{dbPath: dbPath, db: db}, nil
}

func (r *Repository) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *Repository) Reconnect() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.db != nil {
		r.db.Close()
	}
	db, err := sql.Open("sqlite3", r.dbPath+"?mode=ro&cache=shared")
	if err != nil {
		return fmt.Errorf("failed to reopen manifest database: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	r.db = db
	return nil
}

func hashToDBKey(hash uint32) int32 { return int32(hash) }

func (r *Repository) GetCollectibleDefinition(hash uint32) (*bungie.CollectibleDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var blob string
	err := r.db.QueryRow("SELECT json FROM DestinyCollectibleDefinition WHERE id = ?", hashToDBKey(hash)).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query collectible: %w", err)
	}
	var def bungie.CollectibleDefinition
	if err := json.Unmarshal([]byte(blob), &def); err != nil {
		return nil, fmt.Errorf("failed to parse collectible JSON: %w", err)
	}
	return &def, nil
}

func (r *Repository) GetInventoryItemDefinition(hash uint32) (*bungie.InventoryItemDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var blob string
	err := r.db.QueryRow("SELECT json FROM DestinyInventoryItemDefinition WHERE id = ?", hashToDBKey(hash)).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query item: %w", err)
	}
	var def bungie.InventoryItemDefinition
	if err := json.Unmarshal([]byte(blob), &def); err != nil {
		return nil, fmt.Errorf("failed to parse item JSON: %w", err)
	}
	return &def, nil
}

func (r *Repository) GetAllCollectibles() ([]bungie.CollectibleDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.getAllCollectiblesLocked()
}

// getAllCollectiblesLocked is GetAllCollectibles assuming r.mu is already held.
func (r *Repository) getAllCollectiblesLocked() ([]bungie.CollectibleDefinition, error) {
	rows, err := r.db.Query("SELECT json FROM DestinyCollectibleDefinition")
	if err != nil {
		return nil, fmt.Errorf("failed to query collectibles: %w", err)
	}
	defer rows.Close()
	var out []bungie.CollectibleDefinition
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, fmt.Errorf("GetAllCollectibles scan: %w", err)
		}
		var def bungie.CollectibleDefinition
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		if def.DisplayProperties.Name == "" {
			continue
		}
		out = append(out, def)
	}
	return out, rows.Err()
}

// CollectibleWithItem pairs a collectible with its item definition.
type CollectibleWithItem struct {
	Collectible bungie.CollectibleDefinition
	Item        *bungie.InventoryItemDefinition
}

func (r *Repository) GetAllCollectiblesWithItems() ([]CollectibleWithItem, error) {
	// Hold the read lock across both the collectibles query and the item lookups
	// so a manifest swap cannot replace the database between them (which would
	// match collectibles from one manifest against items from another). Repository
	// close/reconnect take the write lock and will wait for this read to finish.
	r.mu.RLock()
	defer r.mu.RUnlock()

	collectibles, err := r.getAllCollectiblesLocked()
	if err != nil {
		return nil, err
	}
	// Batch the item lookups (chunked IN-queries) instead of one query per
	// collectible — a cold collections fetch covers thousands of collectibles.
	hashes := make([]uint32, 0, len(collectibles))
	for _, col := range collectibles {
		if col.ItemHash != 0 {
			hashes = append(hashes, col.ItemHash)
		}
	}
	items, err := r.getItemsByHashesChunkedLocked(hashes)
	if err != nil {
		return nil, err
	}
	results := make([]CollectibleWithItem, 0, len(collectibles))
	for _, col := range collectibles {
		cwi := CollectibleWithItem{Collectible: col}
		if col.ItemHash != 0 {
			cwi.Item = items[col.ItemHash]
		}
		results = append(results, cwi)
	}
	return results, nil
}

// getItemsByHashesChunkedLocked fetches item definitions in IN-clause chunks of
// 500 (SQLite parameter limits) assuming r.mu is already held by the caller.
func (r *Repository) getItemsByHashesChunkedLocked(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error) {
	out := make(map[uint32]*bungie.InventoryItemDefinition, len(hashes))
	const chunkSize = 500
	for i := 0; i < len(hashes); i += chunkSize {
		chunk := hashes[i:min(i+chunkSize, len(hashes))]
		defs, err := r.getItemsByHashesLocked(chunk)
		if err != nil {
			return nil, err
		}
		maps.Copy(out, defs)
	}
	return out, nil
}

// cosmeticItemTypes is the set of itemType values bucketed as cosmetics.
var cosmeticItemTypes = map[int]struct{}{
	14: {}, // Emblem
	21: {}, // Ship
	22: {}, // Sparrow
	23: {}, // Emote
	24: {}, // Ghost
}

// CollectibleCategory classifies an item into one of the collection summary
// buckets, or "" for item types the summary does not count (mods, etc.).
func CollectibleCategory(item *bungie.InventoryItemDefinition) string {
	if item == nil {
		return ""
	}
	if _, isCosmetic := cosmeticItemTypes[item.ItemType]; isCosmetic {
		return "cosmetics"
	}
	isExotic := item.Inventory.TierType == bungie.TierTypeExotic
	switch item.ItemType {
	case bungie.ItemTypeWeapon:
		if isExotic {
			return "exotics"
		}
		return "weapons"
	case bungie.ItemTypeArmor:
		if isExotic {
			return "exotics"
		}
		return "armor"
	}
	return ""
}

// GetItemsByHashes fetches inventory item definitions for a batch of hashes.
// Returns a map from hash to definition. Items not found in the manifest are omitted.
// Hashes are stored as signed int32 in SQLite; the conversion is handled internally.
func (r *Repository) GetItemsByHashes(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error) {
	if len(hashes) == 0 {
		return map[uint32]*bungie.InventoryItemDefinition{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.getItemsByHashesLocked(hashes)
}

// getItemsByHashesLocked is GetItemsByHashes assuming r.mu is already held.
func (r *Repository) getItemsByHashesLocked(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error) {
	if len(hashes) == 0 {
		return map[uint32]*bungie.InventoryItemDefinition{}, nil
	}
	placeholders := make([]string, len(hashes))
	args := make([]any, len(hashes))
	for i, h := range hashes {
		placeholders[i] = "?"
		args[i] = int32(h) // SQLite stores hashes as signed int32
	}
	q := "SELECT id, json FROM DestinyInventoryItemDefinition WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("GetItemsByHashes: %w", err)
	}
	defer rows.Close()

	out := make(map[uint32]*bungie.InventoryItemDefinition)
	for rows.Next() {
		var dbID int32
		var blob string
		if err := rows.Scan(&dbID, &blob); err != nil {
			return nil, fmt.Errorf("GetItemsByHashes scan: %w", err)
		}
		var def bungie.InventoryItemDefinition
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		out[uint32(dbID)] = &def
	}
	return out, rows.Err()
}

// GetMilestoneDefinitions fetches milestone definitions for a batch of hashes.
// Hashes are chunked at 500 to avoid SQLite IN-clause limits.
func (r *Repository) GetMilestoneDefinitions(hashes []uint32) (map[uint32]*bungie.MilestoneDefinition, error) {
	if len(hashes) == 0 {
		return map[uint32]*bungie.MilestoneDefinition{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[uint32]*bungie.MilestoneDefinition, len(hashes))
	const chunkSize = 500
	for i := 0; i < len(hashes); i += chunkSize {
		chunk := hashes[i:min(i+chunkSize, len(hashes))]
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for j, h := range chunk {
			placeholders[j] = "?"
			args[j] = int32(h)
		}
		q := "SELECT json FROM DestinyMilestoneDefinition WHERE id IN (" + strings.Join(placeholders, ",") + ")"
		rows, err := r.db.Query(q, args...)
		if err != nil {
			return nil, fmt.Errorf("GetMilestoneDefinitions: %w", err)
		}
		for rows.Next() {
			var blob string
			if err := rows.Scan(&blob); err != nil {
				rows.Close()
				return nil, err
			}
			var def bungie.MilestoneDefinition
			if err := json.Unmarshal([]byte(blob), &def); err != nil {
				continue
			}
			if def.Hash != 0 {
				out[def.Hash] = &def
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// PresentationNodeDef is a minimal DestinyPresentationNodeDefinition from the manifest.
type PresentationNodeDef struct {
	Hash              uint32                   `json:"hash"`
	DisplayProperties bungie.DisplayProperties `json:"displayProperties"`
	Children          struct {
		PresentationNodes []struct {
			PresentationNodeHash uint32 `json:"presentationNodeHash"`
		} `json:"presentationNodes"`
		Collectibles []struct {
			CollectibleHash uint32 `json:"collectibleHash"`
		} `json:"collectibles"`
		Records []struct {
			RecordHash uint32 `json:"recordHash"`
		} `json:"records"`
	} `json:"children"`
	CompletionRecordHash uint32 `json:"completionRecordHash"`
}

// RecordDef is a minimal DestinyRecordDefinition from the manifest.
type RecordDef struct {
	Hash              uint32                   `json:"hash"`
	DisplayProperties bungie.DisplayProperties `json:"displayProperties"`
	Objectives        struct {
		ObjectiveHashes   []uint32 `json:"objectiveHashes"`
		PerkObjectiveHash uint32   `json:"perkObjectiveHash"`
	} `json:"objectives"`
	IntervalObjectives []struct {
		IntervalObjectiveHash uint32 `json:"intervalObjectiveHash"`
	} `json:"intervalObjectives"`
	ForTitleGilding  bool     `json:"forTitleGilding"`
	ParentNodeHashes []uint32 `json:"parentNodeHashes"`
}

// GetPresentationNodeDefinitions fetches a batch of presentation node definitions by hash.
// Hashes are chunked at 500 to avoid SQLite IN-clause limits.
func (r *Repository) GetPresentationNodeDefinitions(hashes []uint32) (map[uint32]*PresentationNodeDef, error) {
	if len(hashes) == 0 {
		return map[uint32]*PresentationNodeDef{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[uint32]*PresentationNodeDef, len(hashes))
	const chunkSize = 500
	for i := 0; i < len(hashes); i += chunkSize {
		chunk := hashes[i:min(i+chunkSize, len(hashes))]

		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for j, h := range chunk {
			placeholders[j] = "?"
			args[j] = hashToDBKey(h)
		}
		query := fmt.Sprintf("SELECT json FROM DestinyPresentationNodeDefinition WHERE id IN (%s)",
			strings.Join(placeholders, ","))
		rows, err := r.db.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("GetPresentationNodeDefinitions: %w", err)
		}
		for rows.Next() {
			var blob string
			if err := rows.Scan(&blob); err != nil {
				rows.Close()
				return nil, err
			}
			var def PresentationNodeDef
			if err := json.Unmarshal([]byte(blob), &def); err != nil {
				continue
			}
			results[def.Hash] = &def
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return results, nil
}

// GetAllPresentationNodes returns every presentation node keyed by hash. One table
// scan — callers (the collection-tree builder) should cache the result, as it is
// manifest-version-dependent but user-independent.
func (r *Repository) GetAllPresentationNodes() (map[uint32]*PresentationNodeDef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows, err := r.db.Query("SELECT json FROM DestinyPresentationNodeDefinition")
	if err != nil {
		return nil, fmt.Errorf("GetAllPresentationNodes: %w", err)
	}
	defer rows.Close()
	out := make(map[uint32]*PresentationNodeDef)
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, fmt.Errorf("GetAllPresentationNodes scan: %w", err)
		}
		var def PresentationNodeDef
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		if def.Hash != 0 {
			out[def.Hash] = &def
		}
	}
	return out, rows.Err()
}

// GetRecordDefinitions fetches a batch of record definitions by hash.
// Hashes are chunked at 500 to avoid SQLite IN-clause limits.
func (r *Repository) GetRecordDefinitions(hashes []uint32) (map[uint32]*RecordDef, error) {
	if len(hashes) == 0 {
		return map[uint32]*RecordDef{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[uint32]*RecordDef, len(hashes))
	const chunkSize = 500
	for i := 0; i < len(hashes); i += chunkSize {
		chunk := hashes[i:min(i+chunkSize, len(hashes))]

		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for j, h := range chunk {
			placeholders[j] = "?"
			args[j] = hashToDBKey(h)
		}
		query := fmt.Sprintf("SELECT json FROM DestinyRecordDefinition WHERE id IN (%s)",
			strings.Join(placeholders, ","))
		rows, err := r.db.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("GetRecordDefinitions: %w", err)
		}
		for rows.Next() {
			var blob string
			if err := rows.Scan(&blob); err != nil {
				rows.Close()
				return nil, err
			}
			var def RecordDef
			if err := json.Unmarshal([]byte(blob), &def); err != nil {
				continue
			}
			results[def.Hash] = &def
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return results, nil
}

// GetActivityDefinitions fetches activity definitions for a batch of hashes.
// Hashes are chunked at 500 to avoid SQLite IN-clause limits.
func (r *Repository) GetActivityDefinitions(hashes []uint32) (map[uint32]*bungie.ActivityDefinition, error) {
	if len(hashes) == 0 {
		return map[uint32]*bungie.ActivityDefinition{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[uint32]*bungie.ActivityDefinition, len(hashes))
	const chunkSize = 500
	for i := 0; i < len(hashes); i += chunkSize {
		chunk := hashes[i:min(i+chunkSize, len(hashes))]
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for j, h := range chunk {
			placeholders[j] = "?"
			args[j] = int32(h)
		}
		q := "SELECT json FROM DestinyActivityDefinition WHERE id IN (" + strings.Join(placeholders, ",") + ")"
		rows, err := r.db.Query(q, args...)
		if err != nil {
			return nil, fmt.Errorf("GetActivityDefinitions: %w", err)
		}
		for rows.Next() {
			var blob string
			if err := rows.Scan(&blob); err != nil {
				rows.Close()
				return nil, err
			}
			var def bungie.ActivityDefinition
			if err := json.Unmarshal([]byte(blob), &def); err != nil {
				continue
			}
			if def.Hash != 0 {
				out[def.Hash] = &def
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// GetActivityModifierDefinitions fetches activity modifier definitions for a batch of hashes.
// Hashes are chunked at 500 to avoid SQLite IN-clause limits.
func (r *Repository) GetActivityModifierDefinitions(hashes []uint32) (map[uint32]*bungie.ActivityModifierDefinition, error) {
	if len(hashes) == 0 {
		return map[uint32]*bungie.ActivityModifierDefinition{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[uint32]*bungie.ActivityModifierDefinition, len(hashes))
	const chunkSize = 500
	for i := 0; i < len(hashes); i += chunkSize {
		chunk := hashes[i:min(i+chunkSize, len(hashes))]
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for j, h := range chunk {
			placeholders[j] = "?"
			args[j] = int32(h)
		}
		q := "SELECT json FROM DestinyActivityModifierDefinition WHERE id IN (" + strings.Join(placeholders, ",") + ")"
		rows, err := r.db.Query(q, args...)
		if err != nil {
			return nil, fmt.Errorf("GetActivityModifierDefinitions: %w", err)
		}
		for rows.Next() {
			var blob string
			if err := rows.Scan(&blob); err != nil {
				rows.Close()
				return nil, err
			}
			var def bungie.ActivityModifierDefinition
			if err := json.Unmarshal([]byte(blob), &def); err != nil {
				continue
			}
			if def.Hash != 0 {
				out[def.Hash] = &def
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// GetCollectiblesByItemHashes returns collectible definitions keyed by their
// itemHash for a batch of inventory item hashes (e.g. to read sourceString for
// wishlist items). Hashes are chunked at 500 to avoid SQLite IN-clause limits.
func (r *Repository) GetCollectiblesByItemHashes(hashes []uint32) (map[uint32]*bungie.CollectibleDefinition, error) {
	if len(hashes) == 0 {
		return map[uint32]*bungie.CollectibleDefinition{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[uint32]*bungie.CollectibleDefinition, len(hashes))
	const chunkSize = 500
	for i := 0; i < len(hashes); i += chunkSize {
		chunk := hashes[i:min(i+chunkSize, len(hashes))]
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for j, h := range chunk {
			placeholders[j] = "?"
			args[j] = int64(h) // itemHash is stored unsigned inside the JSON blob
		}
		q := "SELECT json FROM DestinyCollectibleDefinition WHERE json_extract(json, '$.itemHash') IN (" +
			strings.Join(placeholders, ",") + ")"
		rows, err := r.db.Query(q, args...)
		if err != nil {
			return nil, fmt.Errorf("GetCollectiblesByItemHashes: %w", err)
		}
		for rows.Next() {
			var blob string
			if err := rows.Scan(&blob); err != nil {
				rows.Close()
				return nil, fmt.Errorf("GetCollectiblesByItemHashes scan: %w", err)
			}
			var def bungie.CollectibleDefinition
			if err := json.Unmarshal([]byte(blob), &def); err != nil {
				continue
			}
			if def.ItemHash != 0 {
				out[def.ItemHash] = &def
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// GetWeaponTypesByName returns a lowercased weapon display name → weapon type
// display name map (e.g. "ace of spades" → "Hand Cannon") covering every weapon
// definition in the manifest. One table scan — callers should cache the result.
func (r *Repository) GetWeaponTypesByName() (map[string]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows, err := r.db.Query(
		"SELECT json FROM DestinyInventoryItemDefinition WHERE json_extract(json, '$.itemType') = ?",
		bungie.ItemTypeWeapon,
	)
	if err != nil {
		return nil, fmt.Errorf("GetWeaponTypesByName: %w", err)
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			continue
		}
		var def struct {
			DisplayProperties struct {
				Name string `json:"name"`
			} `json:"displayProperties"`
			ItemSubType int `json:"itemSubType"`
		}
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		if def.DisplayProperties.Name == "" {
			continue
		}
		out[strings.ToLower(def.DisplayProperties.Name)] = bungie.GetWeaponTypeName(def.ItemSubType)
	}
	return out, rows.Err()
}
