package weekly

import (
	"strconv"

	"guardian-tracker/api-service/services/bungie"
)

// liveVendorAllowlist maps the character-402 rotating vendors we surface as
// "available now" to their display names. Xûr is handled separately (public
// vendor fetch), so it is not in this map.
var liveVendorAllowlist = map[uint32]string{
	bungie.Banshee44VendorHash: "Banshee-44",
	bungie.Ada1VendorHash:      "Ada-1",
	bungie.ShaxxVendorHash:     "Lord Shaxx",
	bungie.ZavalaVendorHash:    "Commander Zavala",
	bungie.DrifterVendorHash:   "The Drifter",
}

// extractVendorItems returns every sale-item hash sold by an allowlisted vendor,
// mapped to that vendor's display name. Non-allowlisted vendors and zero hashes
// are skipped. Unlike getDailyVendorItems (which keeps only mods for the "Do This
// Today" strip), this keeps ALL sale items — the downstream collection
// intersection drops anything that isn't a tracked collectible.
func extractVendorItems(resp *bungie.CharacterVendorsResponse, allowlist map[uint32]string) map[uint32]string {
	out := map[uint32]string{}
	if resp == nil {
		return out
	}
	for vendorKey, sales := range resp.Response.Sales.Data {
		vh, err := strconv.ParseUint(vendorKey, 10, 32)
		if err != nil {
			continue
		}
		name, ok := allowlist[uint32(vh)]
		if !ok {
			continue
		}
		for _, sale := range sales.SaleItems {
			if sale.ItemHash == 0 {
				continue
			}
			out[sale.ItemHash] = name
		}
	}
	return out
}
