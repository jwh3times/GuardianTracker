package search

import (
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	snapshotDir     string
	mu              sync.RWMutex
	entries         []Entry
	builtVersion    string
	building        bool
}

const (
	searchSnapshotPrefix = "search_index_"
	searchSnapshotSuffix = ".json.gz"
	searchSnapshotFormat = 1
)

type indexSnapshot struct {
	Format  int     `json:"format"`
	Version string  `json:"version"`
	Entries []Entry `json:"entries"`
}

// Search result limit bounds. A caller-supplied limit is clamped into
// [1, maxSearchLimit]; out-of-range values fall back to defaultSearchLimit.
const (
	defaultSearchLimit = 20
	maxSearchLimit     = 50
)

// includedItemTypes is the set of Bungie itemType values we index for search.
// 2=Armor, 3=Weapon, 14=Emblem, 21=Ship, 22=Sparrow, 23=Emote, 24=Ghost
var includedItemTypes = map[int]struct{}{
	2: {}, 3: {}, 14: {}, 21: {}, 22: {}, 23: {}, 24: {},
}

// NewService creates a search service backed by the given manifest service and db path.
func NewService(ms *bungie.ManifestService, dbPath string) *Service {
	s := &Service{
		manifestService: ms,
		dbPath:          dbPath,
		snapshotDir:     filepath.Dir(dbPath),
	}
	if ms != nil && s.loadSnapshot(ms.Version()) {
		log.Printf("search: restored index snapshot for manifest %s", ms.Version())
	}
	return s
}

// IsReady returns true when the search index has been built or restored for a
// manifest version.
func (s *Service) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.builtVersion != ""
}

// Search returns up to limit entries matching q (case-insensitive substring).
// Prefix matches are ranked before mid-name matches.
// Returns nil slice (not error) when index not yet built.
func (s *Service) Search(q string, limit int) []Entry {
	if limit <= 0 || limit > maxSearchLimit {
		limit = defaultSearchLimit
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
	// Pre-size by the constant max and the actual match count — neither depends
	// on the caller's limit, so the allocation size can't be driven by user
	// input. The loop below still trims the result to `limit`.
	out := make([]Entry, 0, min(maxSearchLimit, len(matches)))
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
	if err := s.saveSnapshot(version, entries); err != nil {
		log.Printf("search: save snapshot for manifest %s: %v", version, err)
	}
	log.Printf("search: index built — %d items (manifest %s)", len(entries), version)
}

// loadSnapshot restores a snapshot only when its embedded manifest version
// exactly matches the currently installed manifest. Any missing, stale, or
// corrupt snapshot is treated as a cache miss so the normal async rebuild can
// recover the index.
func (s *Service) loadSnapshot(expectedVersion string) bool {
	if expectedVersion == "" {
		return false
	}

	path := s.snapshotPath(expectedVersion)
	file, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("search: open snapshot %s: %v", path, err)
		}
		return false
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		log.Printf("search: ignore corrupt snapshot %s: %v", path, err)
		return false
	}
	defer reader.Close()

	var snapshot indexSnapshot
	if err := json.NewDecoder(reader).Decode(&snapshot); err != nil {
		log.Printf("search: ignore corrupt snapshot %s: %v", path, err)
		return false
	}
	if snapshot.Format != searchSnapshotFormat || snapshot.Version != expectedVersion {
		log.Printf("search: ignore stale or unsupported snapshot %s", path)
		return false
	}
	for i := range snapshot.Entries {
		snapshot.Entries[i].nameLower = strings.ToLower(snapshot.Entries[i].Name)
	}

	s.mu.Lock()
	s.entries = snapshot.Entries
	s.builtVersion = snapshot.Version
	s.mu.Unlock()
	return true
}

// saveSnapshot writes a versioned gzip-JSON index beside the manifest. The
// temporary file plus rename keeps readers from observing a partial snapshot;
// the remove-and-retry fallback handles replacement on Windows.
func (s *Service) saveSnapshot(version string, entries []Entry) error {
	if version == "" {
		return nil
	}
	if err := os.MkdirAll(s.snapshotDir, 0755); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}

	path := s.snapshotPath(version)
	tmp, err := os.CreateTemp(s.snapshotDir, ".search-index-*.tmp")
	if err != nil {
		return fmt.Errorf("create snapshot temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	writer := gzip.NewWriter(tmp)
	snapshot := indexSnapshot{
		Format:  searchSnapshotFormat,
		Version: version,
		Entries: entries,
	}
	if err := json.NewEncoder(writer).Encode(snapshot); err != nil {
		writer.Close()
		tmp.Close()
		return fmt.Errorf("encode snapshot: %w", err)
	}
	if err := writer.Close(); err != nil {
		tmp.Close()
		return fmt.Errorf("close snapshot gzip: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close snapshot: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("replace snapshot: %w", err)
		}
		if err := os.Rename(tmpPath, path); err != nil {
			return fmt.Errorf("rename snapshot: %w", err)
		}
	}

	if err := s.pruneSnapshots(path); err != nil {
		return fmt.Errorf("prune snapshots: %w", err)
	}
	return nil
}

func (s *Service) pruneSnapshots(keepPath string) error {
	paths, err := filepath.Glob(filepath.Join(s.snapshotDir, searchSnapshotPrefix+"*"+searchSnapshotSuffix))
	if err != nil {
		return err
	}
	keepPath = filepath.Clean(keepPath)
	for _, path := range paths {
		if filepath.Clean(path) == keepPath {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale snapshot %s: %w", path, err)
		}
	}
	return nil
}

func (s *Service) snapshotPath(version string) string {
	return filepath.Join(s.snapshotDir, searchSnapshotPrefix+sanitizeVersion(version)+searchSnapshotSuffix)
}

func sanitizeVersion(version string) string {
	var b strings.Builder
	for _, r := range version {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	cleaned := strings.Trim(b.String(), ".-_")
	if cleaned == "" {
		return "unknown"
	}
	return cleaned
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
