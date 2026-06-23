package weekly

import (
	"context"
	"log"
	"strconv"
	"time"

	"guardian-tracker/api-service/services/bungie"
)

// liveVendorItemsCacheKey is the shared daily-cache key for the rotating
// character-402 vendors' sale items (getLiveVendorItems + tests).
const liveVendorItemsCacheKey = "live:vendoritems"

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

// LiveVendorItemHashes returns item hashes currently sold by rotating vendors
// (Xûr, Banshee-44, Ada-1, Shaxx, Zavala, Drifter), mapped to the selling vendor's
// display name. Reuses the existing Xûr (public) and character-402 vendor fetches
// and their reset/daily caches — no new Bungie API calls. Best-effort: returns
// whatever is available (possibly empty) on token/character/fetch failure; never
// errors.
func (s *Service) LiveVendorItemHashes(ctx context.Context, membershipType int, membershipID, bungieToken string) map[uint32]string {
	return s.liveVendorItemHashesAt(ctx, membershipType, membershipID, bungieToken, time.Now().UTC())
}

func (s *Service) liveVendorItemHashesAt(ctx context.Context, membershipType int, membershipID, bungieToken string, now time.Time) map[uint32]string {
	out := map[uint32]string{}
	for h, name := range s.getLiveVendorItems(ctx, membershipType, membershipID, bungieToken, now) {
		out[h] = name
	}
	// Xûr added last so it wins ties (it is the headline weekly vendor).
	for h := range s.xurItemHashesAt(ctx, now) {
		out[h] = "Xûr"
	}
	return out
}

// getLiveVendorItems fetches the allowlisted character-402 vendors' sale items,
// cached daily (shared across users — the rotation is identical for everyone).
// Returns nil when the fetch is impossible (no client/token/character) or fails.
func (s *Service) getLiveVendorItems(ctx context.Context, membershipType int, membershipID, bungieToken string, now time.Time) map[uint32]string {
	const cacheKey = liveVendorItemsCacheKey
	if cached, ok := s.cache.Get(cacheKey); ok {
		if m, ok := cached.(map[uint32]string); ok {
			return m
		}
	}
	if s.bungie == nil || bungieToken == "" {
		return nil
	}
	characterID := s.resolvePrimaryCharacter(ctx, membershipType, membershipID, bungieToken)
	if characterID == "" {
		return nil
	}
	resp, err := s.bungie.GetCharacterVendors(ctx, membershipType, membershipID, characterID, bungieToken)
	if err != nil {
		log.Printf("weekly: LiveVendorItemHashes GetCharacterVendors: %v", err)
		return nil
	}
	items := extractVendorItems(resp, liveVendorAllowlist)
	// Only cache non-empty results — a transient empty response must not poison
	// the cache for all users until the next daily reset.
	if len(items) > 0 {
		ttl := max(NextDailyReset(now).Sub(now), 5*time.Minute)
		s.cache.Set(cacheKey, items, ttl)
	}
	return items
}
