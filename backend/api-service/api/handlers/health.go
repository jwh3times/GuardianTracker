package handlers

import (
	"net/http"
	"time"

	"guardian-tracker/api-service/services/bungie"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health and readiness endpoints.
type HealthHandler struct {
	manifestService *bungie.ManifestService
}

func NewHealthHandler(manifestService *bungie.ManifestService) *HealthHandler {
	return &HealthHandler{manifestService: manifestService}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "api-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	if !h.manifestService.IsReady() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":  false,
			"reason": "Manifest database not ready",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ready": true})
}

func (h *HealthHandler) ManifestStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ready":   h.manifestService.IsReady(),
		"version": h.manifestService.GetCurrentVersion(),
		"dbPath":  h.manifestService.GetDBPath(),
	})
}
