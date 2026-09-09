package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/sources"
	"guardian-tracker/api-service/services/wishlist"

	"github.com/gin-gonic/gin"
)

// --- interfaces (satisfied by concrete service types via structural typing) ---

// wishlistEntries is the wish list capability this handler drives. Satisfied by
// *wishlist.Entries, which owns validation, persistence, and item-existence
// rules — the handler owns none of them.
type wishlistEntries interface {
	List(ctx context.Context, membershipID string) ([]wishlist.StoredEntry, error)
	Add(ctx context.Context, membershipID string, cmd wishlist.AddCommand) (wishlist.StoredEntry, error)
	Update(ctx context.Context, membershipID string, id wishlist.EntryID, patch wishlist.UpdateCommand) (wishlist.StoredEntry, error)
	Remove(ctx context.Context, membershipID string, id wishlist.EntryID) error
	RemoveMany(ctx context.Context, membershipID string, ids []wishlist.EntryID) (wishlist.BulkResult, error)
	SetPriorityMany(ctx context.Context, membershipID string, ids []wishlist.EntryID, priority wishlist.Priority) (wishlist.BulkResult, error)
}

type manifestLookupIface interface {
	GetItemsByHashes(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error)
	GetCollectiblesByItemHashes(hashes []uint32) (map[uint32][]bungie.CollectibleDefinition, error)
}

// liveVendorIface returns itemHash → selling-vendor display name for items
// available right now from rotating vendors (Xûr + Banshee-44 + Ada-1 + ritual
// vendors). Satisfied by *weekly.Service — the same source Collections uses.
type liveVendorIface interface {
	LiveVendorItemHashes(ctx context.Context, membershipType int, membershipID, bungieToken string) map[uint32]string
}

// tokenProvider yields a user's current Bungie access token for the authed
// vendor fetch. Satisfied by *auth.TokenStore. Best-effort: an error means we
// resolve public-only availability (Xûr).
type tokenProvider interface {
	GetValidToken(membershipID string) (string, error)
}

// WishlistHandler handles wishlist endpoints.
type WishlistHandler struct {
	entries     wishlistEntries
	manifest    manifestLookupIface // nil = no enrichment
	liveVendors liveVendorIface     // nil = availability always false
	tokens      tokenProvider       // nil = public-only availability
}

// NewWishlistHandler creates a handler. Entries is required; the completion
// dependencies may be nil when unavailable.
func NewWishlistHandler(entries wishlistEntries, manifest manifestLookupIface, liveVendors liveVendorIface, tokens tokenProvider) *WishlistHandler {
	return &WishlistHandler{entries: entries, manifest: manifest, liveVendors: liveVendors, tokens: tokens}
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
	entries, err := h.entries.List(c.Request.Context(), c.GetString("membership_id"))
	if err != nil {
		handleWishlistError(c, err, "wishlist listing failed")
		return
	}
	c.JSON(http.StatusOK, h.enrichEntries(entries, h.liveVendorMap(c)))
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

	entry, err := h.entries.Add(c.Request.Context(), c.GetString("membership_id"), wishlist.AddCommand{
		ItemHash: body.ItemHash,
		Priority: wishlist.Priority(body.Priority),
		Notes:    body.Notes,
	})
	if err != nil {
		handleWishlistError(c, err, "wishlist item creation failed")
		return
	}
	c.JSON(http.StatusCreated, h.enrichOne(entry, h.liveVendorMap(c)))
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
	entry, err := h.entries.Update(c.Request.Context(), c.GetString("membership_id"), id, patch)
	if err != nil {
		handleWishlistError(c, err, "wishlist item update failed")
		return
	}
	c.JSON(http.StatusOK, h.enrichOne(entry, h.liveVendorMap(c)))
}

// RemoveFromWishlist handles DELETE /api/wishlist/:id
func (h *WishlistHandler) RemoveFromWishlist(c *gin.Context) {
	id, ok := parseEntryID(c)
	if !ok {
		return
	}
	if err := h.entries.Remove(c.Request.Context(), c.GetString("membership_id"), id); err != nil {
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

	membershipID := c.GetString("membership_id")
	var result wishlist.BulkResult
	var err error
	switch body.Action {
	case "delete":
		result, err = h.entries.RemoveMany(c.Request.Context(), membershipID, ids)
	case "set_priority":
		result, err = h.entries.SetPriorityMany(c.Request.Context(), membershipID, ids, wishlist.Priority(body.Priority))
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

func (h *WishlistHandler) enrichEntries(entries []wishlist.StoredEntry, live map[uint32]string) []wishlistResponse {
	if len(entries) == 0 {
		return []wishlistResponse{}
	}
	hashes := make([]uint32, len(entries))
	for i, entry := range entries {
		hashes[i] = entry.ItemHash
	}
	defs := map[uint32]*bungie.InventoryItemDefinition{}
	cols := map[uint32][]bungie.CollectibleDefinition{}
	if h.manifest != nil {
		if m, err := h.manifest.GetItemsByHashes(hashes); err == nil {
			defs = m
		}
		if cs, err := h.manifest.GetCollectiblesByItemHashes(hashes); err == nil {
			cols = cs
		}
	}
	resp := make([]wishlistResponse, len(entries))
	for i, entry := range entries {
		resp[i] = buildResponse(entry, defs[entry.ItemHash], cols[entry.ItemHash], live[entry.ItemHash])
	}
	return resp
}

func (h *WishlistHandler) enrichOne(entry wishlist.StoredEntry, live map[uint32]string) wishlistResponse {
	var def *bungie.InventoryItemDefinition
	var cols []bungie.CollectibleDefinition
	if h.manifest != nil {
		if m, err := h.manifest.GetItemsByHashes([]uint32{entry.ItemHash}); err == nil {
			def = m[entry.ItemHash]
		}
		if cs, err := h.manifest.GetCollectiblesByItemHashes([]uint32{entry.ItemHash}); err == nil {
			cols = cs[entry.ItemHash]
		}
	}
	return buildResponse(entry, def, cols, live[entry.ItemHash])
}

// liveVendorMap resolves item→vendor-name availability for the calling user.
// Best-effort: empty on degraded mode or token failure; never errors.
func (h *WishlistHandler) liveVendorMap(c *gin.Context) map[uint32]string {
	if h.liveVendors == nil {
		return map[uint32]string{}
	}
	membershipID := c.GetString("membership_id")
	membershipType := c.GetInt("membership_type")
	bungieToken := ""
	if h.tokens != nil {
		if t, err := h.tokens.GetValidToken(membershipID); err == nil {
			bungieToken = t
		}
	}
	return h.liveVendors.LiveVendorItemHashes(c.Request.Context(), membershipType, membershipID, bungieToken)
}

func buildResponse(entry wishlist.StoredEntry, def *bungie.InventoryItemDefinition, collectibles []bungie.CollectibleDefinition, vendor string) wishlistResponse {
	name, itemTypeStr, rarity, icon := "Unknown Item", "Item", "Common", ""
	sourceTexts := make([]string, 0, len(collectibles))
	if def != nil {
		name = def.DisplayProperties.Name
		itemTypeStr = bungie.ItemTypeName(def.ItemType, def.ItemSubType)
		rarity = bungie.GetTierName(def.Inventory.TierType)
		icon = def.DisplayProperties.Icon
	}
	for _, col := range collectibles {
		sourceTexts = append(sourceTexts, col.SourceString)
	}
	resp := wishlistResponse{
		ID:                 strconv.FormatInt(int64(entry.ID), 10),
		ItemHash:           entry.ItemHash,
		Name:               name,
		ItemType:           itemTypeStr,
		Rarity:             rarity,
		Icon:               icon,
		Priority:           string(entry.Priority),
		Notes:              entry.Notes,
		AcquisitionSources: sources.DescribeAll(sourceTexts),
		AvailableNow:       vendor != "",
		DateAdded:          entry.CreatedAt.UTC().Format(time.RFC3339),
	}
	if vendor != "" {
		resp.AvailableFrom = vendor
	}
	return resp
}
