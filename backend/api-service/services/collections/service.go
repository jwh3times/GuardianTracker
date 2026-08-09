package collections

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
)

// ManifestRepo is the subset of the manifest repository the collections service
// uses. Satisfied by *manifest.Provider.
type ManifestRepo interface {
	GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error)
	GetAllPresentationNodes() (map[uint32]*manifest.PresentationNodeDef, error)
}

type Service struct {
	bungieClient    *bungie.Client
	manifestService *bungie.ManifestService
	manifest        ManifestRepo
	cache           cache.Cache
	cacheTTL        time.Duration

	treeMu     sync.RWMutex
	treeStruct *TreeStructure // user-independent; rebuilt on manifest swap
}

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

// UserCollections is the canonical collections payload: the Bungie presentation-node
// tree, a shared item-detail map (only on ?include=all), a flat set of owned item
// hashes (the grid's per-item collected state; only on ?include=all), and a derived
// four-category summary for the Dashboard hero and weekly recommender.
type UserCollections struct {
	Tree            []CollectionNode       `json:"tree"`
	Items           map[string]DestinyItem `json:"items,omitempty"`
	CollectedHashes []string               `json:"collectedHashes,omitempty"`
	AvailableNow    map[string]string      `json:"availableNow,omitempty"` // itemHash → vendor name; set by the handler on ?include=all
	Summary         CategorySummary        `json:"summary"`
	FetchedAt       time.Time              `json:"fetchedAt"`
}

// CategoryCount is total/collected for one summary bucket.
type CategoryCount struct {
	Total     int `json:"total"`
	Collected int `json:"collected"`
}

// CategorySummary is the derived four-bucket rollup (Dashboard hero / weekly).
type CategorySummary struct {
	Weapons   CategoryCount `json:"weapons"`
	Armor     CategoryCount `json:"armor"`
	Exotics   CategoryCount `json:"exotics"`
	Cosmetics CategoryCount `json:"cosmetics"`
}

// Lightweight returns a copy with the heavy item data removed: the top-level Items
// map, the CollectedHashes set, and every node's Items hash array. Tree counts,
// summary, and fetchedAt remain. The cached source is never mutated (value receiver
// + fresh node slices).
func (u UserCollections) Lightweight() UserCollections {
	u.Items = nil
	u.CollectedHashes = nil
	u.AvailableNow = nil
	u.Tree = nodesWithoutItems(u.Tree)
	return u
}

func nodesWithoutItems(nodes []CollectionNode) []CollectionNode {
	if nodes == nil {
		return nil
	}
	out := make([]CollectionNode, len(nodes))
	for i, n := range nodes {
		n.Items = nil
		n.Children = nodesWithoutItems(n.Children)
		out[i] = n
	}
	return out
}

// buildCategorySummary aggregates collectibles into the four summary buckets using
// the shared manifest classifier. A pure projection over the same item set as the
// tree (single source of truth). owned is itemHash-keyed (see deriveOwnedItems):
// the manifest carries multiple collectible rows for some re-issued itemHashes, so
// counting is deduped by itemHash — Total counts distinct items, Collected counts
// owned ones — rather than once per collectible row.
func buildCategorySummary(collectibles []manifest.CollectibleWithItem, owned map[uint32]bool) CategorySummary {
	var sum CategorySummary
	seen := make(map[uint32]bool)
	for _, cwi := range collectibles {
		if cwi.Item == nil || cwi.Item.DisplayProperties.Name == "" {
			continue
		}
		if seen[cwi.Item.Hash] {
			continue
		}
		seen[cwi.Item.Hash] = true
		var cc *CategoryCount
		switch manifest.CollectibleCategory(cwi.Item) {
		case "weapons":
			cc = &sum.Weapons
		case "armor":
			cc = &sum.Armor
		case "exotics":
			cc = &sum.Exotics
		case "cosmetics":
			cc = &sum.Cosmetics
		default:
			continue
		}
		cc.Total++
		if owned[cwi.Item.Hash] {
			cc.Collected++
		}
	}
	return sum
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
	FarmOnly    bool     `json:"farmOnly"`
	Sources     []string `json:"sources"`
	IsExotic    bool     `json:"isExotic"`
}

// analysis is the cached, per-user canonical dataset that both projections read.
//
// Treat an *analysis as immutable once cached: concurrent requests share the
// pointer without a lock, so refreshing it means building a replacement, never
// mutating in place (see refreshManifestParts).
type analysis struct {
	collectibles []manifest.CollectibleWithItem
	collected    map[uint32]bool // collectible-hash-keyed, straight from the profile response
	owned        map[uint32]bool // itemHash-keyed: true if ANY of the item's collectibles is acquired (see deriveOwnedItems)
	tree         *TreeStructure
	fetchedAt    time.Time
	// manifestVersion stamps which manifest the manifest-derived fields above
	// (collectibles, tree, owned) were built from. A cache hit whose stamp no
	// longer matches the installed manifest is refreshed in place of an
	// eviction — see getAnalysis.
	manifestVersion string
}

// deriveOwnedItems collapses collectible-hash-level acquisition to itemHash-level
// ownership. The manifest carries multiple DestinyCollectibleDefinition rows for
// some re-issued itemHashes (e.g. Choir of One), and the profile response only ever
// marks the specific collectible row the player actually earned — never every
// duplicate — so an item is owned if ANY collectible sharing its itemHash is
// acquired.
func deriveOwnedItems(collectibles []manifest.CollectibleWithItem, collected map[uint32]bool) map[uint32]bool {
	owned := make(map[uint32]bool)
	for _, cwi := range collectibles {
		if cwi.Item == nil {
			continue
		}
		if collected[cwi.Collectible.Hash] {
			owned[cwi.Item.Hash] = true
		}
	}
	return owned
}

func (s *Service) getAnalysis(ctx context.Context, membershipType int, membershipID, accessToken string) (*analysis, error) {
	cacheKey := fmt.Sprintf("collections:%d:%s", membershipType, membershipID)
	if cached, found := s.cache.Get(cacheKey); found {
		if a, ok := cached.(*analysis); ok {
			// The manifest-derived half of a cached analysis goes stale on a
			// manifest swap, but the expensive half — `collected`, a rate-limited
			// Bungie profile fetch — does not. Rebuild only what the manifest
			// owns and keep the profile data, so a swap costs local SQLite reads
			// instead of a refetch storm across every active user. This is why
			// `collections:*` is not evicted by OnVersionChanged.
			refreshed, err := s.refreshManifestParts(a)
			if err != nil {
				return nil, err
			}
			if refreshed != a {
				s.cache.Set(cacheKey, refreshed, s.cacheTTL)
			}
			return refreshed, nil
		}
	}

	if err := s.manifestService.EnsureReady(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrManifestNotReady, err)
	}

	logger := observability.Logger(ctx)
	logger.LogAttrs(ctx, slog.LevelInfo, "fetching collections",
		slog.Int("membership_type", membershipType),
		observability.ID("membership", membershipID),
	)
	profile, err := s.bungieClient.GetProfile(ctx, membershipType, membershipID, accessToken, []int{bungie.ComponentCollectibles})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}

	collected := make(map[uint32]bool)
	for hashStr, col := range profile.Response.ProfileCollectibles.Data.Collectibles {
		if col.IsCollected() {
			if h, err := strconv.ParseUint(hashStr, 10, 32); err == nil {
				collected[uint32(h)] = true
			}
		}
	}
	for _, charData := range profile.Response.CharacterCollectibles.Data {
		for hashStr, col := range charData.Collectibles {
			if col.IsCollected() {
				if h, err := strconv.ParseUint(hashStr, 10, 32); err == nil {
					collected[uint32(h)] = true
				}
			}
		}
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "collection ownership loaded",
		slog.Int("collected_items", len(collected)),
		observability.ID("membership", membershipID),
	)

	collectibles, err := s.manifest.GetAllCollectiblesWithItems()
	if err != nil {
		if errors.Is(err, ErrManifestNotReady) {
			return nil, err
		}
		return nil, fmt.Errorf("collections: manifest query failed: %w", err)
	}

	tree, err := s.getTreeStructure(collectibles)
	if err != nil {
		if errors.Is(err, ErrManifestNotReady) {
			return nil, err
		}
		return nil, fmt.Errorf("collections: tree build failed: %w", err)
	}

	owned := deriveOwnedItems(collectibles, collected)

	a := &analysis{
		collectibles:    collectibles,
		collected:       collected,
		owned:           owned,
		tree:            tree,
		fetchedAt:       time.Now().UTC(),
		manifestVersion: s.manifestVersion(),
	}
	s.cache.Set(cacheKey, a, s.cacheTTL)
	return a, nil
}

// getTreeStructure returns the cached user-independent tree, building it once from
// the (already-loaded) collectibles + all presentation nodes. Invalidated on swap.
func (s *Service) getTreeStructure(collectibles []manifest.CollectibleWithItem) (*TreeStructure, error) {
	s.treeMu.RLock()
	ts := s.treeStruct
	s.treeMu.RUnlock()
	if ts != nil {
		return ts, nil
	}
	nodes, err := s.manifest.GetAllPresentationNodes()
	if err != nil {
		return nil, err
	}
	ts = buildTreeStructure(nodes, collectibles)
	s.treeMu.Lock()
	s.treeStruct = ts
	s.treeMu.Unlock()
	return ts, nil
}

func (s *Service) GetUserCollections(ctx context.Context, membershipType int, membershipID, accessToken string) (*UserCollections, error) {
	a, err := s.getAnalysis(ctx, membershipType, membershipID, accessToken)
	if err != nil {
		return nil, err
	}
	// Per-item collected state for the grid's missing-only toggle: the item hashes
	// the user owns. Stripped on the lightweight (default) response.
	collectedHashes := make([]string, 0)
	for _, cwi := range a.collectibles {
		if cwi.Item == nil || cwi.Item.DisplayProperties.Name == "" {
			continue
		}
		if a.collected[cwi.Collectible.Hash] {
			collectedHashes = append(collectedHashes, strconv.FormatUint(uint64(cwi.Item.Hash), 10))
		}
	}
	return &UserCollections{
		Tree:            a.tree.overlay(a.owned),
		Items:           a.tree.Items,
		CollectedHashes: collectedHashes,
		Summary:         buildCategorySummary(a.collectibles, a.owned),
		FetchedAt:       a.fetchedAt,
	}, nil
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
		di.ItemType = bungie.ItemTypeName(item.ItemType, item.ItemSubType)
	}
	if col.SourceString != "" {
		di.Sources = append(di.Sources, col.SourceString)
	}
	di.Difficulty = ClassifyDifficulty(col.SourceString, di.IsExotic)
	di.FarmOnly = strings.Contains(strings.ToLower(col.SourceString), "cannot be reacquired")
	return di
}

// Difficulty keyword tiers — verified against the real manifest source-string
// histogram (2026-06-30). Checked Challenging → Moderate → Easy, first hit wins, so
// "Grandmaster Nightfall" scores Challenging before the Moderate "nightfall" rule.
// Every entry is a positive match; anything unmatched is honestly "Unrated" (never a
// catch-all "Easy"). Extend these lists freely.
var (
	challengingDiffKeywords = []string{
		"raid", "vault of glass", "king's fall", "root of nightmares", "crota",
		"deep stone", "garden of salvation", "last wish", "vow of the disciple",
		"salvation's edge", "desert perpetual",
		"trials", "competitive", "iron banner", "glory", "grandmaster", "pantheon",
	}
	moderateDiffKeywords = []string{
		"dungeon", "prophecy", "grasp of avarice", "duality", "spire of the watcher",
		"shattered throne", "pit of heresy", "ghosts of the deep", "warlord", "vesper",
		"sundered doctrine",
		"nightfall", "lost sector", "exotic quest", "exotic mission", "dreaming city",
		"season of", "seasonal", "episode", "into the light", "solstice", "exploring",
		"kepler", "wellspring", "dares", "black armory", "sparrow racing",
	}
	easyDiffKeywords = []string{
		"rank-up package", "rank up package", "earned while leveling", "engram",
		"season pass", "eververse", "bright dust", "focusing", "world drop",
		"complete crucible", "complete gambit", "complete strikes", "complete vanguard",
		"rank-up", "vendor", "banshee", "ada-1", "xûr", "xur", "tower", "monument",
		"deluxe edition", "pre-order", "preorder", "charity", "special offer",
		"rewards pass", "new monarchy", "dead orbit", "future war cult", "saint-14",
	}
)

// ClassifyDifficulty infers an acquisition-difficulty estimate from a collectible's
// source string. Every non-Unrated result is a real keyword match; unmatched sources
// (and empty / "cannot be reacquired" ones) return "Unrated" rather than a misleading
// default. isExotic is retained for a future exotic-aware tie-break; it is not used as
// a default today.
func ClassifyDifficulty(source string, isExotic bool) string {
	s := strings.ToLower(strings.TrimSpace(source))
	switch {
	case s == "":
		return "Unrated"
	// Checked before the keyword tiers: reacquirable raid/dungeon collectibles never carry this string (verified against the manifest), so this short-circuit can't steal a real rating.
	case strings.Contains(s, "cannot be reacquired"):
		return "Unrated"
	case matchesAnyKeyword(s, challengingDiffKeywords):
		return "Challenging"
	case matchesAnyKeyword(s, moderateDiffKeywords):
		return "Moderate"
	case matchesAnyKeyword(s, easyDiffKeywords):
		return "Easy"
	default:
		return "Unrated"
	}
}

func matchesAnyKeyword(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// GetMissingItemHashes returns not-collected weapon/armor/exotic item hashes
// (cosmetics excluded), reusing the cached analysis — no extra Bungie call.
// Ownership is itemHash-level (a.owned): an item hash is excluded from the missing
// set if ANY of its (possibly several, re-issued) collectibles is acquired.
func (s *Service) GetMissingItemHashes(ctx context.Context, membershipType int, membershipID, accessToken string) (map[uint32]struct{}, error) {
	a, err := s.getAnalysis(ctx, membershipType, membershipID, accessToken)
	if err != nil {
		return nil, err
	}
	missing := make(map[uint32]struct{})
	for _, cwi := range a.collectibles {
		if cwi.Item == nil || cwi.Item.DisplayProperties.Name == "" {
			continue
		}
		switch manifest.CollectibleCategory(cwi.Item) {
		case "weapons", "armor", "exotics":
			if !a.owned[cwi.Item.Hash] {
				missing[cwi.Item.Hash] = struct{}{}
			}
		}
	}
	return missing, nil
}

func (s *Service) InvalidateCache(membershipType int, membershipID string) {
	s.cache.Delete(fmt.Sprintf("collections:%d:%s", membershipType, membershipID))
}

// InvalidateTreeCache drops the shared, user-independent tree so it rebuilds
// from the new manifest.
//
// This alone is not sufficient: each cached per-user analysis holds its own
// pointer to the tree it was built with, so dropping the shared copy does not
// reach them. refreshManifestParts is what repairs those, lazily, on read.
func (s *Service) InvalidateTreeCache() {
	s.treeMu.Lock()
	s.treeStruct = nil
	s.treeMu.Unlock()
}

// manifestVersion returns the installed manifest version, or "" when no version
// is knowable — no manifest service (tests) or nothing downloaded yet. An empty
// version disables the staleness check rather than forcing a rebuild on every
// read, since "" can never equal a real stamp.
func (s *Service) manifestVersion() string {
	if s.manifestService == nil {
		return ""
	}
	return s.manifestService.Version()
}

// OnVersionChanged drops the shared tree so the next request rebuilds it from
// the new manifest. Implements bungie.ManifestObserver.
//
// Per-user `collections:*` entries are deliberately NOT evicted — see the
// comment in getAnalysis. Evicting them would discard a rate-limited Bungie
// profile fetch per active user on every hourly swap; refreshManifestParts
// rebuilds just the manifest-derived half instead.
func (s *Service) OnVersionChanged(version string) error {
	s.InvalidateTreeCache()
	return nil
}

// refreshManifestParts returns an analysis whose manifest-derived fields match
// the installed manifest, reusing the caller's profile data. It returns `a`
// unchanged when the stamp already matches, so the common path allocates
// nothing.
//
// It never mutates `a`: concurrent requests share that pointer without a lock,
// so a replacement is built and the caller re-caches it.
func (s *Service) refreshManifestParts(a *analysis) (*analysis, error) {
	current := s.manifestVersion()
	if current == "" || a.manifestVersion == current {
		return a, nil
	}

	collectibles, err := s.manifest.GetAllCollectiblesWithItems()
	if err != nil {
		if errors.Is(err, ErrManifestNotReady) {
			// Mid-swap. Serving the previous manifest's labels for one more
			// request beats a 503 on data we already hold.
			return a, nil
		}
		return nil, fmt.Errorf("collections: manifest query failed: %w", err)
	}
	tree, err := s.getTreeStructure(collectibles)
	if err != nil {
		if errors.Is(err, ErrManifestNotReady) {
			return a, nil
		}
		return nil, fmt.Errorf("collections: tree build failed: %w", err)
	}

	return &analysis{
		collectibles:    collectibles,
		collected:       a.collected,
		owned:           deriveOwnedItems(collectibles, a.collected),
		tree:            tree,
		fetchedAt:       a.fetchedAt,
		manifestVersion: current,
	}, nil
}
