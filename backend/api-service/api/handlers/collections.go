package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"slices"
	"strconv"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/characters"
	"guardian-tracker/api-service/services/collections"
	"guardian-tracker/api-service/services/records"

	"github.com/gin-gonic/gin"
)

// liveAvailabilityProvider returns itemHash → vendor name for items obtainable
// right now from rotating vendors. Satisfied by *weekly.Service.
type liveAvailabilityProvider interface {
	LiveVendorItemHashes(ctx context.Context, membershipType int, membershipID, bungieToken string) map[uint32]string
}

// CollectionsHandler handles collection-related endpoints.
type CollectionsHandler struct {
	collectionsService *collections.Service
	charactersService  *characters.Service
	recordsService     *records.Service
	tokenStore         *auth.TokenStore
	liveVendors        liveAvailabilityProvider
}

func NewCollectionsHandler(svc *collections.Service, chars *characters.Service, recs *records.Service, ts *auth.TokenStore, live liveAvailabilityProvider) *CollectionsHandler {
	return &CollectionsHandler{
		collectionsService: svc,
		charactersService:  chars,
		recordsService:     recs,
		tokenStore:         ts,
		liveVendors:        live,
	}
}

// GetCollections handles GET /api/collections/:membershipType/:membershipId
// Requires jwtHelper.Middleware() on the route (validates JWT, enforces access token type).
func (h *CollectionsHandler) GetCollections(c *gin.Context) {
	membershipType, membershipID, ok := parseMembershipParams(c)
	if !ok {
		return
	}

	if !ownershipCheck(c, membershipID) {
		return
	}

	bungieToken, ok := getBungieToken(c, membershipID, h.tokenStore)
	if !ok {
		return
	}

	result, err := h.collectionsService.GetUserCollections(c.Request.Context(), membershipType, membershipID, bungieToken)
	if err != nil {
		handleBungieError(c, err)
		return
	}

	// The item-detail map and per-item collected hashes ride along only when asked
	// for (?include=all). Lightweight copies by value, so the cached *UserCollections
	// is never mutated.
	if c.Query("include") != "all" {
		c.JSON(http.StatusOK, result.Lightweight())
		return
	}
	// Stamp live "available now" availability (best-effort enrichment — never fails
	// the request). result is a fresh per-request value; result.Items is read-only.
	if h.liveVendors != nil {
		live := h.liveVendors.LiveVendorItemHashes(c.Request.Context(), membershipType, membershipID, bungieToken)
		result.AvailableNow = availableNowOverlay(result.Items, live)
	}
	c.JSON(http.StatusOK, result)
}

// RefreshCollections handles POST /api/collections/:membershipType/:membershipId/refresh
func (h *CollectionsHandler) RefreshCollections(c *gin.Context) {
	membershipType, membershipID, ok := parseMembershipParams(c)
	if !ok {
		return
	}

	if !ownershipCheck(c, membershipID) {
		return
	}

	// Each service owns its own cache key — invalidate through them so a key
	// format change can never silently leave a stale entry behind.
	h.collectionsService.InvalidateCache(membershipType, membershipID)
	h.charactersService.InvalidateCache(membershipType, membershipID)
	h.recordsService.InvalidateCache(membershipType, membershipID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache invalidated. Next request will fetch fresh data.",
	})
}

// parseMembershipParams parses and validates :membershipType and :membershipId path params.
func parseMembershipParams(c *gin.Context) (int, string, bool) {
	membershipType, err := strconv.Atoi(c.Param("membershipType"))
	if err != nil || !isValidMembershipType(membershipType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid membership type"})
		return 0, "", false
	}
	membershipID := c.Param("membershipId")
	if !isValidMembershipID(membershipID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid membership ID"})
		return 0, "", false
	}
	return membershipType, membershipID, true
}

func isValidMembershipType(t int) bool {
	return slices.Contains([]int{1, 2, 3, 4, 5, 6, 10, 254}, t)
}

func isValidMembershipID(id string) bool {
	if len(id) < 10 || len(id) > 25 {
		return false
	}
	for _, ch := range id {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func handleBungieError(c *gin.Context, err error) {
	if errors.Is(err, collections.ErrManifestNotReady) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "The item database is still downloading — try again in a moment.",
			"code":  "MANIFEST_NOT_READY",
		})
		return
	}
	var bungieErr *bungie.BungieError
	if errors.As(err, &bungieErr) {
		switch bungieErr.ErrorCode {
		case 5:
			c.JSON(http.StatusForbidden, gin.H{"error": "User has their Destiny 2 profile set to private", "code": "PRIVACY_RESTRICTION"})
		case 7:
			c.JSON(http.StatusNotFound, gin.H{"error": "Destiny 2 account not found", "code": "ACCOUNT_NOT_FOUND"})
		case 36:
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Bungie API rate limit exceeded. Please try again later.", "code": "RATE_LIMITED", "retryAfter": bungieErr.ThrottleSeconds})
		default:
			log.Printf("Bungie API error: %v", bungieErr)
			c.JSON(http.StatusBadGateway, gin.H{"error": "Error communicating with Bungie API", "code": "BUNGIE_ERROR"})
		}
		return
	}
	log.Printf("Internal error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error", "code": "INTERNAL_ERROR"})
}

// availableNowOverlay returns the subset of the live-vendor availability map whose
// items appear in the collection (keyed by item-hash string), so only tracked
// collectibles get an availability stamp. Iterates the small vendor set, not the
// ~12k-item collection.
func availableNowOverlay(items map[string]collections.DestinyItem, live map[uint32]string) map[string]string {
	out := make(map[string]string)
	for itemHash, vendor := range live {
		hs := strconv.FormatUint(uint64(itemHash), 10)
		if _, ok := items[hs]; ok {
			out[hs] = vendor
		}
	}
	return out
}
