package handlers

import (
	"net/http"
	"strconv"

	"guardian-tracker/api-service/services/search"

	"github.com/gin-gonic/gin"
)

// SearchHandler handles GET /api/items/search
type SearchHandler struct {
	svc *search.Service
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(svc *search.Service) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// Search handles GET /api/items/search?q=<term>&limit=<n>
// Returns up to 50 matching items from the in-memory search index.
// Requires at least 2 characters in q to avoid returning the entire index.
func (h *SearchHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if len(q) < 2 {
		c.JSON(http.StatusOK, []search.Entry{})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 50 {
		limit = 50
	}

	if !h.svc.IsReady() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Search index not ready — manifest may still be downloading",
			"code":  "SEARCH_NOT_READY",
		})
		return
	}

	results := h.svc.Search(q, limit)
	if results == nil {
		results = []search.Entry{}
	}
	c.JSON(http.StatusOK, results)
}
