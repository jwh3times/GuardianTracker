package auth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Role tiers, ordered. standard < beta < alpha < admin. The integer values are
// what persist in users.role (see db migration 0002). Authorization always reads
// the role from the DB-backed revocation cache, never from client-supplied data.
const (
	RoleStandard = 0
	RoleBeta     = 1
	RoleAlpha    = 2
	RoleAdmin    = 3
)

var roleNames = map[int]string{
	RoleStandard: "standard",
	RoleBeta:     "beta",
	RoleAlpha:    "alpha",
	RoleAdmin:    "admin",
}

var roleByName = map[string]int{
	"standard": RoleStandard,
	"beta":     RoleBeta,
	"alpha":    RoleAlpha,
	"admin":    RoleAdmin,
}

// RoleName maps a role tier to its string label ("standard" for unknown values).
func RoleName(role int) string {
	if n, ok := roleNames[role]; ok {
		return n
	}
	return "standard"
}

// ParseRole maps a string label to its role tier, reporting whether it was valid.
func ParseRole(name string) (int, bool) {
	r, ok := roleByName[name]
	return r, ok
}

// Authz builds tier-gating middleware. enabled is false in degraded mode (no DB):
// role-gated endpoints then return 503 because roles cannot be resolved.
type Authz struct{ enabled bool }

// NewAuthz creates an Authz. Pass enabled=true only when a DB-backed user store
// is available (roles are authoritative).
func NewAuthz(enabled bool) *Authz { return &Authz{enabled: enabled} }

// RequireAdmin aborts the request unless the authenticated user is an admin.
// Must compose after JWT.Middleware, which sets "role" in the Gin context.
func (a *Authz) RequireAdmin() gin.HandlerFunc {
	return a.RequireTier(RoleAdmin)
}

// RequireTier aborts the request unless the authenticated user's role meets the
// minimum tier. In degraded mode it returns 503 (roles are unavailable).
func (a *Authz) RequireTier(minTier int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.enabled {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				// Same wording as handlers.HandleStoreError: one DB_UNAVAILABLE
				// shape whether the check fails in middleware or in a handler.
				"error": "This feature needs the account database, which isn't configured on this server.",
				"code":  "DB_UNAVAILABLE",
			})
			return
		}
		if c.GetInt("role") < minTier {
			code := "TIER_LOCKED"
			msg := "You don't have access to this feature yet."
			if minTier >= RoleAdmin {
				code = "FORBIDDEN"
				msg = "Admin access required."
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": msg, "code": code})
			return
		}
		c.Next()
	}
}

// FlagResolver reports a feature flag's state for the caller. Implemented by the
// handlers package (which owns the flag store + cache); declared here with only
// primitive returns so auth does not import db.
type FlagResolver interface {
	Resolve(ctx context.Context, key string) (enabled bool, minTier int, found bool, err error)
}

// RequireFlag aborts the request unless the named feature flag is enabled and the
// caller's role meets its min tier. Unlike RequireTier it FAILS OPEN whenever the
// flag cannot be resolved (nil resolver, degraded mode, store error, or unknown
// key): feature flags are rollout/upsell controls, not security boundaries, so a
// flag-table hiccup must not 503 core pages. Admin endpoints stay hard-gated by
// RequireAdmin. Must compose after JWT.Middleware, which sets "role" in the context.
func (a *Authz) RequireFlag(flags FlagResolver, key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if flags == nil {
			c.Next()
			return
		}
		enabled, minTier, found, err := flags.Resolve(c.Request.Context(), key)
		if err != nil || !found {
			c.Next() // fail open
			return
		}
		if !enabled {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "This feature isn't available.",
				"code":  "FEATURE_DISABLED",
			})
			return
		}
		if c.GetInt("role") < minTier {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "You don't have access to this feature yet.",
				"code":  "TIER_LOCKED",
			})
			return
		}
		c.Next()
	}
}
