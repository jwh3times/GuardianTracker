package collections

import (
	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/sources"
)

// The types below keep their JSON tags because the HTTP adapter passes them
// through verbatim. Only the item-derived fields — `items`, `collectedHashes`,
// and `availableNow` — are assembled at the boundary from a [Full] result; the
// tree, the category rollup, and fetchedAt are serialized as they stand here.

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
