package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"guardian-tracker/api-service/services/manifest"
)

// itemProvider returns manifest-derived item detail. Satisfied by *items.Service.
type itemProvider interface {
	GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error)
	GetItem(itemHash uint32) (*manifest.ItemView, error)
	GetCatalysts(itemHash uint32) ([]manifest.WeaponCatalyst, error)
}

// ItemsHandler serves manifest-derived item detail (perk pools, item views).
type ItemsHandler struct {
	items itemProvider
}

func NewItemsHandler(p itemProvider) *ItemsHandler {
	return &ItemsHandler{items: p}
}

// GetPerks handles GET /api/items/:itemHash/perks.
// Requires jwtHelper.Middleware() on the route. Non-weapons / unknown hashes
// return 200 with empty perkColumns/catalysts arrays; manifest-warming returns
// 503. catalysts is populated only for exotics with a detected catalyst socket.
func (h *ItemsHandler) GetPerks(c *gin.Context) {
	hash64, err := strconv.ParseUint(c.Param("itemHash"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item hash"})
		return
	}
	itemHash := uint32(hash64)

	cols, err := h.items.GetWeaponPerks(itemHash)
	if err != nil {
		handleBungieError(c, err) // maps manifest.ErrNotReady → 503 MANIFEST_NOT_READY
		return
	}
	if cols == nil {
		cols = []manifest.PerkColumn{} // serialize as [] not null
	}

	cats, err := h.items.GetCatalysts(itemHash)
	if err != nil {
		handleBungieError(c, err)
		return
	}
	if cats == nil {
		cats = []manifest.WeaponCatalyst{} // serialize as [] not null
	}

	c.JSON(http.StatusOK, gin.H{
		"itemHash":    c.Param("itemHash"),
		"perkColumns": cols,
		"catalysts":   cats,
	})
}

// GetItem handles GET /api/items/:itemHash — a minimal manifest item view for
// deep-linked non-collectible items. Requires jwtHelper.Middleware() on the route.
func (h *ItemsHandler) GetItem(c *gin.Context) {
	hash64, err := strconv.ParseUint(c.Param("itemHash"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item hash"})
		return
	}
	view, err := h.items.GetItem(uint32(hash64))
	if err != nil {
		handleBungieError(c, err) // manifest.ErrNotReady → 503
		return
	}
	if view == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	c.JSON(http.StatusOK, view)
}
