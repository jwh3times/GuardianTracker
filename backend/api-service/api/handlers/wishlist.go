package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/bungie"

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
}

type manifestLookupIface interface {
	GetItemsByHashes(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error)
	GetCollectiblesByItemHashes(hashes []uint32) (map[uint32]*bungie.CollectibleDefinition, error)
}

// xurInventoryIface is satisfied by *weekly.Service — used to flag wishlist
// items Xûr currently sells (B6).
type xurInventoryIface interface {
	XurItemHashes(ctx context.Context) map[uint32]struct{}
}

type prefsStoreIface interface {
	Get(ctx context.Context, userID int64) (*db.UserPreferences, error)
	Upsert(ctx context.Context, userID int64, cardStyle string, personalize bool) (*db.UserPreferences, error)
}

// WishlistHandler handles wishlist and preferences endpoints.
type WishlistHandler struct {
	store    wishlistStoreIface  // nil = degraded mode
	manifest manifestLookupIface // nil = no enrichment
	prefs    prefsStoreIface     // nil = degraded mode
	xur      xurInventoryIface   // nil = availability always false
}

// NewWishlistHandler creates a handler. Any argument may be nil for degraded-mode operation.
func NewWishlistHandler(store wishlistStoreIface, manifest manifestLookupIface, prefs prefsStoreIface, xur xurInventoryIface) *WishlistHandler {
	return &WishlistHandler{store: store, manifest: manifest, prefs: prefs, xur: xur}
}

// wishlistResponse is the JSON shape returned to clients.
type wishlistResponse struct {
	ID            string   `json:"id"`
	ItemHash      uint32   `json:"itemHash"`
	Name          string   `json:"name"`
	ItemType      string   `json:"itemType"`
	Rarity        string   `json:"rarity"`
	Icon          string   `json:"icon"`
	Priority      string   `json:"priority"`
	Notes         string   `json:"notes"`
	Sources       []string `json:"sources"`
	AvailableNow  bool     `json:"availableNow"`
	AvailableFrom string   `json:"availableFrom,omitempty"`
	DateAdded     string   `json:"dateAdded"`
}

// GetWishlist handles GET /api/wishlist
func (h *WishlistHandler) GetWishlist(c *gin.Context) {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	membershipID := c.GetString("membership_id")
	userID, err := h.store.GetUserID(c.Request.Context(), membershipID)
	if err != nil {
		log.Printf("GetWishlist: GetUserID(%s): %v", membershipID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	items, err := h.store.List(c.Request.Context(), userID)
	if err != nil {
		log.Printf("GetWishlist: List(%d): %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, h.enrichItems(c.Request.Context(), items))
}

// AddToWishlist handles POST /api/wishlist
func (h *WishlistHandler) AddToWishlist(c *gin.Context) {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
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
				log.Printf("AddToWishlist: manifest lookup error for hash %d: %v", body.ItemHash, err)
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
		log.Printf("AddToWishlist: GetUserID(%s): %v", membershipID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	item, err := h.store.Add(c.Request.Context(), userID, body.ItemHash, prio, body.Notes)
	if err != nil {
		if isDuplicate(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "item already in wishlist"})
			return
		}
		log.Printf("AddToWishlist: Add: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusCreated, h.enrichOne(c.Request.Context(), *item))
}

// UpdateWishlistItem handles PUT /api/wishlist/:id
func (h *WishlistHandler) UpdateWishlistItem(c *gin.Context) {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
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
		log.Printf("UpdateWishlistItem: GetUserID(%s): %v", membershipID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	item, err := h.store.Update(c.Request.Context(), userID, id, prio, body.Notes)
	if err != nil {
		if isNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wishlist item not found"})
			return
		}
		log.Printf("UpdateWishlistItem: Update: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, h.enrichOne(c.Request.Context(), *item))
}

// RemoveFromWishlist handles DELETE /api/wishlist/:id
func (h *WishlistHandler) RemoveFromWishlist(c *gin.Context) {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	membershipID := c.GetString("membership_id")
	userID, err := h.store.GetUserID(c.Request.Context(), membershipID)
	if err != nil {
		log.Printf("RemoveFromWishlist: GetUserID(%s): %v", membershipID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	found, err := h.store.Delete(c.Request.Context(), userID, id)
	if err != nil {
		log.Printf("RemoveFromWishlist: Delete: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "wishlist item not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetPreferences handles GET /api/preferences
func (h *WishlistHandler) GetPreferences(c *gin.Context) {
	if h.prefs == nil {
		// Return defaults when DB not configured
		c.JSON(http.StatusOK, gin.H{"cardStyle": "framed", "personalize": true})
		return
	}
	membershipID := c.GetString("membership_id")
	userID, err := h.getUserID(c.Request.Context(), membershipID)
	if err != nil {
		log.Printf("GetPreferences: getUserID(%s): %v", membershipID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	p, err := h.prefs.Get(c.Request.Context(), userID)
	if err != nil {
		log.Printf("GetPreferences: Get(%d): %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cardStyle": p.CardStyle, "personalize": p.Personalize})
}

// UpdatePreferences handles PUT /api/preferences
func (h *WishlistHandler) UpdatePreferences(c *gin.Context) {
	if h.prefs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	var body struct {
		CardStyle   string `json:"cardStyle"`
		Personalize *bool  `json:"personalize"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if body.CardStyle != "" && body.CardStyle != "framed" && body.CardStyle != "compact" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cardStyle must be 'framed' or 'compact'"})
		return
	}
	membershipID := c.GetString("membership_id")
	userID, err := h.getUserID(c.Request.Context(), membershipID)
	if err != nil {
		log.Printf("UpdatePreferences: getUserID(%s): %v", membershipID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	// Read current prefs to fill in defaults for missing fields
	current, err := h.prefs.Get(c.Request.Context(), userID)
	if err != nil {
		log.Printf("UpdatePreferences: Get(%d): %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	cardStyle := current.CardStyle
	if body.CardStyle != "" {
		cardStyle = body.CardStyle
	}
	personalize := current.Personalize
	if body.Personalize != nil {
		personalize = *body.Personalize
	}
	p, err := h.prefs.Upsert(c.Request.Context(), userID, cardStyle, personalize)
	if err != nil {
		log.Printf("UpdatePreferences: Upsert(%d): %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cardStyle": p.CardStyle, "personalize": p.Personalize})
}

// --- helpers ---

// getUserID resolves a membershipID to a DB user ID using the wishlist store.
func (h *WishlistHandler) getUserID(ctx context.Context, membershipID string) (int64, error) {
	if h.store != nil {
		return h.store.GetUserID(ctx, membershipID)
	}
	return 0, fmt.Errorf("no store available")
}

func (h *WishlistHandler) enrichItems(ctx context.Context, items []db.WishlistItem) []wishlistResponse {
	if len(items) == 0 {
		return []wishlistResponse{}
	}
	hashes := make([]uint32, len(items))
	for i, it := range items {
		hashes[i] = it.ItemHash
	}
	defs := map[uint32]*bungie.InventoryItemDefinition{}
	cols := map[uint32]*bungie.CollectibleDefinition{}
	if h.manifest != nil {
		if m, err := h.manifest.GetItemsByHashes(hashes); err == nil {
			defs = m
		}
		if cs, err := h.manifest.GetCollectiblesByItemHashes(hashes); err == nil {
			cols = cs
		}
	}
	xur := h.xurHashes(ctx)
	resp := make([]wishlistResponse, len(items))
	for i, it := range items {
		_, atXur := xur[it.ItemHash]
		resp[i] = buildResponse(it, defs[it.ItemHash], cols[it.ItemHash], atXur)
	}
	return resp
}

func (h *WishlistHandler) enrichOne(ctx context.Context, it db.WishlistItem) wishlistResponse {
	var def *bungie.InventoryItemDefinition
	var col *bungie.CollectibleDefinition
	if h.manifest != nil {
		if m, err := h.manifest.GetItemsByHashes([]uint32{it.ItemHash}); err == nil {
			def = m[it.ItemHash]
		}
		if cs, err := h.manifest.GetCollectiblesByItemHashes([]uint32{it.ItemHash}); err == nil {
			col = cs[it.ItemHash]
		}
	}
	_, atXur := h.xurHashes(ctx)[it.ItemHash]
	return buildResponse(it, def, col, atXur)
}

// xurHashes returns Xûr's current inventory hashes, or empty in degraded mode.
// Served from the weekly service's reset-aligned cache; the first call after a
// reset fetches public vendors inline (bounded by Bungie's response time).
func (h *WishlistHandler) xurHashes(ctx context.Context) map[uint32]struct{} {
	if h.xur == nil {
		return map[uint32]struct{}{}
	}
	return h.xur.XurItemHashes(ctx)
}

func buildResponse(it db.WishlistItem, def *bungie.InventoryItemDefinition, col *bungie.CollectibleDefinition, atXur bool) wishlistResponse {
	name, itemTypeStr, rarity, icon := "Unknown Item", "Item", "Common", ""
	sources := []string{}
	if def != nil {
		name = def.DisplayProperties.Name
		itemTypeStr = bungie.ItemTypeName(def.ItemType, def.ItemSubType)
		rarity = bungie.GetTierName(def.Inventory.TierType)
		icon = def.DisplayProperties.Icon
	}
	if col != nil && col.SourceString != "" {
		sources = append(sources, col.SourceString)
	}
	resp := wishlistResponse{
		ID:           strconv.FormatInt(it.ID, 10),
		ItemHash:     it.ItemHash,
		Name:         name,
		ItemType:     itemTypeStr,
		Rarity:       rarity,
		Icon:         icon,
		Priority:     priorityToStr[it.Priority],
		Notes:        it.Notes,
		Sources:      sources,
		AvailableNow: atXur,
		DateAdded:    it.CreatedAt.UTC().Format(time.RFC3339),
	}
	if atXur {
		resp.AvailableFrom = "Xûr"
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
