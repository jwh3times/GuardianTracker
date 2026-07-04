package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes caps every request body at n bytes via http.MaxBytesReader;
// reads past the cap fail, so ShouldBindJSON returns an error the handler maps
// to 400. GET/HEAD requests are unaffected (no body).
func MaxBodyBytes(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		}
		c.Next()
	}
}
