package handlers

import (
	"context"
	"errors"
	"net/http"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/observability"

	"github.com/gin-gonic/gin"
)

// roleSelfStore is the self opt-in slice of db.UserStore.
type roleSelfStore interface {
	SetRole(ctx context.Context, membershipID string, role int16) error
}

// flagLister is the read slice of db.FlagStore.
type flagLister interface {
	List(ctx context.Context) ([]db.FeatureFlag, error)
}

// UserHandler serves Guardian Tracker user role opt-in and resolved feature-flag state.
type UserHandler struct {
	users    roleSelfStore
	resolver *FlagResolver
	cache    cache.Cache // for evicting tver: entries
	audit    AuditLogger // nil = best-effort (no-op)
}

func NewUserHandler(users roleSelfStore, flags flagLister, c cache.Cache, audit AuditLogger) *UserHandler {
	return &UserHandler{users: users, resolver: NewFlagResolver(flags, c), cache: c, audit: audit}
}

// SetRole handles PUT /api/account/role — self-service tier opt-in.
// Allows standard/beta/alpha only; admin is never self-assignable, and an admin
// caller is refused (so they can't accidentally demote themselves via opt-in).
func (h *UserHandler) SetRole(c *gin.Context) {
	var body struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role is required"})
		return
	}
	role, ok := auth.ParseRole(body.Role)
	if !ok || role == auth.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only switch between standard, beta, and alpha.", "code": "ROLE_NOT_ALLOWED"})
		return
	}
	if c.GetInt("role") == auth.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admins manage roles in the console. Switching tiers here would drop your admin access.", "code": "ADMIN_OPT_IN"})
		return
	}
	membershipID := c.GetString("membership_id")
	if err := h.users.SetRole(c.Request.Context(), membershipID, int16(role)); err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		HandleStoreError(c, err, "self-service role update failed")
		return
	}
	// Evict the revocation cache entry so the new role is read from the DB on the
	// next request — no token_version bump, so the session is preserved.
	if h.cache != nil {
		h.cache.Delete("tver:" + membershipID)
	}
	if h.audit != nil {
		oldRole := c.GetInt("role")
		_ = h.audit.Log(c.Request.Context(), db.AuditEvent{
			EventType:         "role.optin",
			ActorMembershipID: membershipID,
			IP:                c.ClientIP(),
			UserAgent:         c.Request.UserAgent(),
			Details:           map[string]any{"oldRole": oldRole, "newRole": int(role)},
		})
	}
	c.JSON(http.StatusOK, gin.H{"role": auth.RoleName(role)})
}

// resolvedFlag is the per-caller flag state, a direct port of store.jsx flagState.
type resolvedFlag struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	Category   string `json:"category"`
	MinTier    string `json:"minTier"`
	Enabled    bool   `json:"enabled"`
	Accessible bool   `json:"accessible"`
	Locked     bool   `json:"locked"`
}

// GetFlags handles GET /api/flags — resolved feature-flag state for the caller.
// Fail-open: a DB hiccup or degraded mode returns an empty list so the frontend
// treats every shipped feature as accessible (nothing hidden). Actual access to
// gated APIs is enforced server-side by auth.RequireFlag, not by this UI hint (TODO 13.4, done).
func (h *UserHandler) GetFlags(c *gin.Context) {
	role := c.GetInt("role")
	resp := gin.H{"role": auth.RoleName(role), "flags": []resolvedFlag{}}
	flags, err := h.resolver.List(c.Request.Context())
	if err != nil {
		observability.Logger(c.Request.Context()).WarnContext(c.Request.Context(), "feature flag resolution failed open", observability.Err(err))
		c.JSON(http.StatusOK, resp) // fail open — don't hide features on a DB hiccup
		return
	}
	out := make([]resolvedFlag, 0, len(flags))
	for _, f := range flags {
		accessible := f.Enabled && role >= int(f.MinTier)
		out = append(out, resolvedFlag{
			Key:        f.Key,
			Name:       f.Name,
			Desc:       f.Description,
			Category:   f.Category,
			MinTier:    auth.RoleName(int(f.MinTier)),
			Enabled:    f.Enabled,
			Accessible: accessible,
			Locked:     f.Enabled && !accessible,
		})
	}
	resp["flags"] = out
	c.JSON(http.StatusOK, resp)
}
