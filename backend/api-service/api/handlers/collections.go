package handlers

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/collections"

	"github.com/gin-gonic/gin"
)

// CollectionsHandler adapts the complete Collections capability to HTTP.
//
// It owns route binding, authentication, Bungie-token resolution,
// error-to-status mapping, and serialization — and nothing else. Choosing a
// projection, joining live availability, and deciding what a membership refresh
// touches all belong to collections.Service (ADR 0018), which is why this type
// no longer holds Characters, Records, or Weekly.
type CollectionsHandler struct {
	collections *collections.Service
	tokenStore  *auth.TokenStore
}

func NewCollectionsHandler(svc *collections.Service, ts *auth.TokenStore) *CollectionsHandler {
	return &CollectionsHandler{collections: svc, tokenStore: ts}
}

// collectionsResponse is the wire shape of a collections read.
//
// The item-derived fields are absent from a summary and present on
// `?include=all`, which is the same distinction the two service operations
// make. They are assembled from the ordered items of a Full result; everything
// else is serialized straight through.
type collectionsResponse struct {
	Tree            []collections.CollectionNode       `json:"tree"`
	Items           map[string]collections.DestinyItem `json:"items,omitempty"`
	CollectedHashes []string                           `json:"collectedHashes,omitempty"`
	AvailableNow    map[string]string                  `json:"availableNow,omitempty"`
	Summary         collections.CategorySummary        `json:"summary"`
	FetchedAt       time.Time                          `json:"fetchedAt"`
}

// GetCollections handles GET /api/collections/:membershipType/:membershipId
// Requires jwtHelper.Middleware() on the route (validates JWT, enforces access token type).
func (h *CollectionsHandler) GetCollections(c *gin.Context) {
	membershipType, membershipID, ok := parseMembershipParams(c)
	if !ok {
		return
	}

	if !ownershipCheck(c, membershipType, membershipID) {
		return
	}

	bungieToken, ok := getBungieToken(c, membershipID, h.tokenStore)
	if !ok {
		return
	}

	req := collections.MembershipRequest{
		MembershipType: membershipType,
		MembershipID:   membershipID,
		AccessToken:    bungieToken,
	}

	// The heavy item surface rides along only when asked for. These are two
	// different requests to the service, not one result trimmed afterwards, so
	// the default read never pays for items or a vendor call at all.
	if c.Query("include") != "all" {
		summary, err := h.collections.GetSummary(c.Request.Context(), req)
		if err != nil {
			handleBungieError(c, err)
			return
		}
		c.JSON(http.StatusOK, collectionsResponse{
			Tree:      summary.Tree,
			Summary:   summary.Totals,
			FetchedAt: summary.FetchedAt,
		})
		return
	}

	full, err := h.collections.GetFull(c.Request.Context(), req)
	if err != nil {
		handleBungieError(c, err)
		return
	}
	c.JSON(http.StatusOK, fullResponse(full))
}

// fullResponse spreads one complete collection across the three item-derived
// wire fields the frontend has always received.
//
// It is a transcription, not a decision: every item appears in `items`, an item
// the membership owns also contributes to `collectedHashes`, and one on sale
// also contributes to `availableNow`. Because Full.Items arrives in ascending
// item-hash order, `collectedHashes` comes out in that order too without
// sorting anything here.
func fullResponse(full collections.Full) collectionsResponse {
	resp := collectionsResponse{
		Tree:            full.Tree,
		Items:           make(map[string]collections.DestinyItem, len(full.Items)),
		CollectedHashes: make([]string, 0, len(full.Items)),
		AvailableNow:    make(map[string]string),
		Summary:         full.Totals,
		FetchedAt:       full.FetchedAt,
	}
	for _, item := range full.Items {
		resp.Items[item.Item.ItemHash] = item.Item
		if item.Collected {
			resp.CollectedHashes = append(resp.CollectedHashes, item.Item.ItemHash)
		}
		if item.AvailableFrom != "" {
			resp.AvailableNow[item.Item.ItemHash] = item.AvailableFrom
		}
	}
	return resp
}

// RefreshCollections handles POST /api/collections/:membershipType/:membershipId/refresh
func (h *CollectionsHandler) RefreshCollections(c *gin.Context) {
	membershipType, membershipID, ok := parseMembershipParams(c)
	if !ok {
		return
	}

	if !ownershipCheck(c, membershipType, membershipID) {
		return
	}

	// Which owners a membership refresh reaches is a domain question, not a
	// routing one: the route is named for Collections, but the refresh has
	// always been broader than that.
	if err := h.collections.RefreshMembership(c.Request.Context(), collections.Membership{
		MembershipType: membershipType,
		MembershipID:   membershipID,
	}); err != nil {
		handleBungieError(c, err)
		return
	}
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
			ctx := handlerContext(c)
			observability.Logger(ctx).WarnContext(ctx, "Bungie API request failed",
				"bungie_error_code", bungieErr.ErrorCode, observability.Err(bungieErr))
			c.JSON(http.StatusBadGateway, gin.H{"error": "Error communicating with Bungie API", "code": "BUNGIE_ERROR"})
		}
		return
	}
	ctx := handlerContext(c)
	observability.Logger(ctx).ErrorContext(ctx, "collections request failed", observability.Err(err))
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error", "code": "INTERNAL_ERROR"})
}
