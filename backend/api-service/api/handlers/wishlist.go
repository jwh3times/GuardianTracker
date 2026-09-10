package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/sources"
	"guardian-tracker/api-service/services/wishlist"

	"github.com/gin-gonic/gin"
)

// wishlistService is the complete wish list capability this handler drives.
// Satisfied by *wishlist.Service, which owns validation, persistence,
// item-existence rules, Item completion, tombstones, credentials, and the
// availability join — the handler owns none of them.
type wishlistService interface {
	List(ctx context.Context, m wishlist.Membership) ([]wishlist.Entry, error)
	Add(ctx context.Context, m wishlist.Membership, cmd wishlist.AddCommand) (wishlist.Entry, error)
	Update(ctx context.Context, m wishlist.Membership, id wishlist.EntryID, patch wishlist.UpdateCommand) (wishlist.Entry, error)
	Remove(ctx context.Context, m wishlist.Membership, id wishlist.EntryID) error
	DeleteMany(ctx context.Context, m wishlist.Membership, ids []wishlist.EntryID) (wishlist.BulkResult, error)
	SetPriorityMany(ctx context.Context, m wishlist.Membership, ids []wishlist.EntryID, priority wishlist.Priority) (wishlist.BulkResult, error)
}

// WishlistHandler adapts the complete wish list capability to HTTP.
//
// It owns request binding, the bulk action vocabulary, error-to-status mapping,
// and serialization — and nothing else. Resolving what an item is, whether it
// is on sale, and which credential to use all belong to wishlist.Service
// (ADR 0019), which is why this type no longer holds the Manifest, Weekly, or
// the token store.
type WishlistHandler struct {
	wishlist wishlistService
}

func NewWishlistHandler(svc wishlistService) *WishlistHandler {
	return &WishlistHandler{wishlist: svc}
}

// membershipOf reads the caller's Destiny membership from the JWT the
// middleware validated. The wish list is never addressed by client-supplied
// identity: there is no membership on the route or in any request body.
func membershipOf(c *gin.Context) wishlist.Membership {
	return wishlist.Membership{
		MembershipType: c.GetInt("membership_type"),
		MembershipID:   c.GetString("membership_id"),
	}
}

// wishlistResponse is the JSON shape returned to clients.
type wishlistResponse struct {
	ID                 string                      `json:"id"`
	ItemHash           uint32                      `json:"itemHash"`
	Name               string                      `json:"name"`
	ItemType           string                      `json:"itemType"`
	Rarity             string                      `json:"rarity"`
	Icon               string                      `json:"icon"`
	Priority           string                      `json:"priority"`
	Notes              string                      `json:"notes"`
	AcquisitionSources []sources.AcquisitionSource `json:"acquisitionSources"`
	AvailableNow       bool                        `json:"availableNow"`
	AvailableFrom      string                      `json:"availableFrom,omitempty"`
	DateAdded          string                      `json:"dateAdded"`
}

// GetWishlist handles GET /api/wishlist
func (h *WishlistHandler) GetWishlist(c *gin.Context) {
	entries, err := h.wishlist.List(c.Request.Context(), membershipOf(c))
	if err != nil {
		handleWishlistError(c, err, "wishlist listing failed")
		return
	}
	c.JSON(http.StatusOK, wishlistResponses(entries))
}

// AddToWishlist handles POST /api/wishlist
func (h *WishlistHandler) AddToWishlist(c *gin.Context) {
	var body struct {
		ItemHash uint32 `json:"itemHash" binding:"required"`
		Priority string `json:"priority"`
		Notes    string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "itemHash is required"})
		return
	}

	entry, err := h.wishlist.Add(c.Request.Context(), membershipOf(c), wishlist.AddCommand{
		ItemHash: body.ItemHash,
		Priority: wishlist.Priority(body.Priority),
		Notes:    body.Notes,
	})
	if err != nil {
		handleWishlistError(c, err, "wishlist item creation failed")
		return
	}
	c.JSON(http.StatusCreated, wishlistResponseOf(entry))
}

// UpdateWishlistItem handles PUT /api/wishlist/:id
func (h *WishlistHandler) UpdateWishlistItem(c *gin.Context) {
	id, ok := parseEntryID(c)
	if !ok {
		return
	}
	var body struct {
		Priority *string `json:"priority"`
		Notes    *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	patch := wishlist.UpdateCommand{Notes: body.Notes}
	if body.Priority != nil {
		priority := wishlist.Priority(*body.Priority)
		patch.Priority = &priority
	}
	entry, err := h.wishlist.Update(c.Request.Context(), membershipOf(c), id, patch)
	if err != nil {
		handleWishlistError(c, err, "wishlist item update failed")
		return
	}
	c.JSON(http.StatusOK, wishlistResponseOf(entry))
}

// RemoveFromWishlist handles DELETE /api/wishlist/:id
func (h *WishlistHandler) RemoveFromWishlist(c *gin.Context) {
	id, ok := parseEntryID(c)
	if !ok {
		return
	}
	if err := h.wishlist.Remove(c.Request.Context(), membershipOf(c), id); err != nil {
		handleWishlistError(c, err, "wishlist item deletion failed")
		return
	}
	c.Status(http.StatusNoContent)
}

// BulkUpdate handles POST /api/wishlist/bulk — delete or set-priority on a set of
// items in one request. Partial success: foreign/missing ids are silently skipped
// and counted. Body: {action, ids, priority?}; response: {updated, skipped}.
func (h *WishlistHandler) BulkUpdate(c *gin.Context) {
	var body struct {
		Action   string  `json:"action"`
		IDs      []int64 `json:"ids"`
		Priority string  `json:"priority"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ids := make([]wishlist.EntryID, len(body.IDs))
	for i, id := range body.IDs {
		ids[i] = wishlist.EntryID(id)
	}

	membership := membershipOf(c)
	var result wishlist.BulkResult
	var err error
	switch body.Action {
	case "delete":
		result, err = h.wishlist.DeleteMany(c.Request.Context(), membership, ids)
	case "set_priority":
		result, err = h.wishlist.SetPriorityMany(c.Request.Context(), membership, ids, wishlist.Priority(body.Priority))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be 'delete' or 'set_priority'"})
		return
	}
	if err != nil {
		handleWishlistError(c, err, "wishlist bulk update failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": result.Updated, "skipped": result.Skipped})
}

// --- helpers ---

// parseEntryID reads the :id route parameter. The wire carries entry ids as
// strings; anything that is not one is a malformed request, not a missing entry.
func parseEntryID(c *gin.Context) (wishlist.EntryID, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return wishlist.EntryID(id), true
}

// handleWishlistError maps the wish list's error vocabulary to the existing
// wire. Each case is a distinction the domain drew deliberately, and flattening
// any two of them here would put the meaning back in the handler.
func handleWishlistError(c *gin.Context, err error, logMsg string) {
	switch {
	case errors.Is(err, wishlist.ErrUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "This feature needs the user data database, which isn't configured on this server.",
			"code":  "DB_UNAVAILABLE",
		})
	case errors.Is(err, wishlist.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "wishlist item not found"})
	case errors.Is(err, wishlist.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "item already in wishlist"})
	case errors.Is(err, wishlist.ErrItemsUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "The item database is still downloading — try again in a moment.",
			"code":  "MANIFEST_NOT_READY",
		})
	case wishlist.IsValidationError(err):
		c.JSON(http.StatusBadRequest, gin.H{"error": validationMessage(err)})
	default:
		ctx := handlerContext(c)
		observability.Logger(ctx).ErrorContext(ctx, logMsg, observability.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error", "code": "INTERNAL_ERROR"})
	}
}

// validationMessage keeps the exact refusal text clients already receive. The
// domain errors carry a package prefix for logs; the wire never has.
func validationMessage(err error) string {
	switch {
	case errors.Is(err, wishlist.ErrInvalidPriority):
		return "priority must be LOW, MEDIUM, HIGH, or URGENT"
	case errors.Is(err, wishlist.ErrNotesTooLong):
		return "notes must be 500 characters or fewer"
	case errors.Is(err, wishlist.ErrNoEntries):
		return "ids must be a non-empty list"
	case errors.Is(err, wishlist.ErrTooManyEntries):
		return "at most 100 ids per request"
	case errors.Is(err, wishlist.ErrUnknownItem):
		return "unknown item hash"
	}
	return "invalid request"
}

// wishlistResponses transcribes complete entries onto the wire.
//
// A transcription, not a decision: what an item is called, what a tombstone
// looks like, and whether something is on sale are all resolved before they get
// here. The handler only chooses field names.
func wishlistResponses(entries []wishlist.Entry) []wishlistResponse {
	out := make([]wishlistResponse, len(entries))
	for i, entry := range entries {
		out[i] = wishlistResponseOf(entry)
	}
	return out
}

func wishlistResponseOf(entry wishlist.Entry) wishlistResponse {
	return wishlistResponse{
		ID:                 strconv.FormatInt(int64(entry.ID), 10),
		ItemHash:           entry.ItemHash,
		Name:               entry.Item.Name(),
		ItemType:           entry.Item.ItemType(),
		Rarity:             entry.Item.Rarity(),
		Icon:               entry.Item.Icon(),
		Priority:           string(entry.Priority),
		Notes:              entry.Notes,
		AcquisitionSources: entry.Item.AcquisitionSources(),
		AvailableNow:       entry.AvailableFrom != "",
		AvailableFrom:      entry.AvailableFrom,
		DateAdded:          entry.CreatedAt.UTC().Format(time.RFC3339),
	}
}
