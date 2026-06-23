package weekly

import (
	"context"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"guardian-tracker/api-service/services/bungie"
)

// vendorRole maps a rotating vendor's display name to its flavor role label.
var vendorRole = map[string]string{
	"Banshee-44":       "Gunsmith",
	"Ada-1":            "Advanced Armory",
	"Lord Shaxx":       "Crucible",
	"Commander Zavala": "Vanguard",
	"The Drifter":      "Gambit",
}

// vendorOrder is the fixed display order of the rotating vendor cards.
var vendorOrder = []string{"Banshee-44", "Ada-1", "Lord Shaxx", "Commander Zavala", "The Drifter"}

const vendorItemCap = 20

func vendorSlug(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

// buildVendorRotations returns one VendorRotation per rotating character-402 vendor
// that currently offers collectible gear (legendary/exotic weapons & armor), reusing
// the cached getLiveVendorItems fetch (no new Bungie call). Missing counts are computed
// against the caller's missing-collection set; items are missing-first, capped, and
// empty vendors are dropped. Best-effort: returns nil when no live vendor data.
func (s *Service) buildVendorRotations(ctx context.Context, membershipType int, membershipID, bungieToken string, missingHashes map[uint32]struct{}, now time.Time) []VendorRotation {
	live := s.getLiveVendorItems(ctx, membershipType, membershipID, bungieToken, now)
	if len(live) == 0 {
		return nil
	}

	hashes := make([]uint32, 0, len(live))
	for h := range live {
		hashes = append(hashes, h)
	}
	defs, err := s.manifest.GetItemsByHashes(hashes)
	if err != nil {
		log.Printf("weekly: buildVendorRotations GetItemsByHashes: %v", err)
		return nil
	}

	type tmpItem struct {
		item    VendorItem
		missing bool
	}
	byVendor := map[string][]tmpItem{}
	missingCount := map[string]int{}
	for h, vendorName := range live {
		def := defs[h]
		if def == nil || def.DisplayProperties.Name == "" {
			continue
		}
		if def.ItemType != bungie.ItemTypeWeapon && def.ItemType != bungie.ItemTypeArmor {
			continue
		}
		if def.Inventory.TierType != bungie.TierTypeLegendary && def.Inventory.TierType != bungie.TierTypeExotic {
			continue
		}
		_, isMissing := missingHashes[h]
		byVendor[vendorName] = append(byVendor[vendorName], tmpItem{
			item:    VendorItem{Hash: strconv.FormatUint(uint64(h), 10), Name: def.DisplayProperties.Name},
			missing: isMissing,
		})
		if isMissing {
			missingCount[vendorName]++
		}
	}

	out := []VendorRotation{}
	for _, name := range vendorOrder {
		items := byVendor[name]
		if len(items) == 0 {
			continue
		}
		sort.SliceStable(items, func(i, j int) bool { return items[i].missing && !items[j].missing })
		if len(items) > vendorItemCap {
			items = items[:vendorItemCap]
		}
		vis := make([]VendorItem, len(items))
		for i, ti := range items {
			vis[i] = ti.item
		}
		out = append(out, VendorRotation{
			ID:      "v-" + vendorSlug(name),
			Name:    name,
			Role:    vendorRole[name],
			Missing: missingCount[name],
			Items:   vis,
		})
	}
	return out
}
