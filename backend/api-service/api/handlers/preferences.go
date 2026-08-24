package handlers

import (
	"errors"
	"net/http"
	"time"

	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/preferences"

	"github.com/gin-gonic/gin"
)

// PreferencesHandler adapts the Preferences service to HTTP.
type PreferencesHandler struct {
	service *preferences.Service
}

// NewPreferencesHandler constructs the dedicated Preferences HTTP adapter.
func NewPreferencesHandler(service *preferences.Service) *PreferencesHandler {
	return &PreferencesHandler{service: service}
}

type preferencesValuesResponse struct {
	CardStyle   preferences.CardStyle `json:"cardStyle"`
	Personalize bool                  `json:"personalize"`
	OnboardedAt *time.Time            `json:"onboardedAt"`
}

type preferencesReadResponse struct {
	preferencesValuesResponse
	Persisted bool `json:"persisted"`
}

// GetPreferences handles GET /api/preferences.
func (h *PreferencesHandler) GetPreferences(c *gin.Context) {
	result, err := h.service.Get(c.Request.Context(), c.GetString("membership_id"))
	if err != nil {
		ctx := handlerContext(c)
		observability.Logger(ctx).ErrorContext(ctx, "preference listing failed", observability.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error", "code": "INTERNAL_ERROR"})
		return
	}
	c.JSON(http.StatusOK, preferencesReadResponse{
		preferencesValuesResponse: toPreferencesResponse(result.Values),
		Persisted:                 result.Persisted,
	})
}

// UpdatePreferences handles PUT /api/preferences.
func (h *PreferencesHandler) UpdatePreferences(c *gin.Context) {
	var body struct {
		CardStyle          *preferences.CardStyle `json:"cardStyle"`
		Personalize        *bool                  `json:"personalize"`
		OnboardingComplete *bool                  `json:"onboardingComplete"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	// Preserve the existing wire contract: an explicit empty cardStyle is an
	// omitted field, not a request to store an invalid style.
	if body.CardStyle != nil && *body.CardStyle == "" {
		body.CardStyle = nil
	}
	values, err := h.service.Apply(c.Request.Context(), c.GetString("membership_id"), preferences.Patch{
		CardStyle:          body.CardStyle,
		Personalize:        body.Personalize,
		OnboardingComplete: body.OnboardingComplete,
	})
	if err != nil {
		handlePreferencesError(c, err, "preference update failed")
		return
	}
	c.JSON(http.StatusOK, toPreferencesResponse(values))
}

func handlePreferencesError(c *gin.Context, err error, logMsg string) {
	switch {
	case errors.Is(err, preferences.ErrInvalidCardStyle), errors.Is(err, preferences.ErrOnboardingReset):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, preferences.ErrUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "This feature needs the user data database, which isn't configured on this server.",
			"code":  "DB_UNAVAILABLE",
		})
	default:
		ctx := handlerContext(c)
		observability.Logger(ctx).ErrorContext(ctx, logMsg, observability.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
			"code":  "INTERNAL_ERROR",
		})
	}
}

func toPreferencesResponse(values preferences.Values) preferencesValuesResponse {
	return preferencesValuesResponse{
		CardStyle:   values.CardStyle,
		Personalize: values.Personalize,
		OnboardedAt: values.OnboardedAt,
	}
}
