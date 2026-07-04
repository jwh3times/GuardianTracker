package handlers

import (
	"context"
	"net/http"
	"time"

	"guardian-tracker/api-service/services/bungie"

	"github.com/gin-gonic/gin"
)

// DBPinger is the health probe view of the DB pool (*pgxpool.Pool satisfies it
// natively — no adapter needed). nil = degraded mode (no DB configured); the
// readiness check skips the DB probe in that case.
type DBPinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler handles health and readiness endpoints.
type HealthHandler struct {
	manifestService *bungie.ManifestService
	db              DBPinger
}

func NewHealthHandler(manifestService *bungie.ManifestService, db DBPinger) *HealthHandler {
	return &HealthHandler{manifestService: manifestService, db: db}
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
	if h.db != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := h.db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"ready":  false,
				"reason": "Database unreachable",
			})
			return
		}
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
