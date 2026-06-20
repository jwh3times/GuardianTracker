package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
)

// flagsCacheKey caches the full flag list so GET /api/flags is one DB read per
// minute, not per request. Evicted on any admin flag change.
const flagsCacheKey = "flags:all"
const flagsCacheTTL = 60 * time.Second

// roleSelfStore is the self opt-in slice of db.UserStore.
type roleSelfStore interface {
	SetRole(ctx context.Context, membershipID string, role int16) error
}

// flagLister is the read slice of db.FlagStore.
type flagLister interface {
	List(ctx context.Context) ([]db.FeatureFlag, error)
}

// AccountHandler serves the per-user role opt-in and resolved feature-flag state.
type AccountHandler struct {
	users roleSelfStore // nil = degraded mode
	flags flagLister    // nil = degraded mode
	cache cache.Cache   // for evicting tver: entries + caching flag rows
	audit AuditLogger   // nil = best-effort (no-op)
}

func NewAccountHandler(users roleSelfStore, flags flagLister, c cache.Cache, audit AuditLogger) *AccountHandler {
	return &AccountHandler{users: users, flags: flags, cache: c, audit: audit}
}

// SetRole handles PUT /api/account/role — self-service tier opt-in.
// Allows standard/beta/alpha only; admin is never self-assignable, and an admin
// caller is refused (so they can't accidentally demote themselves via opt-in).
func (h *AccountHandler) SetRole(c *gin.Context) {
	if h.users == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Account roles require the database, which is not configured.", "code": "DB_UNAVAILABLE"})
		return
	}
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
		log.Printf("SetRole(%s): %v", membershipID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
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
// All flags are returned with per-caller accessible/locked so the frontend can
// distinguish "disabled" (hide everywhere) from "locked" (upsell) from an unknown
// key (fail-open: show). Actual access to gated APIs is enforced server-side with
// RequireTier, not by this UI hint (TODO 13.4).
func (h *AccountHandler) GetFlags(c *gin.Context) {
	role := c.GetInt("role")
	resp := gin.H{"role": auth.RoleName(role), "flags": []resolvedFlag{}}
	if h.flags == nil {
		// Degraded mode — no flag table. An empty list makes the frontend treat
		// every shipped feature as accessible (nothing hidden).
		c.JSON(http.StatusOK, resp)
		return
	}
	flags, err := h.cachedFlags(c.Request.Context())
	if err != nil {
		log.Printf("GetFlags: list: %v", err)
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

// cachedFlags returns the flag list from a 60s cache, reading the DB on a miss.
func (h *AccountHandler) cachedFlags(ctx context.Context) ([]db.FeatureFlag, error) {
	if h.cache != nil {
		if v, ok := h.cache.Get(flagsCacheKey); ok {
			if flags, ok := v.([]db.FeatureFlag); ok {
				return flags, nil
			}
		}
	}
	flags, err := h.flags.List(ctx)
	if err != nil {
		return nil, err
	}
	if h.cache != nil {
		h.cache.Set(flagsCacheKey, flags, flagsCacheTTL)
	}
	return flags, nil
}
