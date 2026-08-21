package items

import (
	"context"
	"slices"
	"sort"

	"guardian-tracker/api-service/services/bungie"
)

// Lookup resolves canonical facts for the requested item hashes.
//
// Unknown hashes are simply absent from the result: an item that is not in the
// manifest is an answer about the request, not a failure. Manifest
// unavailability, cancellation, and query failure are errors with no partial
// result — an unavailable manifest must never be mistaken for "this item has no
// sources", which is how the previous per-consumer joins produced
// plausible-looking wrong answers.
func (s *Service) Lookup(ctx context.Context, hashes []uint32) (map[uint32]AcquisitionFacts, error) {
	if len(hashes) == 0 {
		return map[uint32]AcquisitionFacts{}, nil
	}

	wanted := slices.Clone(hashes)
	sort.Slice(wanted, func(i, j int) bool { return wanted[i] < wanted[j] })
	wanted = slices.Compact(wanted)

	// Serve from cache only when the whole batch hits. A partial hit is
	// deliberately discarded rather than topped up: combining facts cached under
	// one manifest generation with facts queried under another would produce a
	// result that never existed in any single manifest.
	attempt := s.publication.Begin()
	if cached, ok := s.cachedBatch(wanted); ok {
		return cached, nil
	}

	rows, err := s.repo.GetAcquisitionRows(wanted)
	if err != nil {
		return nil, err
	}

	out := make(map[uint32]AcquisitionFacts, len(wanted))
	for _, h := range wanted {
		item := rows.Items[h]
		if item == nil {
			continue // unknown hash: absent, not an error
		}
		cols := rows.Collectibles[h]
		if _, disagreed := representativeFarmOnly(cols); disagreed {
			logDisagreement(ctx, h, cols)
		}
		out[h] = buildFacts(item, cols)
	}

	// Only the current generation may leave anything behind. A batch loaded
	// against a manifest that has since been replaced still answers the request
	// that asked for it.
	attempt.Publish(func() {
		for h, f := range out {
			s.facts.put(h, f)
		}
	})

	return cloneFactsMap(out), nil
}

// Catalog returns canonical facts for every item linked to at least one
// collectible, ordered by item hash.
//
// The result is one immutable publication per manifest generation: the whole
// catalog is a single value, so a caller can never see half of one generation
// and half of another. Callers receive a defensive copy.
func (s *Service) Catalog(ctx context.Context) ([]AcquisitionFacts, error) {
	attempt := s.publication.Begin()
	if cached := s.loadCatalog(); cached != nil {
		return cloneFacts(cached), nil
	}

	rows, err := s.repo.GetAllCollectiblesWithItems()
	if err != nil {
		return nil, err
	}

	// Group by item first: an item with several collectibles is one catalog
	// entry whose sources are the union, not several competing entries.
	colsByItem := make(map[uint32][]bungie.CollectibleDefinition)
	itemsByHash := make(map[uint32]*bungie.InventoryItemDefinition)
	for _, cwi := range rows {
		if cwi.Item == nil || cwi.Item.DisplayProperties.Name == "" {
			continue
		}
		h := cwi.Item.Hash
		itemsByHash[h] = cwi.Item
		colsByItem[h] = append(colsByItem[h], cwi.Collectible)
	}

	built := make([]AcquisitionFacts, 0, len(itemsByHash))
	for h, item := range itemsByHash {
		cols := colsByItem[h]
		if _, disagreed := representativeFarmOnly(cols); disagreed {
			logDisagreement(ctx, h, cols)
		}
		built = append(built, buildFacts(item, cols))
	}
	sort.Slice(built, func(i, j int) bool { return built[i].ItemHash < built[j].ItemHash })

	attempt.Publish(func() { s.storeCatalog(built) })

	return cloneFacts(built), nil
}

// cachedBatch returns the requested facts only if every hash is present.
func (s *Service) cachedBatch(wanted []uint32) (map[uint32]AcquisitionFacts, bool) {
	out := make(map[uint32]AcquisitionFacts, len(wanted))
	for _, h := range wanted {
		f, ok := s.facts.get(h)
		if !ok {
			return nil, false
		}
		out[h] = f.clone()
	}
	return out, true
}

func cloneFacts(in []AcquisitionFacts) []AcquisitionFacts {
	out := make([]AcquisitionFacts, len(in))
	for i, f := range in {
		out[i] = f.clone()
	}
	return out
}

func cloneFactsMap(in map[uint32]AcquisitionFacts) map[uint32]AcquisitionFacts {
	out := make(map[uint32]AcquisitionFacts, len(in))
	for h, f := range in {
		out[h] = f.clone()
	}
	return out
}
