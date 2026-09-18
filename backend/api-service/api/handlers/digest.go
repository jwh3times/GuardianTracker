package handlers

import (
	"net/http"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/digest"

	"github.com/gin-gonic/gin"
)

// DigestHandler adapts the since-last-visit Digest service to HTTP
// (ADR 0023). It binds, authenticates, resolves the Bungie token, and
// serializes; every visit-boundary, diff, and privacy-gate decision belongs to
// digest.Service.
type DigestHandler struct {
	digest     *digest.Service
	tokenStore *auth.TokenStore
}

func NewDigestHandler(svc *digest.Service, ts *auth.TokenStore) *DigestHandler {
	return &DigestHandler{digest: svc, tokenStore: ts}
}

type acquiredItemResponse struct {
	ItemHash uint32 `json:"itemHash"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	ItemType string `json:"itemType"`
}

type digestResponse struct {
	Status          digest.Status          `json:"status"`
	VisitStartedAt  time.Time              `json:"visitStartedAt"`
	PreviousVisitAt *time.Time             `json:"previousVisitAt,omitempty"`
	Acquired        []acquiredItemResponse `json:"acquired"`
}

// GetDigest handles GET /api/digest/:membershipType/:membershipId.
func (h *DigestHandler) GetDigest(c *gin.Context) {
	membershipType, membershipID, ok := parseMembershipParams(c)
	if !ok {
		return
	}
	if !ownershipCheck(c, membershipType, membershipID) {
		return
	}
	bungieToken, ok := getBungieToken(c, membershipID, h.tokenStore)
	if !ok {
		return
	}

	result, err := h.digest.GetDigest(c.Request.Context(), membershipType, membershipID, bungieToken)
	if err != nil {
		ctx := handlerContext(c)
		observability.Logger(ctx).ErrorContext(ctx, "digest request failed", observability.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error", "code": "INTERNAL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, toDigestResponse(result))
}

func toDigestResponse(result digest.Result) digestResponse {
	acquired := make([]acquiredItemResponse, len(result.Acquired))
	for i, a := range result.Acquired {
		acquired[i] = acquiredItemResponse{ItemHash: a.ItemHash, Name: a.Name, Icon: a.Icon, ItemType: a.ItemType}
	}
	return digestResponse{
		Status:          result.Status,
		VisitStartedAt:  result.VisitStartedAt,
		PreviousVisitAt: result.PreviousVisitAt,
		Acquired:        acquired,
	}
}
