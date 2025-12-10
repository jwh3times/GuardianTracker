package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"guardian-tracker/bungie-service/services/bungie"
	"guardian-tracker/bungie-service/services/collections"

	"github.com/gin-gonic/gin"
)

// CollectionsHandler handles collection-related endpoints
type CollectionsHandler struct {
	collectionsService *collections.Service
}

// NewCollectionsHandler creates a new collections handler
func NewCollectionsHandler(svc *collections.Service) *CollectionsHandler {
	return &CollectionsHandler{
		collectionsService: svc,
	}
}

// GetCollections handles GET /api/collections/:membershipType/:membershipId
func (h *CollectionsHandler) GetCollections(c *gin.Context) {
	// Parse path parameters
	membershipTypeStr := c.Param("membershipType")
	membershipID := c.Param("membershipId")

	membershipType, err := strconv.Atoi(membershipTypeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid membership type",
		})
		return
	}

	// Validate membership type
	if !isValidMembershipType(membershipType) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid membership type",
		})
		return
	}

	// Validate membership ID
	if !isValidMembershipID(membershipID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid membership ID",
		})
		return
	}

	// Extract access token from Authorization header
	accessToken := extractBearerToken(c.GetHeader("Authorization"))
	if accessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authorization required",
		})
		return
	}

	// Get collections
	result, err := h.collectionsService.GetUserCollections(c.Request.Context(), membershipType, membershipID, accessToken)
	if err != nil {
		handleBungieError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// RefreshCollections handles POST /api/collections/:membershipType/:membershipId/refresh
func (h *CollectionsHandler) RefreshCollections(c *gin.Context) {
	membershipTypeStr := c.Param("membershipType")
	membershipID := c.Param("membershipId")

	membershipType, err := strconv.Atoi(membershipTypeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid membership type",
		})
		return
	}

	// Invalidate cache
	h.collectionsService.InvalidateCache(membershipType, membershipID)

	// Return success - next GetCollections will fetch fresh data
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache invalidated. Next request will fetch fresh data.",
	})
}

// Helper functions

func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

func isValidMembershipType(t int) bool {
	validTypes := []int{1, 2, 3, 4, 5, 6, 10, 254}
	for _, valid := range validTypes {
		if t == valid {
			return true
		}
	}
	return false
}

func isValidMembershipID(id string) bool {
	if len(id) < 10 || len(id) > 25 {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func handleBungieError(c *gin.Context, err error) {
	var bungieErr *bungie.BungieError
	if errors.As(err, &bungieErr) {
		switch bungieErr.ErrorCode {
		case 5: // DestinyPrivacyRestriction
			c.JSON(http.StatusForbidden, gin.H{
				"error": "User has their Destiny 2 profile set to private",
				"code":  "PRIVACY_RESTRICTION",
			})
			return
		case 7: // DestinyAccountNotFound
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Destiny 2 account not found",
				"code":  "ACCOUNT_NOT_FOUND",
			})
			return
		case 36: // ThrottleLimitExceeded
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Bungie API rate limit exceeded. Please try again later.",
				"code":        "RATE_LIMITED",
				"retryAfter":  bungieErr.ThrottleSeconds,
			})
			return
		case 1601: // DestinyUnexpectedError
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "Bungie API returned an unexpected error. Please try again.",
				"code":  "BUNGIE_ERROR",
			})
			return
		default:
			log.Printf("Bungie API error: %v", bungieErr)
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "Error communicating with Bungie API",
				"code":  "BUNGIE_ERROR",
			})
			return
		}
	}

	log.Printf("Internal error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "Internal server error",
		"code":  "INTERNAL_ERROR",
	})
}
