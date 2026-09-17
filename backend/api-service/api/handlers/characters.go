package handlers

import (
	"errors"
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

	if !ownershipCheck(c, membershipType, membershipID) {
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

// GetEquipment handles
// GET /api/characters/:membershipType/:membershipId/:characterId/equipment.
func (h *CharactersHandler) GetEquipment(c *gin.Context) {
	membershipType, membershipID, ok := parseMembershipParams(c)
	if !ok {
		return
	}
	if !ownershipCheck(c, membershipType, membershipID) {
		return
	}
	characterID := c.Param("characterId")
	if !isValidMembershipID(characterID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid character ID"})
		return
	}

	bungieToken, ok := getBungieToken(c, membershipID, h.tokenStore)
	if !ok {
		return
	}

	result, err := h.charactersService.GetEquipment(c.Request.Context(), membershipType, membershipID, characterID, bungieToken)
	if errors.Is(err, characters.ErrCharacterNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guardian not found", "code": "CHARACTER_NOT_FOUND"})
		return
	}
	if err != nil {
		handleBungieError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetActivityHistory handles
// GET /api/characters/:membershipType/:membershipId/:characterId/activity-history.
func (h *CharactersHandler) GetActivityHistory(c *gin.Context) {
	membershipType, membershipID, ok := parseMembershipParams(c)
	if !ok {
		return
	}
	if !ownershipCheck(c, membershipType, membershipID) {
		return
	}
	characterID := c.Param("characterId")
	if !isValidMembershipID(characterID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid character ID"})
		return
	}

	bungieToken, ok := getBungieToken(c, membershipID, h.tokenStore)
	if !ok {
		return
	}

	result, err := h.charactersService.GetActivityHistory(c.Request.Context(), membershipType, membershipID, characterID, bungieToken)
	if errors.Is(err, characters.ErrCharacterNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guardian not found", "code": "CHARACTER_NOT_FOUND"})
		return
	}
	if err != nil {
		handleBungieError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetCurrentActivity handles
// GET /api/characters/:membershipType/:membershipId/:characterId/current-activity.
// A Bungie failure is an error response; the frontend presents it as
// unavailable without affecting recent history.
func (h *CharactersHandler) GetCurrentActivity(c *gin.Context) {
	membershipType, membershipID, ok := parseMembershipParams(c)
	if !ok {
		return
	}
	if !ownershipCheck(c, membershipType, membershipID) {
		return
	}
	characterID := c.Param("characterId")
	if !isValidMembershipID(characterID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid character ID"})
		return
	}

	bungieToken, ok := getBungieToken(c, membershipID, h.tokenStore)
	if !ok {
		return
	}

	result, err := h.charactersService.GetCurrentActivity(c.Request.Context(), membershipType, membershipID, characterID, bungieToken)
	if errors.Is(err, characters.ErrCharacterNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guardian not found", "code": "CHARACTER_NOT_FOUND"})
		return
	}
	if err != nil {
		handleBungieError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
