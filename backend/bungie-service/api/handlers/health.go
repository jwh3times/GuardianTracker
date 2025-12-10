package handlers

import (
	"net/http"
	"time"

	"guardian-tracker/bungie-service/services/bungie"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health-related endpoints
type HealthHandler struct {
	manifestService *bungie.ManifestService
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(manifestService *bungie.ManifestService) *HealthHandler {
	return &HealthHandler{
		manifestService: manifestService,
	}
}

// Health handles GET /health
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "bungie-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready handles GET /ready (Kubernetes readiness probe)
func (h *HealthHandler) Ready(c *gin.Context) {
	if !h.manifestService.IsReady() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"reason":  "Manifest database not ready",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ready": true,
	})
}

// ManifestStatus handles GET /api/manifest/status
func (h *HealthHandler) ManifestStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ready":   h.manifestService.IsReady(),
		"version": h.manifestService.GetCurrentVersion(),
		"dbPath":  h.manifestService.GetDBPath(),
	})
}
