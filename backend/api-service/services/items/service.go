// Package items serves manifest-derived, non-user-specific item detail —
// currently weapon perk pools and item views — with an in-memory cache cleared
// on manifest swap.
package items

import (
	"sync"

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
	repo          itemRepo
	mu            sync.RWMutex
	cache         map[uint32][]manifest.PerkColumn
	viewCache     map[uint32]*manifest.ItemView
	catalystCache map[uint32][]manifest.WeaponCatalyst
}

func NewService(repo itemRepo) *Service {
	return &Service{
		repo:          repo,
		cache:         map[uint32][]manifest.PerkColumn{},
		viewCache:     map[uint32]*manifest.ItemView{},
		catalystCache: map[uint32][]manifest.WeaponCatalyst{},
	}
}

// GetWeaponPerks returns cached columns or computes and caches them. Errors
// (including manifest-not-ready) are never cached.
func (s *Service) GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error) {
	s.mu.RLock()
	cols, ok := s.cache[itemHash]
	s.mu.RUnlock()
	if ok {
		return cols, nil
	}

	cols, err := s.repo.GetWeaponPerks(itemHash)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	if _, exists := s.cache[itemHash]; !exists && len(s.cache) >= maxCacheEntries {
		// Bound memory: the cache is keyed by the client-supplied item hash, so an
		// authenticated caller hitting many distinct hashes could otherwise grow it
		// without limit between manifest swaps. Evict an arbitrary entry at capacity;
		// legitimate traffic (a few thousand real weapons) never reaches the cap.
		for k := range s.cache {
			delete(s.cache, k)
			break
		}
	}
	s.cache[itemHash] = cols
	s.mu.Unlock()
	return cols, nil
}

// GetItem returns a cached minimal item view or computes and caches it. (nil,nil) for
// an unknown hash (not cached). Errors (incl. manifest-not-ready) are never cached.
func (s *Service) GetItem(itemHash uint32) (*manifest.ItemView, error) {
	s.mu.RLock()
	v, ok := s.viewCache[itemHash]
	s.mu.RUnlock()
	if ok {
		return v, nil
	}

	v, err := s.repo.GetItemView(itemHash)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}

	s.mu.Lock()
	if _, exists := s.viewCache[itemHash]; !exists && len(s.viewCache) >= maxCacheEntries {
		for k := range s.viewCache {
			delete(s.viewCache, k)
			break
		}
	}
	s.viewCache[itemHash] = v
	s.mu.Unlock()
	return v, nil
}

// GetCatalysts returns a cached catalyst pool or computes and caches it.
// (nil, nil) for non-exotics / weapons without a catalyst socket (cached, like
// GetWeaponPerks's non-weapon case). Errors (incl. manifest-not-ready) are never
// cached.
func (s *Service) GetCatalysts(itemHash uint32) ([]manifest.WeaponCatalyst, error) {
	s.mu.RLock()
	cats, ok := s.catalystCache[itemHash]
	s.mu.RUnlock()
	if ok {
		return cats, nil
	}

	cats, err := s.repo.GetWeaponCatalysts(itemHash)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	if _, exists := s.catalystCache[itemHash]; !exists && len(s.catalystCache) >= maxCacheEntries {
		for k := range s.catalystCache {
			delete(s.catalystCache, k)
			break
		}
	}
	s.catalystCache[itemHash] = cats
	s.mu.Unlock()
	return cats, nil
}

// InvalidateCache drops every cached entry in all three caches. Wired to the manifest swap hook.
func (s *Service) InvalidateCache() {
	s.mu.Lock()
	s.cache = map[uint32][]manifest.PerkColumn{}
	s.viewCache = map[uint32]*manifest.ItemView{}
	s.catalystCache = map[uint32][]manifest.WeaponCatalyst{}
	s.mu.Unlock()
}
