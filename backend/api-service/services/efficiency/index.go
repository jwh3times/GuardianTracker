package efficiency

import (
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
	"guardian-tracker/api-service/services/sources"
)

// BucketItem is one distinct item in a source bucket, with the rarity used for
// weighting. CollectibleHash identifies the representative linked collectible.
type BucketItem struct {
	CollectibleHash uint32
	ItemHash        uint32
	Rarity          string // "Exotic","Legendary","Rare",...
}

// Bucket groups distinct items that share a sourceHash (one in-game source).
type Bucket struct {
	SourceHash   uint32
	Label        string // cleaned, e.g. "Vault of Glass"
	SourceString string // raw, for difficulty mapping by the caller
	Kind         sources.Kind
	Items        []BucketItem
}

// buildBuckets groups joined collectible+item rows by sourceHash.
func buildBuckets(rows []manifest.CollectibleWithItem) map[uint32]*Bucket {
	buckets := make(map[uint32]*Bucket)
	seenItems := make(map[uint32]map[uint32]struct{})
	for _, cwi := range rows {
		col := cwi.Collectible
		b := buckets[col.SourceHash]
		if b == nil {
			label := cleanLabel(col.SourceString)
			b = &Bucket{
				SourceHash:   col.SourceHash,
				Label:        label,
				SourceString: col.SourceString,
				Kind:         classifyBucket(col.SourceHash, col.SourceString),
			}
			buckets[col.SourceHash] = b
			seenItems[col.SourceHash] = make(map[uint32]struct{})
		}
		if _, duplicate := seenItems[col.SourceHash][col.ItemHash]; duplicate {
			continue
		}
		seenItems[col.SourceHash][col.ItemHash] = struct{}{}
		rarity := ""
		if cwi.Item != nil {
			rarity = bungie.GetTierName(cwi.Item.Inventory.TierType)
		}
		b.Items = append(b.Items, BucketItem{
			CollectibleHash: col.Hash,
			ItemHash:        col.ItemHash,
			Rarity:          rarity,
		})
	}
	return buckets
}
