package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/collections"

	"github.com/gin-gonic/gin"
)

// CollectionsHandler handles collection-related endpoints.
type CollectionsHandler struct {
	collectionsService *collections.Service
	tokenStore         *auth.TokenStore
}

func NewCollectionsHandler(svc *collections.Service, ts *auth.TokenStore) *CollectionsHandler {
	return &CollectionsHandler{collectionsService: svc, tokenStore: ts}
}

// GetCollections handles GET /api/collections/:membershipType/:membershipId
// Requires jwtHelper.Middleware() on the route (validates JWT, enforces access token type).
func (h *CollectionsHandler) GetCollections(c *gin.Context) {
	membershipType, membershipID, ok := parseMembershipParams(c)
	if !ok {
		return
	}

	if !h.ownershipCheck(c, membershipID) {
		return
	}

	bungieToken, ok := h.getBungieToken(c, membershipID)
	if !ok {
		return
	}

	result, err := h.collectionsService.GetUserCollections(c.Request.Context(), membershipType, membershipID, bungieToken)
	if err != nil {
		handleBungieError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// RefreshCollections handles POST /api/collections/:membershipType/:membershipId/refresh
func (h *CollectionsHandler) RefreshCollections(c *gin.Context) {
	membershipType, membershipID, ok := parseMembershipParams(c)
	if !ok {
		return
	}

	if !h.ownershipCheck(c, membershipID) {
		return
	}

	h.collectionsService.InvalidateCache(membershipType, membershipID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache invalidated. Next request will fetch fresh data.",
	})
}

// ownershipCheck confirms the authenticated user (set by jwtHelper.Middleware()) owns this resource.
func (h *CollectionsHandler) ownershipCheck(c *gin.Context, membershipID string) bool {
	if c.GetString("membership_id") != membershipID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only access your own data", "code": "FORBIDDEN"})
		return false
	}
	return true
}

// getBungieToken retrieves a valid Bungie OAuth token from the in-process token store.
func (h *CollectionsHandler) getBungieToken(c *gin.Context, membershipID string) (string, bool) {
	token, err := h.tokenStore.GetValidToken(membershipID)
	if err != nil {
		log.Printf("Failed to get Bungie token for user %s: %v", membershipID, err)
		if strings.Contains(err.Error(), "re-authentication required") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Your Bungie session has expired. Please log in again.",
				"code":  "BUNGIE_TOKEN_EXPIRED",
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Unable to retrieve Bungie authorization. Please try again.",
				"code":  "TOKEN_ERROR",
			})
		}
		return "", false
	}
	return token, true
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
	for _, v := range []int{1, 2, 3, 4, 5, 6, 10, 254} {
		if t == v {
			return true
		}
	}
	return false
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
