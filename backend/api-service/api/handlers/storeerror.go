package handlers

import (
	"errors"
	"net/http"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/observability"

	"github.com/gin-gonic/gin"
)

// HandleStoreError maps a persistence failure to a response.
//
// The mirror of handleBungieError, and it exists for the same reason: before
// this, "the database isn't there" produced three different bodies across the
// handlers — two spellings of DB_UNAVAILABLE and, in the seven wishlist
// handlers, a bare {"error": "database not configured"} with no code at all,
// which the frontend could not recognise and rendered as generic copy.
//
// Returns true when it handled the error, so callers read:
//
//	if err != nil {
//	    HandleStoreError(c, err, "wishlist list failed")
//	    return
//	}
func HandleStoreError(c *gin.Context, err error, logMsg string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, db.ErrUnavailable) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "This feature needs the account database, which isn't configured on this server.",
			"code":  "DB_UNAVAILABLE",
		})
		return true
	}
	ctx := handlerContext(c)
	observability.Logger(ctx).ErrorContext(ctx, logMsg, observability.Err(err))
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "Internal server error",
		"code":  "INTERNAL_ERROR",
	})
	return true
}
