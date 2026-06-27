package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"guardian-tracker/api-service/services/manifest"
)

// weaponPerksProvider returns a weapon's possible-perk columns. Satisfied by *items.Service.
type weaponPerksProvider interface {
	GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error)
}

// ItemsHandler serves manifest-derived item detail (perk pools).
type ItemsHandler struct {
	perks weaponPerksProvider
}

func NewItemsHandler(p weaponPerksProvider) *ItemsHandler {
	return &ItemsHandler{perks: p}
}

// GetPerks handles GET /api/items/:itemHash/perks.
// Requires jwtHelper.Middleware() on the route. Non-weapons / unknown hashes
// return 200 with an empty perkColumns array; manifest-warming returns 503.
func (h *ItemsHandler) GetPerks(c *gin.Context) {
	hash64, err := strconv.ParseUint(c.Param("itemHash"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item hash"})
		return
	}

	cols, err := h.perks.GetWeaponPerks(uint32(hash64))
	if err != nil {
		handleBungieError(c, err) // maps manifest.ErrNotReady → 503 MANIFEST_NOT_READY
		return
	}
	if cols == nil {
		cols = []manifest.PerkColumn{} // serialize as [] not null
	}

	c.JSON(http.StatusOK, gin.H{
		"itemHash":    c.Param("itemHash"),
		"perkColumns": cols,
	})
}
