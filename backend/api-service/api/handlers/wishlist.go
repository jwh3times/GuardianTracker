package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// WishlistHandler handles wishlist endpoints.
// All responses are stubs — persistence will be added when the DB layer lands.
type WishlistHandler struct{}

func NewWishlistHandler() *WishlistHandler { return &WishlistHandler{} }

// GetWishlist handles GET /api/wishlist
func (h *WishlistHandler) GetWishlist(c *gin.Context) {
	membershipID := c.GetString("membership_id")
	c.JSON(http.StatusOK, []gin.H{
		{"id": "1", "itemId": "1498876634", "itemName": "Fatebringer", "itemType": "Hand Cannon", "addedAt": "2025-08-02T22:30:00Z", "userId": membershipID},
		{"id": "2", "itemId": "3653573172", "itemName": "Vex Mythoclast", "itemType": "Fusion Rifle", "addedAt": "2025-08-01T15:20:00Z", "userId": membershipID},
	})
}

// AddToWishlist handles POST /api/wishlist
func (h *WishlistHandler) AddToWishlist(c *gin.Context) {
	membershipID := c.GetString("membership_id")
	itemID := c.PostForm("itemId")
	if itemID == "" || len(itemID) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Item added to wishlist successfully", "itemId": itemID, "userId": membershipID})
}

// RemoveFromWishlist handles DELETE /api/wishlist/:id
func (h *WishlistHandler) RemoveFromWishlist(c *gin.Context) {
	itemID := c.Param("id")
	if len(itemID) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Item removed from wishlist", "itemId": itemID, "userId": c.GetString("membership_id")})
}
