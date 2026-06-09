package manifest

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
	rows, err := r.db.Query("SELECT json FROM DestinyCollectibleDefinition")
	if err != nil {
		return nil, fmt.Errorf("failed to query collectibles: %w", err)
	}
	defer rows.Close()
	var out []bungie.CollectibleDefinition
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			continue
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
	collectibles, err := r.GetAllCollectibles()
	if err != nil {
		return nil, err
	}
	var results []CollectibleWithItem
	for _, col := range collectibles {
		cwi := CollectibleWithItem{Collectible: col}
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

// FilteredCollectibles holds collectibles split into categories.
type FilteredCollectibles struct {
	Weapons []CollectibleWithItem
	Armor   []CollectibleWithItem
	Exotics []CollectibleWithItem
}

func (r *Repository) GetFilteredCollectibles() (*FilteredCollectibles, error) {
	all, err := r.GetAllCollectiblesWithItems()
	if err != nil {
		return nil, err
	}
	result := &FilteredCollectibles{
		Weapons: make([]CollectibleWithItem, 0),
		Armor:   make([]CollectibleWithItem, 0),
		Exotics: make([]CollectibleWithItem, 0),
	}
	for _, cwi := range all {
		if cwi.Item == nil || cwi.Item.DisplayProperties.Name == "" {
			continue
		}
		isExotic := cwi.Item.Inventory.TierType == bungie.TierTypeExotic
		switch cwi.Item.ItemType {
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

// SearchItems does a case-insensitive name search against the manifest.
func (r *Repository) SearchItems(query string, limit int) ([]bungie.InventoryItemDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows, err := r.db.Query(
		"SELECT json FROM DestinyInventoryItemDefinition WHERE json LIKE ? LIMIT ?",
		"%"+query+"%", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search items: %w", err)
	}
	defer rows.Close()
	var items []bungie.InventoryItemDefinition
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			continue
		}
		var def bungie.InventoryItemDefinition
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		if def.DisplayProperties.Name != "" &&
			(def.ItemType == bungie.ItemTypeWeapon || def.ItemType == bungie.ItemTypeArmor) {
			items = append(items, def)
		}
	}
	return items, rows.Err()
}
