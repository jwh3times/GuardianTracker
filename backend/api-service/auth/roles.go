package auth

import (
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
				"error": "This feature requires the account database, which is not configured.",
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
