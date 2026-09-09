package handlers

import (
	"context"
	"errors"
	"net/http"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/observability"

	"github.com/gin-gonic/gin"
)

func handlerContext(c *gin.Context) context.Context {
	if c != nil && c.Request != nil {
		return c.Request.Context()
	}
	return context.Background()
}

// ownershipCheck verifies that the authenticated user (set by JWT middleware)
// owns the requested Destiny membership.
//
// A Destiny membership is the pair (type, id), not the id alone — the platform
// is part of the identity, and Bungie reports both for every membership on an
// account. Comparing only the id, as this did before ADR 0018, accepted a
// request naming a platform the caller never authenticated against.
//
// That did not expose anyone else's data: the id half was always pinned to the
// caller's own claim, and Bungie allocates membership ids from one id space, so
// a mismatched pair names a membership that does not exist rather than someone
// else's. What it did do was spend a Bungie call and key per-membership cache
// entries on a pair the JWT never vouched for. This is identity-model
// correctness, not a repaired data-isolation hole.
//
// Both halves are compared here, before any token lookup or service call, so a
// mismatch costs nothing and touches no credentials. The claim has carried
// membership_type since the backend was consolidated and survives a refresh, so
// no live token reaches this with the type missing.
func ownershipCheck(c *gin.Context, membershipType int, membershipID string) bool {
	if c.GetString("membership_id") != membershipID || c.GetInt("membership_type") != membershipType {
		// Abort, not a bare write: the refusal's header is already flushed, so a
		// call site that forgot to return would otherwise append the caller's
		// data to the body of this 403.
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You can only access your own data", "code": "FORBIDDEN"})
		return false
	}
	return true
}

// getBungieToken retrieves a valid Bungie OAuth token for membershipID from the token store.
func getBungieToken(c *gin.Context, membershipID string, tokenStore *auth.TokenStore) (string, bool) {
	token, err := tokenStore.GetValidToken(membershipID)
	if err != nil {
		observability.Logger(c.Request.Context()).WarnContext(c.Request.Context(), "Bungie token unavailable",
			observability.ID("membership", membershipID), observability.Err(err))
		// Aborts for the same reason as ownershipCheck above.
		if errors.Is(err, auth.ErrBungieReauthorizationRequired) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Your Bungie authorization has expired. Reconnect Bungie to continue.",
				"code":  "BUNGIE_REAUTH_REQUIRED",
			})
		} else {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "Unable to retrieve Bungie authorization. Please try again.",
				"code":  "TOKEN_ERROR",
			})
		}
		return "", false
	}
	return token, true
}
