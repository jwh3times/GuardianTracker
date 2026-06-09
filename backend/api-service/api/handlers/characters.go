package handlers

import (
	"log"
	"net/http"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/services/characters"

	"github.com/gin-gonic/gin"
)

// CharactersHandler handles character-related endpoints.
type CharactersHandler struct {
	charactersService *characters.Service
	tokenStore        *auth.TokenStore
}

func NewCharactersHandler(svc *characters.Service, ts *auth.TokenStore) *CharactersHandler {
	return &CharactersHandler{charactersService: svc, tokenStore: ts}
}

// GetCharacters handles GET /api/characters/:membershipType/:membershipId
// Requires jwtHelper.Middleware() on the route (sets membership_id in context).
func (h *CharactersHandler) GetCharacters(c *gin.Context) {
	membershipType, membershipID, ok := parseMembershipParams(c)
	if !ok {
		return
	}

	// Ownership check — middleware already validated the JWT and token type.
	if c.GetString("membership_id") != membershipID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only access your own character data", "code": "FORBIDDEN"})
		return
	}

	bungieToken, err := h.tokenStore.GetValidToken(membershipID)
	if err != nil {
		log.Printf("Failed to get Bungie token for user %s: %v", membershipID, err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Your Bungie session has expired. Please log in again.",
			"code":  "BUNGIE_TOKEN_EXPIRED",
		})
		return
	}

	result, err := h.charactersService.GetCharacters(c.Request.Context(), membershipType, membershipID, bungieToken)
	if err != nil {
		handleBungieError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
