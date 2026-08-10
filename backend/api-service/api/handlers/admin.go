package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/observability"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// adminUserStore is the admin slice of db.UserStore.
type adminUserStore interface {
	ListUsers(ctx context.Context, q string, limit int) ([]db.AdminUser, error)
	SetRoleByID(ctx context.Context, actorMembershipID string, targetUserID int64, newRole int16) (*db.RoleChange, error)
}

// adminFlagStore is the admin slice of db.FlagStore.
type adminFlagStore interface {
	List(ctx context.Context) ([]db.FeatureFlag, error)
	Update(ctx context.Context, key string, enabled *bool, minTier *int16, actorUserID *int64, actorMembershipID string) (*db.FeatureFlag, error)
}

// AdminHandler serves the admin console: user roster + role management and the
// feature-flag config. All routes are mounted behind Authz.RequireAdmin.
type AdminHandler struct {
	users adminUserStore
	flags adminFlagStore
	cache cache.Cache
}

func NewAdminHandler(users adminUserStore, flags adminFlagStore, c cache.Cache) *AdminHandler {
	return &AdminHandler{users: users, flags: flags, cache: c}
}

type adminUserResponse struct {
	ID           string `json:"id"`
	DisplayName  string `json:"displayName"`
	MembershipID string `json:"membershipId"`
	Platform     string `json:"platform"`
	Role         string `json:"role"`
	LastActive   string `json:"lastActive"`
}

// ListUsers handles GET /api/admin/users?q=
func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.users.ListUsers(c.Request.Context(), c.Query("q"), 200)
	if err != nil {
		HandleStoreError(c, err, "admin user listing failed")
		return
	}
	out := make([]adminUserResponse, len(users))
	for i, u := range users {
		out[i] = adminUserResponse{
			ID:           strconv.FormatInt(u.ID, 10),
			DisplayName:  u.DisplayName,
			MembershipID: u.MembershipID,
			Platform:     auth.GetPlatformName(int(u.MembershipType)),
			Role:         auth.RoleName(int(u.Role)),
			LastActive:   u.LastLoginAt.UTC().Format(time.RFC3339),
		}
	}
	c.JSON(http.StatusOK, out)
}

// SetUserRole handles PUT /api/admin/users/:id/role — any role incl. admin, with
// last-admin protection. Bumps the target's token_version and evicts its
// revocation cache entry so stale sessions re-sync; writes an audit_log row.
func (h *AdminHandler) SetUserRole(c *gin.Context) {
	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role is required"})
		return
	}
	newRole, ok := auth.ParseRole(body.Role)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be standard, beta, alpha, or admin"})
		return
	}
	actor := c.GetString("membership_id")
	change, err := h.users.SetRoleByID(c.Request.Context(), actor, targetID, int16(newRole))
	if err != nil {
		switch {
		case errors.Is(err, db.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		case errors.Is(err, db.ErrLastAdmin):
			c.JSON(http.StatusConflict, gin.H{"error": "Can't remove the last admin. Promote another admin first.", "code": "LAST_ADMIN"})
		default:
			observability.Logger(c.Request.Context()).ErrorContext(c.Request.Context(), "admin role update failed",
				observability.IntID("user", targetID), observability.Err(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}
	if h.cache != nil {
		h.cache.Delete("tver:" + change.TargetMembershipID)
	}
	c.JSON(http.StatusOK, gin.H{
		"id":           strconv.FormatInt(change.TargetUserID, 10),
		"role":         auth.RoleName(int(change.NewRole)),
		"previousRole": auth.RoleName(int(change.OldRole)),
	})
}

type adminFlagResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	MinTier     string `json:"minTier"`
	Enabled     bool   `json:"enabled"`
	UpdatedAt   string `json:"updatedAt"`
}

func toAdminFlag(f *db.FeatureFlag) adminFlagResponse {
	return adminFlagResponse{
		Key:         f.Key,
		Name:        f.Name,
		Description: f.Description,
		Category:    f.Category,
		MinTier:     auth.RoleName(int(f.MinTier)),
		Enabled:     f.Enabled,
		UpdatedAt:   f.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// ListFlags handles GET /api/admin/flags
func (h *AdminHandler) ListFlags(c *gin.Context) {
	flags, err := h.flags.List(c.Request.Context())
	if err != nil {
		HandleStoreError(c, err, "admin feature flag listing failed")
		return
	}
	out := make([]adminFlagResponse, len(flags))
	for i := range flags {
		out[i] = toAdminFlag(&flags[i])
	}
	c.JSON(http.StatusOK, out)
}

// UpdateFlag handles PUT /api/admin/flags/:key — toggle enabled and/or set minTier.
func (h *AdminHandler) UpdateFlag(c *gin.Context) {
	key := c.Param("key")
	var body struct {
		Enabled *bool   `json:"enabled"`
		MinTier *string `json:"minTier"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	var minTier *int16
	if body.MinTier != nil {
		t, ok := auth.ParseRole(*body.MinTier)
		if !ok || t > auth.RoleAlpha {
			c.JSON(http.StatusBadRequest, gin.H{"error": "minTier must be standard, beta, or alpha"})
			return
		}
		mt := int16(t)
		minTier = &mt
	}
	if body.Enabled == nil && minTier == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nothing to update"})
		return
	}
	actorMID := c.GetString("membership_id")
	flag, err := h.flags.Update(c.Request.Context(), key, body.Enabled, minTier, actorUserIDFromContext(c), actorMID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown flag"})
			return
		}
		HandleStoreError(c, err, "admin feature flag update failed")
		return
	}
	// Evict the cached flag list so GET /api/flags reflects the change immediately.
	if h.cache != nil {
		h.cache.Delete(flagsCacheKey)
	}
	c.JSON(http.StatusOK, toAdminFlag(flag))
}

// actorUserIDFromContext returns the numeric user id when the middleware stored
// one, else nil. The audit actor falls back to the denormalized membership id.
func actorUserIDFromContext(c *gin.Context) *int64 {
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(int64); ok {
			return &id
		}
	}
	return nil
}
