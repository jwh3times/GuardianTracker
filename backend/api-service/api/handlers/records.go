package handlers

import (
	"net/http"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/services/records"

	"github.com/gin-gonic/gin"
)

// RecordsHandler handles catalysts, crafting, and seals endpoints.
type RecordsHandler struct {
	svc        *records.Service
	tokenStore *auth.TokenStore
}

// NewRecordsHandler creates a new RecordsHandler.
func NewRecordsHandler(svc *records.Service, ts *auth.TokenStore) *RecordsHandler {
	return &RecordsHandler{svc: svc, tokenStore: ts}
}

// GetCatalysts handles GET /api/catalysts/:membershipType/:membershipId
func (h *RecordsHandler) GetCatalysts(c *gin.Context) {
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
	result, fetchedAt, err := h.svc.GetCatalysts(c.Request.Context(), membershipType, membershipID, bungieToken)
	if err != nil {
		handleBungieError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": result, "fetchedAt": fetchedAt})
}

// GetCrafting handles GET /api/crafting/:membershipType/:membershipId
func (h *RecordsHandler) GetCrafting(c *gin.Context) {
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
	result, fetchedAt, err := h.svc.GetCrafting(c.Request.Context(), membershipType, membershipID, bungieToken)
	if err != nil {
		handleBungieError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": result, "fetchedAt": fetchedAt})
}

// GetSeals handles GET /api/seals/:membershipType/:membershipId
func (h *RecordsHandler) GetSeals(c *gin.Context) {
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
	result, fetchedAt, err := h.svc.GetSeals(c.Request.Context(), membershipType, membershipID, bungieToken)
	if err != nil {
		handleBungieError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": result, "fetchedAt": fetchedAt})
}
