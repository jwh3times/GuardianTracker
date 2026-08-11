package weekly

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
)

const characterVendorsCacheTTL = 5 * time.Minute

// liveVendorAllowlist maps the character-scoped rotating vendors we surface as
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
	return s.liveVendorItemHashesAt(ctx, membershipType, membershipID, bungieToken, s.nowUTC())
}

func (s *Service) liveVendorItemHashesAt(ctx context.Context, membershipType int, membershipID, bungieToken string, now time.Time) map[uint32]string {
	characterID, _ := s.resolveCharacter(ctx, membershipType, membershipID, bungieToken, "")
	return s.liveVendorItemHashesAtCharacter(ctx, membershipType, membershipID, characterID, bungieToken, now)
}

func (s *Service) liveVendorItemHashesAtCharacter(ctx context.Context, membershipType int, membershipID, characterID, bungieToken string, now time.Time) map[uint32]string {
	out := map[uint32]string{}
	for h, name := range s.getLiveVendorItems(ctx, membershipType, membershipID, characterID, bungieToken, now) {
		out[h] = name
	}
	// Xûr added last so it wins ties (it is the headline weekly vendor).
	for h := range s.xurItemHashesAt(ctx, now) {
		out[h] = "Xûr"
	}
	return out
}

// characterVendorsCacheKey holds the raw authenticated vendor response. Not
// scoped by manifest version: the value carries no manifest-resolved labels, so
// it deliberately survives a swap (asserted in swap_test.go).
func characterVendorsCacheKey(membershipType int, membershipID, characterID string) string {
	return fmt.Sprintf("vendors:character:%d:%s:%s", membershipType, membershipID, characterID)
}

// getCharacterVendors returns the authenticated 400+402 vendor response for a
// user's validated selected character. The short character-scoped cache lets
// Xûr location, daily actions, and availability enrichment share one response
// without treating potentially account-specific vendor state as global.
func (s *Service) getCharacterVendors(ctx context.Context, membershipType int, membershipID, characterID, bungieToken string) *bungie.CharacterVendorsResponse {
	if bungieToken == "" || characterID == "" {
		return nil
	}
	resp, err := cache.Load(ctx, s.cache, characterVendorsCacheKey(membershipType, membershipID, characterID), characterVendorsCacheTTL,
		func() (*bungie.CharacterVendorsResponse, error) {
			if s.bungie == nil {
				return nil, errUpstreamUnavailable
			}
			return s.bungie.GetCharacterVendors(ctx, membershipType, membershipID, characterID, bungieToken)
		})
	switch {
	case errors.Is(err, errUpstreamUnavailable):
		// Not a failure worth reporting: a service built without a client can
		// still serve everything a seeded cache answers.
		return nil
	case err != nil:
		observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "weekly character vendors fetch failed",
			slog.Int("membership_type", membershipType),
			observability.ID("membership", membershipID),
			observability.ID("character", characterID),
			observability.Err(err),
		)
		return nil
	}
	return resp
}

// errUpstreamUnavailable marks a load that produced nothing worth caching:
// there is no Bungie client, or an upstream call already logged its own
// failure. It is a miss, not a new failure, and must never be logged again.
var errUpstreamUnavailable = errors.New("weekly: upstream unavailable")

// liveVendorItemsCacheKey holds allowlisted vendor sale items. Not scoped by
// manifest version: the vendor names come from a static in-code allowlist, not
// from the manifest.
func liveVendorItemsCacheKey(membershipType int, membershipID, characterID string) string {
	return fmt.Sprintf("live:vendoritems:%d:%s:%s", membershipType, membershipID, characterID)
}

// getLiveVendorItems fetches the allowlisted character vendors' sale items,
// cached daily per character because component 402 can be class-specific.
// Returns nil when the fetch is impossible (no client/token/character) or fails.
func (s *Service) getLiveVendorItems(ctx context.Context, membershipType int, membershipID, characterID, bungieToken string, now time.Time) map[uint32]string {
	// Only a non-empty result is cached — a transient empty response must not
	// poison the cache for all users until the next daily reset.
	items, err := cache.LoadIf(ctx, s.cache, liveVendorItemsCacheKey(membershipType, membershipID, characterID),
		max(NextDailyReset(now).Sub(now), 5*time.Minute),
		func() (map[uint32]string, error) {
			resp := s.getCharacterVendors(ctx, membershipType, membershipID, characterID, bungieToken)
			if resp == nil {
				return nil, errUpstreamUnavailable
			}
			return extractVendorItems(resp, liveVendorAllowlist), nil
		},
		cache.NonEmptyMap)
	if err != nil {
		return nil
	}
	return items
}
