package handlers

import (
	"context"
	"log"
	"net/http"

	"guardian-tracker/api-service/services/weekly"

	"github.com/gin-gonic/gin"
)

// WeeklyHandler handles GET /api/weekly/recommendations
type WeeklyHandler struct {
	service    weeklyService
	tokenStore weeklyTokenStore
}

type weeklyService interface {
	GetWeekly(ctx context.Context, membershipType int, membershipID, bungieToken, characterID string) (*weekly.Weekly, error)
}

type weeklyTokenStore interface {
	GetValidToken(membershipID string) (string, error)
}

// NewWeeklyHandler creates a new WeeklyHandler.
func NewWeeklyHandler(svc weeklyService, ts weeklyTokenStore) *WeeklyHandler {
	return &WeeklyHandler{service: svc, tokenStore: ts}
}

// GetWeekly handles GET /api/weekly/recommendations
func (h *WeeklyHandler) GetWeekly(c *gin.Context) {
	membershipID := c.GetString("membership_id")
	membershipType := c.GetInt("membership_type")
	characterID := c.Query("characterId")

	bungieToken, err := h.tokenStore.GetValidToken(membershipID)
	if err != nil {
		log.Printf("weekly: GetValidToken for %s: %v", membershipID, err)
		// Don't fail — weekly content is partly public; serve without per-user flags
		bungieToken = ""
	}

	result, err := h.service.GetWeekly(c.Request.Context(), membershipType, membershipID, bungieToken, characterID)
	if err != nil {
		log.Printf("weekly: GetWeekly for %s: %v", membershipID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch weekly data"})
		return
	}
	c.JSON(http.StatusOK, result)
}
