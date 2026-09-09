package collections

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"sync"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/manifest"
	"guardian-tracker/api-service/services/manifeststate"
)

// CatalogReader is the entire Items surface Collections consumes: the
// canonical, user-independent acquisition facts for every item linked to at
// least one collectible (ADR 0015). Satisfied by *items.Service.
//
// Consumer-side and deliberately narrower than items.AcquisitionFactsReader:
// Collections never resolves an individual item hash, it overlays one
// membership onto the whole catalog. Raw item definitions and collectible rows
// no longer cross from the manifest into this package.
type CatalogReader interface {
	Catalog(ctx context.Context) ([]items.AcquisitionFacts, error)
}

// PresentationNodeReader is the manifest surface Collections still owns
// directly: the presentation tree the catalog is placed into. Satisfied by
// *manifest.Provider.
type PresentationNodeReader interface {
	GetAllPresentationNodes() (map[uint32]*manifest.PresentationNodeDef, error)
}

// MembershipAnalysis owns the reusable membership-and-manifest analysis:
// item-level ownership, the presentation tree, tree counts, the four-category
// summary, and the missing-item set.
//
// It is the inner of the two Collections construction stages in ADR 0018.
// Weekly consumes this core through its own MissingItemReader interface, which
// is what lets the complete Collections service depend on Weekly's live
// availability without a construction cycle.
type MembershipAnalysis struct {
	bungieClient    *bungie.Client
	manifestService *bungie.ManifestService
	catalog         CatalogReader
	nodes           PresentationNodeReader
	cache           cache.Cache
	cacheTTL        time.Duration

	// publication fences every manifest-derived projection this owner keeps:
	// analysis work that began under one manifest generation may not install
	// itself once a swap has replaced that generation (ADR 0014).
	publication *manifeststate.Publication

	treeMu     sync.RWMutex
	treeStruct *TreeStructure // user-independent; rebuilt after a manifest swap
}

func NewMembershipAnalysis(
	bungieClient *bungie.Client,
	manifestService *bungie.ManifestService,
	catalog CatalogReader,
	nodes PresentationNodeReader,
	c cache.Cache,
	cacheTTL time.Duration,
) *MembershipAnalysis {
	m := &MembershipAnalysis{
		bungieClient:    bungieClient,
		manifestService: manifestService,
		catalog:         catalog,
		nodes:           nodes,
		cache:           c,
		cacheTTL:        cacheTTL,
	}
	// The callback runs inside the publication's critical section, so it only
	// drops the shared tree pointer and never calls back into the publication.
	m.publication = manifeststate.New(m.dropTree)
	return m
}

// ErrManifestNotReady marks failures caused by the manifest database not being
// usable yet (still downloading or mid-swap) — handlers map it to 503
// MANIFEST_NOT_READY instead of a 500. It aliases manifest.ErrNotReady so the
// records service and the provider share one sentinel. A query error against an
// open-but-corrupt database is NOT this error and surfaces as a real 500, so the
// client sees a genuine failure rather than an endless "still downloading".
var ErrManifestNotReady = manifest.ErrNotReady

// analysis is the cached, per-membership dataset every projection reads.
//
// Treat an *analysis as immutable once cached: concurrent requests share the
// pointer without a lock, so refreshing it means building a replacement, never
// mutating in place (see refreshManifestParts).
type analysis struct {
	catalog   []items.AcquisitionFacts
	collected map[uint32]bool // collectible-hash-keyed, straight from the profile response
	owned     map[uint32]bool // itemHash-keyed: true if ANY of the item's collectibles is acquired (see deriveOwnedItems)
	tree      *TreeStructure
	fetchedAt time.Time
	// builtUnder is the publication attempt the manifest-derived fields above
	// (catalog, owned, tree) were built under. A cache hit whose attempt is no
	// longer current is refreshed in place of an eviction — see getAnalysis.
	builtUnder manifeststate.Attempt
}

// deriveOwnedItems collapses collectible-level acquisition to itemHash-level
// ownership. The manifest carries multiple DestinyCollectibleDefinition rows for
// some re-issued itemHashes (e.g. Choir of One), and the profile response only ever
// marks the specific collectible row the player actually earned — never every
// duplicate — so an item is owned if ANY collectible linked to it is acquired.
func deriveOwnedItems(catalog []items.AcquisitionFacts, collected map[uint32]bool) map[uint32]bool {
	owned := make(map[uint32]bool)
	for _, f := range catalog {
		for _, collectibleHash := range f.CollectibleHashes {
			if collected[collectibleHash] {
				owned[f.ItemHash] = true
				break
			}
		}
	}
	return owned
}

// buildCategorySummary aggregates the catalog into the four summary buckets. A
// pure projection over the same item set as the tree (single source of truth),
// counted per item: the catalog is already one entry per item hash, so a
// re-issued item with several collectible rows counts once.
func buildCategorySummary(catalog []items.AcquisitionFacts, owned map[uint32]bool) CategorySummary {
	var sum CategorySummary
	for _, f := range catalog {
		var cc *CategoryCount
		switch f.Category {
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
		if owned[f.ItemHash] {
			cc.Collected++
		}
	}
	return sum
}

func analysisCacheKey(membershipType int, membershipID string) string {
	return fmt.Sprintf("collections:%d:%s", membershipType, membershipID)
}

func itemHashString(itemHash uint32) string {
	return strconv.FormatUint(uint64(itemHash), 10)
}

func (m *MembershipAnalysis) getAnalysis(ctx context.Context, membershipType int, membershipID, accessToken string) (*analysis, error) {
	cacheKey := analysisCacheKey(membershipType, membershipID)
	if cached, found := m.cache.Get(cacheKey); found {
		if a, ok := cached.(*analysis); ok {
			// The manifest-derived half of a cached analysis goes stale on a
			// manifest swap, but the expensive half — `collected`, a rate-limited
			// Bungie profile fetch — does not. Rebuild only what the manifest
			// owns and keep the profile data, so a swap costs a catalog read
			// instead of a refetch storm across every active user. This is why
			// `collections:*` is not evicted when the manifest version changes.
			return m.refreshManifestParts(ctx, cacheKey, a)
		}
	}

	if err := m.manifestService.EnsureReady(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrManifestNotReady, err)
	}

	logger := observability.Logger(ctx)
	logger.LogAttrs(ctx, slog.LevelInfo, "fetching collections",
		slog.Int("membership_type", membershipType),
		observability.ID("membership", membershipID),
	)
	profile, err := m.bungieClient.GetProfile(ctx, membershipType, membershipID, accessToken, []int{bungie.ComponentCollectibles})
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

	// Captured after the profile fetch and before the first catalog read: the
	// fence covers the manifest-derived work, and a swap during a slow Bungie
	// call must not retire an analysis whose manifest half has not started yet.
	attempt := m.publication.Begin()
	catalog, tree, err := m.manifestParts(ctx, attempt)
	if err != nil {
		// A cold read reports the manifest as unavailable rather than
		// returning an empty collection as a success.
		return nil, err
	}

	a := &analysis{
		catalog:    catalog,
		collected:  collected,
		owned:      deriveOwnedItems(catalog, collected),
		tree:       tree,
		fetchedAt:  time.Now().UTC(),
		builtUnder: attempt,
	}
	attempt.Publish(func() { m.cache.Set(cacheKey, a, m.cacheTTL) })
	return a, nil
}

// manifestParts builds the manifest-derived half of an analysis — the catalog
// and the shared presentation tree — under one publication attempt.
//
// One attempt spans catalog access, presentation-node access, tree
// construction, and the tree's own publication, so the two halves can never be
// paired across a manifest swap.
func (m *MembershipAnalysis) manifestParts(ctx context.Context, attempt manifeststate.Attempt) ([]items.AcquisitionFacts, *TreeStructure, error) {
	catalog, err := m.catalog.Catalog(ctx)
	if err != nil {
		if errors.Is(err, ErrManifestNotReady) {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("collections: item catalog unavailable: %w", err)
	}

	tree, err := m.treeStructure(attempt, catalog)
	if err != nil {
		if errors.Is(err, ErrManifestNotReady) {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("collections: tree build failed: %w", err)
	}
	return catalog, tree, nil
}

// treeStructure returns the cached user-independent tree, building it once from
// the (already-loaded) catalog plus all presentation nodes. A tree built under a
// retired generation still answers this request but is not left behind.
func (m *MembershipAnalysis) treeStructure(attempt manifeststate.Attempt, catalog []items.AcquisitionFacts) (*TreeStructure, error) {
	m.treeMu.RLock()
	ts := m.treeStruct
	m.treeMu.RUnlock()
	if ts != nil {
		return ts, nil
	}
	nodes, err := m.nodes.GetAllPresentationNodes()
	if err != nil {
		return nil, err
	}
	ts = buildTreeStructure(nodes, catalog)
	attempt.Publish(func() { m.storeTree(ts) })
	return ts, nil
}

func (m *MembershipAnalysis) storeTree(ts *TreeStructure) {
	m.treeMu.Lock()
	defer m.treeMu.Unlock()
	m.treeStruct = ts
}

// dropTree is the publication's invalidation callback: it drops the shared,
// user-independent tree so it rebuilds from the new manifest.
//
// It runs inside the publication's critical section, so it stays bounded and
// must not call back into the publication.
//
// This alone is not sufficient: each cached per-membership analysis holds its
// own pointer to the tree it was built with, so dropping the shared copy does
// not reach them. refreshManifestParts is what repairs those, lazily, on read.
func (m *MembershipAnalysis) dropTree() {
	m.storeTree(nil)
}

// GetMembershipCollections returns the full collection projection for one
// membership.
func (m *MembershipAnalysis) GetMembershipCollections(ctx context.Context, membershipType int, membershipID, accessToken string) (*MembershipCollections, error) {
	a, err := m.getAnalysis(ctx, membershipType, membershipID, accessToken)
	if err != nil {
		return nil, err
	}
	// Per-item collected state for the grid's missing-only toggle: the item hashes
	// the membership owns. Stripped on the lightweight (default) response.
	ownedHashes := make([]uint32, 0, len(a.owned))
	for itemHash, owned := range a.owned {
		if owned {
			ownedHashes = append(ownedHashes, itemHash)
		}
	}
	slices.Sort(ownedHashes)
	collectedHashes := make([]string, len(ownedHashes))
	for i, itemHash := range ownedHashes {
		collectedHashes[i] = itemHashString(itemHash)
	}
	return &MembershipCollections{
		Tree:            a.tree.overlay(a.owned),
		Items:           a.tree.Items,
		CollectedHashes: collectedHashes,
		Summary:         buildCategorySummary(a.catalog, a.owned),
		FetchedAt:       a.fetchedAt,
	}, nil
}

// GetMissingItemHashes returns not-collected weapon/armor/exotic item hashes
// (cosmetics excluded), reusing the cached analysis — no extra Bungie call.
// Ownership is itemHash-level (a.owned): an item hash is excluded from the missing
// set if ANY of its (possibly several, re-issued) collectibles is acquired.
func (m *MembershipAnalysis) GetMissingItemHashes(ctx context.Context, membershipType int, membershipID, accessToken string) (map[uint32]struct{}, error) {
	a, err := m.getAnalysis(ctx, membershipType, membershipID, accessToken)
	if err != nil {
		return nil, err
	}
	missing := make(map[uint32]struct{})
	for _, f := range a.catalog {
		switch f.Category {
		case "weapons", "armor", "exotics":
			if !a.owned[f.ItemHash] {
				missing[f.ItemHash] = struct{}{}
			}
		}
	}
	return missing, nil
}

func (m *MembershipAnalysis) InvalidateCache(membershipType int, membershipID string) {
	m.cache.Delete(analysisCacheKey(membershipType, membershipID))
}

// OnVersionChanged retires every in-flight analysis and drops the shared tree,
// as one transition. Implements bungie.ManifestObserver.
//
// Per-membership `collections:*` entries are deliberately NOT evicted — see the
// comment in getAnalysis. Evicting them would discard a rate-limited Bungie
// profile fetch per active user on every hourly swap; refreshManifestParts
// rebuilds just the manifest-derived half instead.
func (m *MembershipAnalysis) OnVersionChanged(version string) error {
	return m.publication.Advance(version)
}

// refreshManifestParts returns an analysis whose manifest-derived fields belong
// to the current manifest generation, reusing the caller's profile data, and
// re-caches it when it had to rebuild. It returns `a` unchanged when the
// generation it was built under is still current, so the common path allocates
// nothing.
//
// It never mutates `a`: concurrent requests share that pointer without a lock,
// so a replacement is built and cached in its place.
func (m *MembershipAnalysis) refreshManifestParts(ctx context.Context, cacheKey string, a *analysis) (*analysis, error) {
	// One attempt covers the staleness question and everything the answer
	// causes, so a swap landing mid-rebuild cannot leave the replacement behind.
	attempt := m.publication.Begin()
	if a.builtUnder.Current() {
		return a, nil
	}

	catalog, tree, err := m.manifestParts(ctx, attempt)
	if err != nil {
		if errors.Is(err, ErrManifestNotReady) {
			// Mid-swap. Serving the previous manifest's labels for one more
			// request beats a 503 on data we already hold.
			observability.Logger(ctx).DebugContext(ctx, "collections: manifest unavailable mid-refresh; serving the previous analysis")
			return a, nil
		}
		return nil, err
	}

	refreshed := &analysis{
		catalog:    catalog,
		collected:  a.collected,
		owned:      deriveOwnedItems(catalog, a.collected),
		tree:       tree,
		fetchedAt:  a.fetchedAt,
		builtUnder: attempt,
	}
	attempt.Publish(func() { m.cache.Set(cacheKey, refreshed, m.cacheTTL) })
	return refreshed, nil
}
