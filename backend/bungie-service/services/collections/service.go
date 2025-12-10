package collections

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"guardian-tracker/bungie-service/cache"
	"guardian-tracker/bungie-service/services/bungie"
	"guardian-tracker/bungie-service/services/manifest"
)

// Service handles collection analysis and data aggregation
type Service struct {
	bungieClient    *bungie.Client
	manifestService *bungie.ManifestService
	manifestRepo    *manifest.Repository
	cache           cache.Cache
	cacheTTL        time.Duration
}

// NewService creates a new collections service
func NewService(
	bungieClient *bungie.Client,
	manifestService *bungie.ManifestService,
	manifestRepo *manifest.Repository,
	cache cache.Cache,
	cacheTTL time.Duration,
) *Service {
	return &Service{
		bungieClient:    bungieClient,
		manifestService: manifestService,
		manifestRepo:    manifestRepo,
		cache:           cache,
		cacheTTL:        cacheTTL,
	}
}

// UserCollections represents the complete collection data for a user
type UserCollections struct {
	Weapons  CollectionSummary `json:"weapons"`
	Armor    CollectionSummary `json:"armor"`
	Exotics  CollectionSummary `json:"exotics"`
}

// CollectionSummary contains stats and missing items for a collection category
type CollectionSummary struct {
	Total     int           `json:"total"`
	Collected int           `json:"collected"`
	Missing   []DestinyItem `json:"missing"`
}

// DestinyItem represents an item in the format expected by the frontend
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

// GetUserCollections retrieves and analyzes a user's collections
func (s *Service) GetUserCollections(ctx context.Context, membershipType int, membershipID string, accessToken string) (*UserCollections, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("collections:%d:%s", membershipType, membershipID)
	if cached, found := s.cache.Get(cacheKey); found {
		if collections, ok := cached.(*UserCollections); ok {
			return collections, nil
		}
	}

	// Ensure manifest is ready
	if err := s.manifestService.EnsureReady(ctx); err != nil {
		return nil, fmt.Errorf("manifest not ready: %w", err)
	}

	// Get user's collected items from Bungie API
	log.Printf("Fetching collections for membership %d/%s", membershipType, membershipID)

	profile, err := s.bungieClient.GetProfile(ctx, membershipType, membershipID, accessToken, []int{bungie.ComponentCollectibles})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}

	// Build a set of collected item hashes
	collectedHashes := make(map[uint32]bool)

	// Profile-level collectibles
	for hashStr, collectible := range profile.Response.ProfileCollectibles.Data.Collectibles {
		if collectible.IsCollected() {
			if hash, err := strconv.ParseUint(hashStr, 10, 32); err == nil {
				collectedHashes[uint32(hash)] = true
			}
		}
	}

	// Character-level collectibles (some collectibles are per-character)
	for _, charData := range profile.Response.CharacterCollectibles.Data {
		for hashStr, collectible := range charData.Collectibles {
			if collectible.IsCollected() {
				if hash, err := strconv.ParseUint(hashStr, 10, 32); err == nil {
					collectedHashes[uint32(hash)] = true
				}
			}
		}
	}

	log.Printf("User has %d collected items", len(collectedHashes))

	// Get all collectibles from manifest
	filtered, err := s.manifestRepo.GetFilteredCollectibles()
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest collectibles: %w", err)
	}

	// Build the result
	result := &UserCollections{
		Weapons:  s.buildCollectionSummary(filtered.Weapons, collectedHashes),
		Armor:    s.buildCollectionSummary(filtered.Armor, collectedHashes),
		Exotics:  s.buildCollectionSummary(filtered.Exotics, collectedHashes),
	}

	// Cache the result
	s.cache.Set(cacheKey, result, s.cacheTTL)

	return result, nil
}

// buildCollectionSummary creates a summary for a category of items
func (s *Service) buildCollectionSummary(items []manifest.CollectibleWithItem, collected map[uint32]bool) CollectionSummary {
	summary := CollectionSummary{
		Total:   len(items),
		Missing: make([]DestinyItem, 0),
	}

	for _, cwi := range items {
		// Check if the collectible hash is in the collected set
		isCollected := collected[cwi.Collectible.Hash]

		if isCollected {
			summary.Collected++
		} else {
			// Add to missing items
			destinyItem := s.convertToDestinyItem(&cwi)
			summary.Missing = append(summary.Missing, destinyItem)
		}
	}

	return summary
}

// convertToDestinyItem converts a CollectibleWithItem to the frontend format
func (s *Service) convertToDestinyItem(cwi *manifest.CollectibleWithItem) DestinyItem {
	item := cwi.Item
	collectible := cwi.Collectible

	destinyItem := DestinyItem{
		ItemHash:    strconv.FormatUint(uint64(item.Hash), 10),
		Name:        item.DisplayProperties.Name,
		Description: item.DisplayProperties.Description,
		Icon:        item.DisplayProperties.Icon,
		TierType:    item.Inventory.TierType,
		Rarity:      bungie.GetTierName(item.Inventory.TierType),
		IsExotic:    item.Inventory.TierType == bungie.TierTypeExotic,
		Sources:     []string{},
	}

	// Determine item type name
	switch item.ItemType {
	case bungie.ItemTypeWeapon:
		destinyItem.ItemType = bungie.GetWeaponTypeName(item.ItemSubType)
	case bungie.ItemTypeArmor:
		destinyItem.ItemType = bungie.GetArmorTypeName(item.EquippingBlock.EquipmentSlotTypeHash)
	default:
		destinyItem.ItemType = "Unknown"
	}

	// Add source information
	if collectible.SourceString != "" {
		destinyItem.Sources = append(destinyItem.Sources, collectible.SourceString)
	}

	// Classify difficulty based on source
	destinyItem.Difficulty = classifyDifficulty(collectible.SourceString, destinyItem.IsExotic)

	return destinyItem
}

// classifyDifficulty determines acquisition difficulty based on source
func classifyDifficulty(sourceString string, isExotic bool) string {
	sourceLower := strings.ToLower(sourceString)

	// Check for raid sources
	raidKeywords := []string{"raid", "vault of glass", "king's fall", "root of nightmares", "crota", "deep stone", "garden of salvation", "last wish"}
	for _, keyword := range raidKeywords {
		if strings.Contains(sourceLower, keyword) {
			return "Challenging"
		}
	}

	// Check for dungeon sources
	dungeonKeywords := []string{"dungeon", "prophecy", "grasp of avarice", "duality", "spire of the watcher", "shattered throne", "pit of heresy"}
	for _, keyword := range dungeonKeywords {
		if strings.Contains(sourceLower, keyword) {
			return "Moderate"
		}
	}

	// Check for trials/competitive sources
	pvpKeywords := []string{"trials", "competitive", "iron banner", "glory rank"}
	for _, keyword := range pvpKeywords {
		if strings.Contains(sourceLower, keyword) {
			return "Challenging"
		}
	}

	// Check for nightfall/strike sources
	if strings.Contains(sourceLower, "nightfall") {
		return "Moderate"
	}

	// Exotics from quests are typically moderate
	if isExotic && strings.Contains(sourceLower, "quest") {
		return "Moderate"
	}

	// Default
	if isExotic {
		return "Moderate"
	}

	return "Easy"
}

// InvalidateCache removes a user's cached collections
func (s *Service) InvalidateCache(membershipType int, membershipID string) {
	cacheKey := fmt.Sprintf("collections:%d:%s", membershipType, membershipID)
	s.cache.Delete(cacheKey)
}
