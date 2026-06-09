package collections

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
)

// Service handles collection analysis and data aggregation.
type Service struct {
	bungieClient    *bungie.Client
	manifestService *bungie.ManifestService
	dbPath          string
	repoMu          sync.Mutex
	repo            *manifest.Repository
	cache           cache.Cache
	cacheTTL        time.Duration
}

// NewService creates a collection service. The manifest repository is opened lazily
// on the first request after the manifest database is available.
func NewService(
	bungieClient *bungie.Client,
	manifestService *bungie.ManifestService,
	dbPath string,
	c cache.Cache,
	cacheTTL time.Duration,
) *Service {
	return &Service{
		bungieClient:    bungieClient,
		manifestService: manifestService,
		dbPath:          dbPath,
		cache:           c,
		cacheTTL:        cacheTTL,
	}
}

// Close releases the manifest database connection if it was opened.
func (s *Service) Close() {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()
	if s.repo != nil {
		if err := s.repo.Close(); err != nil {
			log.Printf("Warning: error closing manifest repository: %v", err)
		}
		s.repo = nil
	}
}

// ensureRepo lazily opens the manifest repository once the database file is present.
func (s *Service) ensureRepo() (*manifest.Repository, error) {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()
	if s.repo != nil {
		return s.repo, nil
	}
	if !s.manifestService.IsReady() {
		return nil, fmt.Errorf("manifest not ready")
	}
	repo, err := manifest.NewRepository(s.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest database: %w", err)
	}
	s.repo = repo
	return repo, nil
}

// UserCollections is the complete collection result for a user.
type UserCollections struct {
	Weapons CollectionSummary `json:"weapons"`
	Armor   CollectionSummary `json:"armor"`
	Exotics CollectionSummary `json:"exotics"`
}

// CollectionSummary holds stats and missing items for one category.
type CollectionSummary struct {
	Total     int           `json:"total"`
	Collected int           `json:"collected"`
	Missing   []DestinyItem `json:"missing"`
}

// DestinyItem is the frontend-facing item representation.
type DestinyItem struct {
	ItemHash    string   `json:"itemHash"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	ItemType    string   `json:"itemType"`
	TierType    int      `json:"tierType"`
	Rarity      string   `json:"rarity"`
	Difficulty  string   `json:"difficulty"`
	Sources     []string `json:"sources"`
	IsExotic    bool     `json:"isExotic"`
}

func (s *Service) GetUserCollections(ctx context.Context, membershipType int, membershipID, accessToken string) (*UserCollections, error) {
	cacheKey := fmt.Sprintf("collections:%d:%s", membershipType, membershipID)
	if cached, found := s.cache.Get(cacheKey); found {
		if cols, ok := cached.(*UserCollections); ok {
			return cols, nil
		}
	}

	if err := s.manifestService.EnsureReady(ctx); err != nil {
		return nil, fmt.Errorf("manifest not ready: %w", err)
	}

	repo, err := s.ensureRepo()
	if err != nil {
		return nil, fmt.Errorf("manifest database unavailable: %w", err)
	}

	log.Printf("Fetching collections for membership %d/%s", membershipType, membershipID)
	profile, err := s.bungieClient.GetProfile(ctx, membershipType, membershipID, accessToken, []int{bungie.ComponentCollectibles})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}

	collected := make(map[uint32]bool)
	for hashStr, c := range profile.Response.ProfileCollectibles.Data.Collectibles {
		if c.IsCollected() {
			if h, err := strconv.ParseUint(hashStr, 10, 32); err == nil {
				collected[uint32(h)] = true
			}
		}
	}
	for _, charData := range profile.Response.CharacterCollectibles.Data {
		for hashStr, c := range charData.Collectibles {
			if c.IsCollected() {
				if h, err := strconv.ParseUint(hashStr, 10, 32); err == nil {
					collected[uint32(h)] = true
				}
			}
		}
	}
	log.Printf("User has %d collected items", len(collected))

	filtered, err := repo.GetFilteredCollectibles()
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest collectibles: %w", err)
	}

	result := &UserCollections{
		Weapons: s.buildSummary(filtered.Weapons, collected),
		Armor:   s.buildSummary(filtered.Armor, collected),
		Exotics: s.buildSummary(filtered.Exotics, collected),
	}
	s.cache.Set(cacheKey, result, s.cacheTTL)
	return result, nil
}

func (s *Service) buildSummary(items []manifest.CollectibleWithItem, collected map[uint32]bool) CollectionSummary {
	summary := CollectionSummary{
		Total:   len(items),
		Missing: make([]DestinyItem, 0),
	}
	for _, cwi := range items {
		if collected[cwi.Collectible.Hash] {
			summary.Collected++
		} else {
			summary.Missing = append(summary.Missing, s.toDestinyItem(&cwi))
		}
	}
	return summary
}

func (s *Service) toDestinyItem(cwi *manifest.CollectibleWithItem) DestinyItem {
	item := cwi.Item
	col := cwi.Collectible
	di := DestinyItem{
		ItemHash:    strconv.FormatUint(uint64(item.Hash), 10),
		Name:        item.DisplayProperties.Name,
		Description: item.DisplayProperties.Description,
		Icon:        item.DisplayProperties.Icon,
		TierType:    item.Inventory.TierType,
		Rarity:      bungie.GetTierName(item.Inventory.TierType),
		IsExotic:    item.Inventory.TierType == bungie.TierTypeExotic,
		Sources:     []string{},
	}
	switch item.ItemType {
	case bungie.ItemTypeWeapon:
		di.ItemType = bungie.GetWeaponTypeName(item.ItemSubType)
	case bungie.ItemTypeArmor:
		di.ItemType = bungie.GetArmorTypeName(item.EquippingBlock.EquipmentSlotTypeHash)
	default:
		di.ItemType = "Unknown"
	}
	if col.SourceString != "" {
		di.Sources = append(di.Sources, col.SourceString)
	}
	di.Difficulty = classifyDifficulty(col.SourceString, di.IsExotic)
	return di
}

func classifyDifficulty(source string, isExotic bool) string {
	s := strings.ToLower(source)
	for _, kw := range []string{"raid", "vault of glass", "king's fall", "root of nightmares", "crota", "deep stone", "garden of salvation", "last wish"} {
		if strings.Contains(s, kw) {
			return "Challenging"
		}
	}
	for _, kw := range []string{"dungeon", "prophecy", "grasp of avarice", "duality", "spire of the watcher", "shattered throne", "pit of heresy"} {
		if strings.Contains(s, kw) {
			return "Moderate"
		}
	}
	for _, kw := range []string{"trials", "competitive", "iron banner", "glory rank"} {
		if strings.Contains(s, kw) {
			return "Challenging"
		}
	}
	if strings.Contains(s, "nightfall") {
		return "Moderate"
	}
	if isExotic && strings.Contains(s, "quest") {
		return "Moderate"
	}
	if isExotic {
		return "Moderate"
	}
	return "Easy"
}

func (s *Service) InvalidateCache(membershipType int, membershipID string) {
	s.cache.Delete(fmt.Sprintf("collections:%d:%s", membershipType, membershipID))
}
