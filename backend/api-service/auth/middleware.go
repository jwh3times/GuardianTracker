package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Middleware returns a Gin handler that requires a valid access token and checks revocation.
// It sets user_id, membership_id, membership_type, display_name, platform, and token_version
// in the request context for downstream handlers.
func (j *JWT) Middleware(revoker *RevocationChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ExtractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}
		claims, err := j.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		if claims.TokenType != "access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Access token required"})
			return
		}
		// Resolve role from the DB-backed cache (authoritative), not from the JWT.
		// This also enforces account-wide revocation (token_version) and per-device
		// session validity (sid). In degraded mode (revoker nil) every user is
		// treated as standard and neither check runs.
		role := RoleStandard
		if revoker != nil {
			r, err := revoker.Resolve(c.Request.Context(), claims.MembershipID, claims.TokenVersion, claims.SessionID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
				return
			}
			role = r
		}
		c.Set("user_id", claims.UserID)
		c.Set("membership_id", claims.MembershipID)
		c.Set("membership_type", claims.MembershipType)
		c.Set("display_name", claims.DisplayName)
		c.Set("platform", claims.Platform)
		c.Set("token_version", claims.TokenVersion)
		c.Set("session_id", claims.SessionID)
		c.Set("role", role)
		c.Next()
	}
}
