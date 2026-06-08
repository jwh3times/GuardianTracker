package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"guardian-tracker/bungie-service/services/auth"
	"guardian-tracker/bungie-service/services/characters"

	"github.com/gin-gonic/gin"
)

// CharactersHandler handles character-related endpoints
type CharactersHandler struct {
	charactersService *characters.Service
	authClient        *auth.Client
}

// NewCharactersHandler creates a new characters handler
func NewCharactersHandler(svc *characters.Service, authClient *auth.Client) *CharactersHandler {
	return &CharactersHandler{
		charactersService: svc,
		authClient:        authClient,
	}
}

// GetCharacters handles GET /api/characters/:membershipType/:membershipId
func (h *CharactersHandler) GetCharacters(c *gin.Context) {
	membershipTypeStr := c.Param("membershipType")
	membershipID := c.Param("membershipId")

	membershipType, err := strconv.Atoi(membershipTypeStr)
	if err != nil || !isValidMembershipType(membershipType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid membership type"})
		return
	}

	if !isValidMembershipID(membershipID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid membership ID"})
		return
	}

	// Extract and validate the app JWT
	appToken := auth.ExtractBearerToken(c.GetHeader("Authorization"))
	if appToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authorization required",
			"code":  "AUTH_REQUIRED",
		})
		return
	}

	claims, err := h.authClient.ValidateAppJWT(appToken)
	if err != nil {
		log.Printf("JWT validation failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid or expired token",
			"code":  "INVALID_TOKEN",
		})
		return
	}

	// Verify the user is requesting their own data
	if claims.MembershipID != membershipID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You can only access your own character data",
			"code":  "FORBIDDEN",
		})
		return
	}

	// Fetch the Bungie OAuth token from auth-service
	bungieToken, err := h.authClient.GetBungieToken(membershipID)
	if err != nil {
		log.Printf("Failed to get Bungie token for user %s: %v", membershipID, err)
		if strings.Contains(err.Error(), "re-authenticate") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Your Bungie session has expired. Please log in again.",
				"code":  "BUNGIE_TOKEN_EXPIRED",
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Unable to retrieve Bungie authorization. Please try again.",
				"code":  "AUTH_SERVICE_ERROR",
			})
		}
		return
	}

	result, err := h.charactersService.GetCharacters(c.Request.Context(), membershipType, membershipID, bungieToken)
	if err != nil {
		handleBungieError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
