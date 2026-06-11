package search

import (
	"database/sql"
	"encoding/json"
	"log"
	"sort"
	"strings"
	"sync"

	"guardian-tracker/api-service/services/bungie"

	_ "github.com/mattn/go-sqlite3"
)

// Entry is a single item in the search index, suitable for JSON responses.
type Entry struct {
	Hash      uint32 `json:"hash"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Type      string `json:"type"`
	Rarity    string `json:"rarity"`
	nameLower string
}

// Service holds the in-memory search index built from the manifest SQLite database.
type Service struct {
	manifestService *bungie.ManifestService
	dbPath          string
	mu              sync.RWMutex
	entries         []Entry
	builtVersion    string
	building        bool
}

// includedItemTypes is the set of Bungie itemType values we index for search.
// 2=Armor, 3=Weapon, 14=Emblem, 21=Ship, 22=Sparrow, 23=Emote, 24=Ghost
var includedItemTypes = map[int]struct{}{
	2: {}, 3: {}, 14: {}, 21: {}, 22: {}, 23: {}, 24: {},
}

// NewService creates a search service backed by the given manifest service and db path.
func NewService(ms *bungie.ManifestService, dbPath string) *Service {
	return &Service{manifestService: ms, dbPath: dbPath}
}

// IsReady returns true when the search index has been built at least once.
func (s *Service) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries) > 0
}

// Search returns up to limit entries matching q (case-insensitive substring).
// Prefix matches are ranked before mid-name matches.
// Returns nil slice (not error) when index not yet built.
func (s *Service) Search(q string, limit int) []Entry {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	s.ensureIndex()
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.entries) == 0 {
		return nil
	}
	lower := strings.ToLower(q)
	type scored struct {
		e     Entry
		score int // 0 = prefix match, 1 = substring match
	}
	var matches []scored
	for _, e := range s.entries {
		if strings.HasPrefix(e.nameLower, lower) {
			matches = append(matches, scored{e, 0})
		} else if strings.Contains(e.nameLower, lower) {
			matches = append(matches, scored{e, 1})
		}
		// Collect a larger candidate pool before trimming, so sort has more prefix hits to promote
		if len(matches) >= limit*3 {
			break
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].score < matches[j].score
	})
	out := make([]Entry, 0, limit)
	for i := range matches {
		if i >= limit {
			break
		}
		out = append(out, matches[i].e)
	}
	return out
}

// BuildIndex builds the search index from the manifest SQLite database.
// Safe to call concurrently — a concurrent call is a no-op if a build is already running.
func (s *Service) BuildIndex() {
	s.mu.Lock()
	if s.building {
		s.mu.Unlock()
		return
	}
	s.building = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.building = false
		s.mu.Unlock()
	}()

	// If the manifest service or DB isn't ready yet, bail gracefully.
	if s.manifestService == nil {
		return
	}
	version := s.manifestService.Version()
	if version == "" {
		return
	}

	db, err := sql.Open("sqlite3", s.dbPath+"?mode=ro&cache=shared")
	if err != nil {
		log.Printf("search: open manifest db: %v", err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT json FROM DestinyInventoryItemDefinition")
	if err != nil {
		log.Printf("search: query DestinyInventoryItemDefinition: %v", err)
		return
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			continue
		}
		var def bungie.InventoryItemDefinition
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		if _, ok := includedItemTypes[def.ItemType]; !ok {
			continue
		}
		name := def.DisplayProperties.Name
		if name == "" {
			continue
		}
		entries = append(entries, Entry{
			Hash:      def.Hash,
			Name:      name,
			nameLower: strings.ToLower(name),
			Icon:      def.DisplayProperties.Icon,
			Type:      itemTypeName(def),
			Rarity:    bungie.GetTierName(def.Inventory.TierType),
		})
	}
	if err := rows.Err(); err != nil {
		log.Printf("search: scan error: %v", err)
		return
	}

	s.mu.Lock()
	s.entries = entries
	s.builtVersion = version
	s.mu.Unlock()
	log.Printf("search: index built — %d items (manifest %s)", len(entries), version)
}

// ensureIndex kicks an async rebuild if the manifest version changed since last build.
func (s *Service) ensureIndex() {
	if s.manifestService == nil {
		return
	}
	s.mu.RLock()
	current := s.manifestService.Version()
	needsRebuild := current != "" && s.builtVersion != current && !s.building
	s.mu.RUnlock()
	if needsRebuild {
		go s.BuildIndex()
	}
}

func itemTypeName(def bungie.InventoryItemDefinition) string {
	switch def.ItemType {
	case 3:
		return bungie.GetWeaponTypeName(def.ItemSubType)
	case 2:
		return bungie.GetArmorTypeName(def.EquippingBlock.EquipmentSlotTypeHash)
	case 14:
		return "Emblem"
	case 21:
		return "Ship"
	case 22:
		return "Sparrow"
	case 23:
		return "Emote"
	case 24:
		return "Ghost"
	default:
		return "Item"
	}
}
