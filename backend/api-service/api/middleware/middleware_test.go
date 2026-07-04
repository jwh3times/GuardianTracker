package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPerIPRateLimit_Blocks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", PerIPRateLimit(1, 2), func(c *gin.Context) { c.Status(200) })

	codes := []int{}
	for i := 0; i < 4; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/x", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		r.ServeHTTP(w, req)
		codes = append(codes, w.Code)
	}
	// burst 2 → first two pass, then 429s.
	if codes[0] != 200 || codes[1] != 200 || codes[2] != 429 || codes[3] != 429 {
		t.Fatalf("codes = %v, want [200 200 429 429]", codes)
	}
	// A different IP has its own bucket.
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("second IP got %d, want 200", w.Code)
	}
}

func TestMaxBodyBytes_Rejects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodyBytes(64))
	r.POST("/x", func(c *gin.Context) {
		var v map[string]any
		if err := c.ShouldBindJSON(&v); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "body too large or invalid"})
			return
		}
		c.Status(200)
	})

	small := httptest.NewRecorder()
	r.ServeHTTP(small, httptest.NewRequest("POST", "/x", strings.NewReader(`{"a":1}`)))
	if small.Code != 200 {
		t.Fatalf("small body got %d, want 200", small.Code)
	}

	// VALID JSON that exceeds the 64-byte cap — an invalid payload would 400
	// from JSON syntax alone and prove nothing about MaxBytesReader.
	oversized := `{"a":"` + strings.Repeat("a", 128) + `"}`
	big := httptest.NewRecorder()
	r.ServeHTTP(big, httptest.NewRequest("POST", "/x", strings.NewReader(oversized)))
	if big.Code != 400 {
		t.Fatalf("big body got %d, want 400", big.Code)
	}
}
