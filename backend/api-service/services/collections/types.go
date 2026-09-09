package collections

import (
	"time"

	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/sources"
)

// MembershipCollections is the canonical collections payload: the Bungie presentation-node
// tree, a shared item-detail map (only on ?include=all), a flat set of owned item
// hashes (the grid's per-item collected state; only on ?include=all), and a derived
// four-category summary for the Dashboard hero and weekly recommender.
type MembershipCollections struct {
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
func (u MembershipCollections) Lightweight() MembershipCollections {
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

// DestinyItem is the frontend-facing item representation.
type DestinyItem struct {
	ItemHash           string                      `json:"itemHash"`
	Name               string                      `json:"name"`
	Description        string                      `json:"description"`
	Icon               string                      `json:"icon"`
	ItemType           string                      `json:"itemType"`
	TierType           int                         `json:"tierType"`
	Rarity             string                      `json:"rarity"`
	FarmOnly           bool                        `json:"farmOnly"`
	AcquisitionSources []sources.AcquisitionSource `json:"acquisitionSources"`
	IsExotic           bool                        `json:"isExotic"`
}

// destinyItem renders canonical Item facts into the collections wire shape.
//
// It is a rename, not a projection with opinions: every field comes from Items
// (ADR 0015), so what the Collections grid says about an item is the same thing
// the Wish List and item detail say about it. The source union, the
// slot-specific item type, and the farm-only representative are decided there.
func destinyItem(f items.AcquisitionFacts) DestinyItem {
	return DestinyItem{
		ItemHash:           itemHashString(f.ItemHash),
		Name:               f.Name,
		Description:        f.Description,
		Icon:               f.Icon,
		ItemType:           f.ItemType,
		TierType:           f.TierType,
		Rarity:             f.Rarity,
		FarmOnly:           f.FarmOnly,
		AcquisitionSources: f.AcquisitionSources,
		IsExotic:           f.IsExotic,
	}
}
