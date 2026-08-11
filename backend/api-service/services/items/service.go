// Package items serves manifest-derived, non-user-specific item detail —
// currently weapon perk pools and item views — with an in-memory cache cleared
// on manifest swap.
package items

import (
	"guardian-tracker/api-service/services/manifest"
)

// maxCacheEntries bounds each cache so a flood of distinct (e.g. invalid)
// item hashes cannot grow it without limit between manifest swaps.
const maxCacheEntries = 4096

type itemRepo interface {
	GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error)
	GetItemView(itemHash uint32) (*manifest.ItemView, error)
	GetWeaponCatalysts(itemHash uint32) ([]manifest.WeaponCatalyst, error)
}

// Service caches weapon perk columns, item views, and catalyst pools keyed by
// item hash. Data is static for a given manifest version, so entries live until
// the next manifest swap calls InvalidateCache.
type Service struct {
	repo      itemRepo
	perks     *boundedCache[uint32, []manifest.PerkColumn]
	views     *boundedCache[uint32, *manifest.ItemView]
	catalysts *boundedCache[uint32, []manifest.WeaponCatalyst]
}

func NewService(repo itemRepo) *Service {
	return &Service{
		repo:      repo,
		perks:     newBoundedCache[uint32, []manifest.PerkColumn](maxCacheEntries),
		views:     newBoundedCache[uint32, *manifest.ItemView](maxCacheEntries),
		catalysts: newBoundedCache[uint32, []manifest.WeaponCatalyst](maxCacheEntries),
	}
}

// GetWeaponPerks returns cached columns or computes and caches them. A weapon
// with no perk columns caches as such — the answer is "none", not "unknown".
// Errors (including manifest-not-ready) are never cached.
func (s *Service) GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error) {
	return s.perks.load(itemHash, s.repo.GetWeaponPerks, nil)
}

// GetItem returns a cached minimal item view or computes and caches it. An
// unknown hash returns (nil, nil) and is deliberately NOT cached: unlike a
// weapon with no perks, "no such item" is an answer about the request rather
// than about the manifest, and caching it would hold a bogus hash in a bounded
// cache that real items are competing for.
func (s *Service) GetItem(itemHash uint32) (*manifest.ItemView, error) {
	return s.views.load(itemHash, s.repo.GetItemView, func(v *manifest.ItemView) bool { return v != nil })
}

// GetCatalysts returns a cached catalyst pool or computes and caches it.
// (nil, nil) for non-exotics and weapons without a catalyst socket, cached like
// GetWeaponPerks's non-weapon case. Errors are never cached.
func (s *Service) GetCatalysts(itemHash uint32) ([]manifest.WeaponCatalyst, error) {
	return s.catalysts.load(itemHash, s.repo.GetWeaponCatalysts, nil)
}

// OnVersionChanged drops every cached projection so it rebuilds from the new
// manifest. Implements bungie.ManifestObserver.
func (s *Service) OnVersionChanged(version string) error {
	s.InvalidateCache()
	return nil
}

// InvalidateCache drops every cached entry in all three caches.
func (s *Service) InvalidateCache() {
	s.perks.clear()
	s.views.clear()
	s.catalysts.clear()
}
