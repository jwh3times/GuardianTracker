package handlers

import (
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

	if !ownershipCheck(c, membershipID) {
		return
	}

	bungieToken, ok := getBungieToken(c, membershipID, h.tokenStore)
	if !ok {
		return
	}

	result, err := h.charactersService.GetCharacters(c.Request.Context(), membershipType, membershipID, bungieToken)
	if err != nil {
		handleBungieError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
