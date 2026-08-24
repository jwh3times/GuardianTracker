package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/sources"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Priority mapping between string labels and DB int16 values.
var priorityToInt = map[string]int16{"LOW": 0, "MEDIUM": 1, "HIGH": 2, "URGENT": 3}
var priorityToStr = map[int16]string{0: "LOW", 1: "MEDIUM", 2: "HIGH", 3: "URGENT"}

// --- interfaces (satisfied by concrete db/manifest types via structural typing) ---

type wishlistStoreIface interface {
	GetUserID(ctx context.Context, membershipID string) (int64, error)
	List(ctx context.Context, userID int64) ([]db.WishlistItem, error)
	Add(ctx context.Context, userID int64, hash uint32, prio int16, notes string) (*db.WishlistItem, error)
	Update(ctx context.Context, userID, id int64, prio *int16, notes *string) (*db.WishlistItem, error)
	Delete(ctx context.Context, userID, id int64) (bool, error)
	BulkDelete(ctx context.Context, userID int64, ids []int64) (int64, error)
	BulkSetPriority(ctx context.Context, userID int64, ids []int64, prio int16) (int64, error)
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
	store       wishlistStoreIface  // degraded implementation when no database exists
	manifest    manifestLookupIface // nil = no enrichment
	liveVendors liveVendorIface     // nil = availability always false
	tokens      tokenProvider       // nil = public-only availability
}

// NewWishlistHandler creates a handler. Store is required and remains non-nil
// in degraded mode; enrichment dependencies may be nil when unavailable.
func NewWishlistHandler(store wishlistStoreIface, manifest manifestLookupIface, liveVendors liveVendorIface, tokens tokenProvider) *WishlistHandler {
	return &WishlistHandler{store: store, manifest: manifest, liveVendors: liveVendors, tokens: tokens}
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
	membershipID := c.GetString("membership_id")
	userID, err := h.store.GetUserID(c.Request.Context(), membershipID)
	if err != nil {
		HandleStoreError(c, err, "wishlist user lookup failed")
		return
	}
	items, err := h.store.List(c.Request.Context(), userID)
	if err != nil {
		HandleStoreError(c, err, "wishlist listing failed")
		return
	}
	c.JSON(http.StatusOK, h.enrichItems(items, h.liveVendorMap(c)))
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
	// Validate item exists in manifest (best-effort; skip if manifest unavailable)
	if h.manifest != nil {
		defs, err := h.manifest.GetItemsByHashes([]uint32{body.ItemHash})
		if err != nil || defs[body.ItemHash] == nil {
			if err != nil {
				observability.Logger(c.Request.Context()).WarnContext(c.Request.Context(), "wishlist manifest lookup failed",
					"item_hash", body.ItemHash, observability.Err(err))
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown item hash"})
				return
			}
		}
	}
	prio := priorityToInt["MEDIUM"]
	if body.Priority != "" {
		p, ok := priorityToInt[body.Priority]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "priority must be LOW, MEDIUM, HIGH, or URGENT"})
			return
		}
		prio = p
	}
	if len(body.Notes) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notes must be 500 characters or fewer"})
		return
	}
	membershipID := c.GetString("membership_id")
	userID, err := h.store.GetUserID(c.Request.Context(), membershipID)
	if err != nil {
		HandleStoreError(c, err, "wishlist user lookup failed")
		return
	}
	item, err := h.store.Add(c.Request.Context(), userID, body.ItemHash, prio, body.Notes)
	if err != nil {
		if isDuplicate(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "item already in wishlist"})
			return
		}
		HandleStoreError(c, err, "wishlist item creation failed")
		return
	}
	c.JSON(http.StatusCreated, h.enrichOne(*item, h.liveVendorMap(c)))
}

// UpdateWishlistItem handles PUT /api/wishlist/:id
func (h *WishlistHandler) UpdateWishlistItem(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
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
	var prio *int16
	if body.Priority != nil {
		p, ok := priorityToInt[*body.Priority]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "priority must be LOW, MEDIUM, HIGH, or URGENT"})
			return
		}
		prio = &p
	}
	if body.Notes != nil && len(*body.Notes) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notes must be 500 characters or fewer"})
		return
	}
	membershipID := c.GetString("membership_id")
	userID, err := h.store.GetUserID(c.Request.Context(), membershipID)
	if err != nil {
		HandleStoreError(c, err, "wishlist user lookup failed")
		return
	}
	item, err := h.store.Update(c.Request.Context(), userID, id, prio, body.Notes)
	if err != nil {
		if isNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wishlist item not found"})
			return
		}
		HandleStoreError(c, err, "wishlist item update failed")
		return
	}
	c.JSON(http.StatusOK, h.enrichOne(*item, h.liveVendorMap(c)))
}

// RemoveFromWishlist handles DELETE /api/wishlist/:id
func (h *WishlistHandler) RemoveFromWishlist(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	membershipID := c.GetString("membership_id")
	userID, err := h.store.GetUserID(c.Request.Context(), membershipID)
	if err != nil {
		HandleStoreError(c, err, "wishlist user lookup failed")
		return
	}
	found, err := h.store.Delete(c.Request.Context(), userID, id)
	if err != nil {
		HandleStoreError(c, err, "wishlist item deletion failed")
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "wishlist item not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

const bulkMaxIDs = 100

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
	// Dedupe ids, preserving nothing but uniqueness.
	seen := make(map[int64]struct{}, len(body.IDs))
	ids := make([]int64, 0, len(body.IDs))
	for _, id := range body.IDs {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids must be a non-empty list"})
		return
	}
	if len(ids) > bulkMaxIDs {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("at most %d ids per request", bulkMaxIDs)})
		return
	}

	// Validate the action (and priority) before resolving the user, so a malformed
	// request doesn't cost a DB lookup.
	var prio int16
	switch body.Action {
	case "delete":
		// no priority needed
	case "set_priority":
		p, ok := priorityToInt[body.Priority]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "priority must be LOW, MEDIUM, HIGH, or URGENT"})
			return
		}
		prio = p
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be 'delete' or 'set_priority'"})
		return
	}

	membershipID := c.GetString("membership_id")
	userID, err := h.store.GetUserID(c.Request.Context(), membershipID)
	if err != nil {
		HandleStoreError(c, err, "wishlist user lookup failed")
		return
	}

	var updated int64
	switch body.Action {
	case "delete":
		updated, err = h.store.BulkDelete(c.Request.Context(), userID, ids)
	case "set_priority":
		updated, err = h.store.BulkSetPriority(c.Request.Context(), userID, ids, prio)
	}
	if err != nil {
		HandleStoreError(c, err, "wishlist bulk update failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": updated, "skipped": int64(len(ids)) - updated})
}

// --- helpers ---

func (h *WishlistHandler) enrichItems(items []db.WishlistItem, live map[uint32]string) []wishlistResponse {
	if len(items) == 0 {
		return []wishlistResponse{}
	}
	hashes := make([]uint32, len(items))
	for i, it := range items {
		hashes[i] = it.ItemHash
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
	resp := make([]wishlistResponse, len(items))
	for i, it := range items {
		resp[i] = buildResponse(it, defs[it.ItemHash], cols[it.ItemHash], live[it.ItemHash])
	}
	return resp
}

func (h *WishlistHandler) enrichOne(it db.WishlistItem, live map[uint32]string) wishlistResponse {
	var def *bungie.InventoryItemDefinition
	var cols []bungie.CollectibleDefinition
	if h.manifest != nil {
		if m, err := h.manifest.GetItemsByHashes([]uint32{it.ItemHash}); err == nil {
			def = m[it.ItemHash]
		}
		if cs, err := h.manifest.GetCollectiblesByItemHashes([]uint32{it.ItemHash}); err == nil {
			cols = cs[it.ItemHash]
		}
	}
	return buildResponse(it, def, cols, live[it.ItemHash])
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

func buildResponse(it db.WishlistItem, def *bungie.InventoryItemDefinition, collectibles []bungie.CollectibleDefinition, vendor string) wishlistResponse {
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
		ID:                 strconv.FormatInt(it.ID, 10),
		ItemHash:           it.ItemHash,
		Name:               name,
		ItemType:           itemTypeStr,
		Rarity:             rarity,
		Icon:               icon,
		Priority:           priorityToStr[it.Priority],
		Notes:              it.Notes,
		AcquisitionSources: sources.DescribeAll(sourceTexts),
		AvailableNow:       vendor != "",
		DateAdded:          it.CreatedAt.UTC().Format(time.RFC3339),
	}
	if vendor != "" {
		resp.AvailableFrom = vendor
	}
	return resp
}

func isDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
