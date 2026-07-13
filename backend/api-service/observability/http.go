package observability

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// HTTPMiddleware owns request IDs, request-scoped logging, access records, and
// panic recovery. It intentionally records route templates rather than URLs.
func HTTPMiddleware(base *slog.Logger) gin.HandlerFunc {
	if base == nil {
		base = slog.Default()
	}
	return func(c *gin.Context) {
		requestID := uuid.NewString()
		c.Header(RequestIDHeader, requestID)
		requestLogger := base.With(slog.String("request_id", requestID))
		requestContext := WithLogger(c.Request.Context(), requestLogger)
		c.Request = c.Request.WithContext(requestContext)
		c.Set("request_id", requestID)
		started := time.Now()

		defer func() {
			if recover() != nil {
				// Never log the panic value: it can contain request or secret data.
				requestLogger.ErrorContext(requestContext, "panic recovered")
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"error": "Internal server error",
						"code":  "INTERNAL_ERROR",
					})
				} else {
					c.Abort()
				}
			}

			route := c.FullPath()
			if route == "" {
				route = "unmatched"
			}
			status := c.Writer.Status()
			responseBytes := c.Writer.Size()
			if responseBytes < 0 {
				responseBytes = 0
			}
			level := accessLevel(route, status)
			requestLogger.LogAttrs(requestContext, level, "http request",
				slog.String("method", c.Request.Method),
				slog.String("route", route),
				slog.Int("status", status),
				slog.Duration("duration", time.Since(started)),
				slog.Int("response_bytes", responseBytes),
			)
		}()

		c.Next()
	}
}

func accessLevel(route string, status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	case route == "/health" || route == "/ready":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}
