package handlers

import (
	"log"
	"net/http"
	"strings"

	"guardian-tracker/api-service/auth"

	"github.com/gin-gonic/gin"
)

// ownershipCheck verifies the authenticated user (set by JWT middleware) matches the requested membershipID.
func ownershipCheck(c *gin.Context, membershipID string) bool {
	if c.GetString("membership_id") != membershipID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only access your own data", "code": "FORBIDDEN"})
		return false
	}
	return true
}

// getBungieToken retrieves a valid Bungie OAuth token for membershipID from the token store.
func getBungieToken(c *gin.Context, membershipID string, tokenStore *auth.TokenStore) (string, bool) {
	token, err := tokenStore.GetValidToken(membershipID)
	if err != nil {
		log.Printf("getBungieToken for %s: %v", membershipID, err)
		if strings.Contains(err.Error(), "re-authentication required") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Your Bungie session has expired. Please log in again.",
				"code":  "BUNGIE_TOKEN_EXPIRED",
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Unable to retrieve Bungie authorization. Please try again.",
				"code":  "TOKEN_ERROR",
			})
		}
		return "", false
	}
	return token, true
}
