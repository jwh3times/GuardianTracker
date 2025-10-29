package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		tokenString, err := ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Store claims in context for use in handlers
		c.Set("user_id", claims.UserID)
		c.Set("membership_id", claims.MembershipID)
		c.Set("membership_type", claims.MembershipType)
		c.Set("display_name", claims.DisplayName)
		c.Set("platform", claims.Platform)
		c.Set("token_type", claims.TokenType)

		c.Next()
	}
}

// OptionalAuthMiddleware attempts to validate JWT but doesn't require it
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString, err := ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.Next()
			return
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		// Store claims in context if valid
		c.Set("user_id", claims.UserID)
		c.Set("membership_id", claims.MembershipID)
		c.Set("membership_type", claims.MembershipType)
		c.Set("display_name", claims.DisplayName)
		c.Set("platform", claims.Platform)
		c.Set("token_type", claims.TokenType)

		c.Next()
	}
}
