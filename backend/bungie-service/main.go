package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Setup Gin router
	if os.Getenv("GO_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"service":   "bungie-service",
			"timestamp": gin.H{"time": "now"},
		})
	})

	// API routes placeholder
	api := router.Group("/api")
	{
		api.GET("/collections/:membershipType/:membershipId", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"weapons": gin.H{
					"total":     500,
					"collected": 450,
					"missing":   []gin.H{},
				},
				"armor": gin.H{
					"total":     300,
					"collected": 280,
					"missing":   []gin.H{},
				},
				"exotics": gin.H{
					"total":     100,
					"collected": 85,
					"missing":   []gin.H{},
				},
			})
		})

		api.GET("/items/search", func(c *gin.Context) {
			c.JSON(200, []gin.H{})
		})

		api.GET("/weekly/recommendations", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"vendors":    []gin.H{},
				"activities": []gin.H{},
				"pursuits":   []gin.H{},
			})
		})
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Bungie service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
