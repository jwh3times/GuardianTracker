package manifest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"guardian-tracker/bungie-service/services/bungie"

	_ "github.com/mattn/go-sqlite3"
)

// Repository handles queries to the manifest SQLite database
type Repository struct {
	dbPath string
	db     *sql.DB
	mu     sync.RWMutex
}

// NewRepository creates a new manifest repository
func NewRepository(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro&cache=shared")
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest database: %w", err)
	}

	// Set connection pool settings for read-only access
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to manifest database: %w", err)
	}

	return &Repository{
		dbPath: dbPath,
		db:     db,
	}, nil
}

// Close closes the database connection
func (r *Repository) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// Reconnect reopens the database connection (useful after manifest update)
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

// hashToDBKey converts a uint32 hash to the signed int32 used in Bungie's SQLite
func hashToDBKey(hash uint32) int32 {
	return int32(hash)
}

// GetCollectibleDefinition retrieves a collectible definition by hash
func (r *Repository) GetCollectibleDefinition(hash uint32) (*bungie.CollectibleDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var jsonBlob string
	err := r.db.QueryRow(
		"SELECT json FROM DestinyCollectibleDefinition WHERE id = ?",
		hashToDBKey(hash),
	).Scan(&jsonBlob)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query collectible: %w", err)
	}

	var def bungie.CollectibleDefinition
	if err := json.Unmarshal([]byte(jsonBlob), &def); err != nil {
		return nil, fmt.Errorf("failed to parse collectible JSON: %w", err)
	}

	return &def, nil
}

// GetInventoryItemDefinition retrieves an item definition by hash
func (r *Repository) GetInventoryItemDefinition(hash uint32) (*bungie.InventoryItemDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var jsonBlob string
	err := r.db.QueryRow(
		"SELECT json FROM DestinyInventoryItemDefinition WHERE id = ?",
		hashToDBKey(hash),
	).Scan(&jsonBlob)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query item: %w", err)
	}

	var def bungie.InventoryItemDefinition
	if err := json.Unmarshal([]byte(jsonBlob), &def); err != nil {
		return nil, fmt.Errorf("failed to parse item JSON: %w", err)
	}

	return &def, nil
}

// GetAllCollectibles retrieves all collectible definitions
func (r *Repository) GetAllCollectibles() ([]bungie.CollectibleDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.Query("SELECT json FROM DestinyCollectibleDefinition")
	if err != nil {
		return nil, fmt.Errorf("failed to query collectibles: %w", err)
	}
	defer rows.Close()

	var collectibles []bungie.CollectibleDefinition
	for rows.Next() {
		var jsonBlob string
		if err := rows.Scan(&jsonBlob); err != nil {
			continue
		}

		var def bungie.CollectibleDefinition
		if err := json.Unmarshal([]byte(jsonBlob), &def); err != nil {
			continue
		}

		// Skip items without names (placeholder entries)
		if def.DisplayProperties.Name == "" {
			continue
		}

		collectibles = append(collectibles, def)
	}

	return collectibles, rows.Err()
}

// CollectibleWithItem combines a collectible with its item definition
type CollectibleWithItem struct {
	Collectible bungie.CollectibleDefinition
	Item        *bungie.InventoryItemDefinition
}

// GetAllCollectiblesWithItems retrieves all collectibles with their corresponding item definitions
func (r *Repository) GetAllCollectiblesWithItems() ([]CollectibleWithItem, error) {
	collectibles, err := r.GetAllCollectibles()
	if err != nil {
		return nil, err
	}

	var results []CollectibleWithItem
	for _, col := range collectibles {
		cwi := CollectibleWithItem{Collectible: col}

		// Get the item definition if available
		if col.ItemHash != 0 {
			item, err := r.GetInventoryItemDefinition(col.ItemHash)
			if err == nil && item != nil {
				cwi.Item = item
			}
		}

		results = append(results, cwi)
	}

	return results, nil
}

// FilteredCollectibles holds categorized collectibles
type FilteredCollectibles struct {
	Weapons  []CollectibleWithItem
	Armor    []CollectibleWithItem
	Exotics  []CollectibleWithItem
}

// GetFilteredCollectibles retrieves and categorizes all collectibles
func (r *Repository) GetFilteredCollectibles() (*FilteredCollectibles, error) {
	allItems, err := r.GetAllCollectiblesWithItems()
	if err != nil {
		return nil, err
	}

	result := &FilteredCollectibles{
		Weapons:  make([]CollectibleWithItem, 0),
		Armor:    make([]CollectibleWithItem, 0),
		Exotics:  make([]CollectibleWithItem, 0),
	}

	for _, cwi := range allItems {
		// Skip items without proper item definition
		if cwi.Item == nil {
			continue
		}

		item := cwi.Item

		// Skip items without names
		if item.DisplayProperties.Name == "" {
			continue
		}

		// Check if exotic (goes into exotics category regardless of type)
		isExotic := item.Inventory.TierType == bungie.TierTypeExotic

		// Categorize by item type
		switch item.ItemType {
		case bungie.ItemTypeWeapon:
			if isExotic {
				result.Exotics = append(result.Exotics, cwi)
			} else {
				result.Weapons = append(result.Weapons, cwi)
			}
		case bungie.ItemTypeArmor:
			if isExotic {
				result.Exotics = append(result.Exotics, cwi)
			} else {
				result.Armor = append(result.Armor, cwi)
			}
		}
	}

	return result, nil
}

// SearchItems searches for items by name (case-insensitive)
func (r *Repository) SearchItems(query string, limit int) ([]bungie.InventoryItemDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Use LIKE for basic search
	rows, err := r.db.Query(`
		SELECT json FROM DestinyInventoryItemDefinition
		WHERE json LIKE ?
		LIMIT ?
	`, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search items: %w", err)
	}
	defer rows.Close()

	var items []bungie.InventoryItemDefinition
	for rows.Next() {
		var jsonBlob string
		if err := rows.Scan(&jsonBlob); err != nil {
			continue
		}

		var def bungie.InventoryItemDefinition
		if err := json.Unmarshal([]byte(jsonBlob), &def); err != nil {
			continue
		}

		// Only include items with names and valid types
		if def.DisplayProperties.Name != "" &&
			(def.ItemType == bungie.ItemTypeWeapon || def.ItemType == bungie.ItemTypeArmor) {
			items = append(items, def)
		}
	}

	return items, rows.Err()
}
