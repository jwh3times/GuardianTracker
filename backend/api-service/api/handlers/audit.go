package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
)

// auditReadStore is the read slice of db.AuditStore.
type auditReadStore interface {
	List(ctx context.Context, f db.AuditFilter) ([]db.AuditEntry, string, error)
}

// AuditHandler serves the admin audit feed. Mounted behind RequireAdmin.
type AuditHandler struct {
	store auditReadStore // nil in degraded mode
}

func NewAuditHandler(store auditReadStore) *AuditHandler { return &AuditHandler{store: store} }

type auditPartyResponse struct {
	MembershipID string `json:"membershipId"`
	DisplayName  string `json:"displayName"`
}

type auditEntryResponse struct {
	ID        string              `json:"id"`
	EventType string              `json:"eventType"`
	Outcome   string              `json:"outcome"`
	Actor     auditPartyResponse  `json:"actor"`
	Target    *auditPartyResponse `json:"target,omitempty"`
	IP        string              `json:"ip,omitempty"`
	UserAgent string              `json:"userAgent,omitempty"`
	Details   map[string]any      `json:"details"`
	CreatedAt string              `json:"createdAt"`
}

// ListAudit handles GET /api/admin/audit (admin-gated upstream).
func (h *AuditHandler) ListAudit(c *gin.Context) {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Audit log requires the database, which is not configured.", "code": "DB_UNAVAILABLE"})
		return
	}

	f := db.AuditFilter{
		EventType: c.Query("type"),
		Actor:     c.Query("actor"),
		Target:    c.Query("target"),
		Outcome:   c.Query("outcome"),
		Cursor:    c.Query("cursor"),
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Limit = n
		}
	}
	if v := c.Query("after"); v != "" {
		if ts, err := time.Parse(time.RFC3339, v); err == nil {
			f.After = ts
		}
	}
	if v := c.Query("before"); v != "" {
		if ts, err := time.Parse(time.RFC3339, v); err == nil {
			f.Before = ts
		}
	}

	entries, nextCursor, err := h.store.List(c.Request.Context(), f)
	if err != nil {
		log.Printf("admin ListAudit: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	out := make([]auditEntryResponse, len(entries))
	for i, e := range entries {
		resp := auditEntryResponse{
			ID:        strconv.FormatInt(e.ID, 10),
			EventType: e.EventType,
			Outcome:   e.Outcome,
			Actor:     auditPartyResponse{MembershipID: e.ActorMembershipID, DisplayName: e.ActorDisplayName},
			IP:        e.IP,
			UserAgent: e.UserAgent,
			Details:   e.Details,
			CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
		}
		if resp.Details == nil {
			resp.Details = map[string]any{}
		}
		if e.TargetMembershipID != "" {
			resp.Target = &auditPartyResponse{MembershipID: e.TargetMembershipID, DisplayName: e.TargetDisplayName}
		}
		out[i] = resp
	}
	c.JSON(http.StatusOK, gin.H{"entries": out, "nextCursor": nextCursor})
}
