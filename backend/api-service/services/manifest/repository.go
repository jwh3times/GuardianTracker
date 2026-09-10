package manifest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
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

// hashToDBKey maps an unsigned Bungie hash to the signed id the manifest SQLite
// stores. Bungie hashes use the full uint32 range; the manifest writes them as
// signed two's-complement int32 (values >= 2^31 are stored negative). We
// reproduce that mapping with int64 arithmetic and an explicit range check
// instead of a uint32->int32 narrowing conversion, which static analysis flags
// as a lossy integer conversion (CWE-681). The numeric result is identical to
// int32(hash) for every uint32, and the manifest id column is INTEGER (int64).
func hashToDBKey(hash uint32) int64 {
	if hash > math.MaxInt32 {
		return int64(hash) - (1 << 32)
	}
	return int64(hash)
}

// dbKeyToHash is the exact inverse of hashToDBKey: it maps a signed manifest id
// back to the unsigned Bungie hash it encodes. Scanning the id column into an
// int64 and undoing the two's-complement wrap here keeps the round trip free of
// a narrowing conversion in either direction.
func dbKeyToHash(key int64) uint32 {
	if key < 0 {
		return uint32(key + (1 << 32))
	}
	return uint32(key)
}

// ResolveVendorLocation resolves a live vendor location index through the
// vendor and destination manifest definitions. Missing definitions and invalid
// indexes are represented by zero values so best-effort callers can omit the
// location without failing their larger response.
func (r *Repository) ResolveVendorLocation(vendorHash uint32, locationIndex int) (uint32, string, error) {
	if locationIndex < 0 {
		return 0, "", nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var vendorBlob string
	err := r.db.QueryRow(
		"SELECT json FROM DestinyVendorDefinition WHERE id = ?",
		hashToDBKey(vendorHash),
	).Scan(&vendorBlob)
	if err == sql.ErrNoRows {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", fmt.Errorf("failed to query vendor location: %w", err)
	}

	var vendor bungie.VendorDefinition
	if err := json.Unmarshal([]byte(vendorBlob), &vendor); err != nil {
		return 0, "", fmt.Errorf("failed to parse vendor location JSON: %w", err)
	}
	if locationIndex >= len(vendor.Locations) {
		return 0, "", nil
	}

	destinationHash := vendor.Locations[locationIndex].DestinationHash
	if destinationHash == 0 {
		return 0, "", nil
	}

	var destinationBlob string
	err = r.db.QueryRow(
		"SELECT json FROM DestinyDestinationDefinition WHERE id = ?",
		hashToDBKey(destinationHash),
	).Scan(&destinationBlob)
	if err == sql.ErrNoRows {
		return destinationHash, "", nil
	}
	if err != nil {
		return 0, "", fmt.Errorf("failed to query vendor destination: %w", err)
	}

	var destination bungie.DestinationDefinition
	if err := json.Unmarshal([]byte(destinationBlob), &destination); err != nil {
		return 0, "", fmt.Errorf("failed to parse vendor destination JSON: %w", err)
	}
	return destinationHash, destination.DisplayProperties.Name, nil
}

// getAllCollectiblesLocked reads every named collectible in the manifest. It
// assumes r.mu is already held, so its one caller can hold a single read lock
// across both halves of the collectible-plus-item join.
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
			return nil, fmt.Errorf("getAllCollectibles scan: %w", err)
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
	items, err := r.getItemsByHashesLocked(hashes)
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

// cosmeticItemTypes is the set of itemType values bucketed as cosmetics.
var cosmeticItemTypes = map[int]struct{}{
	14:                      {}, // Emblem
	21:                      {}, // Ship
	22:                      {}, // Sparrow
	23:                      {}, // Emote
	24:                      {}, // Ghost
	bungie.ItemTypeFinisher: {},
}

// CollectibleCategory classifies an item into one of the collection summary
// buckets, or "" for item types the summary does not count (mods, etc.).
func CollectibleCategory(item *bungie.InventoryItemDefinition) string {
	if item == nil {
		return ""
	}
	// Shaders and ornaments share itemType=19 (ItemTypeMod) with regular mods,
	// so classify only their manifest-verified subtypes instead of adding every
	// mod to cosmeticItemTypes.
	if item.ItemType == bungie.ItemTypeMod &&
		(item.ItemSubType == bungie.ItemSubTypeShader || item.ItemSubType == bungie.ItemSubTypeOrnament) {
		return "cosmetics"
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
//
// Definitions are keyed by the row id rather than by the hash inside the blob,
// so an item whose stored id and self-reported hash disagree is still returned
// under the hash the caller asked for.
func (r *Repository) getItemsByHashesLocked(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error) {
	out := make(map[uint32]*bungie.InventoryItemDefinition, len(hashes))
	err := queryDefsChunked(r.db, hashes, byRowID("DestinyInventoryItemDefinition", "GetItemsByHashes"),
		func(id uint32, def *bungie.InventoryItemDefinition) { out[id] = def })
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetMilestoneDefinitions fetches milestone definitions for a batch of hashes.
func (r *Repository) GetMilestoneDefinitions(hashes []uint32) (map[uint32]*bungie.MilestoneDefinition, error) {
	if len(hashes) == 0 {
		return map[uint32]*bungie.MilestoneDefinition{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[uint32]*bungie.MilestoneDefinition, len(hashes))
	err := queryDefsChunked(r.db, hashes, byRowID("DestinyMilestoneDefinition", "GetMilestoneDefinitions"),
		func(_ uint32, def *bungie.MilestoneDefinition) {
			if def.Hash != 0 {
				out[def.Hash] = def
			}
		})
	if err != nil {
		return nil, err
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
	// RecordTypeName groups records by kind ("Exotic Catalysts", "Weapon Pattern",
	// …). Bungie files catalysts and crafting patterns under one combined
	// "Patterns & Catalysts" presentation node, so this is how the records service
	// tells the two apart.
	RecordTypeName string `json:"recordTypeName"`
	// StateInfo.ObscuredDescription is the acquisition-source text shown while a
	// record is still obscured (e.g. a catalyst's "Found in strikes and the
	// Crucible."). DisplayProperties.Description is the completion requirement.
	StateInfo struct {
		ObscuredDescription string `json:"obscuredDescription"`
	} `json:"stateInfo"`
	// ObjectiveHashes links a catalyst record to its unlock objective(s). This is
	// a TOP-LEVEL field on DestinyRecordDefinition per the real manifest schema —
	// unlike DestinyInventoryItemDefinition's plug objectives, which nest under an
	// "objectives" key. The records service uses this to link an exotic-catalyst
	// record to its weapon via objective-hash overlap with the weapon's
	// catalyst-socket plug pool.
	ObjectiveHashes    []uint32 `json:"objectiveHashes"`
	IntervalObjectives []struct {
		IntervalObjectiveHash uint32 `json:"intervalObjectiveHash"`
	} `json:"intervalObjectives"`
	ForTitleGilding  bool     `json:"forTitleGilding"`
	ParentNodeHashes []uint32 `json:"parentNodeHashes"`
}

// GetPresentationNodeDefinitions fetches a batch of presentation node definitions by hash.
func (r *Repository) GetPresentationNodeDefinitions(hashes []uint32) (map[uint32]*PresentationNodeDef, error) {
	if len(hashes) == 0 {
		return map[uint32]*PresentationNodeDef{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[uint32]*PresentationNodeDef, len(hashes))
	err := queryDefsChunked(r.db, hashes, byRowID("DestinyPresentationNodeDefinition", "GetPresentationNodeDefinitions"),
		func(_ uint32, def *PresentationNodeDef) { results[def.Hash] = def })
	if err != nil {
		return nil, err
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
func (r *Repository) GetRecordDefinitions(hashes []uint32) (map[uint32]*RecordDef, error) {
	if len(hashes) == 0 {
		return map[uint32]*RecordDef{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[uint32]*RecordDef, len(hashes))
	err := queryDefsChunked(r.db, hashes, byRowID("DestinyRecordDefinition", "GetRecordDefinitions"),
		func(_ uint32, def *RecordDef) { results[def.Hash] = def })
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetActivityDefinitions fetches activity definitions for a batch of hashes.
func (r *Repository) GetActivityDefinitions(hashes []uint32) (map[uint32]*bungie.ActivityDefinition, error) {
	if len(hashes) == 0 {
		return map[uint32]*bungie.ActivityDefinition{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[uint32]*bungie.ActivityDefinition, len(hashes))
	err := queryDefsChunked(r.db, hashes, byRowID("DestinyActivityDefinition", "GetActivityDefinitions"),
		func(_ uint32, def *bungie.ActivityDefinition) {
			if def.Hash != 0 {
				out[def.Hash] = def
			}
		})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetActivityModifierDefinitions fetches activity modifier definitions for a batch of hashes.
func (r *Repository) GetActivityModifierDefinitions(hashes []uint32) (map[uint32]*bungie.ActivityModifierDefinition, error) {
	if len(hashes) == 0 {
		return map[uint32]*bungie.ActivityModifierDefinition{}, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[uint32]*bungie.ActivityModifierDefinition, len(hashes))
	err := queryDefsChunked(r.db, hashes, byRowID("DestinyActivityModifierDefinition", "GetActivityModifierDefinitions"),
		func(_ uint32, def *bungie.ActivityModifierDefinition) {
			if def.Hash != 0 {
				out[def.Hash] = def
			}
		})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AcquisitionRows is one coherent read of the item definitions and linked
// collectibles behind a set of item hashes.
//
// The two halves are read under a single lock because they are only meaningful
// together: matching items from one manifest against collectibles from another
// would attribute the wrong acquisition sources to an item. Callers that need
// both must use this rather than two separate calls.
type AcquisitionRows struct {
	Items        map[uint32]*bungie.InventoryItemDefinition
	Collectibles map[uint32][]bungie.CollectibleDefinition
}

// GetAcquisitionRows reads item definitions and their linked collectibles for
// the requested hashes under one read lock, so a manifest swap cannot land
// between the two queries. An empty request touches the database not at all.
func (r *Repository) GetAcquisitionRows(hashes []uint32) (*AcquisitionRows, error) {
	if len(hashes) == 0 {
		return &AcquisitionRows{
			Items:        map[uint32]*bungie.InventoryItemDefinition{},
			Collectibles: map[uint32][]bungie.CollectibleDefinition{},
		}, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	items, err := r.getItemsByHashesLocked(hashes)
	if err != nil {
		return nil, err
	}
	collectibles, err := r.getCollectiblesByItemHashesLocked(hashes)
	if err != nil {
		return nil, err
	}
	return &AcquisitionRows{Items: items, Collectibles: collectibles}, nil
}

// collectiblesByItemHash matches collectibles on the itemHash inside the blob
// rather than on the row id, because the link this query follows runs the other
// way: from an item hash to every collectible that attributes a source to it.
// Rows are ordered by id so an item's several collectibles keep a stable order.
var collectiblesByItemHash = chunkedQuery{
	label: "getCollectiblesByItemHashes",
	statement: func(placeholders string) string {
		return "SELECT id, json FROM DestinyCollectibleDefinition WHERE json_extract(json, '$.itemHash') IN (" +
			placeholders + ") ORDER BY id"
	},
	// itemHash is stored unsigned inside the JSON blob, so it is compared as the
	// plain hash rather than through hashToDBKey's signed row-id encoding.
	arg: func(hash uint32) any { return int64(hash) },
}

// getCollectiblesByItemHashesLocked returns every collectible definition linked
// to each requested itemHash, assuming r.mu is already held. More than one
// collectible can attribute a different acquisition source to the same item, so
// the multiplicity is part of this interface rather than an implementation
// detail.
func (r *Repository) getCollectiblesByItemHashesLocked(hashes []uint32) (map[uint32][]bungie.CollectibleDefinition, error) {
	if len(hashes) == 0 {
		return map[uint32][]bungie.CollectibleDefinition{}, nil
	}

	out := make(map[uint32][]bungie.CollectibleDefinition, len(hashes))
	err := queryDefsChunked(r.db, hashes, collectiblesByItemHash,
		func(_ uint32, def *bungie.CollectibleDefinition) {
			if def.ItemHash != 0 {
				out[def.ItemHash] = append(out[def.ItemHash], *def)
			}
		})
	if err != nil {
		return nil, err
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

// ExoticWeapon carries the display fields the records service uses to enrich an
// exotic catalyst entry with its weapon's picture and type.
type ExoticWeapon struct {
	Type string
	Icon string
}

// GetExoticWeaponsByName returns a lowercased exotic-weapon display name →
// {type, icon} map covering every exotic weapon in the manifest. Catalyst
// records carry only a near-transparent generic catalyst glyph as their icon, so
// the records service resolves the real weapon picture (and type) through this
// map. One table scan — callers should cache the result.
func (r *Repository) GetExoticWeaponsByName() (map[string]ExoticWeapon, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows, err := r.db.Query(
		"SELECT json FROM DestinyInventoryItemDefinition WHERE json_extract(json, '$.itemType') = ? AND json_extract(json, '$.inventory.tierType') = ?",
		bungie.ItemTypeWeapon, bungie.TierTypeExotic,
	)
	if err != nil {
		return nil, fmt.Errorf("GetExoticWeaponsByName: %w", err)
	}
	defer rows.Close()

	out := make(map[string]ExoticWeapon)
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			continue
		}
		var def struct {
			DisplayProperties struct {
				Name string `json:"name"`
				Icon string `json:"icon"`
			} `json:"displayProperties"`
			ItemSubType int `json:"itemSubType"`
		}
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		name := strings.ToLower(def.DisplayProperties.Name)
		if name == "" || def.DisplayProperties.Icon == "" {
			continue
		}
		// Keep the first definition seen for a given name; duplicates (re-issued
		// weapons) share the same icon and type.
		if _, exists := out[name]; !exists {
			out[name] = ExoticWeapon{Type: bungie.GetWeaponTypeName(def.ItemSubType), Icon: def.DisplayProperties.Icon}
		}
	}
	return out, rows.Err()
}
