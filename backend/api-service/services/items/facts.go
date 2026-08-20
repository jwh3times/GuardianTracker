package items

import (
	"context"
	"errors"
	"slices"
	"sort"

	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
	"guardian-tracker/api-service/services/sources"
)

// AcquisitionFacts is the canonical, user-independent projection of one Item:
// everything Guardian Tracker knows about an item from the manifest alone.
//
// It deliberately carries no user state, no live vendor availability, no
// presentation-tree position, and no raw Bungie definitions. Those have other
// owners, and mixing them here is what let the previous per-consumer
// projections drift apart.
//
// Values are immutable by contract. The slices are allocated per result, so a
// caller cannot mutate state another caller is about to read.
type AcquisitionFacts struct {
	ItemHash    uint32
	Name        string
	Description string
	Icon        string

	// ItemType is slot-specific: weapons resolve to their weapon type and
	// armor to its equipment slot ("Helmet", "Chest Armor"), never a generic
	// "Armor". This is the projection Collections already used; making it
	// canonical is what stops the Wish List and item-detail views disagreeing
	// with Collections about the same item.
	ItemType string

	TierType int
	Rarity   string
	IsExotic bool

	// Category is the four-bucket collection rollup: weapons, armor, exotics,
	// or cosmetics. Empty when the item belongs to none.
	Category string

	// CollectibleHashes is every collectible linked to this item, unique and
	// ascending. An item with no linked collectible has an allocated empty
	// slice, not nil.
	CollectibleHashes []uint32

	// AcquisitionSources is the deterministic union contributed by every linked
	// collectible, ordered by visible text. Difficulty and raid/dungeon are
	// facets of each source; an item with several sources has no aggregate
	// difficulty.
	AcquisitionSources []sources.AcquisitionSource

	// FarmOnly retains the representative-collectible semantics described in
	// ADR 0015. See representativeFarmOnly for what "representative" means and
	// why the multi-collectible case is watched rather than resolved.
	FarmOnly bool
}

// clone returns a copy whose slices no caller can use to mutate shared state.
func (f AcquisitionFacts) clone() AcquisitionFacts {
	f.CollectibleHashes = slices.Clone(f.CollectibleHashes)
	f.AcquisitionSources = slices.Clone(f.AcquisitionSources)
	return f
}

// AcquisitionFactsReader is the canonical Item facts seam. Collections, the
// Wish List, and item-detail handling all read Item meaning through it rather
// than each joining the manifest themselves.
//
// There is deliberately no single-item convenience method, no projection flags,
// and no open-ended query: those are the shapes that let a seam accumulate
// storage-flavoured policy until it stops being about items at all.
type AcquisitionFactsReader interface {
	// Lookup resolves any inventory item, including items with no collectible.
	// An unknown hash is absent from the result rather than an error. Empty
	// input returns an allocated empty map without touching the manifest.
	Lookup(ctx context.Context, hashes []uint32) (map[uint32]AcquisitionFacts, error)

	// Catalog returns every distinct item linked to at least one collectible,
	// ordered by item hash.
	Catalog(ctx context.Context) ([]AcquisitionFacts, error)
}

// ErrDisagreeingFarmOnly reports the manifest state ADR 0015 left deliberately
// unresolved: one item whose linked collectibles disagree about whether it can
// be reacquired.
//
// It is never returned to a caller. It exists so the condition has a name in
// logs and in tests. See representativeFarmOnly.
var ErrDisagreeingFarmOnly = errors.New("items: linked collectibles disagree on farm-only")

// representativeFarmOnly derives FarmOnly from an item's linked collectibles,
// and reports whether they disagreed.
//
// ADR 0015 keeps the existing representative-collectible behaviour: FarmOnly
// comes from one collectible, not from a union. Before this seam the
// representative was whichever row an unordered `SELECT` happened to return
// first; here it is the lowest collectible hash, matching the ascending order
// the rest of this type uses. Verification against manifest
// 244213.26.06.29.2000-1-bnet.65583 confirmed that switching to a deterministic
// order changes the representative for 17 items and changes the farm-only
// outcome for none of them.
//
// The disagreement flag exists because that verification also showed the
// agreement is coincidental rather than structural: every farm-only item in
// that manifest has exactly one collectible, so farm-only and multi-collectible
// are disjoint populations that simply do not meet. If a future manifest ships
// a farm-only item with two collectibles, the representative choice becomes
// observable again — and without this, it would become observable silently.
// Callers log it; nobody fails a request over a manifest anomaly, because the
// blast radius of refusing to serve Collections is far worse than serving a
// deterministic answer while saying loudly that the answer is now a choice.
func representativeFarmOnly(cols []bungie.CollectibleDefinition) (farmOnly, disagreed bool) {
	if len(cols) == 0 {
		return false, false
	}

	lowest := cols[0]
	sawTrue, sawFalse := false, false
	for _, c := range cols {
		if isUnreacquirable(c) {
			sawTrue = true
		} else {
			sawFalse = true
		}
		if c.Hash < lowest.Hash {
			lowest = c
		}
	}
	return isUnreacquirable(lowest), sawTrue && sawFalse
}

func isUnreacquirable(c bungie.CollectibleDefinition) bool {
	return sources.IsUnreacquirable(c.SourceString)
}

// buildFacts projects one item plus its linked collectibles into canonical
// facts. cols may be empty: an item with no collectible is a valid, complete
// answer with empty sources, not a missing one.
func buildFacts(item *bungie.InventoryItemDefinition, cols []bungie.CollectibleDefinition) AcquisitionFacts {
	hashes := make([]uint32, 0, len(cols))
	texts := make([]string, 0, len(cols))
	for _, c := range cols {
		hashes = append(hashes, c.Hash)
		texts = append(texts, c.SourceString)
	}
	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
	hashes = slices.Compact(hashes)

	farmOnly, _ := representativeFarmOnly(cols)

	return AcquisitionFacts{
		ItemHash:           item.Hash,
		Name:               item.DisplayProperties.Name,
		Description:        item.DisplayProperties.Description,
		Icon:               item.DisplayProperties.Icon,
		ItemType:           slotSpecificItemType(item),
		TierType:           item.Inventory.TierType,
		Rarity:             bungie.GetTierName(item.Inventory.TierType),
		IsExotic:           item.Inventory.TierType == bungie.TierTypeExotic,
		Category:           manifest.CollectibleCategory(item),
		CollectibleHashes:  hashes,
		AcquisitionSources: sources.DescribeAll(texts),
		FarmOnly:           farmOnly,
	}
}

// slotSpecificItemType resolves the item type the way Collections always has:
// weapons by weapon type, armor by equipment slot, everything else by the
// shared type/subtype naming.
func slotSpecificItemType(item *bungie.InventoryItemDefinition) string {
	switch item.ItemType {
	case bungie.ItemTypeWeapon:
		return bungie.GetWeaponTypeName(item.ItemSubType)
	case bungie.ItemTypeArmor:
		return bungie.GetArmorTypeName(item.EquippingBlock.EquipmentSlotTypeHash)
	default:
		return bungie.ItemTypeName(item.ItemType, item.ItemSubType)
	}
}

// logDisagreement reports the watched manifest anomaly once per affected item.
func logDisagreement(ctx context.Context, itemHash uint32, cols []bungie.CollectibleDefinition) {
	hashes := make([]uint32, 0, len(cols))
	for _, c := range cols {
		hashes = append(hashes, c.Hash)
	}
	observability.Logger(ctx).ErrorContext(ctx,
		"linked collectibles disagree on farm-only; representative choice is now observable",
		"item_hash", itemHash,
		"collectible_hashes", hashes,
		observability.Err(ErrDisagreeingFarmOnly),
	)
}
