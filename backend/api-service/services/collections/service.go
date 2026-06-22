package collections

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
)

// ManifestRepo is the subset of the manifest repository the collections service
// uses. Satisfied by *manifest.Provider, which opens lazily and reconnects
// across manifest swaps.
type ManifestRepo interface {
	GetFilteredCollectibles() (*manifest.FilteredCollectibles, error)
}

// Service handles collection analysis and data aggregation.
type Service struct {
	bungieClient    *bungie.Client
	manifestService *bungie.ManifestService
	manifest        ManifestRepo
	cache           cache.Cache
	cacheTTL        time.Duration
}

// NewService creates a collection service.
func NewService(
	bungieClient *bungie.Client,
	manifestService *bungie.ManifestService,
	m ManifestRepo,
	c cache.Cache,
	cacheTTL time.Duration,
) *Service {
	return &Service{
		bungieClient:    bungieClient,
		manifestService: manifestService,
		manifest:        m,
		cache:           c,
		cacheTTL:        cacheTTL,
	}
}

// ErrManifestNotReady marks failures caused by the manifest database not being
// usable yet (still downloading or mid-swap) — handlers map it to 503
// MANIFEST_NOT_READY instead of a 500. It aliases manifest.ErrNotReady so the
// records service and the provider share one sentinel. A query error against an
// open-but-corrupt database is NOT this error and surfaces as a real 500, so the
// client sees a genuine failure rather than an endless "still downloading".
var ErrManifestNotReady = manifest.ErrNotReady

// UserCollections is the complete collection result for a user.
type UserCollections struct {
	Weapons   CollectionSummary `json:"weapons"`
	Armor     CollectionSummary `json:"armor"`
	Exotics   CollectionSummary `json:"exotics"`
	Cosmetics CollectionSummary `json:"cosmetics"`
	FetchedAt time.Time         `json:"fetchedAt"` // when this data was fetched from Bungie (B8)
}

// CollectionSummary holds stats and missing items for one category.
// CollectedItems is populated in the cached result but only serialized when the
// client asks for ?include=all (WithoutCollectedItems strips it otherwise).
type CollectionSummary struct {
	Total          int           `json:"total"`
	Collected      int           `json:"collected"`
	Missing        []DestinyItem `json:"missing"`
	CollectedItems []DestinyItem `json:"collectedItems,omitempty"`
}

// WithoutCollectedItems returns a copy with every category's CollectedItems
// cleared — the default response shape when the client did not ask for
// ?include=all. Lives next to the struct so a new category added above is
// stripped here too. The cached *UserCollections is never mutated (value copy).
func (u UserCollections) WithoutCollectedItems() UserCollections {
	u.Weapons.CollectedItems = nil
	u.Armor.CollectedItems = nil
	u.Exotics.CollectedItems = nil
	u.Cosmetics.CollectedItems = nil
	return u
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
		return nil, fmt.Errorf("%w: %v", ErrManifestNotReady, err)
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

	filtered, err := s.manifest.GetFilteredCollectibles()
	if err != nil {
		if errors.Is(err, ErrManifestNotReady) {
			return nil, err
		}
		// The manifest opened but the query failed (e.g. a corrupt database) —
		// surface a real error (→ 500) instead of an endless "still downloading".
		return nil, fmt.Errorf("collections: manifest query failed: %w", err)
	}

	result := &UserCollections{
		Weapons:   s.buildSummary(filtered.Weapons, collected),
		Armor:     s.buildSummary(filtered.Armor, collected),
		Exotics:   s.buildSummary(filtered.Exotics, collected),
		Cosmetics: s.buildSummary(filtered.Cosmetics, collected),
		FetchedAt: time.Now().UTC(),
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
			summary.CollectedItems = append(summary.CollectedItems, toDestinyItem(&cwi))
		} else {
			summary.Missing = append(summary.Missing, toDestinyItem(&cwi))
		}
	}
	return summary
}

// toDestinyItem maps a manifest collectible+item pair into the frontend item shape.
func toDestinyItem(cwi *manifest.CollectibleWithItem) DestinyItem {
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

// GetMissingItemHashes returns the set of item hashes not yet collected by the user.
// It reuses the cached GetUserCollections result — no extra Bungie call.
func (s *Service) GetMissingItemHashes(ctx context.Context, membershipType int, membershipID, accessToken string) (map[uint32]struct{}, error) {
	result, err := s.GetUserCollections(ctx, membershipType, membershipID, accessToken)
	if err != nil {
		return nil, err
	}
	missing := make(map[uint32]struct{})
	for _, item := range result.Weapons.Missing {
		if h, err := strconv.ParseUint(item.ItemHash, 10, 32); err == nil {
			missing[uint32(h)] = struct{}{}
		}
	}
	for _, item := range result.Armor.Missing {
		if h, err := strconv.ParseUint(item.ItemHash, 10, 32); err == nil {
			missing[uint32(h)] = struct{}{}
		}
	}
	for _, item := range result.Exotics.Missing {
		if h, err := strconv.ParseUint(item.ItemHash, 10, 32); err == nil {
			missing[uint32(h)] = struct{}{}
		}
	}
	return missing, nil
}

func (s *Service) InvalidateCache(membershipType int, membershipID string) {
	s.cache.Delete(fmt.Sprintf("collections:%d:%s", membershipType, membershipID))
}
