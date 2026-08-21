package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/manifest"
)

// itemProvider returns manifest-derived item detail. Satisfied by *items.Service.
type itemProvider interface {
	GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error)
	GetCatalysts(itemHash uint32) ([]manifest.WeaponCatalyst, error)
	Lookup(ctx context.Context, hashes []uint32) (map[uint32]items.AcquisitionFacts, error)
}

// itemViewResponse is the wire shape of GET /api/items/:itemHash. It is
// unchanged apart from itemType, which now carries the canonical slot-specific
// value ("Helmet" rather than "Armor") that Collections has always used — the
// two disagreed about the same item until this handler started reading
// canonical facts.
type itemViewResponse struct {
	ItemHash    string `json:"itemHash"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	ItemType    string `json:"itemType"`
	TierType    int    `json:"tierType"`
	Rarity      string `json:"rarity"`
	Description string `json:"description"`
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
	itemHash := uint32(hash64)
	facts, err := h.items.Lookup(c.Request.Context(), []uint32{itemHash})
	if err != nil {
		handleBungieError(c, err) // manifest.ErrNotReady → 503
		return
	}
	// An unknown hash is absent from a successful lookup rather than an error,
	// which is exactly the 404 case — and is kept distinct from the failure
	// above, where the manifest could not answer at all.
	f, ok := facts[itemHash]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	c.JSON(http.StatusOK, itemViewResponse{
		ItemHash:    c.Param("itemHash"),
		Name:        f.Name,
		Icon:        f.Icon,
		ItemType:    f.ItemType,
		TierType:    f.TierType,
		Rarity:      f.Rarity,
		Description: f.Description,
	})
}
