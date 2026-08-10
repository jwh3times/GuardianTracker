package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// APISecurityHeaders adds baseline browser hardening to every /api response,
// including handler errors. Health probes remain unchanged.
func APISecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/api" || strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("X-Content-Type-Options", "nosniff")
			c.Header("Referrer-Policy", "no-referrer")
		}
		c.Next()
	}
}

// NoStore prevents browsers and intermediaries from retaining auth responses.
func NoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}

// OriginAllowed performs the same exact, trimmed allowlist match for CORS and
// the state-changing unauthenticated auth endpoints. Wildcards are never
// interpreted here.
func OriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	return false
}

// CORS answers preflights and reflects an exact allowlisted Origin. It always
// sets Vary: Origin, including on a denied origin, so a shared cache cannot
// serve one origin's allow header to another.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		c.Header("Vary", "Origin")
		if OriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// RequireAllowedOrigin rejects requests without an exact allowlisted Origin.
// It is applied to callback and refresh because both establish/rotate a
// credential-bearing cookie without requiring an access token.
func RequireAllowedOrigin(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !OriginAllowed(c.GetHeader("Origin"), allowedOrigins) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Origin not allowed"})
			return
		}
		c.Next()
	}
}
