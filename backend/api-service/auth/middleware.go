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
		if revoker != nil {
			if err := revoker.Check(c.Request.Context(), claims.MembershipID, claims.TokenVersion); err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
				return
			}
		}
		c.Set("user_id", claims.UserID)
		c.Set("membership_id", claims.MembershipID)
		c.Set("membership_type", claims.MembershipType)
		c.Set("display_name", claims.DisplayName)
		c.Set("platform", claims.Platform)
		c.Set("token_version", claims.TokenVersion)
		c.Next()
	}
}

// OptionalMiddleware is like Middleware but does not abort on missing/invalid tokens.
// Revocation is not checked — this path is for unauthenticated access with optional personalization.
func (j *JWT) OptionalMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ExtractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			c.Next()
			return
		}
		claims, err := j.ValidateToken(token)
		if err != nil || claims.TokenType != "access" {
			c.Next()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("membership_id", claims.MembershipID)
		c.Set("membership_type", claims.MembershipType)
		c.Set("display_name", claims.DisplayName)
		c.Set("platform", claims.Platform)
		c.Set("token_version", claims.TokenVersion)
		c.Next()
	}
}
