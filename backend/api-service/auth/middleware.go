package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Middleware returns a Gin handler that requires a valid access token.
// It sets user_id, membership_id, membership_type, display_name, and platform
// in the request context for downstream handlers.
func (j *JWT) Middleware() gin.HandlerFunc {
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
		c.Set("user_id", claims.UserID)
		c.Set("membership_id", claims.MembershipID)
		c.Set("membership_type", claims.MembershipType)
		c.Set("display_name", claims.DisplayName)
		c.Set("platform", claims.Platform)
		c.Next()
	}
}

// OptionalMiddleware is like Middleware but does not abort on missing/invalid tokens.
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
		c.Next()
	}
}
