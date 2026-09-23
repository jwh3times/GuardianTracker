// Package items serves manifest-derived, non-user-specific item detail —
// currently weapon perk pools and item views — with an in-memory cache cleared
// on manifest swap.
package items

import (
	"maps"
	"strings"
	"sync"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
	"guardian-tracker/api-service/services/manifeststate"
)

// maxCacheEntries bounds each cache so a flood of distinct (e.g. invalid)
// item hashes cannot grow it without limit between manifest swaps.
const maxCacheEntries = 4096

type itemRepo interface {
	GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error)
	GetItemsByHashes(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error)
	GetWeaponCatalysts(itemHash uint32) ([]manifest.WeaponCatalyst, error)
	GetAcquisitionRows(hashes []uint32) (*manifest.AcquisitionRows, error)
	GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error)
	GetWeaponHashes() ([]uint32, error)
}

// Service owns manifest-derived, user-independent item detail: weapon perk
// pools, catalyst pools, and the canonical acquisition facts every other module
// reads item meaning through.
//
// Every cached projection is guarded by one publication, so work that began
// under an older manifest generation cannot install itself after a swap. Data
// is static for a given manifest version, so entries live until that swap.
type Service struct {
	repo      itemRepo
	perks     *boundedCache[uint32, []manifest.PerkColumn]
	catalysts *boundedCache[uint32, []manifest.WeaponCatalyst]
	facts     *boundedCache[uint32, AcquisitionFacts]

	publication *manifeststate.Publication

	catalogMu sync.RWMutex
	catalog   []AcquisitionFacts

	perkNamesMu sync.RWMutex
	perkNames   map[string]string
}

func NewService(repo itemRepo) *Service {
	s := &Service{
		repo:      repo,
		perks:     newBoundedCache[uint32, []manifest.PerkColumn](maxCacheEntries),
		catalysts: newBoundedCache[uint32, []manifest.WeaponCatalyst](maxCacheEntries),
		facts:     newBoundedCache[uint32, AcquisitionFacts](maxCacheEntries),
	}
	// The callback runs inside the publication's critical section, so it does
	// only bounded map and slice clearing and never calls back into the
	// publication.
	s.publication = manifeststate.New(s.clearAll)
	return s
}

func (s *Service) loadCatalog() []AcquisitionFacts {
	s.catalogMu.RLock()
	defer s.catalogMu.RUnlock()
	return s.catalog
}

func (s *Service) storeCatalog(c []AcquisitionFacts) {
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	s.catalog = c
}

func (s *Service) loadPerkNames() map[string]string {
	s.perkNamesMu.RLock()
	defer s.perkNamesMu.RUnlock()
	return s.perkNames
}

func (s *Service) storePerkNames(n map[string]string) {
	s.perkNamesMu.Lock()
	defer s.perkNamesMu.Unlock()
	s.perkNames = n
}

// WeaponPerkNames returns every name any weapon's perk columns carry, keyed by
// lower case with the manifest's spelling as the value.
//
// Building it reads every weapon definition (about 2,200, a few seconds against
// the real manifest), so the set is one publication per manifest generation,
// like Catalog. It reads the repository directly rather than through the
// per-weapon cache, which is bounded and would evict. A failed read returns the
// error and caches nothing: an empty set would refuse every name as unknown.
// Callers receive a copy.
func (s *Service) WeaponPerkNames() (map[string]string, error) {
	attempt := s.publication.Begin()
	if cached := s.loadPerkNames(); cached != nil {
		return maps.Clone(cached), nil
	}

	hashes, err := s.repo.GetWeaponHashes()
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, h := range hashes {
		cols, err := s.repo.GetWeaponPerks(h)
		if err != nil {
			return nil, err
		}
		for _, c := range cols {
			for _, p := range c.Perks {
				names[strings.ToLower(p)] = p
			}
		}
	}

	attempt.Publish(func() { s.storePerkNames(names) })

	return maps.Clone(names), nil
}

// GetWeaponPerks returns cached columns or computes and caches them. A weapon
// with no perk columns caches as such — the answer is "none", not "unknown".
// Errors (including manifest-not-ready) are never cached.
func (s *Service) GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error) {
	return s.perks.load(itemHash, s.repo.GetWeaponPerks, nil)
}

// GetCatalysts returns a cached catalyst pool or computes and caches it.
// (nil, nil) for non-exotics and weapons without a catalyst socket, cached like
// GetWeaponPerks's non-weapon case. Errors are never cached.
// PlugNames resolves plug item hashes to their display names.
//
// It exists for callers that hold a plug hash with no weapon to place it in —
// a roll target that names perks without naming a weapon. Resolution is safe in
// this direction: many plugs share one display name, but each hash has exactly
// one, and a perk's base and enhanced variants share theirs, so either hash
// normalises to the same name. A hash the manifest does not know is absent from
// the result rather than mapped to an empty name.
func (s *Service) PlugNames(hashes []uint32) (map[uint32]string, error) {
	if len(hashes) == 0 {
		return map[uint32]string{}, nil
	}
	defs, err := s.repo.GetItemsByHashes(hashes)
	if err != nil {
		return nil, err
	}
	out := make(map[uint32]string, len(defs))
	for hash, def := range defs {
		if def == nil || def.DisplayProperties.Name == "" {
			continue
		}
		out[hash] = def.DisplayProperties.Name
	}
	return out, nil
}

func (s *Service) GetCatalysts(itemHash uint32) ([]manifest.WeaponCatalyst, error) {
	return s.catalysts.load(itemHash, s.repo.GetWeaponCatalysts, nil)
}

// OnVersionChanged retires every in-flight load and drops every cached
// projection, so both rebuild from the new manifest. Implements
// bungie.ManifestObserver.
//
// Advancing the publication is what makes this more than a cache clear: a
// request that started reading under the old manifest can no longer install its
// result afterwards. The two happen as one transition, so no loader can observe
// the new generation over the old cache.
func (s *Service) OnVersionChanged(version string) error {
	return s.publication.Advance(version)
}

// InvalidateCache drops every cached entry without advancing the generation.
//
// It stays for callers that want a cold cache rather than a new manifest — a
// version change must go through OnVersionChanged, or in-flight work would keep
// its claim on a generation whose data has been thrown away.
func (s *Service) InvalidateCache() {
	s.clearAll()
}

// clearAll is the publication's invalidation callback. It runs inside the
// publication's critical section, so it must stay bounded and must not call
// back into the publication.
func (s *Service) clearAll() {
	s.perks.clear()
	s.catalysts.clear()
	s.facts.clear()
	s.storeCatalog(nil)
	s.storePerkNames(nil)
}
