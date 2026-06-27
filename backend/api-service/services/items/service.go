// Package items serves manifest-derived, non-user-specific item detail —
// currently weapon perk pools — with an in-memory cache cleared on manifest swap.
package items

import (
	"sync"

	"guardian-tracker/api-service/services/manifest"
)

type weaponPerksRepo interface {
	GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error)
}

// Service caches weapon perk columns keyed by item hash. Perk pools are static
// for a given manifest version, so entries live until the next manifest swap
// calls InvalidateCache.
type Service struct {
	repo  weaponPerksRepo
	mu    sync.RWMutex
	cache map[uint32][]manifest.PerkColumn
}

func NewService(repo weaponPerksRepo) *Service {
	return &Service{repo: repo, cache: map[uint32][]manifest.PerkColumn{}}
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
	s.cache[itemHash] = cols
	s.mu.Unlock()
	return cols, nil
}

// InvalidateCache drops every cached entry. Wired to the manifest swap hook.
func (s *Service) InvalidateCache() {
	s.mu.Lock()
	s.cache = map[uint32][]manifest.PerkColumn{}
	s.mu.Unlock()
}
