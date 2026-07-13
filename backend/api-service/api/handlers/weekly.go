package handlers

import (
	"context"
	"net/http"

	"guardian-tracker/api-service/observability"
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
		observability.Logger(c.Request.Context()).WarnContext(c.Request.Context(), "weekly personalization token unavailable",
			observability.ID("membership", membershipID), observability.Err(err))
		// Don't fail — weekly content is partly public; serve without per-user flags
		bungieToken = ""
	}

	result, err := h.service.GetWeekly(c.Request.Context(), membershipType, membershipID, bungieToken, characterID)
	if err != nil {
		logger := observability.Logger(c.Request.Context())
		if characterID != "" {
			logger = logger.With(observability.ID("character", characterID))
		}
		logger.ErrorContext(c.Request.Context(), "weekly recommendations failed",
			observability.ID("membership", membershipID), observability.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch weekly data"})
		return
	}
	c.JSON(http.StatusOK, result)
}
